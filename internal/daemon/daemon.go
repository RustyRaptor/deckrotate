package daemon

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sync/atomic"
	"time"

	"github.com/godbus/dbus/v5"

	"github.com/RustyRaptor/deckrotate/internal/config"
	"github.com/RustyRaptor/deckrotate/internal/core"
	"github.com/RustyRaptor/deckrotate/internal/display"
	"github.com/RustyRaptor/deckrotate/internal/imu"
	"github.com/RustyRaptor/deckrotate/internal/overlay"
	"github.com/RustyRaptor/deckrotate/internal/service"
)

type Options struct {
	UIDir   string // holds overlay.qml
	QMLBin  string
	SysRoot string
}

func Run(ctx context.Context, o Options) error {
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		return fmt.Errorf("session bus: %w", err)
	}
	defer conn.Close()

	var svc atomic.Pointer[service.Service]
	var settle atomic.Int64
	var ctrl *core.Controller
	prompt := overlay.NewProcessPrompt(
		func(ctx context.Context, p overlay.Placement) *exec.Cmd {
			args := append([]string{filepath.Join(o.UIDir, "overlay.qml"), "--"}, p.Args()...)
			cmd := exec.CommandContext(ctx, o.QMLBin, args...)
			cmd.Stderr = os.Stderr
			return cmd
		},
		func() {
			c, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			if err := ctrl.Accept(c); err != nil {
				log.Printf("accept: %v", err)
			}
		},
	)
	ctrl = core.New(core.Deps{
		Screen: display.NewKScreen(display.ExecRunner),
		Prompt: prompt,
		Store:  config.NewFileStore(config.DefaultPath()),
		OnChange: func(st core.State) {
			settle.Store(int64(st.Settle))
			if s := svc.Load(); s != nil {
				s.Publish(st)
			}
		},
	})
	if err := ctrl.Init(ctx); err != nil {
		return fmt.Errorf("init: %w", err)
	}
	s, err := service.Export(conn, ctrl)
	if err != nil {
		return err
	}
	svc.Store(s)
	s.Publish(ctrl.State())
	st := ctrl.State()
	log.Printf("deckrotate running: mode=%v locked=%v screen=%v", st.Mode, st.Locked, st.Screen)

	go readIMU(ctx, o.SysRoot, NewTracker(ctrl, func() time.Duration { return time.Duration(settle.Load()) }))
	<-ctx.Done()
	prompt.Hide()
	return nil
}

// readIMU keeps a reader running, reopening the device after suspend/resume
// or any other read failure.
func readIMU(ctx context.Context, sysRoot string, tr *Tracker) {
	const retry = 2 * time.Second
	for ctx.Err() == nil {
		if err := readOnce(ctx, sysRoot, tr); err != nil && ctx.Err() == nil {
			log.Printf("imu: %v (retrying in %v)", err, retry)
		}
		tr.Lost()
		select {
		case <-ctx.Done():
		case <-time.After(retry):
		}
	}
}

func readOnce(ctx context.Context, sysRoot string, tr *Tracker) error {
	path, err := imu.Find(sysRoot)
	if err != nil {
		return err
	}
	dev, err := imu.OpenHIDRaw(path)
	if err != nil {
		return err
	}
	stop := context.AfterFunc(ctx, func() { dev.Close() })
	defer stop()
	defer dev.Close()
	// ~250 reports/s: re-enable after half a second without IMU data.
	return imu.NewReader(dev, 125).Run(func(s imu.Sample) { tr.Sample(time.Now(), s) })
}

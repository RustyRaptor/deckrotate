// Command deckrotate rotates the Steam Deck screen in KDE Plasma desktop mode.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"math"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/godbus/dbus/v5"

	"github.com/RustyRaptor/deckrotate/internal/core"
	"github.com/RustyRaptor/deckrotate/internal/daemon"
	"github.com/RustyRaptor/deckrotate/internal/orient"
	"github.com/RustyRaptor/deckrotate/internal/service"
)

const usage = `usage: deckrotate <command>

  daemon [flags]          run the orientation daemon (normally via systemd)
  status [--json]         show mode, lock and orientations
  lock | unlock | toggle-lock
  mode auto|manual|toggle
  rotate cw|ccw|flip      turn the screen from where it is now
  rotate landscape|portrait-cw|landscape-flipped|portrait-ccw
  settle <duration>       how long the Deck must hold still before it counts (100ms-3s)
  prompt-timeout <dur>    how long the manual-mode button stays up (2s-30s)
  accept                  take the rotation the manual-mode button offers
  dismiss                 hide the manual-mode button
`

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "deckrotate:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 || args[0] == "-h" || args[0] == "--help" || args[0] == "help" {
		fmt.Print(usage)
		return nil
	}
	switch args[0] {
	case "daemon":
		return runDaemon(args[1:])
	case "status":
		return runStatus(args[1:])
	}
	method, params, err := parseCommand(args)
	if err != nil {
		return err
	}
	cl, err := dial()
	if err != nil {
		return err
	}
	return cl.Call(method, params...)
}

func parseCommand(args []string) (string, []any, error) {
	if len(args) == 0 {
		return "", nil, errors.New("no command (try --help)")
	}
	arg := func() (string, error) {
		if len(args) != 2 {
			return "", fmt.Errorf("%s takes exactly one argument", args[0])
		}
		return args[1], nil
	}
	noArgs := func(method string, params ...any) (string, []any, error) {
		if len(args) != 1 {
			return "", nil, fmt.Errorf("%s takes no arguments", args[0])
		}
		return method, params, nil
	}
	switch args[0] {
	case "lock":
		return noArgs("SetLocked", true)
	case "unlock":
		return noArgs("SetLocked", false)
	case "toggle-lock":
		return noArgs("ToggleLock")
	case "accept":
		return noArgs("Accept")
	case "dismiss":
		return noArgs("Dismiss")
	case "mode":
		m, err := arg()
		if err != nil {
			return "", nil, err
		}
		if m == "toggle" {
			return "ToggleMode", nil, nil
		}
		if _, err := core.ParseMode(m); err != nil {
			return "", nil, err
		}
		return "SetMode", []any{m}, nil
	case "settle", "prompt-timeout":
		v, err := arg()
		if err != nil {
			return "", nil, err
		}
		d, err := time.ParseDuration(v)
		if err != nil {
			return "", nil, err
		}
		ms := d.Milliseconds()
		if ms <= 0 || ms > math.MaxInt32 {
			return "", nil, fmt.Errorf("%s: %v is out of range", args[0], d)
		}
		method := map[string]string{"settle": "SetSettleMs", "prompt-timeout": "SetPromptTimeoutMs"}[args[0]]
		return method, []any{int32(ms)}, nil
	case "rotate":
		r, err := arg()
		if err != nil {
			return "", nil, err
		}
		if turns, ok := map[string]int32{"cw": 1, "ccw": -1, "flip": 2}[r]; ok {
			return "RotateBy", []any{turns}, nil
		}
		if _, err := orient.Parse(r); err != nil {
			return "", nil, err
		}
		return "RotateTo", []any{r}, nil
	}
	return "", nil, fmt.Errorf("unknown command %q (try --help)", args[0])
}

func dial() (*service.Client, error) {
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		return nil, err
	}
	return service.NewClient(conn), nil
}

func formatStatus(s service.Status) string {
	yn := map[bool]string{true: "yes", false: "no"}
	imu := map[bool]string{true: "ok", false: "unavailable"}
	sug := s.Suggested
	if sug == "" {
		sug = "-"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "mode:      %s\nlocked:    %s\ndevice:    %s\nscreen:    %s\nsuggested: %s\nimu:       %s\n",
		s.Mode, yn[s.Locked], s.Device, s.Screen, sug, imu[s.IMU])
	ms := func(v int32) time.Duration { return time.Duration(v) * time.Millisecond }
	fmt.Fprintf(&b, "settle:    %v\nbutton:    %v\n", ms(s.SettleMs), ms(s.PromptTimeoutMs))
	return b.String()
}

func runStatus(args []string) error {
	fs := flag.NewFlagSet("status", flag.ContinueOnError)
	asJSON := fs.Bool("json", false, "print JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	cl, err := dial()
	if err != nil {
		return err
	}
	s, err := cl.Status()
	if err != nil {
		return err
	}
	if *asJSON {
		return json.NewEncoder(os.Stdout).Encode(s)
	}
	fmt.Print(formatStatus(s))
	return nil
}

func defaultUIDir() string {
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Join(filepath.Dir(exe), "..", "share", "deckrotate", "ui")
		if _, err := os.Stat(filepath.Join(dir, "overlay.qml")); err == nil {
			return dir
		}
	}
	return filepath.Join(os.Getenv("HOME"), ".local", "share", "deckrotate", "ui")
}

func runDaemon(args []string) error {
	fs := flag.NewFlagSet("daemon", flag.ContinueOnError)
	o := daemon.Options{SysRoot: "/sys"}
	fs.StringVar(&o.UIDir, "ui-dir", defaultUIDir(), "directory containing overlay.qml")
	fs.StringVar(&o.QMLBin, "qml", "/usr/lib/qt6/bin/qml", "QML runtime used for the overlay")
	if err := fs.Parse(args); err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	return daemon.Run(ctx, o)
}

// Command testservice serves a real controller with an in-memory screen on
// the session bus, for the plasmoid's D-Bus integration tests.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/godbus/dbus/v5"

	"github.com/RustyRaptor/deckrotate/internal/core"
	"github.com/RustyRaptor/deckrotate/internal/orient"
	"github.com/RustyRaptor/deckrotate/internal/service"
)

type memScreen struct{ o orient.Orientation }

func (m *memScreen) Current(context.Context) (orient.Orientation, error) { return m.o, nil }
func (m *memScreen) Apply(_ context.Context, o orient.Orientation) error {
	m.o = o
	return nil
}

type noPrompt struct{}

func (noPrompt) Show(_, _ orient.Orientation) {}
func (noPrompt) Hide()                        {}

type memStore struct{ s core.Settings }

func (m *memStore) Load() (core.Settings, error) { return m.s, nil }
func (m *memStore) Save(s core.Settings) error   { m.s = s; return nil }

func main() {
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		log.Fatal(err)
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	var svc *service.Service
	ctrl := core.New(core.Deps{
		Screen: &memScreen{o: orient.Landscape},
		Prompt: noPrompt{},
		Store:  &memStore{},
		OnChange: func(st core.State) {
			if svc != nil {
				svc.Publish(st)
			}
		},
	})
	if err := ctrl.Init(ctx); err != nil {
		log.Fatal(err)
	}
	ctrl.SetIMUAvailable(true)
	ctrl.SetLocked(ctx, true) // don't let the device report rotate the screen
	ctrl.DeviceTurned(ctx, orient.Landscape)
	ctrl.SetLocked(ctx, false)
	if svc, err = service.Export(conn, ctrl); err != nil {
		log.Fatal(err)
	}
	fmt.Fprintln(os.Stdout, "ready")
	<-ctx.Done()
}

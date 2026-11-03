// Package service exposes the controller on the D-Bus session bus.
package service

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5/introspect"
	"github.com/godbus/dbus/v5/prop"

	"github.com/RustyRaptor/deckrotate/internal/core"
	"github.com/RustyRaptor/deckrotate/internal/orient"
)

const (
	BusName    = "io.github.RustyRaptor.DeckRotate"
	Interface  = BusName
	ObjectPath = dbus.ObjectPath("/io/github/RustyRaptor/DeckRotate")

	callTimeout = 10 * time.Second
)

var ErrAlreadyRunning = errors.New("service: another deckrotate daemon owns the bus name")

type Controller interface {
	State() core.State
	SetMode(context.Context, core.Mode) error
	ToggleMode(context.Context) error
	SetLocked(context.Context, bool) error
	ToggleLock(context.Context) error
	RotateTo(context.Context, orient.Orientation) error
	RotateBy(context.Context, int) error
	Accept(context.Context) error
	Dismiss()
}

// Status is the property set as plain values.
type Status struct {
	Mode      string
	Locked    bool
	Device    string
	Screen    string
	Suggested string // empty when nothing is being offered
	IMU       bool
}

func statusOf(s core.State) Status {
	sug := ""
	if s.Suggested != orient.Unknown {
		sug = s.Suggested.String()
	}
	return Status{
		Mode: s.Mode.String(), Locked: s.Locked,
		Device: s.Device.String(), Screen: s.Screen.String(),
		Suggested: sug, IMU: s.IMU,
	}
}

func (s Status) props() map[string]any {
	return map[string]any{
		"Mode":                 s.Mode,
		"Locked":               s.Locked,
		"DeviceOrientation":    s.Device,
		"ScreenOrientation":    s.Screen,
		"SuggestedOrientation": s.Suggested,
		"ImuAvailable":         s.IMU,
	}
}

type Service struct {
	props *prop.Properties
	mu    sync.Mutex
	last  Status
}

// object holds the exported methods; godbus exports every method of it.
type object struct{ c Controller }

func fail(err error) *dbus.Error {
	if err == nil {
		return nil
	}
	return dbus.MakeFailedError(err)
}

func ctx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), callTimeout)
}

func (o object) SetMode(m string) *dbus.Error {
	mode, err := core.ParseMode(m)
	if err != nil {
		return fail(err)
	}
	c, cancel := ctx()
	defer cancel()
	return fail(o.c.SetMode(c, mode))
}

func (o object) ToggleMode() *dbus.Error {
	c, cancel := ctx()
	defer cancel()
	return fail(o.c.ToggleMode(c))
}

func (o object) SetLocked(locked bool) *dbus.Error {
	c, cancel := ctx()
	defer cancel()
	return fail(o.c.SetLocked(c, locked))
}

func (o object) ToggleLock() *dbus.Error {
	c, cancel := ctx()
	defer cancel()
	return fail(o.c.ToggleLock(c))
}

func (o object) RotateTo(name string) *dbus.Error {
	or, err := orient.Parse(name)
	if err != nil {
		return fail(err)
	}
	c, cancel := ctx()
	defer cancel()
	return fail(o.c.RotateTo(c, or))
}

func (o object) RotateBy(turns int32) *dbus.Error {
	c, cancel := ctx()
	defer cancel()
	return fail(o.c.RotateBy(c, int(turns)))
}

func (o object) Accept() *dbus.Error {
	c, cancel := ctx()
	defer cancel()
	return fail(o.c.Accept(c))
}

func (o object) Dismiss() *dbus.Error {
	o.c.Dismiss()
	return nil
}

var methods = []introspect.Method{
	{Name: "SetMode", Args: []introspect.Arg{{Name: "mode", Type: "s", Direction: "in"}}},
	{Name: "ToggleMode"},
	{Name: "SetLocked", Args: []introspect.Arg{{Name: "locked", Type: "b", Direction: "in"}}},
	{Name: "ToggleLock"},
	{Name: "RotateTo", Args: []introspect.Arg{{Name: "orientation", Type: "s", Direction: "in"}}},
	{Name: "RotateBy", Args: []introspect.Arg{{Name: "quarterTurnsClockwise", Type: "i", Direction: "in"}}},
	{Name: "Accept"},
	{Name: "Dismiss"},
}

// Export publishes c on conn and claims the well-known bus name.
func Export(conn *dbus.Conn, c Controller) (*Service, error) {
	if err := conn.Export(object{c}, ObjectPath, Interface); err != nil {
		return nil, err
	}
	st := statusOf(c.State())
	spec := map[string]*prop.Prop{}
	for name, v := range st.props() {
		spec[name] = &prop.Prop{Value: v, Emit: prop.EmitTrue}
	}
	props, err := prop.Export(conn, ObjectPath, prop.Map{Interface: spec})
	if err != nil {
		return nil, err
	}
	node := &introspect.Node{
		Name: string(ObjectPath),
		Interfaces: []introspect.Interface{
			introspect.IntrospectData,
			prop.IntrospectData,
			{Name: Interface, Methods: methods, Properties: props.Introspection(Interface)},
		},
	}
	if err := conn.Export(introspect.NewIntrospectable(node), ObjectPath, "org.freedesktop.DBus.Introspectable"); err != nil {
		return nil, err
	}
	reply, err := conn.RequestName(BusName, dbus.NameFlagDoNotQueue)
	if err != nil {
		return nil, err
	}
	if reply != dbus.RequestNameReplyPrimaryOwner {
		return nil, ErrAlreadyRunning
	}
	return &Service{props: props, last: st}, nil
}

// Publish updates the properties, emitting PropertiesChanged for each one
// whose value actually changed.
func (s *Service) Publish(st core.State) {
	s.mu.Lock()
	defer s.mu.Unlock()
	next := statusOf(st)
	prev := s.last.props()
	for name, v := range next.props() {
		if prev[name] != v {
			s.props.SetMust(Interface, name, v)
		}
	}
	s.last = next
}

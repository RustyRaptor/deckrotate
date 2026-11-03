package service

import (
	"fmt"

	"github.com/godbus/dbus/v5"
)

// Client talks to a running daemon.
type Client struct{ conn *dbus.Conn }

func NewClient(conn *dbus.Conn) *Client { return &Client{conn: conn} }

func (c *Client) obj() dbus.BusObject { return c.conn.Object(BusName, ObjectPath) }

func (c *Client) Call(method string, args ...any) error {
	return c.obj().Call(Interface+"."+method, 0, args...).Err
}

func (c *Client) Status() (Status, error) {
	var all map[string]dbus.Variant
	if err := c.obj().Call("org.freedesktop.DBus.Properties.GetAll", 0, Interface).Store(&all); err != nil {
		return Status{}, err
	}
	var s Status
	for name, dst := range map[string]any{
		"Mode": &s.Mode, "Locked": &s.Locked,
		"DeviceOrientation": &s.Device, "ScreenOrientation": &s.Screen,
		"SuggestedOrientation": &s.Suggested, "ImuAvailable": &s.IMU,
		"SettleMs": &s.SettleMs, "PromptTimeoutMs": &s.PromptTimeoutMs,
	} {
		v, ok := all[name]
		if !ok {
			return s, fmt.Errorf("daemon did not report %s", name)
		}
		if err := v.Store(dst); err != nil {
			return s, fmt.Errorf("%s: %w", name, err)
		}
	}
	return s, nil
}

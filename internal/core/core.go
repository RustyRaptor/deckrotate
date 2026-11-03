// Package core decides when the screen rotates: auto, manual (prompted) or locked.
package core

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/RustyRaptor/deckrotate/internal/orient"
)

type Mode int

const (
	Auto Mode = iota
	Manual
)

func (m Mode) String() string {
	if m == Manual {
		return "manual"
	}
	return "auto"
}

func ParseMode(s string) (Mode, error) {
	switch s {
	case "auto":
		return Auto, nil
	case "manual":
		return Manual, nil
	}
	return Auto, fmt.Errorf("unknown mode %q (want auto or manual)", s)
}

func (m Mode) MarshalText() ([]byte, error) { return []byte(m.String()), nil }

func (m *Mode) UnmarshalText(b []byte) (err error) {
	*m, err = ParseMode(string(b))
	return err
}

// Settings are the user choices that survive restarts.
type Settings struct {
	Mode   Mode `json:"mode"`
	Locked bool `json:"locked"`
}

type State struct {
	Mode      Mode
	Locked    bool
	Device    orient.Orientation // physical orientation, Unknown until first reading
	Screen    orient.Orientation // what the panel is rotated for
	Suggested orient.Orientation // manual-mode prompt target, Unknown when none
	IMU       bool               // accelerometer is delivering data
}

type Screen interface {
	Current(ctx context.Context) (orient.Orientation, error)
	Apply(ctx context.Context, o orient.Orientation) error
}

// Prompt shows the manual-mode "rotate to target" button. Implementations
// must not block.
type Prompt interface {
	Show(target, current orient.Orientation)
	Hide()
}

type Store interface {
	Load() (Settings, error)
	Save(Settings) error
}

type Timer interface{ Stop() bool }

type Deps struct {
	Screen        Screen
	Prompt        Prompt
	Store         Store
	AfterFunc     func(time.Duration, func()) Timer
	PromptTimeout time.Duration
	OnChange      func(State)
}

var ErrNothingToAccept = errors.New("no rotation is being offered")

type Controller struct {
	d     Deps
	mu    sync.Mutex
	st    State
	timer Timer
	gen   int // bumps on every prompt change so stale timeouts are ignored
}

func New(d Deps) *Controller {
	if d.AfterFunc == nil {
		d.AfterFunc = func(t time.Duration, f func()) Timer { return time.AfterFunc(t, f) }
	}
	if d.OnChange == nil {
		d.OnChange = func(State) {}
	}
	return &Controller{d: d, st: State{Device: orient.Unknown, Screen: orient.Unknown, Suggested: orient.Unknown}}
}

func (c *Controller) Init(ctx context.Context) error {
	s, err := c.d.Store.Load()
	if err != nil {
		return err
	}
	return c.update(func() error {
		c.st.Mode, c.st.Locked = s.Mode, s.Locked
		return c.refreshScreen(ctx)
	})
}

func (c *Controller) State() State {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.st
}

// update runs f under the lock, then reports the resulting state.
func (c *Controller) update(f func() error) error {
	c.mu.Lock()
	err := f()
	st := c.st
	c.mu.Unlock()
	c.d.OnChange(st)
	return err
}

func (c *Controller) refreshScreen(ctx context.Context) error {
	o, err := c.d.Screen.Current(ctx)
	if err != nil {
		return err
	}
	c.st.Screen = o
	return nil
}

func (c *Controller) apply(ctx context.Context, o orient.Orientation) error {
	if err := c.d.Screen.Apply(ctx, o); err != nil {
		return err
	}
	c.st.Screen = o
	return nil
}

func (c *Controller) save() error {
	return c.d.Store.Save(Settings{Mode: c.st.Mode, Locked: c.st.Locked})
}

func (c *Controller) withdraw() {
	if c.st.Suggested == orient.Unknown {
		return
	}
	c.gen++
	if c.timer != nil {
		c.timer.Stop()
		c.timer = nil
	}
	c.st.Suggested = orient.Unknown
	c.d.Prompt.Hide()
}

func (c *Controller) offer(o orient.Orientation) {
	if c.st.Suggested == o {
		return
	}
	if c.timer != nil {
		c.timer.Stop()
	}
	c.gen++
	gen := c.gen
	c.st.Suggested = o
	c.d.Prompt.Show(o, c.st.Screen)
	c.timer = c.d.AfterFunc(c.d.PromptTimeout, func() {
		c.update(func() error {
			if c.gen == gen {
				c.withdraw()
			}
			return nil
		})
	})
}

// follow reacts to the current device orientation according to mode and lock.
func (c *Controller) follow(ctx context.Context) error {
	dev := c.st.Device
	if c.st.Locked || dev == orient.Unknown || dev == c.st.Screen {
		c.withdraw()
		return nil
	}
	if c.st.Mode == Auto {
		c.withdraw()
		return c.apply(ctx, dev)
	}
	c.offer(dev)
	return nil
}

// DeviceTurned is called when the debounced physical orientation changes.
func (c *Controller) DeviceTurned(ctx context.Context, o orient.Orientation) error {
	return c.update(func() error {
		c.st.Device = o
		if err := c.refreshScreen(ctx); err != nil {
			return err
		}
		return c.follow(ctx)
	})
}

func (c *Controller) SetIMUAvailable(ok bool) {
	c.update(func() error { c.st.IMU = ok; return nil })
}

func (c *Controller) SetMode(ctx context.Context, m Mode) error {
	return c.update(func() error {
		c.st.Mode = m
		if err := c.save(); err != nil {
			return err
		}
		return c.follow(ctx)
	})
}

func (c *Controller) ToggleMode(ctx context.Context) error {
	m := Auto
	if c.State().Mode == Auto {
		m = Manual
	}
	return c.SetMode(ctx, m)
}

func (c *Controller) SetLocked(ctx context.Context, locked bool) error {
	return c.update(func() error {
		c.st.Locked = locked
		if err := c.save(); err != nil {
			return err
		}
		return c.follow(ctx)
	})
}

func (c *Controller) ToggleLock(ctx context.Context) error {
	return c.SetLocked(ctx, !c.State().Locked)
}

// RotateTo is an explicit user request; it works regardless of mode and lock.
func (c *Controller) RotateTo(ctx context.Context, o orient.Orientation) error {
	return c.update(func() error {
		c.withdraw()
		return c.apply(ctx, o)
	})
}

// RotateBy turns the screen by clockwise quarter turns from where it is now.
func (c *Controller) RotateBy(ctx context.Context, turns int) error {
	return c.update(func() error {
		if err := c.refreshScreen(ctx); err != nil {
			return err
		}
		from := c.st.Screen
		if from == orient.Unknown {
			from = orient.Landscape
		}
		c.withdraw()
		return c.apply(ctx, from.Rotate(turns))
	})
}

// Accept takes the rotation the manual-mode prompt is offering.
func (c *Controller) Accept(ctx context.Context) error {
	return c.update(func() error {
		o := c.st.Suggested
		if o == orient.Unknown {
			return ErrNothingToAccept
		}
		c.withdraw()
		return c.apply(ctx, o)
	})
}

func (c *Controller) Dismiss() {
	c.update(func() error { c.withdraw(); return nil })
}

package core

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/RustyRaptor/deckrotate/internal/orient"
)

type fakeScreen struct {
	cur     orient.Orientation
	applied []orient.Orientation
	err     error
}

func (s *fakeScreen) Current(context.Context) (orient.Orientation, error) { return s.cur, nil }
func (s *fakeScreen) Apply(_ context.Context, o orient.Orientation) error {
	if s.err != nil {
		return s.err
	}
	s.applied = append(s.applied, o)
	s.cur = o
	return nil
}

type promptCall struct {
	show            bool
	target, current orient.Orientation
}

type fakePrompt struct{ calls []promptCall }

func (p *fakePrompt) Show(target, current orient.Orientation) {
	p.calls = append(p.calls, promptCall{true, target, current})
}
func (p *fakePrompt) Hide() { p.calls = append(p.calls, promptCall{show: false}) }

func (p *fakePrompt) visible() bool {
	return len(p.calls) > 0 && p.calls[len(p.calls)-1].show
}

type fakeStore struct {
	s     Settings
	saved []Settings
}

func (f *fakeStore) Load() (Settings, error) { return f.s, nil }
func (f *fakeStore) Save(s Settings) error {
	f.saved = append(f.saved, s)
	f.s = s
	return nil
}

type fakeTimer struct {
	d       time.Duration
	f       func()
	stopped bool
}

func (t *fakeTimer) Stop() bool { t.stopped = true; return true }

type fakeClock struct{ timers []*fakeTimer }

func (c *fakeClock) AfterFunc(d time.Duration, f func()) Timer {
	t := &fakeTimer{d: d, f: f}
	c.timers = append(c.timers, t)
	return t
}

func (c *fakeClock) fireLast() {
	t := c.timers[len(c.timers)-1]
	if !t.stopped {
		t.f()
	}
}

type rig struct {
	c      *Controller
	screen *fakeScreen
	prompt *fakePrompt
	store  *fakeStore
	clock  *fakeClock
	states []State
}

func newRig(t *testing.T, s Settings) *rig {
	t.Helper()
	r := &rig{
		screen: &fakeScreen{cur: orient.Landscape},
		prompt: &fakePrompt{},
		store:  &fakeStore{s: s},
		clock:  &fakeClock{},
	}
	r.c = New(Deps{
		Screen:        r.screen,
		Prompt:        r.prompt,
		Store:         r.store,
		AfterFunc:     r.clock.AfterFunc,
		PromptTimeout: 5 * time.Second,
		OnChange:      func(s State) { r.states = append(r.states, s) },
	})
	if err := r.c.Init(context.Background()); err != nil {
		t.Fatal(err)
	}
	return r
}

var ctx = context.Background()

func TestInitLoadsSettingsAndScreen(t *testing.T) {
	r := newRig(t, Settings{Mode: Manual, Locked: true})
	got := r.c.State()
	want := State{Mode: Manual, Locked: true, Device: orient.Unknown, Screen: orient.Landscape, Suggested: orient.Unknown}
	if got != want {
		t.Errorf("State = %+v, want %+v", got, want)
	}
}

func TestAutoRotatesImmediately(t *testing.T) {
	r := newRig(t, Settings{Mode: Auto})
	r.c.DeviceTurned(ctx, orient.PortraitCW)
	if !reflect.DeepEqual(r.screen.applied, []orient.Orientation{orient.PortraitCW}) {
		t.Errorf("applied = %v", r.screen.applied)
	}
	if s := r.c.State(); s.Screen != orient.PortraitCW || s.Device != orient.PortraitCW {
		t.Errorf("state = %+v", s)
	}
	if len(r.prompt.calls) != 0 {
		t.Errorf("auto mode must not prompt: %v", r.prompt.calls)
	}
}

func TestAutoNoOpWhenAlreadyMatching(t *testing.T) {
	r := newRig(t, Settings{Mode: Auto})
	r.c.DeviceTurned(ctx, orient.Landscape)
	if len(r.screen.applied) != 0 {
		t.Errorf("applied = %v", r.screen.applied)
	}
}

func TestLockedIgnoresTurns(t *testing.T) {
	for _, m := range []Mode{Auto, Manual} {
		r := newRig(t, Settings{Mode: m, Locked: true})
		r.c.DeviceTurned(ctx, orient.PortraitCCW)
		if len(r.screen.applied) != 0 || len(r.prompt.calls) != 0 {
			t.Errorf("%v: locked but applied=%v prompt=%v", m, r.screen.applied, r.prompt.calls)
		}
		if r.c.State().Device != orient.PortraitCCW {
			t.Errorf("%v: device orientation should still be tracked", m)
		}
	}
}

func TestManualShowsPromptThenAccept(t *testing.T) {
	r := newRig(t, Settings{Mode: Manual})
	r.c.DeviceTurned(ctx, orient.PortraitCCW)
	if len(r.screen.applied) != 0 {
		t.Fatalf("manual mode applied %v", r.screen.applied)
	}
	want := []promptCall{{true, orient.PortraitCCW, orient.Landscape}}
	if !reflect.DeepEqual(r.prompt.calls, want) {
		t.Fatalf("prompt = %v, want %v", r.prompt.calls, want)
	}
	if r.c.State().Suggested != orient.PortraitCCW {
		t.Errorf("Suggested = %v", r.c.State().Suggested)
	}

	if err := r.c.Accept(ctx); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(r.screen.applied, []orient.Orientation{orient.PortraitCCW}) {
		t.Errorf("applied = %v", r.screen.applied)
	}
	if r.prompt.visible() || r.c.State().Suggested != orient.Unknown {
		t.Errorf("prompt should be gone after accept: %+v", r.c.State())
	}
	if !r.clock.timers[0].stopped {
		t.Error("prompt timeout should be cancelled")
	}
}

func TestManualPromptTimesOut(t *testing.T) {
	r := newRig(t, Settings{Mode: Manual})
	r.c.DeviceTurned(ctx, orient.PortraitCW)
	if len(r.clock.timers) != 1 || r.clock.timers[0].d != 5*time.Second {
		t.Fatalf("timers = %+v", r.clock.timers)
	}
	r.clock.fireLast()
	if r.prompt.visible() || r.c.State().Suggested != orient.Unknown {
		t.Errorf("prompt still up after timeout")
	}
	if err := r.c.Accept(ctx); !errors.Is(err, ErrNothingToAccept) {
		t.Errorf("Accept after timeout = %v", err)
	}
}

func TestManualTurnBackHidesPrompt(t *testing.T) {
	r := newRig(t, Settings{Mode: Manual})
	r.c.DeviceTurned(ctx, orient.PortraitCW)
	r.c.DeviceTurned(ctx, orient.Landscape)
	if r.prompt.visible() || r.c.State().Suggested != orient.Unknown {
		t.Errorf("turning back should withdraw the prompt: %v", r.prompt.calls)
	}
}

func TestManualSecondTurnReplacesPrompt(t *testing.T) {
	r := newRig(t, Settings{Mode: Manual})
	r.c.DeviceTurned(ctx, orient.PortraitCW)
	r.c.DeviceTurned(ctx, orient.LandscapeFlipped)
	last := r.prompt.calls[len(r.prompt.calls)-1]
	if last != (promptCall{true, orient.LandscapeFlipped, orient.Landscape}) {
		t.Errorf("last prompt = %v", last)
	}
	if !r.clock.timers[0].stopped || len(r.clock.timers) != 2 {
		t.Errorf("old timeout must be replaced: %+v", r.clock.timers)
	}
}

func TestStaleTimerDoesNotHideNewPrompt(t *testing.T) {
	r := newRig(t, Settings{Mode: Manual})
	r.c.DeviceTurned(ctx, orient.PortraitCW)
	first := r.clock.timers[0]
	r.c.DeviceTurned(ctx, orient.LandscapeFlipped)
	first.f() // fires anyway, racing with Stop
	if r.c.State().Suggested != orient.LandscapeFlipped || !r.prompt.visible() {
		t.Errorf("stale timer hid the current prompt: %+v", r.c.State())
	}
}

func TestLockingHidesPrompt(t *testing.T) {
	r := newRig(t, Settings{Mode: Manual})
	r.c.DeviceTurned(ctx, orient.PortraitCW)
	r.c.SetLocked(ctx, true)
	if r.prompt.visible() || r.c.State().Suggested != orient.Unknown {
		t.Error("lock should withdraw the prompt")
	}
	if got := r.store.s; !got.Locked || got.Mode != Manual {
		t.Errorf("saved %+v", got)
	}
}

func TestUnlockCatchesUp(t *testing.T) {
	r := newRig(t, Settings{Mode: Auto, Locked: true})
	r.c.DeviceTurned(ctx, orient.PortraitCW)
	r.c.SetLocked(ctx, false)
	if !reflect.DeepEqual(r.screen.applied, []orient.Orientation{orient.PortraitCW}) {
		t.Errorf("auto unlock should rotate to the device: %v", r.screen.applied)
	}

	r = newRig(t, Settings{Mode: Manual, Locked: true})
	r.c.DeviceTurned(ctx, orient.PortraitCW)
	r.c.ToggleLock(ctx)
	if !r.prompt.visible() {
		t.Error("manual unlock should offer the device orientation")
	}
}

func TestSwitchingToAutoCatchesUp(t *testing.T) {
	r := newRig(t, Settings{Mode: Manual})
	r.c.DeviceTurned(ctx, orient.LandscapeFlipped)
	r.c.SetMode(ctx, Auto)
	if !reflect.DeepEqual(r.screen.applied, []orient.Orientation{orient.LandscapeFlipped}) {
		t.Errorf("applied = %v", r.screen.applied)
	}
	if r.prompt.visible() {
		t.Error("prompt should be gone in auto mode")
	}
	if r.store.s.Mode != Auto {
		t.Errorf("mode not saved: %+v", r.store.s)
	}
	r.c.ToggleMode(ctx)
	if r.c.State().Mode != Manual {
		t.Errorf("ToggleMode -> %v", r.c.State().Mode)
	}
}

func TestExplicitRotateWorksWhileLocked(t *testing.T) {
	r := newRig(t, Settings{Mode: Auto, Locked: true})
	if err := r.c.RotateTo(ctx, orient.PortraitCCW); err != nil {
		t.Fatal(err)
	}
	if err := r.c.RotateBy(ctx, 1); err != nil {
		t.Fatal(err)
	}
	want := []orient.Orientation{orient.PortraitCCW, orient.Landscape}
	if !reflect.DeepEqual(r.screen.applied, want) {
		t.Errorf("applied = %v, want %v", r.screen.applied, want)
	}
	if !r.c.State().Locked {
		t.Error("explicit rotation must not change the lock")
	}
}

func TestRotateRefreshesScreenFirst(t *testing.T) {
	r := newRig(t, Settings{Mode: Auto, Locked: true})
	r.screen.cur = orient.PortraitCW // changed behind our back in System Settings
	r.c.RotateBy(ctx, 1)
	if last := r.screen.applied[len(r.screen.applied)-1]; last != orient.LandscapeFlipped {
		t.Errorf("RotateBy(1) from portrait-cw applied %v", last)
	}
}

func TestApplyErrorKeepsState(t *testing.T) {
	r := newRig(t, Settings{Mode: Auto, Locked: true})
	r.screen.err = errors.New("kscreen-doctor missing")
	if err := r.c.RotateTo(ctx, orient.PortraitCW); err == nil {
		t.Fatal("expected error")
	}
	if r.c.State().Screen != orient.Landscape {
		t.Errorf("Screen = %v after failed apply", r.c.State().Screen)
	}
}

func TestOnChangeFires(t *testing.T) {
	r := newRig(t, Settings{Mode: Auto})
	n := len(r.states)
	r.c.DeviceTurned(ctx, orient.PortraitCW)
	if len(r.states) == n {
		t.Fatal("OnChange not called")
	}
	if last := r.states[len(r.states)-1]; last.Screen != orient.PortraitCW {
		t.Errorf("last state = %+v", last)
	}
	r.c.SetIMUAvailable(true)
	if !r.states[len(r.states)-1].IMU {
		t.Error("IMU flag not reported")
	}
}

func TestModeStrings(t *testing.T) {
	for _, m := range []Mode{Auto, Manual} {
		got, err := ParseMode(m.String())
		if err != nil || got != m {
			t.Errorf("ParseMode(%q) = %v, %v", m.String(), got, err)
		}
	}
	if _, err := ParseMode("sometimes"); err == nil {
		t.Error("ParseMode(sometimes) should fail")
	}
}

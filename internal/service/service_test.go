package service

import (
	"bufio"
	"context"
	"errors"
	"os/exec"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/godbus/dbus/v5"

	"github.com/RustyRaptor/deckrotate/internal/core"
	"github.com/RustyRaptor/deckrotate/internal/orient"
)

// privateBus starts a throwaway session bus and returns its address.
func privateBus(t *testing.T) string {
	t.Helper()
	path, err := exec.LookPath("dbus-daemon")
	if err != nil {
		t.Skip("dbus-daemon not available")
	}
	cmd := exec.Command(path, "--session", "--nofork", "--print-address=1")
	out, _ := cmd.StdoutPipe()
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { cmd.Process.Kill(); cmd.Wait() })
	addr, err := bufio.NewReader(out).ReadString('\n')
	if err != nil {
		t.Fatal(err)
	}
	return strings.TrimSpace(addr)
}

func connect(t *testing.T, addr string) *dbus.Conn {
	t.Helper()
	conn, err := dbus.Connect(addr)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	return conn
}

type fakeController struct {
	mu    sync.Mutex
	calls []string
	st    core.State
	err   error
}

func (f *fakeController) rec(s string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, s)
	return f.err
}

func (f *fakeController) recorded() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.calls...)
}

func (f *fakeController) setErr(err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.err = err
}

func (f *fakeController) State() core.State { return f.st }
func (f *fakeController) SetMode(_ context.Context, m core.Mode) error {
	return f.rec("mode " + m.String())
}
func (f *fakeController) ToggleMode(context.Context) error { return f.rec("toggle-mode") }
func (f *fakeController) SetLocked(_ context.Context, l bool) error {
	if l {
		return f.rec("lock")
	}
	return f.rec("unlock")
}
func (f *fakeController) ToggleLock(context.Context) error { return f.rec("toggle-lock") }
func (f *fakeController) RotateTo(_ context.Context, o orient.Orientation) error {
	return f.rec("to " + o.String())
}
func (f *fakeController) RotateBy(_ context.Context, n int) error {
	return f.rec("by " + map[int]string{1: "1", -1: "-1"}[n])
}
func (f *fakeController) Accept(context.Context) error { return f.rec("accept") }
func (f *fakeController) Dismiss()                     { f.rec("dismiss") }

func setup(t *testing.T) (*fakeController, *Service, *Client) {
	t.Helper()
	addr := privateBus(t)
	fc := &fakeController{st: core.State{
		Mode: core.Manual, Locked: true, Device: orient.PortraitCW,
		Screen: orient.Landscape, Suggested: orient.Unknown, IMU: true,
	}}
	svc, err := Export(connect(t, addr), fc)
	if err != nil {
		t.Fatal(err)
	}
	return fc, svc, NewClient(connect(t, addr))
}

func TestMethodsReachController(t *testing.T) {
	fc, _, cl := setup(t)
	steps := []struct {
		method string
		args   []any
	}{
		{"SetMode", []any{"auto"}},
		{"ToggleMode", nil},
		{"SetLocked", []any{true}},
		{"SetLocked", []any{false}},
		{"ToggleLock", nil},
		{"RotateTo", []any{"portrait-ccw"}},
		{"RotateBy", []any{int32(1)}},
		{"RotateBy", []any{int32(-1)}},
		{"Accept", nil},
		{"Dismiss", nil},
	}
	for _, s := range steps {
		if err := cl.Call(s.method, s.args...); err != nil {
			t.Fatalf("%s: %v", s.method, err)
		}
	}
	want := []string{"mode auto", "toggle-mode", "lock", "unlock", "toggle-lock",
		"to portrait-ccw", "by 1", "by -1", "accept", "dismiss"}
	if got := fc.recorded(); !reflect.DeepEqual(got, want) {
		t.Errorf("calls = %v\nwant    %v", got, want)
	}
}

func TestBadArgumentsAreDBusErrors(t *testing.T) {
	fc, _, cl := setup(t)
	if err := cl.Call("SetMode", "sometimes"); err == nil {
		t.Error("SetMode(sometimes) should fail")
	}
	if err := cl.Call("RotateTo", "diagonal"); err == nil {
		t.Error("RotateTo(diagonal) should fail")
	}
	if got := fc.recorded(); len(got) != 0 {
		t.Errorf("controller called with bad args: %v", got)
	}
	fc.setErr(errors.New("kscreen-doctor exploded"))
	err := cl.Call("ToggleLock")
	if err == nil || !strings.Contains(err.Error(), "kscreen-doctor exploded") {
		t.Errorf("controller error not forwarded: %v", err)
	}
}

func TestInitialPropertiesAndStatus(t *testing.T) {
	_, _, cl := setup(t)
	got, err := cl.Status()
	if err != nil {
		t.Fatal(err)
	}
	want := Status{Mode: "manual", Locked: true, Device: "portrait-cw", Screen: "landscape", Suggested: "", IMU: true}
	if got != want {
		t.Errorf("Status = %+v, want %+v", got, want)
	}
}

func TestPublishEmitsPropertiesChanged(t *testing.T) {
	addr := privateBus(t)
	svc, err := Export(connect(t, addr), &fakeController{st: core.State{Device: orient.Unknown, Screen: orient.Landscape, Suggested: orient.Unknown}})
	if err != nil {
		t.Fatal(err)
	}
	watcher := connect(t, addr)
	if err := watcher.AddMatchSignal(dbus.WithMatchInterface("org.freedesktop.DBus.Properties"), dbus.WithMatchMember("PropertiesChanged")); err != nil {
		t.Fatal(err)
	}
	ch := make(chan *dbus.Signal, 16)
	watcher.Signal(ch)

	svc.Publish(core.State{Mode: core.Auto, Device: orient.PortraitCCW, Screen: orient.Landscape, Suggested: orient.PortraitCCW, IMU: true})

	changed := map[string]any{}
	deadline := time.After(3 * time.Second)
	for len(changed) < 3 {
		select {
		case sig := <-ch:
			if sig.Body[0] != Interface {
				continue
			}
			for k, v := range sig.Body[1].(map[string]dbus.Variant) {
				changed[k] = v.Value()
			}
		case <-deadline:
			t.Fatalf("only saw %v", changed)
		}
	}
	want := map[string]any{"DeviceOrientation": "portrait-ccw", "SuggestedOrientation": "portrait-ccw", "ImuAvailable": true}
	if !reflect.DeepEqual(changed, want) {
		t.Errorf("changed = %v, want %v", changed, want)
	}
}

func TestSecondInstanceRefused(t *testing.T) {
	addr := privateBus(t)
	if _, err := Export(connect(t, addr), &fakeController{}); err != nil {
		t.Fatal(err)
	}
	if _, err := Export(connect(t, addr), &fakeController{}); !errors.Is(err, ErrAlreadyRunning) {
		t.Errorf("second Export err = %v, want ErrAlreadyRunning", err)
	}
}

func TestIntrospectionListsMethods(t *testing.T) {
	_, _, cl := setup(t)
	var xml string
	if err := cl.conn.Object(BusName, ObjectPath).Call("org.freedesktop.DBus.Introspectable.Introspect", 0).Store(&xml); err != nil {
		t.Fatal(err)
	}
	for _, m := range []string{"ToggleLock", "RotateBy", "Accept", "ScreenOrientation"} {
		if !strings.Contains(xml, `"`+m+`"`) {
			t.Errorf("introspection is missing %s", m)
		}
	}
}

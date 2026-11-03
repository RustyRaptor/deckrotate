package display

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/RustyRaptor/deckrotate/internal/orient"
)

func TestRotationMappingRoundTrips(t *testing.T) {
	// Verified on hardware: the Deck's panel is natively portrait with its top
	// at the Deck's left edge, so "none" is upright with the right grip down.
	want := map[orient.Orientation]string{
		orient.Landscape:        "right",
		orient.PortraitCW:       "none",
		orient.LandscapeFlipped: "left",
		orient.PortraitCCW:      "inverted",
	}
	for o, name := range want {
		if got := RotationFor(o); got != name {
			t.Errorf("RotationFor(%v) = %q, want %q", o, got, name)
		}
		if got := orientationForKScreen(kscreenValue[name]); got != o {
			t.Errorf("orientationForKScreen(%q) = %v, want %v", name, got, o)
		}
	}
}

type call struct {
	name string
	args []string
}

type fakeRunner struct {
	calls []call
	out   []byte
	err   error
}

func (f *fakeRunner) run(_ context.Context, name string, args ...string) ([]byte, error) {
	f.calls = append(f.calls, call{name, args})
	return f.out, f.err
}

func TestCurrentReadsInternalPanel(t *testing.T) {
	out, err := os.ReadFile(filepath.Join("testdata", "kscreen_handheld.json"))
	if err != nil {
		t.Fatal(err)
	}
	f := &fakeRunner{out: out}
	k := NewKScreen(f.run)
	o, err := k.Current(context.Background())
	if err != nil || o != orient.Landscape {
		t.Fatalf("Current = %v, %v; want landscape", o, err)
	}
	if !reflect.DeepEqual(f.calls, []call{{"kscreen-doctor", []string{"-j"}}}) {
		t.Errorf("calls = %v", f.calls)
	}
}

const dockedJSON = `{"outputs":[
 {"name":"DP-1","type":14,"enabled":true,"connected":true,"rotation":1},
 {"name":"eDP-1","type":7,"enabled":true,"connected":true,"rotation":2}
]}`

func TestCurrentIgnoresExternalMonitors(t *testing.T) {
	k := NewKScreen((&fakeRunner{out: []byte(dockedJSON)}).run)
	o, err := k.Current(context.Background())
	if err != nil || o != orient.LandscapeFlipped {
		t.Fatalf("Current = %v, %v; want landscape-flipped", o, err)
	}
}

func TestCurrentNoPanel(t *testing.T) {
	k := NewKScreen((&fakeRunner{out: []byte(`{"outputs":[{"name":"DP-1","type":14,"rotation":1}]}`)}).run)
	if _, err := k.Current(context.Background()); !errors.Is(err, ErrNoPanel) {
		t.Fatalf("err = %v, want ErrNoPanel", err)
	}
}

func TestApplyUsesPanelName(t *testing.T) {
	f := &fakeRunner{out: []byte(dockedJSON)}
	k := NewKScreen(f.run)
	if err := k.Apply(context.Background(), orient.PortraitCCW); err != nil {
		t.Fatal(err)
	}
	want := call{"kscreen-doctor", []string{"output.eDP-1.rotation.inverted"}}
	if len(f.calls) != 2 || !reflect.DeepEqual(f.calls[1], want) {
		t.Errorf("calls = %v, want lookup then %v", f.calls, want)
	}
	// The panel name is cached after the first lookup.
	k.Apply(context.Background(), orient.Landscape)
	if len(f.calls) != 3 {
		t.Errorf("expected a single extra call, got %v", f.calls)
	}
}

func TestApplyRejectsUnknown(t *testing.T) {
	k := NewKScreen((&fakeRunner{out: []byte(dockedJSON)}).run)
	if err := k.Apply(context.Background(), orient.Unknown); err == nil {
		t.Error("Apply(Unknown) should fail")
	}
}

func TestRunnerErrorsPropagate(t *testing.T) {
	k := NewKScreen((&fakeRunner{err: errors.New("boom")}).run)
	if _, err := k.Current(context.Background()); err == nil {
		t.Error("expected error")
	}
}

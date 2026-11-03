// Package display reads and sets the Deck panel's rotation through kscreen-doctor.
package display

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"sync"

	"github.com/RustyRaptor/deckrotate/internal/orient"
)

var ErrNoPanel = errors.New("display: no internal panel found")

// The Deck's panel is natively portrait with its top at the Deck's left edge.
var rotationFor = map[orient.Orientation]string{
	orient.Landscape:        "right",
	orient.PortraitCW:       "none",
	orient.LandscapeFlipped: "left",
	orient.PortraitCCW:      "inverted",
}

// KScreen::Output::Rotation values as they appear in kscreen-doctor -j.
var kscreenValue = map[string]int{"none": 1, "left": 2, "inverted": 4, "right": 8}

const kscreenPanelType = 7

// RotationFor is the kscreen rotation that shows content upright when the
// Deck is physically in orientation o.
func RotationFor(o orient.Orientation) string { return rotationFor[o] }

func orientationForKScreen(v int) orient.Orientation {
	for o, name := range rotationFor {
		if kscreenValue[name] == v {
			return o
		}
	}
	return orient.Unknown
}

type Runner func(ctx context.Context, name string, args ...string) ([]byte, error)

// ExecRunner runs a real command and returns its stdout.
func ExecRunner(ctx context.Context, name string, args ...string) ([]byte, error) {
	out, err := exec.CommandContext(ctx, name, args...).Output()
	if ee, ok := err.(*exec.ExitError); ok {
		return out, fmt.Errorf("%s: %w: %s", name, err, strings.TrimSpace(string(ee.Stderr)))
	}
	return out, err
}

type KScreen struct {
	run   Runner
	mu    sync.Mutex
	panel string
}

func NewKScreen(run Runner) *KScreen { return &KScreen{run: run} }

type output struct {
	Name     string `json:"name"`
	Type     int    `json:"type"`
	Rotation int    `json:"rotation"`
}

func (k *KScreen) panelOutput(ctx context.Context) (output, error) {
	out, err := k.run(ctx, "kscreen-doctor", "-j")
	if err != nil {
		return output{}, err
	}
	var cfg struct {
		Outputs []output `json:"outputs"`
	}
	if err := json.Unmarshal(out, &cfg); err != nil {
		return output{}, fmt.Errorf("display: parse kscreen-doctor: %w", err)
	}
	for _, o := range cfg.Outputs {
		if o.Type == kscreenPanelType || strings.HasPrefix(o.Name, "eDP") {
			return o, nil
		}
	}
	return output{}, ErrNoPanel
}

// Current returns the physical orientation the panel's rotation is set for.
func (k *KScreen) Current(ctx context.Context) (orient.Orientation, error) {
	o, err := k.panelOutput(ctx)
	if err != nil {
		return orient.Unknown, err
	}
	k.mu.Lock()
	k.panel = o.Name
	k.mu.Unlock()
	return orientationForKScreen(o.Rotation), nil
}

// Apply rotates the panel so content is upright in orientation o.
func (k *KScreen) Apply(ctx context.Context, o orient.Orientation) error {
	rot, ok := rotationFor[o]
	if !ok {
		return fmt.Errorf("display: cannot apply %v", o)
	}
	k.mu.Lock()
	name := k.panel
	k.mu.Unlock()
	if name == "" {
		out, err := k.panelOutput(ctx)
		if err != nil {
			return err
		}
		name = out.Name
		k.mu.Lock()
		k.panel = name
		k.mu.Unlock()
	}
	_, err := k.run(ctx, "kscreen-doctor", "output."+name+".rotation."+rot)
	return err
}

package overlay

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/RustyRaptor/deckrotate/internal/orient"
)

func TestPlacement(t *testing.T) {
	cases := []struct {
		target, current orient.Orientation
		want            Placement
	}{
		// Not rotated relative to the content: plain bottom-right, upright.
		{orient.Landscape, orient.Landscape, Placement{"bottom", "right", 0}},
		// Turned clockwise (right grip down): the user's bottom-right is the
		// content's top-right, and the icon must turn counter-clockwise.
		{orient.PortraitCW, orient.Landscape, Placement{"top", "right", 270}},
		// Turned counter-clockwise (left grip down).
		{orient.PortraitCCW, orient.Landscape, Placement{"bottom", "left", 90}},
		// Upside down.
		{orient.LandscapeFlipped, orient.Landscape, Placement{"top", "left", 180}},
		// Relative turns are what matter, not absolute ones.
		{orient.Landscape, orient.PortraitCCW, Placement{"top", "right", 270}},
		{orient.PortraitCW, orient.LandscapeFlipped, Placement{"bottom", "left", 90}},
	}
	for _, c := range cases {
		if got := PlacementFor(c.target, c.current); got != c.want {
			t.Errorf("PlacementFor(%v, %v) = %+v, want %+v", c.target, c.current, got, c.want)
		}
	}
}

func TestPlacementArgs(t *testing.T) {
	got := Placement{"top", "left", 180}.Args()
	want := []string{"--vertical", "top", "--horizontal", "left", "--angle", "180"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Args = %v", got)
	}
}

// TestHelperOverlay is not a real test: the process tests below re-run the
// test binary as a stand-in for the QML overlay.
func TestHelperOverlay(t *testing.T) {
	mode := os.Getenv("DECKROTATE_HELPER")
	if mode == "" {
		return
	}
	fmt.Fprintf(os.Stderr, "args=%v\n", os.Args)
	switch mode {
	case "accept":
		os.Exit(AcceptExitCode)
	case "fail":
		os.Exit(1)
	case "wait":
		bufio.NewReader(os.Stdin).ReadString('\n')
		time.Sleep(time.Minute)
	}
	os.Exit(0)
}

func helper(mode string, seen *[][]string, mu *sync.Mutex) CommandFunc {
	return func(ctx context.Context, p Placement) *exec.Cmd {
		mu.Lock()
		*seen = append(*seen, p.Args())
		mu.Unlock()
		cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=TestHelperOverlay")
		cmd.Env = append(os.Environ(), "DECKROTATE_HELPER="+mode)
		return cmd
	}
}

func TestProcessPromptAccept(t *testing.T) {
	var mu sync.Mutex
	var seen [][]string
	accepted := make(chan struct{}, 1)
	p := NewProcessPrompt(helper("accept", &seen, &mu), func() { accepted <- struct{}{} })
	p.Show(orient.PortraitCW, orient.Landscape)
	select {
	case <-accepted:
	case <-time.After(5 * time.Second):
		t.Fatal("OnAccept not called")
	}
	mu.Lock()
	defer mu.Unlock()
	if len(seen) != 1 || strings.Join(seen[0], " ") != "--vertical top --horizontal right --angle 270" {
		t.Errorf("args = %v", seen)
	}
}

func TestProcessPromptOtherExitIsNotAccept(t *testing.T) {
	var mu sync.Mutex
	var seen [][]string
	accepted := make(chan struct{}, 1)
	p := NewProcessPrompt(helper("fail", &seen, &mu), func() { accepted <- struct{}{} })
	p.Show(orient.PortraitCW, orient.Landscape)
	deadline := time.Now().Add(5 * time.Second)
	for p.running() && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	select {
	case <-accepted:
		t.Fatal("a failing overlay must not count as accept")
	default:
	}
}

func TestProcessPromptHideKills(t *testing.T) {
	var mu sync.Mutex
	var seen [][]string
	p := NewProcessPrompt(helper("wait", &seen, &mu), func() { t.Error("unexpected accept") })
	p.Show(orient.PortraitCW, orient.Landscape)
	if !p.running() {
		t.Fatal("overlay not running after Show")
	}
	p.Show(orient.PortraitCCW, orient.Landscape) // replaces the first
	p.Hide()
	deadline := time.Now().Add(5 * time.Second)
	for p.running() && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if p.running() {
		t.Fatal("overlay still running after Hide")
	}
	mu.Lock()
	defer mu.Unlock()
	if len(seen) != 2 {
		t.Errorf("started %d overlays, want 2", len(seen))
	}
}

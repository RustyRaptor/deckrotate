// Package overlay runs the manual-mode "rotate" button as a separate QML process.
package overlay

import (
	"context"
	"errors"
	"log"
	"os/exec"
	"strconv"
	"sync"

	"github.com/RustyRaptor/deckrotate/internal/orient"
)

// Placement says where the button goes on the screen as it is currently
// rotated, and how far (clockwise degrees) to turn its icon so it looks
// upright to someone holding the Deck in the new orientation.
type Placement struct {
	Vertical   string // top or bottom
	Horizontal string // left or right
	Angle      int
}

var direction = [4]string{"top", "right", "bottom", "left"}

// PlacementFor puts the button in the bottom-right corner as seen by someone
// holding the Deck in target, while the content is still drawn for current.
func PlacementFor(target, current orient.Orientation) Placement {
	d := ((int(target)-int(current))%4 + 4) % 4
	userRight := direction[(1-d+4)%4]
	userDown := direction[(2-d+4)%4]
	p := Placement{Angle: (360 - 90*d) % 360}
	if userDown == "top" || userDown == "bottom" {
		p.Vertical, p.Horizontal = userDown, userRight
	} else {
		p.Vertical, p.Horizontal = userRight, userDown
	}
	return p
}

func (p Placement) Args() []string {
	return []string{"--vertical", p.Vertical, "--horizontal", p.Horizontal, "--angle", strconv.Itoa(p.Angle)}
}

type CommandFunc func(ctx context.Context, p Placement) *exec.Cmd

// AcceptExitCode is how the overlay reports a tap; QML has no clean stdout.
const AcceptExitCode = 10

// ProcessPrompt implements core.Prompt by starting the overlay command on
// Show and killing it on Hide.
type ProcessPrompt struct {
	command  CommandFunc
	onAccept func()

	mu     sync.Mutex
	cancel context.CancelFunc
	gen    int
	live   int
}

func NewProcessPrompt(command CommandFunc, onAccept func()) *ProcessPrompt {
	return &ProcessPrompt{command: command, onAccept: onAccept}
}

func (p *ProcessPrompt) Show(target, current orient.Orientation) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.stopLocked()
	ctx, cancel := context.WithCancel(context.Background())
	cmd := p.command(ctx, PlacementFor(target, current))
	if err := cmd.Start(); err != nil {
		cancel()
		log.Printf("overlay: start: %v", err)
		return
	}
	p.cancel = cancel
	p.gen++
	gen := p.gen
	p.live++
	go func() {
		err := cmd.Wait()
		cancel()
		var ee *exec.ExitError
		accepted := errors.As(err, &ee) && ee.ExitCode() == AcceptExitCode
		p.mu.Lock()
		p.live--
		current := p.gen == gen
		p.mu.Unlock()
		if accepted && current {
			p.onAccept()
		}
	}()
}

func (p *ProcessPrompt) Hide() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.stopLocked()
}

func (p *ProcessPrompt) stopLocked() {
	if p.cancel != nil {
		p.cancel()
		p.cancel = nil
		p.gen++
	}
}

func (p *ProcessPrompt) running() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.live > 0
}

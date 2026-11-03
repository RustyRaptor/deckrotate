// Package orient turns accelerometer readings into a physical Deck orientation.
package orient

import (
	"fmt"
	"math"
	"time"
)

// Orientation is how far the Deck is physically turned clockwise from normal
// landscape, in quarter turns, as seen from the front.
type Orientation int

const (
	Unknown          Orientation = -1
	Landscape        Orientation = 0 // top edge up
	PortraitCW       Orientation = 1 // left edge up, right grip down
	LandscapeFlipped Orientation = 2 // bottom edge up
	PortraitCCW      Orientation = 3 // right edge up, left grip down
)

var All = []Orientation{Landscape, PortraitCW, LandscapeFlipped, PortraitCCW}

var names = map[Orientation]string{
	Unknown:          "unknown",
	Landscape:        "landscape",
	PortraitCW:       "portrait-cw",
	LandscapeFlipped: "landscape-flipped",
	PortraitCCW:      "portrait-ccw",
}

func (o Orientation) String() string {
	if n, ok := names[o]; ok {
		return n
	}
	return fmt.Sprintf("orientation(%d)", int(o))
}

func Parse(s string) (Orientation, error) {
	for _, o := range All {
		if names[o] == s {
			return o, nil
		}
	}
	return Unknown, fmt.Errorf("unknown orientation %q", s)
}

// Rotate turns o by the given number of clockwise quarter turns (negative is
// counter-clockwise).
func (o Orientation) Rotate(turns int) Orientation {
	return Orientation(((int(o)+turns)%4 + 4) % 4)
}

// Vec is a gravity reaction vector in g, in Deck coordinates: X toward the
// right edge, Y toward the top edge, Z out of the screen.
type Vec struct{ X, Y, Z float64 }

const (
	minMagnitude = 0.6  // below this the Deck is being swung or dropped
	maxMagnitude = 1.4  // above this it is being shaken
	minPlane     = 0.45 // ~27 degrees from flat; flatter than this has no "up"
	stickyDeg    = 65.0 // stay in the current orientation within this angle
)

// Classify returns the orientation v points to, or Unknown when the Deck is
// too flat or moving too much to tell. current adds hysteresis: a reading has
// to be well past the 45 degree midpoint before it leaves current.
func Classify(current Orientation, v Vec) Orientation {
	mag := math.Sqrt(v.X*v.X + v.Y*v.Y + v.Z*v.Z)
	if mag < minMagnitude || mag > maxMagnitude {
		return Unknown
	}
	if math.Hypot(v.X, v.Y) < minPlane {
		return Unknown
	}
	// 0 = top edge up, increasing as the Deck turns clockwise.
	deg := math.Atan2(-v.X, v.Y) * 180 / math.Pi
	if current != Unknown && angleDist(deg, float64(current)*90) <= stickyDeg {
		return current
	}
	return Orientation(int(math.Round(deg / 90))).Rotate(0)
}

func angleDist(a, b float64) float64 {
	d := math.Mod(math.Abs(a-b), 360)
	if d > 180 {
		d = 360 - d
	}
	return d
}

// Detector debounces Classify: an orientation is reported only after every
// sample for settle time agrees on it.
type Detector struct {
	settle    time.Duration
	current   Orientation
	candidate Orientation
	since     time.Time
}

func NewDetector(settle time.Duration) *Detector {
	return &Detector{settle: settle, current: Unknown, candidate: Unknown}
}

// Current is the last reported orientation, or Unknown before the first.
func (d *Detector) Current() Orientation { return d.current }

// Update feeds one sample taken at t. It returns the new orientation and true
// when the settled orientation changes.
func (d *Detector) Update(t time.Time, v Vec) (Orientation, bool) {
	ref := d.candidate
	if ref == Unknown {
		ref = d.current
	}
	o := Classify(ref, v)
	if o == Unknown || o == d.current {
		d.candidate = Unknown
		return d.current, false
	}
	if o != d.candidate {
		d.candidate, d.since = o, t
	}
	if t.Sub(d.since) < d.settle {
		return d.current, false
	}
	d.current, d.candidate = o, Unknown
	return o, true
}

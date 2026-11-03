package orient

import (
	"math"
	"testing"
	"time"
)

func TestOrientationStringRoundTrip(t *testing.T) {
	for _, o := range All {
		got, err := Parse(o.String())
		if err != nil || got != o {
			t.Errorf("Parse(%q) = %v, %v; want %v", o.String(), got, err, o)
		}
	}
	if _, err := Parse("sideways"); err == nil {
		t.Error("Parse(sideways) should fail")
	}
	if Unknown.String() != "unknown" {
		t.Errorf("Unknown.String() = %q", Unknown.String())
	}
}

func TestRotate(t *testing.T) {
	cases := []struct {
		o     Orientation
		turns int
		want  Orientation
	}{
		{Landscape, 1, PortraitCW},
		{PortraitCW, 1, LandscapeFlipped},
		{PortraitCCW, 1, Landscape},
		{Landscape, -1, PortraitCCW},
		{Landscape, 2, LandscapeFlipped},
		{PortraitCW, -5, Landscape},
	}
	for _, c := range cases {
		if got := c.o.Rotate(c.turns); got != c.want {
			t.Errorf("%v.Rotate(%d) = %v, want %v", c.o, c.turns, got, c.want)
		}
	}
}

// vec builds a 1 g screen-plane vector for a Deck turned deg degrees
// clockwise from landscape (0 = top edge up, 90 = left edge up).
func vec(deg float64) Vec {
	r := deg * math.Pi / 180
	return Vec{X: -math.Sin(r), Y: math.Cos(r)}
}

func TestClassifyCardinal(t *testing.T) {
	cases := []struct {
		v    Vec
		want Orientation
	}{
		{Vec{X: 0, Y: 1}, Landscape},
		{Vec{X: -1, Y: 0}, PortraitCW},
		{Vec{X: 0, Y: -1}, LandscapeFlipped},
		{Vec{X: 1, Y: 0}, PortraitCCW},
	}
	for _, c := range cases {
		if got := Classify(Unknown, c.v); got != c.want {
			t.Errorf("Classify(%+v) = %v, want %v", c.v, got, c.want)
		}
	}
}

func TestClassifyFlatIsUnknown(t *testing.T) {
	// Lying flat: gravity is almost entirely through the screen.
	flat := Vec{X: 0.1, Y: 0.2, Z: 0.97}
	if got := Classify(Landscape, flat); got != Unknown {
		t.Errorf("flat = %v, want unknown", got)
	}
}

func TestClassifyRejectsShaking(t *testing.T) {
	// 2.5 g along +Y is a jolt, not a steady pose.
	if got := Classify(Unknown, Vec{Y: 2.5}); got != Unknown {
		t.Errorf("jolt = %v, want unknown", got)
	}
	if got := Classify(Unknown, Vec{Y: 0.3}); got != Unknown {
		t.Errorf("free-fall-ish = %v, want unknown", got)
	}
}

func TestClassifyTiltedBackStillLandscape(t *testing.T) {
	// Real reading from the Deck held upright, leaning back ~22 degrees.
	if got := Classify(Unknown, Vec{X: 0.04, Y: 0.93, Z: 0.38}); got != Landscape {
		t.Errorf("got %v, want landscape", got)
	}
}

func TestClassifyHysteresis(t *testing.T) {
	// 50 degrees toward PortraitCW: past the 45 degree midpoint, but inside
	// the hysteresis band, so the current orientation sticks.
	if got := Classify(Landscape, vec(50)); got != Landscape {
		t.Errorf("50deg from landscape = %v, want landscape (hysteresis)", got)
	}
	// With no current orientation the nearest one wins.
	if got := Classify(Unknown, vec(50)); got != PortraitCW {
		t.Errorf("50deg fresh = %v, want portrait-cw", got)
	}
	// Well past the band it switches.
	if got := Classify(Landscape, vec(70)); got != PortraitCW {
		t.Errorf("70deg from landscape = %v, want portrait-cw", got)
	}
	// Hysteresis works across the 0/360 wrap too.
	if got := Classify(Landscape, vec(-50)); got != Landscape {
		t.Errorf("-50deg from landscape = %v, want landscape", got)
	}
	if got := Classify(PortraitCCW, vec(-40)); got != PortraitCCW {
		t.Errorf("-40deg from portrait-ccw = %v, want portrait-ccw", got)
	}
}

func TestDetectorNeedsSettleTime(t *testing.T) {
	d := NewDetector(400 * time.Millisecond)
	t0 := time.Unix(1000, 0)

	if _, ok := d.Update(t0, vec(0)); ok {
		t.Fatal("first sample must not report a change yet")
	}
	if _, ok := d.Update(t0.Add(300*time.Millisecond), vec(0)); ok {
		t.Fatal("reported before settle time")
	}
	o, ok := d.Update(t0.Add(400*time.Millisecond), vec(0))
	if !ok || o != Landscape {
		t.Fatalf("got %v,%v; want landscape,true", o, ok)
	}
	if _, ok := d.Update(t0.Add(500*time.Millisecond), vec(0)); ok {
		t.Fatal("must report each orientation once")
	}
	if d.Current() != Landscape {
		t.Fatalf("Current() = %v", d.Current())
	}
}

func TestDetectorIgnoresBriefWobble(t *testing.T) {
	d := NewDetector(400 * time.Millisecond)
	t0 := time.Unix(1000, 0)
	d.Update(t0, vec(0))
	d.Update(t0.Add(400*time.Millisecond), vec(0))

	at := t0.Add(time.Second)
	d.Update(at, vec(90))                           // starts turning
	d.Update(at.Add(200*time.Millisecond), vec(90)) // not long enough
	d.Update(at.Add(300*time.Millisecond), vec(0))  // came back
	if _, ok := d.Update(at.Add(800*time.Millisecond), vec(0)); ok {
		t.Fatal("returning to the same orientation is not a change")
	}
	if d.Current() != Landscape {
		t.Fatalf("Current() = %v", d.Current())
	}
}

func TestDetectorReportsTurn(t *testing.T) {
	d := NewDetector(400 * time.Millisecond)
	t0 := time.Unix(1000, 0)
	d.Update(t0, vec(0))
	d.Update(t0.Add(400*time.Millisecond), vec(0))

	at := t0.Add(time.Second)
	d.Update(at, vec(-90))
	o, ok := d.Update(at.Add(450*time.Millisecond), vec(-88))
	if !ok || o != PortraitCCW {
		t.Fatalf("got %v,%v; want portrait-ccw,true", o, ok)
	}
}

func TestDetectorFlatKeepsOrientationAndResetsCandidate(t *testing.T) {
	d := NewDetector(400 * time.Millisecond)
	t0 := time.Unix(1000, 0)
	d.Update(t0, vec(0))
	d.Update(t0.Add(400*time.Millisecond), vec(0))

	at := t0.Add(time.Second)
	d.Update(at, vec(90))
	d.Update(at.Add(200*time.Millisecond), Vec{Z: 1}) // laid flat mid-turn
	if _, ok := d.Update(at.Add(450*time.Millisecond), vec(90)); ok {
		t.Fatal("flat sample must restart the settle timer")
	}
	if d.Current() != Landscape {
		t.Fatalf("Current() = %v", d.Current())
	}
	if o, ok := d.Update(at.Add(850*time.Millisecond), vec(90)); !ok || o != PortraitCW {
		t.Fatalf("got %v,%v; want portrait-cw,true", o, ok)
	}
}

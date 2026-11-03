// Package daemon wires the IMU, orientation detector, controller, overlay
// and D-Bus service together.
package daemon

import (
	"context"
	"log"
	"time"

	"github.com/RustyRaptor/deckrotate/internal/imu"
	"github.com/RustyRaptor/deckrotate/internal/orient"
)

type Sink interface {
	DeviceTurned(context.Context, orient.Orientation) error
	SetIMUAvailable(bool)
}

// Tracker turns raw IMU samples into settled orientation changes. It is not
// safe for concurrent use; feed it from the single IMU reader goroutine.
type Tracker struct {
	sink  Sink
	det   *orient.Detector
	alive bool
}

func NewTracker(sink Sink, settle time.Duration) *Tracker {
	return &Tracker{sink: sink, det: orient.NewDetector(settle)}
}

func (t *Tracker) Sample(at time.Time, s imu.Sample) {
	if !t.alive {
		t.alive = true
		t.sink.SetIMUAvailable(true)
	}
	if o, changed := t.det.Update(at, s.Vec()); changed {
		log.Printf("deck turned to %v", o)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		if err := t.sink.DeviceTurned(ctx, o); err != nil {
			log.Printf("rotate to %v: %v", o, err)
		}
		cancel()
	}
}

// Lost records that the IMU stopped delivering (device gone, read error).
func (t *Tracker) Lost() {
	if t.alive {
		t.alive = false
		t.sink.SetIMUAvailable(false)
	}
}

package daemon

import (
	"context"
	"testing"
	"time"

	"github.com/RustyRaptor/deckrotate/internal/imu"
	"github.com/RustyRaptor/deckrotate/internal/orient"
)

type fakeSink struct {
	turns []orient.Orientation
	imu   []bool
}

func (f *fakeSink) DeviceTurned(_ context.Context, o orient.Orientation) error {
	f.turns = append(f.turns, o)
	return nil
}
func (f *fakeSink) SetIMUAvailable(ok bool) { f.imu = append(f.imu, ok) }

var (
	landscape  = imu.Sample{Accel: [3]int16{660, 15228, 6174}} // recorded on hardware
	leftGripDn = imu.Sample{Accel: [3]int16{16176, -88, 520}}  // right edge up
	flat       = imu.Sample{Accel: [3]int16{46, 4839, -15603}} // face down on a table
)

func TestTrackerFeedsSettledTurns(t *testing.T) {
	sink := &fakeSink{}
	tr := NewTracker(sink, 300*time.Millisecond)
	t0 := time.Unix(0, 0)
	feed := func(s imu.Sample, from, to time.Duration) {
		for d := from; d <= to; d += 4 * time.Millisecond {
			tr.Sample(t0.Add(d), s)
		}
	}
	feed(landscape, 0, 400*time.Millisecond)
	feed(leftGripDn, 500*time.Millisecond, 900*time.Millisecond)
	feed(flat, time.Second, 2*time.Second)

	want := []orient.Orientation{orient.Landscape, orient.PortraitCCW}
	if len(sink.turns) != 2 || sink.turns[0] != want[0] || sink.turns[1] != want[1] {
		t.Errorf("turns = %v, want %v", sink.turns, want)
	}
	if len(sink.imu) != 1 || !sink.imu[0] {
		t.Errorf("imu availability = %v, want one true", sink.imu)
	}
}

func TestTrackerReportsLostIMU(t *testing.T) {
	sink := &fakeSink{}
	tr := NewTracker(sink, 300*time.Millisecond)
	tr.Sample(time.Unix(0, 0), landscape)
	tr.Lost()
	tr.Lost()
	tr.Sample(time.Unix(1, 0), landscape)
	want := []bool{true, false, true}
	if len(sink.imu) != 3 || sink.imu[0] != want[0] || sink.imu[1] != want[1] || sink.imu[2] != want[2] {
		t.Errorf("imu = %v, want %v", sink.imu, want)
	}
}

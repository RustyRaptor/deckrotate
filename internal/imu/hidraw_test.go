package imu

import (
	"os"
	"testing"
)

func TestHIDIOCSFEATURE(t *testing.T) {
	// From <linux/hidraw.h>: _IOC(_IOC_WRITE|_IOC_READ, 'H', 0x06, len)
	if got := hidiocsfeature(65); got != 0xC0414806 {
		t.Errorf("hidiocsfeature(65) = %#x, want 0xc0414806", got)
	}
}

// TestHardwareSample runs only on a Deck with DECKROTATE_HW=1.
func TestHardwareSample(t *testing.T) {
	if os.Getenv("DECKROTATE_HW") == "" {
		t.Skip("set DECKROTATE_HW=1 to read the real controller")
	}
	path, err := Find("/sys")
	if err != nil {
		t.Fatal(err)
	}
	dev, err := OpenHIDRaw(path)
	if err != nil {
		t.Fatal(err)
	}
	defer dev.Close()
	n := 0
	err = NewReader(dev, 250).Run(func(s Sample) {
		if n++; n == 50 {
			dev.Close()
		}
		if v := s.Vec(); v.X*v.X+v.Y*v.Y+v.Z*v.Z < 0.5 {
			t.Errorf("implausible gravity %+v", v)
		}
	})
	if n < 50 {
		t.Fatalf("got %d samples before error %v", n, err)
	}
}

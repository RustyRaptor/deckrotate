// Package imu reads the Steam Deck controller's accelerometer from hidraw.
package imu

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/RustyRaptor/deckrotate/internal/orient"
)

const (
	ReportSize        = 64
	FeatureReportSize = 64

	deckHIDID       = "0003:000028DE:00001205"
	reportTypeState = 0x09
	accelOffset     = 24 // after header(4), packet num(4), buttons(8), pads(8)
	gyroOffset      = 30
	countsPerG      = 16384.0 // +/-2 g range

	cmdSetSettings  = 0x87
	settingIMUMode  = 0x30
	imuRawAccelGyro = 0x0008 | 0x0010
)

var ErrNotState = errors.New("imu: not a controller state report")

// vendorUsagePage prefixes the report descriptor of the interface carrying
// controller state; the other Deck interfaces are lizard-mode mouse/keyboard.
var vendorUsagePage = []byte{0x06, 0xff, 0xff}

type Sample struct {
	Accel [3]int16
	Gyro  [3]int16
}

// Off reports whether the controller is sending no IMU data, which is what
// Steam leaves it in when the active config doesn't use the gyro.
func (s Sample) Off() bool {
	return s.Accel == [3]int16{} && s.Gyro == [3]int16{}
}

func (s Sample) Vec() orient.Vec {
	return orient.Vec{
		X: float64(s.Accel[0]) / countsPerG,
		Y: float64(s.Accel[1]) / countsPerG,
		Z: float64(s.Accel[2]) / countsPerG,
	}
}

func Decode(r []byte) (Sample, error) {
	if len(r) < ReportSize {
		return Sample{}, fmt.Errorf("imu: short report (%d bytes)", len(r))
	}
	if r[2] != reportTypeState {
		return Sample{}, ErrNotState
	}
	var s Sample
	for i := range 3 {
		s.Accel[i] = int16(binary.LittleEndian.Uint16(r[accelOffset+2*i:]))
		s.Gyro[i] = int16(binary.LittleEndian.Uint16(r[gyroOffset+2*i:]))
	}
	return s, nil
}

// EnableReport is the hidraw feature report (leading report ID 0) that
// switches the controller to streaming raw accelerometer and gyro data.
func EnableReport() []byte {
	b := make([]byte, FeatureReportSize+1)
	b[1] = cmdSetSettings
	b[2] = 3 // one {u8 register, u16 value} setting
	b[3] = settingIMUMode
	binary.LittleEndian.PutUint16(b[4:], imuRawAccelGyro)
	return b
}

// Find returns the /dev path of the Deck controller's state interface.
// sysRoot is normally "/sys".
func Find(sysRoot string) (string, error) {
	dirs, err := filepath.Glob(filepath.Join(sysRoot, "class", "hidraw", "hidraw*"))
	if err != nil {
		return "", err
	}
	for _, d := range dirs {
		uevent, err := os.ReadFile(filepath.Join(d, "device", "uevent"))
		if err != nil || !strings.Contains(strings.ToUpper(string(uevent)), "HID_ID="+deckHIDID) {
			continue
		}
		desc, err := os.ReadFile(filepath.Join(d, "device", "report_descriptor"))
		if err != nil || !bytes.HasPrefix(desc, vendorUsagePage) {
			continue
		}
		return "/dev/" + filepath.Base(d), nil
	}
	return "", fmt.Errorf("imu: no Steam Deck controller found under %s", sysRoot)
}

type Device interface {
	Read(p []byte) (int, error)
	SetFeature(report []byte) error
}

// Reader streams samples and keeps the IMU switched on.
type Reader struct {
	dev           Device
	reenableAfter int
}

// NewReader re-sends the enable report after reenableAfter consecutive
// samples without IMU data (the controller reports at about 250 Hz).
func NewReader(dev Device, reenableAfter int) *Reader {
	return &Reader{dev: dev, reenableAfter: reenableAfter}
}

// Run calls f for every sample that carries IMU data until a read fails.
func (r *Reader) Run(f func(Sample)) error {
	if err := r.dev.SetFeature(EnableReport()); err != nil {
		return fmt.Errorf("imu: enable: %w", err)
	}
	buf := make([]byte, ReportSize)
	empty := 0
	for {
		n, err := r.dev.Read(buf)
		if err != nil {
			return err
		}
		s, err := Decode(buf[:n])
		if err != nil {
			continue
		}
		if !s.Off() {
			empty = 0
			f(s)
			continue
		}
		if empty++; empty >= r.reenableAfter {
			empty = 0
			if err := r.dev.SetFeature(EnableReport()); err != nil {
				return fmt.Errorf("imu: re-enable: %w", err)
			}
		}
	}
}

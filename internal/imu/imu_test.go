package imu

import (
	"bytes"
	"errors"
	"math"
	"os"
	"path/filepath"
	"testing"
)

func fixture(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestDecodeRealReport(t *testing.T) {
	s, err := Decode(fixture(t, "report_imu_on.bin"))
	if err != nil {
		t.Fatal(err)
	}
	if s.Accel != [3]int16{690, 15104, 6079} {
		t.Errorf("Accel = %v", s.Accel)
	}
	if s.Off() {
		t.Error("report with accel data reported as Off")
	}
}

func TestDecodeIMUOffReport(t *testing.T) {
	s, err := Decode(fixture(t, "report_imu_off.bin"))
	if err != nil {
		t.Fatal(err)
	}
	if !s.Off() {
		t.Errorf("all-zero IMU should be Off, got %+v", s)
	}
}

func TestDecodeRejectsOtherReports(t *testing.T) {
	r := make([]byte, ReportSize)
	r[0], r[2] = 0x01, 0x04 // battery/status report type
	if _, err := Decode(r); !errors.Is(err, ErrNotState) {
		t.Errorf("err = %v, want ErrNotState", err)
	}
	if _, err := Decode(make([]byte, 10)); err == nil || errors.Is(err, ErrNotState) {
		t.Errorf("short report err = %v, want a size error", err)
	}
}

func TestSampleVecInG(t *testing.T) {
	v := Sample{Accel: [3]int16{-16384, 8192, 0}}.Vec()
	if math.Abs(v.X+1) > 1e-9 || math.Abs(v.Y-0.5) > 1e-9 || v.Z != 0 {
		t.Errorf("Vec = %+v", v)
	}
}

func TestEnableReport(t *testing.T) {
	got := EnableReport()
	if len(got) != FeatureReportSize+1 {
		t.Fatalf("len = %d, want %d", len(got), FeatureReportSize+1)
	}
	// report ID 0, ID_SET_SETTINGS_VALUES, 3 bytes, SETTING_IMU_MODE = raw accel|gyro (LE u16)
	want := []byte{0x00, 0x87, 0x03, 0x30, 0x18, 0x00}
	if !bytes.Equal(got[:len(want)], want) {
		t.Errorf("prefix = % x, want % x", got[:len(want)], want)
	}
	for i, b := range got[len(want):] {
		if b != 0 {
			t.Fatalf("byte %d = %#x, want zero padding", len(want)+i, b)
		}
	}
}

func TestFindPicksVendorInterface(t *testing.T) {
	root := t.TempDir()
	mk := func(name, hidID string, desc []byte) {
		d := filepath.Join(root, "class", "hidraw", name, "device")
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
		os.WriteFile(filepath.Join(d, "uevent"), []byte("DRIVER=hid-steam\nHID_ID="+hidID+"\n"), 0o644)
		os.WriteFile(filepath.Join(d, "report_descriptor"), desc, 0o644)
	}
	mk("hidraw0", "0003:000028DE:00001205", []byte{0x05, 0x01, 0x09, 0x02}) // lizard mouse
	mk("hidraw2", "0018:00002808:00001015", []byte{0x06, 0xff, 0xff})       // touchscreen
	mk("hidraw3", "0003:000028DE:00001205", []byte{0x06, 0xff, 0xff, 0x09}) // controller state

	got, err := Find(root)
	if err != nil || got != "/dev/hidraw3" {
		t.Errorf("Find = %q, %v; want /dev/hidraw3", got, err)
	}
}

func TestFindNoDeck(t *testing.T) {
	if _, err := Find(t.TempDir()); err == nil {
		t.Error("expected an error with no devices")
	}
}

// fakeDev feeds canned reports and records feature reports written to it.
type fakeDev struct {
	reports  [][]byte
	features [][]byte
}

func (f *fakeDev) Read(p []byte) (int, error) {
	if len(f.reports) == 0 {
		return 0, errors.New("EOF")
	}
	n := copy(p, f.reports[0])
	f.reports = f.reports[1:]
	return n, nil
}

func (f *fakeDev) SetFeature(b []byte) error {
	f.features = append(f.features, append([]byte(nil), b...))
	return nil
}

func TestReaderReEnablesIMUWhenSteamTurnsItOff(t *testing.T) {
	on := fixture(t, "report_imu_on.bin")
	off := fixture(t, "report_imu_off.bin")
	dev := &fakeDev{}
	for i := 0; i < 3; i++ {
		dev.reports = append(dev.reports, on)
	}
	for i := 0; i < 30; i++ {
		dev.reports = append(dev.reports, off)
	}
	r := NewReader(dev, 10) // re-enable after 10 consecutive empty samples

	var got []Sample
	err := r.Run(func(s Sample) { got = append(got, s) })
	if err == nil || err.Error() != "EOF" {
		t.Fatalf("Run err = %v", err)
	}
	if len(got) != 3 {
		t.Errorf("delivered %d samples, want only the 3 with IMU data", len(got))
	}
	// Once at start, then once per 10 empty samples (30 / 10 = 3).
	if len(dev.features) != 4 {
		t.Errorf("sent %d enable reports, want 4", len(dev.features))
	}
	for _, f := range dev.features {
		if !bytes.Equal(f, EnableReport()) {
			t.Errorf("unexpected feature report % x", f[:8])
		}
	}
}

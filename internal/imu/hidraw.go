package imu

import (
	"os"
	"syscall"
	"unsafe"
)

func hidiocsfeature(n int) uintptr {
	const iocRead, iocWrite = 2, 1
	return uintptr((iocRead|iocWrite)<<30 | n<<16 | 'H'<<8 | 0x06)
}

// HIDRaw is a /dev/hidrawN device.
type HIDRaw struct{ f *os.File }

func OpenHIDRaw(path string) (*HIDRaw, error) {
	f, err := os.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		return nil, err
	}
	return &HIDRaw{f: f}, nil
}

func (h *HIDRaw) Read(p []byte) (int, error) { return h.f.Read(p) }
func (h *HIDRaw) Close() error               { return h.f.Close() }

func (h *HIDRaw) SetFeature(report []byte) error {
	rc, err := h.f.SyscallConn()
	if err != nil {
		return err
	}
	var errno syscall.Errno
	err = rc.Control(func(fd uintptr) {
		_, _, errno = syscall.Syscall(syscall.SYS_IOCTL, fd, hidiocsfeature(len(report)), uintptr(unsafe.Pointer(&report[0])))
	})
	if err != nil {
		return err
	}
	if errno != 0 {
		return errno
	}
	return nil
}

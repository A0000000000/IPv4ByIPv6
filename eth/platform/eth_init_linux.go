package platform

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"syscall"
	"unsafe"
)

func CreateTunDeviceInner(name string, segment, number uint32) (error, io.ReadWriteCloser) {
	var arr [0x28]byte
	copy(arr[0:0x10], name)
	arr[0x10] = 0x01
	arr[0x11] = 0x10
	fd, err := syscall.Open("/dev/net/tun", os.O_RDWR|syscall.O_NONBLOCK, 0)
	if err != nil {
		return err, nil
	}
	_, _, err = syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), uintptr(syscall.TUNSETIFF), uintptr(unsafe.Pointer(&arr)))
	if err != nil && !errors.Is(err, syscall.Errno(0x0)) {
		return err, nil
	}
	_, _, err = syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), uintptr(syscall.TUNSETPERSIST), uintptr(0))
	if err != nil && !errors.Is(err, syscall.Errno(0x0)) {
		return err, nil
	}
	cmd1 := exec.Command("ip", "addr", "add", fmt.Sprintf("192.168.%d.%d/24", segment, number), "dev", name)
	cmd2 := exec.Command("ip", "link", "set", "dev", name, "up")
	err = cmd1.Run()
	if err != nil {
		return err, nil
	}
	err = cmd2.Run()
	if err != nil {
		return err, nil
	}
	return nil, os.NewFile(uintptr(fd), "tun")
}

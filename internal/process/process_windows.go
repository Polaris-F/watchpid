// +build windows

package process

import (
	"path/filepath"
	"strconv"
	"syscall"
	"time"
	"unsafe"
)

const (
	processQueryLimitedInformation = 0x1000
	stillActive                    = 259
	errorInvalidParameter          = syscall.Errno(87)
)

var (
	modkernel32                   = syscall.NewLazyDLL("kernel32.dll")
	procOpenProcess               = modkernel32.NewProc("OpenProcess")
	procCloseHandle               = modkernel32.NewProc("CloseHandle")
	procGetProcessTimes           = modkernel32.NewProc("GetProcessTimes")
	procGetExitCodeProcess        = modkernel32.NewProc("GetExitCodeProcess")
	procQueryFullProcessImageName = modkernel32.NewProc("QueryFullProcessImageNameW")
)

func Inspect(pid int) (Snapshot, error) {
	handle, err := openProcessHandle(pid)
	if err != nil {
		if err == errorInvalidParameter {
			return Snapshot{}, ErrNotFound
		}
		return Snapshot{}, err
	}
	defer closeHandle(handle)

	exitCode, err := getExitCode(handle)
	if err != nil {
		return Snapshot{}, err
	}
	if exitCode != stillActive {
		return Snapshot{}, ErrNotFound
	}

	createdAt, err := getProcessCreateTime(handle)
	if err != nil {
		return Snapshot{}, err
	}

	image, _ := getProcessImage(handle)
	name := image
	if name != "" {
		name = filepath.Base(name)
	}
	token := strconv.FormatInt(createdAt.UnixNano(), 10)
	return Snapshot{
		PID:         pid,
		Name:        name,
		Command:     []string{image},
		CreateToken: token,
		StartedAt:   &createdAt,
	}, nil
}

func openProcessHandle(pid int) (syscall.Handle, error) {
	r1, _, err := procOpenProcess.Call(uintptr(processQueryLimitedInformation), uintptr(0), uintptr(uint32(pid)))
	if r1 == 0 {
		if err != syscall.Errno(0) {
			return 0, err
		}
		return 0, syscall.EINVAL
	}
	return syscall.Handle(r1), nil
}

func closeHandle(handle syscall.Handle) {
	_, _, _ = procCloseHandle.Call(uintptr(handle))
}

func getExitCode(handle syscall.Handle) (uint32, error) {
	var code uint32
	r1, _, err := procGetExitCodeProcess.Call(uintptr(handle), uintptr(unsafe.Pointer(&code)))
	if r1 == 0 {
		if err != syscall.Errno(0) {
			return 0, err
		}
		return 0, syscall.EINVAL
	}
	return code, nil
}

func getProcessCreateTime(handle syscall.Handle) (time.Time, error) {
	var createTime syscall.Filetime
	var exitTime syscall.Filetime
	var kernelTime syscall.Filetime
	var userTime syscall.Filetime
	r1, _, err := procGetProcessTimes.Call(
		uintptr(handle),
		uintptr(unsafe.Pointer(&createTime)),
		uintptr(unsafe.Pointer(&exitTime)),
		uintptr(unsafe.Pointer(&kernelTime)),
		uintptr(unsafe.Pointer(&userTime)),
	)
	if r1 == 0 {
		if err != syscall.Errno(0) {
			return time.Time{}, err
		}
		return time.Time{}, syscall.EINVAL
	}
	return time.Unix(0, createTime.Nanoseconds()), nil
}

func getProcessImage(handle syscall.Handle) (string, error) {
	buffer := make([]uint16, syscall.MAX_PATH)
	size := uint32(len(buffer))
	r1, _, err := procQueryFullProcessImageName.Call(
		uintptr(handle),
		uintptr(0),
		uintptr(unsafe.Pointer(&buffer[0])),
		uintptr(unsafe.Pointer(&size)),
	)
	if r1 == 0 {
		if err != syscall.Errno(0) {
			return "", err
		}
		return "", syscall.EINVAL
	}
	return syscall.UTF16ToString(buffer[:size]), nil
}

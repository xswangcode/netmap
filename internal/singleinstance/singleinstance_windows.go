//go:build windows

package singleinstance

import (
	"fmt"
	"syscall"
	"unsafe"
)

var (
	kernel32         = syscall.NewLazyDLL("kernel32.dll")
	procCreateMutexW = kernel32.NewProc("CreateMutexW")
	procCloseHandle  = kernel32.NewProc("CloseHandle")
	procReleaseMutex = kernel32.NewProc("ReleaseMutex")
)

const errorAlreadyExists syscall.Errno = 183

// Instance NetMap 单实例锁。
type Instance struct {
	handle syscall.Handle
}

// Acquire 创建 NetMap 单实例锁。
//
// 如果已经有一个 NetMap 运行，则返回：
// alreadyRunning = true
func Acquire(name string) (*Instance, bool, error) {
	mutexName, err := syscall.UTF16PtrFromString(name)
	if err != nil {
		return nil, false, fmt.Errorf(
			"create mutex name failed: %w",
			err,
		)
	}

	handle, _, callErr := procCreateMutexW.Call(
		0,
		0,
		uintptr(unsafe.Pointer(mutexName)),
	)

	if handle == 0 {
		return nil, false, fmt.Errorf(
			"create mutex failed: %w",
			callErr,
		)
	}

	if callErr == errorAlreadyExists {
		_, _, _ = procCloseHandle.Call(handle)

		return nil, true, nil
	}

	return &Instance{
		handle: syscall.Handle(handle),
	}, false, nil
}

// Release 释放单实例锁。
func (i *Instance) Release() {
	if i == nil || i.handle == 0 {
		return
	}

	_, _, _ = procReleaseMutex.Call(
		uintptr(i.handle),
	)

	_, _, _ = procCloseHandle.Call(
		uintptr(i.handle),
	)

	i.handle = 0
}

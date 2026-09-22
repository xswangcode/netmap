//go:build windows

package singleinstance

import (
	"fmt"
	"syscall"
	"unsafe"

	"netmap/internal/logger"
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
//
// 日志级别约定：
//   - logger.Info ：获取成功 / 检测到已有实例 / 释放
//   - logger.Error：系统调用失败
func Acquire(name string) (*Instance, bool, error) {
	logger.Debug(
		"singleinstance: acquiring mutex: name=%s",
		name,
	)

	mutexName, err := syscall.UTF16PtrFromString(name)
	if err != nil {
		logger.Error(
			"singleinstance: create mutex name failed: name=%s error=%v",
			name,
			err,
		)

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
		logger.Error(
			"singleinstance: create mutex failed: name=%s error=%v",
			name,
			callErr,
		)

		return nil, false, fmt.Errorf(
			"create mutex failed: %w",
			callErr,
		)
	}

	if callErr == errorAlreadyExists {
		_, _, _ = procCloseHandle.Call(handle)

		// 应用层面：检测到已有实例在运行。
		// 这是预期内的业务结果，不是错误。
		logger.Info(
			"singleinstance: another instance is already running: name=%s",
			name,
		)

		return nil, true, nil
	}

	// 应用层面：成功拿到单实例锁。
	logger.Info(
		"singleinstance: mutex acquired: name=%s",
		name,
	)

	return &Instance{
		handle: syscall.Handle(handle),
	}, false, nil
}

// Release 释放单实例锁。
func (i *Instance) Release() {
	if i == nil || i.handle == 0 {
		logger.Debug(
			"singleinstance: release ignored, no handle",
		)

		return
	}

	if _, _, err := procReleaseMutex.Call(
		uintptr(i.handle),
	); err != syscall.Errno(0) {
		// ReleaseMutex 失败通常是句柄已无效，
		// 属于需要关注但不应中断流程的情况。
		logger.Error(
			"singleinstance: release mutex failed: error=%v",
			err,
		)
	}

	if _, _, err := procCloseHandle.Call(
		uintptr(i.handle),
	); err != syscall.Errno(0) {
		logger.Error(
			"singleinstance: close handle failed: error=%v",
			err,
		)
	}

	i.handle = 0

	logger.Info(
		"singleinstance: mutex released",
	)
}

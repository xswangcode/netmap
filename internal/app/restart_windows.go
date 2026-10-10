//go:build windows

package app

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"syscall"
)

const restartAfterFlag = "--netmap-restart-after"
const processSynchronize = 0x00100000
const waitInfinite = 0xffffffff

var (
	kernel32              = syscall.NewLazyDLL("kernel32.dll")
	procOpenRestartTarget = kernel32.NewProc("OpenProcess")
	procWaitRestartTarget = kernel32.NewProc("WaitForSingleObject")
	procCloseRestart      = kernel32.NewProc("CloseHandle")
)

// ScheduleRestart starts a helper that waits for this process to exit before
// relaunching NetMap with the selected configuration.
func ScheduleRestart(configPath string) error {
	executable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve executable path: %w", err)
	}

	command := exec.Command(
		executable,
		restartAfterFlag,
		strconv.Itoa(os.Getpid()),
		"--config",
		configPath,
	)
	if err := command.Start(); err != nil {
		return fmt.Errorf("start restart helper: %w", err)
	}

	return nil
}

// RunRestartHelper handles the private helper invocation before the normal
// single-instance lock is acquired.
func RunRestartHelper(args []string) (bool, error) {
	if len(args) == 0 || args[0] != restartAfterFlag {
		return false, nil
	}
	if len(args) != 4 || args[2] != "--config" {
		return true, fmt.Errorf("invalid restart helper arguments")
	}

	parentPID, err := strconv.Atoi(args[1])
	if err != nil || parentPID <= 0 {
		return true, fmt.Errorf("invalid restart parent process ID %q", args[1])
	}
	if err := waitForProcess(parentPID); err != nil {
		return true, fmt.Errorf("wait for previous process: %w", err)
	}

	executable, err := os.Executable()
	if err != nil {
		return true, fmt.Errorf("resolve executable path: %w", err)
	}
	command := exec.Command(executable, "--config", args[3])
	if err := command.Start(); err != nil {
		return true, fmt.Errorf("start NetMap: %w", err)
	}

	return true, nil
}

func waitForProcess(pid int) error {
	handle, _, callErr := procOpenRestartTarget.Call(
		processSynchronize,
		0,
		uintptr(uint32(pid)),
	)
	if handle == 0 {
		if callErr == syscall.Errno(87) {
			return nil
		}
		return fmt.Errorf("open process %d: %w", pid, callErr)
	}
	defer procCloseRestart.Call(handle)

	result, _, waitErr := procWaitRestartTarget.Call(
		handle,
		waitInfinite,
	)
	if result != 0 {
		return fmt.Errorf("wait for process %d: %w", pid, waitErr)
	}

	return nil
}

//go:build windows

package daemon

import (
	"fmt"
	"os"
	"syscall"
	"time"
)

// procAttrs returns Windows process attributes.
// CREATE_NEW_PROCESS_GROUP + DETACHED_PROCESS fully decouple the child from the
// parent console so it survives after the parent terminal closes.
func procAttrs() *syscall.SysProcAttr {
	const detachedProcess = 0x00000008 // DETACHED_PROCESS
	return &syscall.SysProcAttr{
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP | detachedProcess,
	}
}

// IsRunning checks whether the daemon is alive on Windows.
// Signal(0) is meaningless on Windows (returns EWINDOWS for any non-SIGKILL
// signal), so we use OpenProcess + WaitForSingleObject(0ms) instead.
func IsRunning() (bool, int, error) {
	pid, err := readPIDFile()
	if err != nil || pid == 0 {
		return false, 0, err
	}

	const (
		processQueryLimitedInformation = 0x1000
		waitTimeout                    = 258 // WAIT_TIMEOUT — process still running
	)

	handle, err := syscall.OpenProcess(processQueryLimitedInformation, false, uint32(pid))
	if err != nil {
		// Process not found or access denied — treat as gone.
		return false, pid, nil
	}
	defer syscall.CloseHandle(handle) //nolint:errcheck

	// 0ms timeout: returns immediately.
	// WAIT_OBJECT_0 (0)   → process exited
	// WAIT_TIMEOUT  (258) → process still running
	event, err := syscall.WaitForSingleObject(handle, 0)
	if err != nil {
		return false, pid, nil
	}
	return event == waitTimeout, pid, nil
}

// Stop terminates the daemon process on Windows.
func Stop() error {
	alive, pid, err := IsRunning()
	if err != nil {
		return err
	}
	if !alive {
		return fmt.Errorf("daemon is not running")
	}

	proc, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("finding process %d: %w", pid, err)
	}
	if err := proc.Kill(); err != nil {
		return fmt.Errorf(
			"could not stop daemon (PID %d): %w\n"+
				"If the daemon was started by a different user or as Administrator,\n"+
				"stop it manually:  taskkill /PID %d /F",
			pid, err, pid)
	}

	for i := 0; i < 50; i++ {
		time.Sleep(100 * time.Millisecond)
		if alive, _, _ := IsRunning(); !alive {
			removePID()
			fmt.Println("taildog daemon stopped")
			return nil
		}
	}
	removePID()
	fmt.Println("taildog daemon stopped")
	return nil
}

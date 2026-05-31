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
// We use OpenProcess + GetExitCodeProcess: GetExitCodeProcess requires only
// PROCESS_QUERY_LIMITED_INFORMATION (WaitForSingleObject additionally needs
// SYNCHRONIZE which we don't request, so it would fail with access-denied).
// GetExitCodeProcess returns STILL_ACTIVE (259) for a running process.
func IsRunning() (bool, int, error) {
	pid, err := readPIDFile()
	if err != nil || pid == 0 {
		return false, 0, err
	}

	const processQueryLimitedInformation = 0x1000

	handle, err := syscall.OpenProcess(processQueryLimitedInformation, false, uint32(pid))
	if err != nil {
		// Process not found or access denied — treat as gone.
		return false, pid, nil
	}
	defer syscall.CloseHandle(handle) //nolint:errcheck

	var exitCode uint32
	if err := syscall.GetExitCodeProcess(handle, &exitCode); err != nil {
		return false, pid, nil
	}
	// STILL_ACTIVE (259) means the process has not exited yet.
	return exitCode == 259, pid, nil
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
			fmt.Println("renode daemon stopped")
			return nil
		}
	}
	removePID()
	fmt.Println("renode daemon stopped")
	return nil
}

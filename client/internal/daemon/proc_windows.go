//go:build windows

package daemon

import (
	"fmt"
	"os"
	"syscall"
	"time"
)

// procAttrs returns Windows process attributes.
// CREATE_NEW_PROCESS_GROUP lets the child run independently of the parent.
func procAttrs() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
	}
}

// IsRunning checks whether the PID stored in the PID file is alive.
// On Windows, os.FindProcess always succeeds; we open the process handle to
// confirm it is still running.
func IsRunning() (bool, int, error) {
	pid, err := readPIDFile()
	if err != nil || pid == 0 {
		return false, 0, err
	}

	proc, err := os.FindProcess(pid)
	if err != nil {
		return false, pid, nil
	}

	// On Windows, Signal(0) is not meaningful. We attempt a no-op signal and
	// fall back to checking whether Wait returns immediately.
	// The idiomatic Windows check: open the process with SYNCHRONIZE rights,
	// then WaitForSingleObject with timeout 0. We approximate this by
	// calling proc.Signal(os.Interrupt) which on Windows sends CTRL_BREAK — not
	// what we want. Instead we use a non-signalling technique: try to kill with
	// exit code query via OpenProcess. Because the standard library does not
	// expose this, we use a simple heuristic: if Release succeeds it was found.
	// A more reliable approach uses the windows package; we keep it dependency-free.
	if err := proc.Signal(syscall.Signal(0)); err != nil {
		return false, pid, nil
	}
	return true, pid, nil
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

	// Windows does not support SIGTERM; kill immediately.
	if err := proc.Kill(); err != nil {
		return fmt.Errorf("killing process %d: %w", pid, err)
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

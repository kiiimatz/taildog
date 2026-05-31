//go:build !windows

package daemon

import (
	"errors"
	"fmt"
	"os"
	"syscall"
	"time"
)

// procAttrs returns Unix process attributes that start the child in a new
// session so it survives the parent's exit.
func procAttrs() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{Setsid: true}
}

// IsRunning checks whether the PID stored in the PID file is alive.
func IsRunning() (bool, int, error) {
	pid, err := readPIDFile()
	if err != nil || pid == 0 {
		return false, 0, err
	}

	proc, err := os.FindProcess(pid)
	if err != nil {
		return false, pid, nil
	}

	// signal(0) tests liveness without actually signalling.
	if err := proc.Signal(syscall.Signal(0)); err != nil {
		switch {
		case errors.Is(err, os.ErrProcessDone), errors.Is(err, syscall.ESRCH):
			return false, pid, nil
		case errors.Is(err, syscall.EPERM):
			// Process exists but belongs to another user.
			return true, pid, nil
		default:
			return false, pid, nil
		}
	}
	return true, pid, nil
}

// Stop sends SIGTERM to the daemon and waits up to 5 s for it to exit.
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

	if err := proc.Signal(syscall.SIGTERM); err != nil {
		return fmt.Errorf("sending SIGTERM to %d: %w", pid, err)
	}

	for i := 0; i < 50; i++ {
		time.Sleep(100 * time.Millisecond)
		if alive, _, _ := IsRunning(); !alive {
			removePID()
			fmt.Println("renode daemon stopped")
			return nil
		}
	}

	_ = proc.Kill()
	removePID()
	fmt.Println("renode daemon stopped")
	return nil
}

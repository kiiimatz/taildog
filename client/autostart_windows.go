//go:build windows

package main

import (
	"fmt"
	"os"
	"os/exec"
)

func installAutostart() {
	exe, err := os.Executable()
	if err != nil {
		fmt.Printf("autostart: cannot determine executable path: %v\n", err)
		return
	}

	// Register a Task Scheduler task that runs at logon with highest privileges.
	// /F overwrites any existing task with the same name.
	out, err := exec.Command("schtasks",
		"/Create", "/F",
		"/TN", "Taildog",
		"/SC", "ONLOGON",
		"/TR", fmt.Sprintf(`"%s" up --foreground`, exe),
		"/RL", "HIGHEST",
	).CombinedOutput()
	if err != nil {
		fmt.Printf("autostart: could not register startup task: %v\n%s\n", err, out)
		return
	}
	fmt.Println("taildog registered to start at login (Task Scheduler)")
}

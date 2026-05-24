//go:build windows

package main

import (
	"fmt"
	"os"
	"os/exec"
)

// installAutostart writes a per-user autostart entry to the Windows registry.
// HKCU\...\Run does NOT require administrator privileges and fires every time
// the current user logs in.
func installAutostart() {
	exe, err := os.Executable()
	if err != nil {
		fmt.Printf("autostart: cannot determine executable path: %v\n", err)
		return
	}

	out, err := exec.Command("reg", "add",
		`HKCU\SOFTWARE\Microsoft\Windows\CurrentVersion\Run`,
		"/v", "Taildog",
		"/t", "REG_SZ",
		"/d", fmt.Sprintf(`"%s" up --foreground`, exe),
		"/f",
	).CombinedOutput()
	if err != nil {
		fmt.Printf("autostart: could not register startup entry: %v\n%s\n", err, out)
		return
	}
	fmt.Println("taildog registered to start at login (HKCU registry)")
}

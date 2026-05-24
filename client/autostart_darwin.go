//go:build darwin

package main

import (
	"fmt"
	"os"
	"os/exec"
)

const launchPlist = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.taildog.daemon</string>
    <key>ProgramArguments</key>
    <array>
        <string>%s</string>
        <string>up</string>
        <string>--foreground</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>/var/log/taildog.log</string>
    <key>StandardErrorPath</key>
    <string>/var/log/taildog.log</string>
</dict>
</plist>
`

func installAutostart() {
	exe, err := os.Executable()
	if err != nil {
		fmt.Printf("autostart: cannot determine executable path: %v\n", err)
		return
	}

	const plistPath = "/Library/LaunchDaemons/com.taildog.daemon.plist"
	if err := os.WriteFile(plistPath, []byte(fmt.Sprintf(launchPlist, exe)), 0644); err != nil {
		fmt.Printf("autostart: writing %s: %v\n", plistPath, err)
		return
	}

	// Load immediately (idempotent — bootout first in case it was loaded before).
	exec.Command("launchctl", "bootout", "system/com.taildog.daemon").Run() //nolint:errcheck
	if out, err := exec.Command("launchctl", "bootstrap", "system", plistPath).CombinedOutput(); err != nil {
		fmt.Printf("autostart: launchctl bootstrap: %v\n%s\n", err, out)
		return
	}

	fmt.Println("taildog launch daemon installed — will start automatically at boot")
}

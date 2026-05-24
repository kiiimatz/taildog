//go:build linux

package main

import (
	"fmt"
	"os"
	"os/exec"
)

const systemdUnit = `[Unit]
Description=Taildog tunnel daemon
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=%s up --foreground
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
`

func installAutostart() {
	exe, err := os.Executable()
	if err != nil {
		fmt.Printf("autostart: cannot determine executable path: %v\n", err)
		return
	}

	const unitPath = "/etc/systemd/system/taildog.service"
	if err := os.WriteFile(unitPath, []byte(fmt.Sprintf(systemdUnit, exe)), 0644); err != nil {
		fmt.Printf("autostart: writing %s: %v\n", unitPath, err)
		return
	}

	if out, err := exec.Command("systemctl", "daemon-reload").CombinedOutput(); err != nil {
		fmt.Printf("autostart: systemctl daemon-reload: %v\n%s\n", err, out)
		return
	}
	if out, err := exec.Command("systemctl", "enable", "taildog").CombinedOutput(); err != nil {
		fmt.Printf("autostart: systemctl enable taildog: %v\n%s\n", err, out)
		return
	}

	fmt.Println("taildog service enabled — will start automatically on next boot")
}

package server

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"
)

// ufwAvailable returns true when we are on Linux and the ufw binary is present.
func ufwAvailable() bool {
	if runtime.GOOS != "linux" {
		return false
	}
	_, err := exec.LookPath("ufw")
	return err == nil
}

func runUFW(args ...string) error {
	cmd := exec.Command("sudo", append([]string{"ufw"}, args...)...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// ufwAllow opens port/proto in UFW. No-op on non-Linux or when ufw is absent.
func ufwAllow(port int, proto string) {
	if !ufwAvailable() {
		return
	}
	rule := fmt.Sprintf("%d/%s", port, proto)
	if err := runUFW("allow", rule); err != nil {
		log.Printf("ufw allow %s: %v", rule, err)
	}
}

// ufwDelete removes the allow rule for port/proto from UFW.
func ufwDelete(port int, proto string) {
	if !ufwAvailable() {
		return
	}
	rule := fmt.Sprintf("%d/%s", port, proto)
	if err := runUFW("delete", "allow", rule); err != nil {
		log.Printf("ufw delete allow %s: %v", rule, err)
	}
}

//go:build !windows

package main

import (
	"fmt"
	"os"
)

func requireAdmin() error {
	if os.Getuid() != 0 {
		return fmt.Errorf("renode-relay must be run as root\n\n  sudo %s", os.Args[0])
	}
	return nil
}

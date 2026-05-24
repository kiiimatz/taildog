//go:build !windows && !linux && !darwin

package main

import "fmt"

func installAutostart() {
	fmt.Println("autostart: not supported on this platform")
}

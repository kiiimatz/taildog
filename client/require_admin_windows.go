//go:build windows

package main

import (
	"fmt"
	"syscall"
	"unsafe"
)

func requireAdmin() error {
	// OpenCurrentProcessToken opens the access token of the calling process.
	token, err := syscall.OpenCurrentProcessToken()
	if err != nil {
		// Cannot determine elevation — allow through rather than block.
		return nil
	}
	defer syscall.CloseHandle(syscall.Handle(token))

	// TokenElevation (20) returns a DWORD: 0 = not elevated, 1 = elevated.
	const tokenElevation = 20
	var elevation uint32
	var returnedLen uint32
	if err := syscall.GetTokenInformation(
		token,
		tokenElevation,
		(*byte)(unsafe.Pointer(&elevation)),
		uint32(unsafe.Sizeof(elevation)),
		&returnedLen,
	); err != nil {
		// Cannot determine elevation — allow through.
		return nil
	}
	if elevation == 0 {
		return fmt.Errorf("taildog must be run as Administrator\n\nRight-click your terminal and choose \"Run as administrator\", then try again.")
	}
	return nil
}

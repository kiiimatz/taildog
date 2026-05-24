//go:build windows

package main

import (
	"fmt"
	"syscall"
	"unsafe"
)

func requireAdmin() error {
	token, err := syscall.OpenCurrentProcessToken()
	if err != nil {
		return nil
	}
	defer syscall.CloseHandle(syscall.Handle(token))

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
		return nil
	}
	if elevation == 0 {
		return fmt.Errorf("taildog-relay must be run as Administrator\n\nRight-click your terminal and choose \"Run as administrator\", then try again.")
	}
	return nil
}

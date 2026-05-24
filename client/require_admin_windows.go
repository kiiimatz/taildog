//go:build windows

package main

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"syscall"
	"unsafe"
)

func requireAdmin() error {
	if isElevated() {
		return nil
	}
	return elevate()
}

func isElevated() bool {
	token, err := syscall.OpenCurrentProcessToken()
	if err != nil {
		return false
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
		return false
	}
	return elevation != 0
}

// elevate re-launches the current command with administrator privileges via
// ShellExecuteW("runas"), triggering a UAC prompt, then exits this instance.
func elevate() error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("cannot determine executable path: %w", err)
	}

	// Rebuild the argument string from os.Args (skip program name, quote if needed).
	var parts []string
	for _, a := range os.Args[1:] {
		if strings.ContainsAny(a, " \t") {
			parts = append(parts, `"`+a+`"`)
		} else {
			parts = append(parts, a)
		}
	}
	argsStr := strings.Join(parts, " ")

	shell32 := syscall.NewLazyDLL("shell32.dll")
	shellExecuteW := shell32.NewProc("ShellExecuteW")

	verbPtr, _ := syscall.UTF16PtrFromString("runas")
	exePtr, _ := syscall.UTF16PtrFromString(exe)
	argsPtr, _ := syscall.UTF16PtrFromString(argsStr)

	// SW_SHOWNORMAL = 1 so the elevated console window is visible.
	ret, _, _ := shellExecuteW.Call(
		0,
		uintptr(unsafe.Pointer(verbPtr)),
		uintptr(unsafe.Pointer(exePtr)),
		uintptr(unsafe.Pointer(argsPtr)),
		0,
		1, // SW_SHOWNORMAL
	)
	runtime.KeepAlive(verbPtr)
	runtime.KeepAlive(exePtr)
	runtime.KeepAlive(argsPtr)

	if ret <= 32 {
		return fmt.Errorf("UAC elevation failed (code %d) — right-click your terminal and choose \"Run as administrator\"", ret)
	}

	// The elevated process is now running; exit this non-elevated instance.
	os.Exit(0)
	return nil
}

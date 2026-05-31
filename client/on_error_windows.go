//go:build windows

package main

import (
	"runtime"
	"syscall"
	"unsafe"
)

// platformOnError shows a Windows MessageBox so the error is visible even
// when the console window (launched via UAC/ShellExecuteW) closes immediately.
func platformOnError(msg string) {
	user32 := syscall.NewLazyDLL("user32.dll")
	messageBoxW := user32.NewProc("MessageBoxW")

	titlePtr, _ := syscall.UTF16PtrFromString("Renode Error")
	msgPtr, _ := syscall.UTF16PtrFromString(msg)

	const mbIconError = 0x10 // MB_ICONERROR
	messageBoxW.Call(
		0,
		uintptr(unsafe.Pointer(msgPtr)),
		uintptr(unsafe.Pointer(titlePtr)),
		mbIconError,
	)
	runtime.KeepAlive(titlePtr)
	runtime.KeepAlive(msgPtr)
}

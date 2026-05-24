//go:build !windows

package main

// platformOnError is a no-op on non-Windows platforms — Cobra already
// writes the error to stderr.
func platformOnError(_ string) {}

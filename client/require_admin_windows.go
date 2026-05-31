//go:build windows

package main

// requireAdmin is intentionally a no-op on Windows.
//
// The renode daemon communicates over outbound TCP and reads/writes only to
// the current user's AppData directory — neither of those requires elevated
// privileges.  Running every command through UAC only breaks status, down, and
// logs because ShellExecuteW opens a new console window that closes before
// the user can read the output.
//
// The only Windows-specific operation that *would* need elevation is system-
// wide service registration, but renode uses per-user autostart (HKCU
// registry) instead, which does not require admin.
func requireAdmin() error {
	return nil
}

//go:build !windows

package auth

import (
	"fmt"
	"os"

	"golang.org/x/term"
)

// readPasswordUnix reads a password with echo disabled using golang.org/x/term.
func readPasswordUnix(_ string) (string, error) {
	fd := int(os.Stdin.Fd())
	raw, err := term.ReadPassword(fd)
	if err != nil {
		return "", fmt.Errorf("read password: %w", err)
	}
	fmt.Println() // restore newline suppressed by ReadPassword
	return string(raw), nil
}

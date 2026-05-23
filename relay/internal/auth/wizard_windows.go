//go:build windows

package auth

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// readPasswordUnix is the Windows stub — echo cannot be disabled portably here,
// so we fall through to the plain bufio reader already handled in wizard.go.
// This function is called only from promptPassword which already gates on
// runtime.GOOS, so this body should never execute; it exists only to satisfy
// the build constraint split.
func readPasswordUnix(_ string) (string, error) {
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("read password: %w", err)
	}
	fmt.Println()
	return strings.TrimSpace(line), nil
}

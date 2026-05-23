// Package auth — first-run console wizard.
package auth

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"regexp"
	"runtime"
	"strings"
	"unicode"

	"github.com/kiiimatz/taildog/relay/internal/db"
)

var usernameRE = regexp.MustCompile(`^[a-zA-Z0-9_-]{3,32}$`)

// RunWizard runs an interactive first-time setup wizard that creates the
// initial admin account. It keeps prompting until valid credentials are given.
func RunWizard(database *db.DB) error {
	printBanner()

	var username string
	for {
		u, err := promptLine("  Username: ")
		if err != nil {
			return err
		}
		if !usernameRE.MatchString(u) {
			fmt.Fprintln(os.Stderr, "  ! Username must be 3–32 characters: letters, digits, hyphen, underscore.")
			continue
		}
		username = u
		break
	}

	var password string
	for {
		p, err := promptPassword("  Password: ")
		if err != nil {
			return err
		}
		if err := validatePassword(p); err != nil {
			fmt.Fprintf(os.Stderr, "  ! %s\n", err)
			continue
		}

		c, err := promptPassword("  Confirm password: ")
		if err != nil {
			return err
		}
		if p != c {
			fmt.Fprintln(os.Stderr, "  ! Passwords do not match.")
			continue
		}
		password = p
		break
	}

	fmt.Println()
	fmt.Println("  [Enter to confirm]")

	// Wait for Enter.
	reader := bufio.NewReader(os.Stdin)
	reader.ReadString('\n') //nolint:errcheck

	printDivider()

	hash, err := HashPassword(password)
	// Zero the plaintext immediately after hashing.
	password = strings.Repeat("\x00", len(password))
	_ = password
	if err != nil {
		return fmt.Errorf("wizard: hash password: %w", err)
	}

	if err := database.CreateUser(username, hash, "admin"); err != nil {
		return fmt.Errorf("wizard: create user: %w", err)
	}

	fmt.Printf("  Admin account '%s' created. Enjoy taildog!\n", username)
	printDivider()
	return nil
}

// validatePassword enforces complexity rules.
func validatePassword(p string) error {
	if len(p) < 12 {
		return errors.New("password must be at least 12 characters")
	}
	var hasUpper, hasLower, hasDigit, hasSymbol bool
	for _, r := range p {
		switch {
		case unicode.IsUpper(r):
			hasUpper = true
		case unicode.IsLower(r):
			hasLower = true
		case unicode.IsDigit(r):
			hasDigit = true
		case unicode.IsPunct(r) || unicode.IsSymbol(r):
			hasSymbol = true
		}
	}
	if !hasUpper {
		return errors.New("password must contain at least one uppercase letter")
	}
	if !hasLower {
		return errors.New("password must contain at least one lowercase letter")
	}
	if !hasDigit {
		return errors.New("password must contain at least one digit")
	}
	if !hasSymbol {
		return errors.New("password must contain at least one symbol")
	}
	return nil
}

func printBanner() {
	printDivider()
	fmt.Println("  taildog — First-time setup")
	printDivider()
	fmt.Println("  Welcome! Let's create your admin account.")
	fmt.Println()
}

func printDivider() {
	fmt.Println("─────────────────────────────────────────")
}

// promptLine reads a single line from stdin with a visible prompt.
func promptLine(prompt string) (string, error) {
	fmt.Print(prompt)
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(line), nil
}

// promptPassword reads a password from the terminal with echo disabled on
// Unix systems. Falls back to a plain bufio read on Windows.
func promptPassword(prompt string) (string, error) {
	fmt.Print(prompt)

	if runtime.GOOS == "windows" {
		// Windows fallback: visible input (best effort).
		reader := bufio.NewReader(os.Stdin)
		line, err := reader.ReadString('\n')
		if err != nil {
			return "", err
		}
		fmt.Println() // newline after input
		return strings.TrimSpace(line), nil
	}

	return readPasswordUnix(prompt)
}

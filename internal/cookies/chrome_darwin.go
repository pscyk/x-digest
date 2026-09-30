package cookies

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

func chromeUserDataDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("finding home directory: %w", err)
	}
	return filepath.Join(home, "Library", "Application Support", "Google", "Chrome"), nil
}

func chromeSafeStoragePassword() ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	// Capture stdout in memory; never print or persist the Keychain secret.
	secret, err := exec.CommandContext(ctx, "/usr/bin/security", "find-generic-password",
		"-w", "-s", "Chrome Safe Storage", "-a", "Chrome").Output()
	if err != nil {
		clear(secret)
		return nil, fmt.Errorf("cannot read Chrome Safe Storage; unlock your login Keychain and allow the macOS access prompt, then retry")
	}
	secret = bytes.TrimSuffix(secret, []byte("\n"))
	if len(secret) == 0 {
		return nil, fmt.Errorf("Chrome Safe Storage is empty; open Chrome and retry")
	}
	return secret, nil
}

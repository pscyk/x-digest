//go:build !darwin

package cookies

import "fmt"

func chromeUserDataDir() (string, error) {
	return "", fmt.Errorf("--browser chrome currently supports macOS only; use --browser firefox on this platform")
}

func chromeSafeStoragePassword() ([]byte, error) {
	return nil, fmt.Errorf("Chrome Safe Storage requires macOS")
}

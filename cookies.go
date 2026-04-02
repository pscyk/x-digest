package main

import (
	"bufio"
	"database/sql"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
)

const (
	cookieQuery = `
		SELECT name, value FROM moz_cookies
		WHERE (host = '.x.com' OR host = '.twitter.com')
		  AND name IN ('auth_token', 'ct0')
		  AND originAttributes = ''
	`
	// WAL files needed for consistent reads while Firefox is running.
	dbFile  = "cookies.sqlite"
	walFile = "cookies.sqlite-wal"
	shmFile = "cookies.sqlite-shm"
)

// extractCookies reads auth_token and ct0 from Firefox's cookie store.
func extractCookies(profileOverride string) (authToken, ct0 string, err error) {
	profileDir, err := findFirefoxProfile(profileOverride)
	if err != nil {
		return "", "", err
	}

	dbPath := filepath.Join(profileDir, dbFile)
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		return "", "", fmt.Errorf("cookies.sqlite not found at %s — are you logged into X in Firefox?", dbPath)
	}

	// Copy DB + WAL files to temp dir (Firefox holds a WAL lock while running).
	tmpDir, err := os.MkdirTemp("", "x-digest-")
	if err != nil {
		return "", "", fmt.Errorf("creating temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	for _, name := range []string{dbFile, walFile, shmFile} {
		src := filepath.Join(profileDir, name)
		dst := filepath.Join(tmpDir, name)
		if cpErr := copyFile(src, dst); cpErr != nil && name == dbFile {
			return "", "", fmt.Errorf("copying %s: %w", name, cpErr)
		}
	}

	db, err := sql.Open("sqlite", filepath.Join(tmpDir, dbFile)+"?mode=ro")
	if err != nil {
		return "", "", fmt.Errorf("opening cookie db: %w", err)
	}
	defer db.Close()

	rows, err := db.Query(cookieQuery)
	if err != nil {
		return "", "", fmt.Errorf("querying cookies: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var name, value string
		if err := rows.Scan(&name, &value); err != nil {
			return "", "", fmt.Errorf("scanning cookie row: %w", err)
		}
		switch name {
		case "auth_token":
			authToken = value
		case "ct0":
			ct0 = value
		}
	}
	if err := rows.Err(); err != nil {
		return "", "", fmt.Errorf("iterating cookie rows: %w", err)
	}

	if authToken == "" {
		return "", "", fmt.Errorf("auth_token cookie not found — log into x.com in Firefox first")
	}
	if ct0 == "" {
		return "", "", fmt.Errorf("ct0 cookie not found — log into x.com in Firefox first")
	}

	assert(len(authToken) > 0, "authToken validated but empty")
	assert(len(ct0) > 0, "ct0 validated but empty")

	return authToken, ct0, nil
}

// findFirefoxProfile locates the default Firefox profile directory.
func findFirefoxProfile(override string) (string, error) {
	appData := os.Getenv("APPDATA")
	if appData == "" {
		return "", fmt.Errorf("APPDATA environment variable not set")
	}
	firefoxDir := filepath.Join(appData, "Mozilla", "Firefox")

	if override != "" {
		dir := filepath.Join(firefoxDir, "Profiles", override)
		if _, err := os.Stat(dir); err == nil {
			return dir, nil
		}
		return "", fmt.Errorf("profile %q not found in %s", override, filepath.Join(firefoxDir, "Profiles"))
	}

	// Parse profiles.ini to find the active profile.
	iniPath := filepath.Join(firefoxDir, "profiles.ini")
	if profilePath, err := parseProfilesINI(iniPath); err == nil && profilePath != "" {
		if filepath.IsAbs(profilePath) {
			return profilePath, nil
		}
		return filepath.Join(firefoxDir, profilePath), nil
	}

	// Fallback: glob for common default profile names.
	for _, pattern := range []string{"*.default-release", "*.default"} {
		matches, _ := filepath.Glob(filepath.Join(firefoxDir, "Profiles", pattern))
		if len(matches) > 0 {
			return matches[0], nil
		}
	}

	return "", fmt.Errorf("no Firefox profile found in %s", firefoxDir)
}

// parseProfilesINI reads the Default= path from the [Install*] section.
func parseProfilesINI(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	inInstallSection := false

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "[") {
			inInstallSection = strings.HasPrefix(line, "[Install")
			continue
		}
		if inInstallSection && strings.HasPrefix(line, "Default=") {
			return strings.TrimPrefix(line, "Default="), nil
		}
	}
	return "", fmt.Errorf("no Default path found in profiles.ini")
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

package cookies

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/pbkdf2"
	"crypto/sha1"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ExtractChrome reads only X's session cookies from the selected local Chrome
// profile. Chrome is currently supported on macOS, using Chrome Safe Storage.
// An empty profile selects Local State's last_used profile, or Default.
func ExtractChrome(profile string) (authToken, ct0 string, err error) {
	root, err := chromeUserDataDir()
	if err != nil {
		return "", "", err
	}
	return extractChrome(root, profile, chromeSafeStoragePassword)
}

func chromeProfileDir(root, profile string) (string, error) {
	if profile == "" {
		var state struct {
			Profile struct {
				LastUsed string `json:"last_used"`
			} `json:"profile"`
		}
		data, err := os.ReadFile(filepath.Join(root, "Local State"))
		if err != nil && !os.IsNotExist(err) {
			return "", fmt.Errorf("reading Chrome Local State: %w", err)
		}
		if err == nil {
			if err := json.Unmarshal(data, &state); err != nil {
				return "", fmt.Errorf("reading Chrome profile selection: %w", err)
			}
		}
		profile = state.Profile.LastUsed
		if profile == "" {
			profile = "Default"
		}
	}
	if profile == "." || profile == ".." || strings.ContainsAny(profile, `/\`) || filepath.IsAbs(profile) {
		return "", fmt.Errorf("Chrome --profile must be a directory name, such as Default or 'Profile 1'")
	}
	dir := filepath.Join(root, profile)
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		return "", fmt.Errorf("Chrome profile %q not found in %s; see chrome://version for your Profile Path", profile, root)
	}
	return dir, nil
}

type chromeCookie struct {
	host, name, value string
	encrypted         []byte
}

func extractChrome(root, profile string, password func() ([]byte, error)) (string, string, error) {
	dir, err := chromeProfileDir(root, profile)
	if err != nil {
		return "", "", err
	}
	var dbPath string
	for _, candidate := range []string{filepath.Join(dir, "Network", "Cookies"), filepath.Join(dir, "Cookies")} {
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			dbPath = candidate
			break
		}
	}
	if dbPath == "" {
		return "", "", fmt.Errorf("Chrome Cookies database not found in %s; log into x.com in this Chrome profile first", dir)
	}
	items, version, err := readChromeCookies(dbPath, time.Now())
	if err != nil {
		return "", "", err
	}
	// Never combine tokens from different domains or switch to another profile.
	for _, host := range []string{".x.com", "x.com", ".twitter.com", "twitter.com"} {
		pair := make(map[string]chromeCookie, 2)
		for _, item := range items {
			if item.host == host {
				if _, exists := pair[item.name]; !exists {
					pair[item.name] = item
				}
			}
		}
		if len(pair) != 2 {
			continue
		}
		var key []byte
		for _, name := range []string{"auth_token", "ct0"} {
			item := pair[name]
			if len(item.encrypted) > 0 {
				if key == nil {
					secret, err := password()
					if err != nil {
						return "", "", err
					}
					key, err = pbkdf2.Key(sha1.New, string(secret), []byte("saltysalt"), 1003, 16)
					clear(secret)
					if err != nil {
						return "", "", fmt.Errorf("deriving Chrome cookie key: %w", err)
					}
					defer clear(key)
				}
				item.value, err = decryptChromeCookie(item.encrypted, key, host, version)
				if err != nil {
					return "", "", fmt.Errorf("decrypting Chrome %s cookie: %w", name, err)
				}
			}
			if item.value == "" || strings.ContainsAny(item.value, "\r\n;") {
				return "", "", fmt.Errorf("invalid Chrome %s cookie; log into x.com again", name)
			}
			pair[name] = item
		}
		return pair["auth_token"].value, pair["ct0"].value, nil
	}
	return "", "", fmt.Errorf("no complete, unexpired X session in Chrome profile %q; log into x.com in that profile first", filepath.Base(dir))
}

func readChromeCookies(path string, now time.Time) ([]chromeCookie, int, error) {
	// Read the live DB and its WAL in one SQLite snapshot. Copying them separately
	// can lose recent logins or pair a database with an inconsistent WAL.
	u := url.URL{Scheme: "file", Path: path}
	q := url.Values{"mode": {"ro"}, "_pragma": {"query_only(1)", "busy_timeout(2000)"}}
	u.RawQuery = q.Encode()
	db, err := sql.Open("sqlite", u.String())
	if err != nil {
		return nil, 0, fmt.Errorf("opening Chrome cookies read-only: %w", err)
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	tx, err := db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, 0, fmt.Errorf("reading Chrome cookies (try quitting Chrome and retrying): %w", err)
	}
	defer tx.Rollback()
	var version int
	if err := tx.QueryRowContext(ctx, "SELECT value FROM meta WHERE key = 'version'").Scan(&version); err != nil {
		return nil, 0, fmt.Errorf("reading Chrome cookie schema: %w", err)
	}
	if version < 23 {
		return nil, 0, fmt.Errorf("unsupported Chrome cookie database version %d; update Chrome and retry", version)
	}
	const query = `SELECT host_key, name, value, encrypted_value FROM cookies
		WHERE host_key IN ('.x.com', 'x.com', '.twitter.com', 'twitter.com')
		AND name IN ('auth_token', 'ct0') AND path = '/' AND top_frame_site_key = ''
		AND (has_expires = 0 OR expires_utc > ?)
		ORDER BY last_update_utc DESC, creation_utc DESC`
	const windowsEpochOffset = int64(11644473600)
	rows, err := tx.QueryContext(ctx, query, (now.Unix()+windowsEpochOffset)*1_000_000)
	if err != nil {
		return nil, 0, fmt.Errorf("querying Chrome X cookies: %w", err)
	}
	defer rows.Close()
	items := make([]chromeCookie, 0, 8)
	for rows.Next() {
		var item chromeCookie
		if err := rows.Scan(&item.host, &item.name, &item.value, &item.encrypted); err != nil {
			return nil, 0, fmt.Errorf("reading Chrome X cookie: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("reading Chrome X cookies: %w", err)
	}
	return items, version, nil
}

func decryptChromeCookie(encrypted, key []byte, host string, version int) (string, error) {
	if !bytes.HasPrefix(encrypted, []byte("v10")) {
		return "", fmt.Errorf("unsupported Chrome encryption format")
	}
	data := encrypted[3:]
	if len(data) == 0 || len(data)%aes.BlockSize != 0 {
		return "", fmt.Errorf("invalid encrypted cookie length")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("invalid Chrome cookie key")
	}
	plain := make([]byte, len(data))
	defer clear(plain)
	cipher.NewCBCDecrypter(block, bytes.Repeat([]byte(" "), aes.BlockSize)).CryptBlocks(plain, data)
	padding := int(plain[len(plain)-1])
	if padding == 0 || padding > aes.BlockSize || !bytes.Equal(plain[len(plain)-padding:], bytes.Repeat([]byte{byte(padding)}, padding)) {
		return "", fmt.Errorf("invalid cookie padding (Chrome Safe Storage key may not match)")
	}
	plain = plain[:len(plain)-padding]
	if version >= 24 {
		hash := sha256.Sum256([]byte(host))
		if len(plain) < sha256.Size || !bytes.Equal(plain[:sha256.Size], hash[:]) {
			return "", fmt.Errorf("cookie domain hash mismatch")
		}
		plain = plain[sha256.Size:]
	}
	return string(plain), nil
}

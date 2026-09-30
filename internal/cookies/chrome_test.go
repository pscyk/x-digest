package cookies

import (
	"bytes"
	"database/sql"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Independent vectors: Python hashlib.pbkdf2_hmac('sha1', b'fixture-password',
// b'saltysalt', 1003, 16), then openssl enc -aes-128-cbc with an all-spaces IV.
const fixtureKey = "5d84e88b8d2628e23102b464d77a5bbd"
const fixtureV23 = "7631303e6729894816333843a1a9e7bf44abd7"
const fixtureAuth = "76313041e6df1f72b7c14d1be8086982be0bb6c6711ad662bf2e982974a0c29083695d8287f9081e5b4e4a795bfd84bd0e112b"
const fixtureCSRF = "76313041e6df1f72b7c14d1be8086982be0bb6c6711ad662bf2e982974a0c29083695df7e9679fe67693afc801301a5840232d"

func unhex(t testing.TB, s string) []byte {
	t.Helper()
	b, err := hex.DecodeString(s)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestDecryptChromeCookie(t *testing.T) {
	tests := []struct {
		name, encrypted, host string
		version               int
		want, wantError       string
	}{
		{"legacy", fixtureV23, ".x.com", 23, "test-auth", ""},
		{"domain-bound", fixtureAuth, ".x.com", 24, "test-auth", ""},
		{"csrf", fixtureCSRF, ".x.com", 24, "test-csrf", ""},
		{"wrong-domain", fixtureAuth, ".twitter.com", 24, "", "domain hash"},
		{"missing-domain-hash", fixtureV23, ".x.com", 24, "", "domain hash"},
		{"unsupported-format", "76323000", ".x.com", 24, "", "unsupported"},
		{"truncated", "76313000", ".x.com", 24, "", "length"},
		{"empty", "763130", ".x.com", 24, "", "length"},
		{"bad-padding", "76313000000000000000000000000000000000", ".x.com", 24, "", "padding"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encrypted := unhex(t, tt.encrypted)
			original := bytes.Clone(encrypted)
			got, err := decryptChromeCookie(encrypted, unhex(t, fixtureKey), tt.host, tt.version)
			if tt.wantError != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantError) {
					t.Fatalf("got %q, %v; want error %q", got, err, tt.wantError)
				}
			} else if err != nil || got != tt.want {
				t.Fatalf("got %q, %v; want %q", got, err, tt.want)
			}
			if !bytes.Equal(encrypted, original) {
				t.Fatal("mutated ciphertext")
			}
		})
	}
}

func TestChromeProfileSelection(t *testing.T) {
	root := t.TempDir()
	for _, profile := range []string{"Default", "Profile 1"} {
		if err := os.Mkdir(filepath.Join(root, profile), 0700); err != nil {
			t.Fatal(err)
		}
	}
	check := func(profile, want string) {
		t.Helper()
		got, err := chromeProfileDir(root, profile)
		if err != nil || got != filepath.Join(root, want) {
			t.Fatalf("got %q, %v; want %s", got, err, want)
		}
	}
	check("", "Default")
	if err := os.WriteFile(filepath.Join(root, "Local State"), []byte(`{"profile":{"last_used":"Profile 1"}}`), 0600); err != nil {
		t.Fatal(err)
	}
	check("", "Profile 1")
	check("Default", "Default")
	for _, invalid := range []string{"..", ".", "../Default", "/tmp", `..\Default`, "Missing"} {
		if _, err := chromeProfileDir(root, invalid); err == nil {
			t.Fatalf("accepted %q", invalid)
		}
	}
}

func fixtureDB(t *testing.T, network bool) (string, *sql.DB) {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, "Default")
	if network {
		dir = filepath.Join(dir, "Network")
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", filepath.Join(dir, "Cookies"))
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { db.Close() })
	for _, stmt := range []string{
		"PRAGMA journal_mode=WAL", "PRAGMA wal_autocheckpoint=0",
		"CREATE TABLE meta (key TEXT PRIMARY KEY, value TEXT)",
		"INSERT INTO meta VALUES ('version', '24')",
		`CREATE TABLE cookies (host_key TEXT, name TEXT, value TEXT, encrypted_value BLOB,
    path TEXT DEFAULT '/', top_frame_site_key TEXT DEFAULT '', has_expires INTEGER DEFAULT 0,
    expires_utc INTEGER DEFAULT 0, last_update_utc INTEGER DEFAULT 0, creation_utc INTEGER DEFAULT 0)`,
	} {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatal(err)
		}
	}
	return root, db
}

func addCookie(t *testing.T, db *sql.DB, host, name, value string, encrypted []byte) {
	t.Helper()
	if encrypted == nil {
		encrypted = []byte{}
	}
	if _, err := db.Exec("INSERT INTO cookies (host_key,name,value,encrypted_value) VALUES (?,?,?,?)", host, name, value, encrypted); err != nil {
		t.Fatal(err)
	}
}

func TestExtractChromeLiveWAL(t *testing.T) {
	for _, network := range []bool{false, true} {
		t.Run(map[bool]string{false: "Cookies", true: "Network"}[network], func(t *testing.T) {
			root, db := fixtureDB(t, network)
			addCookie(t, db, ".x.com", "auth_token", "", unhex(t, fixtureAuth))
			addCookie(t, db, ".x.com", "ct0", "", unhex(t, fixtureCSRF))
			// The writer stays open and rows remain in WAL, like a running Chrome profile.
			calls := 0
			secret := []byte("fixture-password")
			auth, csrf, err := extractChrome(root, "", func() ([]byte, error) { calls++; return secret, nil })
			if err != nil || auth != "test-auth" || csrf != "test-csrf" || calls != 1 {
				t.Fatalf("auth/csrf matched: %v/%v, key calls %d, error %v", auth == "test-auth", csrf == "test-csrf", calls, err)
			}
			if !bytes.Equal(secret, make([]byte, len(secret))) {
				t.Fatal("password buffer not cleared")
			}
			var n int
			if err := db.QueryRow("SELECT count(*) FROM cookies").Scan(&n); err != nil || n != 2 {
				t.Fatalf("source database changed: %d, %v", n, err)
			}
		})
	}
}

func TestExtractChromeSessionSelection(t *testing.T) {
	for _, kind := range []string{"missing", "expired", "partitioned", "wrong-path", "mixed-domains", "plaintext", "prefer-x"} {
		t.Run(kind, func(t *testing.T) {
			root, db := fixtureDB(t, false)
			if kind != "missing" {
				addCookie(t, db, ".x.com", "auth_token", "local-auth", nil)
				host := ".x.com"
				if kind == "mixed-domains" {
					host = ".twitter.com"
				}
				addCookie(t, db, host, "ct0", "local-csrf", nil)
			}
			statement := ""
			switch kind {
			case "expired":
				statement = "UPDATE cookies SET has_expires=1, expires_utc=1"
			case "partitioned":
				statement = "UPDATE cookies SET top_frame_site_key='https://other.test'"
			case "wrong-path":
				statement = "UPDATE cookies SET path='/other'"
			case "prefer-x":
				addCookie(t, db, ".twitter.com", "auth_token", "old-auth", nil)
				addCookie(t, db, ".twitter.com", "ct0", "old-csrf", nil)
			}
			if statement != "" {
				if _, err := db.Exec(statement); err != nil {
					t.Fatal(err)
				}
			}
			auth, csrf, err := extractChrome(root, "", func() ([]byte, error) { t.Fatal("unexpected Keychain access"); return nil, nil })
			if kind == "plaintext" || kind == "prefer-x" {
				if err != nil || auth != "local-auth" || csrf != "local-csrf" {
					t.Fatalf("session selection: %v", err)
				}
			} else if err == nil || !strings.Contains(err.Error(), "no complete, unexpired X session") {
				t.Fatalf("got %v", err)
			}
		})
	}
}

func TestExtractChromeKeychainFailure(t *testing.T) {
	root, db := fixtureDB(t, false)
	addCookie(t, db, ".x.com", "auth_token", "", unhex(t, fixtureAuth))
	addCookie(t, db, ".x.com", "ct0", "", unhex(t, fixtureCSRF))
	denied := errors.New("keychain denied")
	auth, csrf, err := extractChrome(root, "", func() ([]byte, error) { return nil, denied })
	if !errors.Is(err, denied) || auth != "" || csrf != "" {
		t.Fatalf("got %v", err)
	}
}

func TestChromeCookieExpiryBoundary(t *testing.T) {
	root, db := fixtureDB(t, false)
	now := time.Unix(1700000000, 0)
	addCookie(t, db, ".x.com", "auth_token", "auth", nil)
	addCookie(t, db, ".x.com", "ct0", "csrf", nil)
	if _, err := db.Exec("UPDATE cookies SET has_expires=1,expires_utc=?", (now.Unix()+11644473600)*1000000); err != nil {
		t.Fatal(err)
	}
	items, _, err := readChromeCookies(filepath.Join(root, "Default", "Cookies"), now)
	if err != nil || len(items) != 0 {
		t.Fatalf("expired rows retained: %d, %v", len(items), err)
	}
}

func FuzzDecryptChromeCookie(f *testing.F) {
	f.Add(unhex(f, fixtureAuth))
	f.Add([]byte("v10"))
	key := unhex(f, fixtureKey)
	f.Fuzz(func(t *testing.T, data []byte) { _, _ = decryptChromeCookie(data, key, ".x.com", 24) })
}

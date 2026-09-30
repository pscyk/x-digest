package main

import (
	"bytes"
	"errors"
	"flag"
	"strings"
	"testing"
)

func TestInvalidInputBeforeCookieAccess(t *testing.T) {
	for _, args := range [][]string{
		{"--browser", "safari"}, {"--count", "0"}, {"--count", "-1"}, {"--count", "101"},
		{"--timeline", "invalid"}, {"--user", "alice", "--bookmarks", "go"}, {"--user", "@"}, {"extra"},
	} {
		var out, stderr bytes.Buffer
		if err := run(args, &out, &stderr); err == nil {
			t.Fatalf("accepted %v", args)
		}
		if out.Len() != 0 {
			t.Fatalf("accessed cookies for %v: %s", args, out.String())
		}
	}
}

func TestHelp(t *testing.T) {
	var out, stderr bytes.Buffer
	if err := run([]string{"--help"}, &out, &stderr); !errors.Is(err, flag.ErrHelp) {
		t.Fatal(err)
	}
	for _, flag := range []string{"-browser", "-profile"} {
		if !strings.Contains(stderr.String(), flag) {
			t.Fatalf("missing %s", flag)
		}
	}
}

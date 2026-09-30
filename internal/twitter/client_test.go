package twitter

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

func TestSessionHeadersDoNotFollowRedirects(t *testing.T) {
	var redirected atomic.Bool
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { redirected.Store(true) }))
	defer target.Close()
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, target.URL, http.StatusFound) }))
	defer source.Close()
	client := NewClient("fixture-auth", "fixture-csrf")
	_, err := client.doRequest(source.URL)
	if err == nil || !strings.Contains(err.Error(), "302") {
		t.Fatalf("got %v", err)
	}
	if redirected.Load() {
		t.Fatal("followed redirect with session credentials")
	}
}

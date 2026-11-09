package main

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

// POST /api/open hands URLs to the system's opener, which also runs files and
// apps: only web, mail and phone links may get through.
func TestOpenURLRefusesOtherLinks(t *testing.T) {
	for _, u := range []string{
		"file:///Applications/Calculator.app",
		"/Applications/Calculator.app",
		"javascript:alert(1)",
		"ssh://example.com",
		"https:relative",
		"mailto:",
		"-a Calculator",
	} {
		r := httptest.NewRequest("POST", "/api/open", strings.NewReader(`{"url": `+strconv.Quote(u)+`}`))
		w := httptest.NewRecorder()
		openURL(w, r)
		if w.Code != http.StatusBadRequest {
			t.Errorf("%q: got %d, want 400", u, w.Code)
		}
	}
}

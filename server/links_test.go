package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

func linkDoc(t *testing.T, contacts, bullet string) Doc {
	t.Helper()
	return parseDoc(t, `{"basics": {"name": "Ada", "headline": [], "contacts": [`+contacts+`]},
		"sections": [{"kind": "entries", "title": "Work", "items": [
			{"type": "entry", "title": "Analyst", "date": "2020", "bullets": [`+bullet+`]}]}],
		"style": {}}`)
}

func TestLinkChecks(t *testing.T) {
	doc := linkDoc(t,
		`{"text": "ada.dev", "url": "ada.dev"},
		 {"text": "ada@example.com", "url": "mailto:ada@example.com"},
		 {"text": "github.com/ada", "url": "https://github.com/someone"},
		 {"text": "www.ada.dev/", "url": "https://ada.dev"},
		 {"text": "Hidden", "url": "https://hidden.dev"},
		 {"text": "", "url": "https://not-on-the-page.dev"}`,
		`"Built [the map](https://map.dev)"`)
	facts := &pdfFacts{URLs: []string{"mailto:ada@example.com", "https://github.com/someone", "https://ada.dev", "https://map.dev"}}
	checks := linkChecks(doc, facts)

	if got := levelOf(t, checks, "Link won’t open: ada.dev"); got != "warn" {
		t.Errorf("no scheme: got %s, want warn", got)
	}
	if got := levelOf(t, checks, "Link text names a different address: github.com/ada"); got != "warn" {
		t.Errorf("label/target mismatch: got %s, want warn", got)
	}
	for _, c := range checks {
		if strings.Contains(c.Title, "www.ada.dev") || strings.Contains(c.Title, "ada@example.com") {
			t.Errorf("matching label flagged: %q", c.Title)
		}
	}
	// The scheme-less link is reported once, not again as unclickable; the
	// contact without text isn't on the page, so it isn't counted.
	c := checks[len(checks)-1]
	if c.Title != "1 of 6 links aren’t clickable in the PDF" || !strings.Contains(c.Detail, "Hidden → https://hidden.dev") {
		t.Errorf("unclickable summary = %q: %s", c.Title, c.Detail)
	}
}

func TestURLProblem(t *testing.T) {
	for u, bad := range map[string]bool{
		"https://ada.dev/x?y=1":       false,
		"mailto:ada@example.com?s=Hi": false,
		"tel:+1 555-555-0100":         false,
		"mailto:ada":                  true,
		"tel:123":                     true,
		"https:///path":               true,
		"linkedin.com/in/ada":         true,
		"javascript:alert(1)":         true,
	} {
		if got := urlProblem(u) != ""; got != bad {
			t.Errorf("urlProblem(%q) = %q, want problem: %v", u, urlProblem(u), bad)
		}
	}
}

func TestLinkChecker(t *testing.T) {
	var hits atomic.Int32
	site := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		switch r.URL.Path {
		case "/moved":
			http.Redirect(w, r, "/ok", http.StatusMovedPermanently)
		case "/ok":
		case "/blocked":
			w.WriteHeader(999)
		default:
			http.NotFound(w, r)
		}
	}))
	defer site.Close()

	links := `"[a](` + site.URL + `/moved) [b](` + site.URL + `/gone) [c](` + site.URL + `/blocked) [d](mailto:a@b.co)"`
	doc := linkDoc(t, "", links)
	lc := newLinkChecker()
	checks, err := lc.check(context.Background(), doc)
	if err != nil {
		t.Fatal(err)
	}
	if got := levelOf(t, checks, "Broken link: "+site.URL+"/gone"); got != "warn" {
		t.Errorf("404: got %s, want warn", got)
	}
	if got := levelOf(t, checks, "Couldn’t confirm: "+site.URL+"/blocked"); got != "info" {
		t.Errorf("999: got %s, want info", got)
	}
	if got := levelOf(t, checks, "1 of 3 web links open"); got != "ok" {
		t.Errorf("redirect to 200: got %s, want ok", got)
	}

	before := hits.Load()
	if _, err := lc.check(context.Background(), doc); err != nil {
		t.Fatal(err)
	}
	if hits.Load() != before {
		t.Errorf("second check requested the site again (%d → %d hits)", before, hits.Load())
	}
}

package main

import (
	"encoding/json"
	"strings"
	"testing"
)

const testDoc = `{
  "basics": {"name": "Ada Lovelace", "headline": [], "contacts": [{"text": "ada@example.com"}]},
  "sections": [{
    "kind": "entries", "title": "Work",
    "items": [{"type": "entry", "title": "Analyst", "date": "JAN 2020 – PRESENT",
               "subtitle": "*Engine* Co", "location": "London",
               "bullets": ["Built [the map](https://x.y) of bike-stress-model routes"]}]
  }],
  "style": {}
}`

func levelOf(t *testing.T, checks []Check, titlePrefix string) string {
	t.Helper()
	for _, c := range checks {
		if strings.HasPrefix(c.Title, titlePrefix) {
			return c.Level
		}
	}
	t.Fatalf("no check starting with %q in %+v", titlePrefix, checks)
	return ""
}

func TestAnalyze(t *testing.T) {
	var doc Doc
	if err := json.Unmarshal([]byte(testDoc), &doc); err != nil {
		t.Fatal(err)
	}

	clean := "ADA LOVELACE\nada@example.com\nWORK\nAnalyst JAN 2020 – PRESENT\nEngine Co London\n" +
		"Built the map of bike-stress-\nmodel routes\n"
	checks := analyze(doc, clean, 1)
	for _, c := range checks {
		if c.Level != "ok" {
			t.Errorf("clean text: unexpected %s check %q: %s", c.Level, c.Title, c.Detail)
		}
	}

	dotted := strings.Replace(clean, "Analyst JAN", "Analyst ……………… JAN", 1)
	if got := levelOf(t, analyze(doc, dotted, 1), "Dot leaders"); got != "error" {
		t.Errorf("typed leaders: got %s, want error", got)
	}

	symbol := strings.Replace(clean, "Built", "❖ Built", 1)
	if got := levelOf(t, analyze(doc, symbol, 1), "Symbols"); got != "warn" {
		t.Errorf("❖ bullet: got %s, want warn", got)
	}

	split := strings.Replace(clean, "Analyst JAN", "Analyst\nJAN", 1)
	if got := levelOf(t, analyze(doc, split, 1), "Title and date"); got != "warn" {
		t.Errorf("split heading: got %s, want warn", got)
	}

	dropped := strings.Replace(clean, "Engine Co London\n", "", 1)
	if got := levelOf(t, analyze(doc, dropped, 1), "2 of"); got != "error" {
		t.Errorf("dropped subtitle+location: got %s, want error", got)
	}
}

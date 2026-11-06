package main

import (
	"strings"
	"testing"
	"time"
)

func TestParseSpan(t *testing.T) {
	now := yearMonth{2026, 11}
	tests := []struct {
		in   string
		want dateSpan
		ok   bool
	}{
		{"SEP 2025 – PRESENT", dateSpan{yearMonth{2025, 9}, now}, true},
		{"Exp. May 2027", dateSpan{yearMonth{2027, 5}, yearMonth{2027, 5}}, true},
		{"(Non-degree) 2021", dateSpan{yearMonth{2021, 0}, yearMonth{2021, 0}}, true},
		{"JAN – MAR 2020", dateSpan{yearMonth{2020, 1}, yearMonth{2020, 3}}, true},
		{"2018-2019", dateSpan{yearMonth{2018, 0}, yearMonth{2019, 0}}, true},
		{"Sept 2019 to June 2020", dateSpan{yearMonth{2019, 9}, yearMonth{2020, 6}}, true},
		{"Summer", dateSpan{}, false},
		{"2020 – Jan", dateSpan{}, false},
	}
	for _, tt := range tests {
		got, ok := parseSpan(tt.in, now)
		if ok != tt.ok || got != tt.want {
			t.Errorf("parseSpan(%q) = %v, %v; want %v, %v", tt.in, got, ok, tt.want, tt.ok)
		}
	}
}

func TestDateChecks(t *testing.T) {
	now := time.Date(2026, 11, 6, 0, 0, 0, 0, time.UTC)
	entry := func(title, date string) DocItem { return DocItem{Type: "entry", Title: title, Date: date} }
	run := func(items ...DocItem) []Check {
		return dateChecks(Doc{Sections: []DocSection{{Title: "Work", Items: items}}}, now)
	}
	issues := func(checks []Check) []string {
		var out []string
		for _, c := range checks {
			if c.Level != "ok" {
				out = append(out, c.Title+": "+c.Detail)
			}
		}
		return out
	}

	ordered := run(
		entry("Degree", "Exp. May 2027"),     // a future end outranks Present
		entry("Lab", "SEP 2025 – PRESENT"),   // Present ties with this month…
		entry("Tutor", "JUN 2025 – PRESENT"), // …so the later start goes first
		entry("Engineer", "2021 – JAN 2025"),
		entry("Analyst", "2019 – 2021"),
		entry("Clerk", "2020 – MAR 2021"), // 2021 may be after March: either order
		entry("Intern", "NOV 2017 – DEC 2017"),
		DocItem{Type: "label", Text: "Non-degree:"}, // a new list may start older than the last
		entry("Exchange", "2021"),
		entry("Summer school", "JULY 2016"),
	)
	if got := issues(ordered); len(got) > 0 {
		t.Errorf("ordered entries: unexpected %q", got)
	}

	swapped := run(entry("Intern", "2017"), entry("Engineer", "2021 – 2025"))
	if got := issues(swapped); len(got) != 1 || !strings.Contains(got[0], "Work: Engineer (2021 – 2025) is listed after Intern (2017)") {
		t.Errorf("oldest first: got %q, want the pair named", got)
	}

	for _, date := range []string{"MAR 2021 – JAN 2021", "PRESENT – 2020"} {
		if got := levelOf(t, run(entry("Engineer", date)), "Dates that end before they start"); got != "warn" {
			t.Errorf("%q: got %s, want warn", date, got)
		}
	}

	if got := run(entry("Engineer", "Summer"), entry("Intern", "")); got != nil {
		t.Errorf("no years: got %+v, want no checks", got)
	}
}

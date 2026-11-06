package main

import (
	"cmp"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// yearMonth is a point an entry date names; month is 0 when only the year is given.
type yearMonth struct{ year, month int }

// compare orders a and b by year, then by month when both give one.
func (a yearMonth) compare(b yearMonth) int {
	if c := cmp.Compare(a.year, b.year); c != 0 || a.month == 0 || b.month == 0 {
		return c
	}
	return cmp.Compare(a.month, b.month)
}

// dateSpan is an entry's first and last date; equal for a single date.
type dateSpan struct{ start, end yearMonth }

// newerThan reports whether s belongs above t in a newest-first list: it
// ends later, or ends on the same date and starts later. Ends that only tie
// because one lacks a month (2021 and MAR 2021) could be in either order.
func (s dateSpan) newerThan(t dateSpan) bool {
	c := s.end.compare(t.end)
	if c == 0 && s.end == t.end {
		c = s.start.compare(t.start)
	}
	return c > 0
}

var (
	dateWordRe = regexp.MustCompile(`[a-z]+|\d+`)
	monthNames = map[string]int{
		"jan": 1, "january": 1, "feb": 2, "february": 2, "mar": 3, "march": 3, "apr": 4, "april": 4,
		"may": 5, "jun": 6, "june": 6, "jul": 7, "july": 7, "aug": 8, "august": 8,
		"sep": 9, "sept": 9, "september": 9, "oct": 10, "october": 10, "nov": 11, "november": 11,
		"dec": 12, "december": 12,
	}
)

// parseSpan reads the dates in a free-text entry date ("SEP 2025 – PRESENT",
// "Exp. May 2027", "JAN – MAR 2020"): month names, 4-digit years, and
// "present" (or current, now, today) as now. A month without its own year
// takes the next date's year. ok is false when a date has no year.
func parseSpan(s string, now yearMonth) (span dateSpan, ok bool) {
	var points []yearMonth
	month := 0
	flush := func() {
		if month != 0 {
			points = append(points, yearMonth{month: month})
			month = 0
		}
	}
	for _, w := range dateWordRe.FindAllString(strings.ToLower(s), -1) {
		if m, isMonth := monthNames[w]; isMonth {
			flush()
			month = m
			continue
		}
		switch w {
		case "present", "current", "now", "today":
			flush()
			points = append(points, now)
			continue
		}
		if y, err := strconv.Atoi(w); err == nil && len(w) == 4 && y >= 1900 && y < 2100 {
			points = append(points, yearMonth{y, month})
			month = 0
		}
	}
	flush()
	for i := len(points) - 2; i >= 0; i-- {
		if points[i].year == 0 {
			points[i].year = points[i+1].year
		}
	}
	if len(points) == 0 || points[0].year == 0 || points[len(points)-1].year == 0 {
		return dateSpan{}, false
	}
	return dateSpan{points[0], points[len(points)-1]}, true
}

// dateChecks reports entry date ranges that end before they start, and
// entries not listed newest first. A label item starts a new list, so a
// section can hold several ("Non-degree education:"). Entries whose date
// has no year are skipped; parserChecks reports those.
func dateChecks(doc Doc, now time.Time) []Check {
	today := yearMonth{now.Year(), int(now.Month())}
	var backwards, unordered []string
	dated := 0
	for _, sec := range doc.Sections {
		var prev string
		var prevSpan dateSpan
		for _, it := range sec.Items {
			if it.Type == "label" {
				prev = ""
				continue
			}
			if it.Type != "entry" {
				continue
			}
			span, ok := parseSpan(it.Date, today)
			if !ok {
				continue
			}
			dated++
			name := entryName(it)
			if span.start.compare(span.end) > 0 {
				backwards = append(backwards, inSection(sec, name))
			}
			if prev != "" && span.newerThan(prevSpan) {
				unordered = append(unordered, inSection(sec, name)+" is listed after "+prev)
			}
			prev, prevSpan = name, span
		}
	}
	if dated == 0 {
		return nil
	}

	l := checkList{group: groupDates}
	if len(backwards) > 0 {
		l.add("warn", "Dates that end before they start", sample(backwards))
	} else {
		l.add("ok", "No dates end before they start", "")
	}
	if len(unordered) > 0 {
		l.add("warn", "Entries out of date order",
			"Resumes list entries newest first: by end date (“Present” is this month), then by start date. "+
				"A label starts a new list.\n"+sample(unordered))
	} else {
		l.add("ok", fmt.Sprintf("All %d dated entries are listed newest first", dated), "")
	}
	return l.checks
}

// entryName identifies an entry in a check's detail: "Analyst (2020 – 2021)".
func entryName(it DocItem) string {
	return fmt.Sprintf("%s (%s)", cmp.Or(it.Title, it.Subtitle, "Untitled entry"), it.Date)
}

func inSection(sec DocSection, s string) string {
	if sec.Title == "" {
		return s
	}
	return sec.Title + ": " + s
}

package main

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
	"unicode"
)

// Doc is the subset of the resume JSON the server needs to understand.
// The frontend owns the full schema; unknown fields pass through untouched.
type Doc struct {
	Basics *struct {
		Name     string   `json:"name"`
		Headline []string `json:"headline"`
		Contacts []struct {
			Text string `json:"text"`
			URL  string `json:"url"`
		} `json:"contacts"`
	} `json:"basics"`
	Sections []DocSection `json:"sections"`
	Style    *Style       `json:"style"`
}

// Hidden sections, items and columns stay in the saved resume but are left
// off the page and out of every export.
type DocSection struct {
	Kind            string      `json:"kind"`
	Hidden          bool        `json:"hidden"`
	Title           string      `json:"title"`
	Paragraphs      []string    `json:"paragraphs"`
	UppercaseTitles bool        `json:"uppercaseTitles"`
	Items           []DocItem   `json:"items"`
	Variant         string      `json:"variant"`
	Columns         []DocColumn `json:"columns"`
}

type DocItem struct {
	Type     string   `json:"type"`
	Hidden   bool     `json:"hidden"`
	Title    string   `json:"title"`
	Date     string   `json:"date"`
	Subtitle string   `json:"subtitle"`
	Location string   `json:"location"`
	Text     string   `json:"text"`
	Bullets  []string `json:"bullets"`
}

type DocColumn struct {
	Hidden bool     `json:"hidden"`
	Title  string   `json:"title"`
	Lines  []string `json:"lines"`
}

// dropHidden removes the hidden sections, items and columns, leaving what
// the page shows.
func (d *Doc) dropHidden() {
	d.Sections = slices.DeleteFunc(d.Sections, func(s DocSection) bool { return s.Hidden })
	for i := range d.Sections {
		s := &d.Sections[i]
		s.Items = slices.DeleteFunc(s.Items, func(it DocItem) bool { return it.Hidden })
		s.Columns = slices.DeleteFunc(s.Columns, func(c DocColumn) bool { return c.Hidden })
	}
}

// Style is the page style; the Typst template and the Word export read it.
type Style struct {
	Accent          string  `json:"accent"`
	Font            string  `json:"font"`
	FontSize        float64 `json:"fontSize"`
	Margin          float64 `json:"margin"`
	Leader          string  `json:"leader"`
	Bullet          string  `json:"bullet"`
	ParagraphMarker string  `json:"paragraphMarker"`
}

type Check struct {
	Group  string `json:"group"` // groupParsers | groupScreenReaders | groupLinks | groupDates
	Level  string `json:"level"` // ok | info | warn | error
	Title  string `json:"title"`
	Detail string `json:"detail,omitempty"`
}

const (
	groupParsers       = "Parsers"
	groupScreenReaders = "Screen readers"
	groupLinks         = "Links"
	groupDates         = "Dates"
)

// checkList collects one group's checks.
type checkList struct {
	group  string
	checks []Check
}

func (l *checkList) add(level, title, detail string) {
	l.checks = append(l.checks, Check{Group: l.group, Level: level, Title: title, Detail: detail})
}

var (
	dotRun       = regexp.MustCompile(`(?:\.\s?){4,}|…`)
	emailRe      = regexp.MustCompile(`[\w.+-]+@[\w-]+(?:\.[\w-]+)+`)
	yearRe       = regexp.MustCompile(`(?i)\b(?:19|20)\d{2}\b|\bpresent\b|\bcurrent\b|\bnow\b`)
	markupLinkRe = regexp.MustCompile(`\[([^\]]+)\]\(([^)\s]+)\)`)
	spaceRe      = regexp.MustCompile(`\s+`)
	wrapRe       = regexp.MustCompile(`([-/])\s+`)
	quotes       = strings.NewReplacer("’", "'", "‘", "'", "“", `"`, "”", `"`)
)

// parserChecks compares the resume content with the PDF text layer
// (pdftotext -raw) and reports what an automated reader would trip over.
func parserChecks(doc Doc, text string, pages int) []Check {
	lines := strings.FieldsFunc(text, func(r rune) bool { return r == '\n' || r == '\f' })
	l := checkList{group: groupParsers}

	// Typed dot leaders.
	if hits := matchingLines(lines, dotRun.MatchString); len(hits) > 0 {
		l.add("error", "Dot leaders are in the text layer",
			"Readers see the dots as words. Draw leaders instead of typing them.\n"+sample(hits))
	} else {
		l.add("ok", "No typed dot leaders", "")
	}

	// Decorative symbols (❖ ● ★ emoji …) that parsers render as noise.
	if syms := symbols(text); len(syms) > 0 {
		hits := matchingLines(lines, func(s string) bool { return strings.ContainsAny(s, strings.Join(syms, "")) })
		l.add("warn", "Symbols in the text layer: "+strings.Join(syms, " "),
			"Parsers may show these as garbage or drop the line. Remove them, or draw them in the template.\n"+sample(hits))
	} else {
		l.add("ok", "No decorative symbols", "")
	}

	normLines := make([]string, len(lines))
	for i, line := range lines {
		normLines[i] = normalize(line)
	}

	// Every entry heading should read as "title … date" on one line.
	var split, badDates []string
	headings := 0
	for _, sec := range doc.Sections {
		for _, it := range sec.Items {
			if it.Type != "entry" || it.Title == "" {
				continue
			}
			if it.Date != "" && !yearRe.MatchString(it.Date) {
				badDates = append(badDates, fmt.Sprintf("%s: %q", it.Title, it.Date))
			}
			if it.Date == "" {
				continue
			}
			headings++
			t, d := normalize(it.Title), normalize(it.Date)
			if !slices.ContainsFunc(normLines, func(s string) bool { return strings.Contains(s, t) && strings.Contains(s, d) }) {
				split = append(split, it.Title)
			}
		}
	}
	if len(split) > 0 {
		l.add("warn", "Title and date read on separate lines",
			"Parsers may not link these dates to their entries:\n"+strings.Join(split, "\n"))
	} else if headings > 0 {
		l.add("ok", fmt.Sprintf("All %d entry headings read as one line (title + date)", headings), "")
	}
	if len(badDates) > 0 {
		l.add("warn", "Dates without a year",
			"Use a month and 4-digit year (or “Present”):\n"+strings.Join(badDates, "\n"))
	}

	// Every piece of content should survive into the text layer.
	pieces := contentPieces(doc)
	if missing, _ := findPieces(pieces, text, normalize); len(missing) > 0 {
		l.add("error", fmt.Sprintf("%d of %d text pieces not found in the text layer", len(missing), len(pieces)),
			sample(missing))
	} else {
		l.add("ok", fmt.Sprintf("All %d text pieces found in the text layer", len(pieces)), "")
	}

	if email := emailRe.FindString(text); email != "" {
		l.add("ok", "Email readable: "+email, "")
	} else {
		l.add("warn", "No email address found in the text layer", "")
	}

	if pages > 2 {
		l.add("info", fmt.Sprintf("%d pages", pages), "Most resumes fit on one or two pages.")
	}
	return l.checks
}

// findPieces looks for each piece in text, in order, comparing their key
// forms. missing pieces aren't in text at all; outOfOrder pieces are, but
// only before an earlier piece.
func findPieces(pieces []string, text string, key func(string) string) (missing, outOfOrder []string) {
	full := key(text)
	pos := 0
	for _, p := range pieces {
		np := key(p)
		if i := strings.Index(full[pos:], np); i >= 0 {
			pos += i + len(np)
		} else if strings.Contains(full, np) {
			outOfOrder = append(outOfOrder, p)
		} else {
			missing = append(missing, p)
		}
	}
	return missing, outOfOrder
}

// contentPieces lists the user-written strings that must appear in the PDF.
func contentPieces(doc Doc) []string {
	var out []string
	add := func(s ...string) {
		for _, v := range s {
			if strings.TrimSpace(v) != "" {
				out = append(out, v)
			}
		}
	}
	if doc.Basics != nil {
		add(doc.Basics.Name)
		add(doc.Basics.Headline...)
		for _, c := range doc.Basics.Contacts {
			add(c.Text)
		}
	}
	for _, sec := range doc.Sections {
		add(sec.Title)
		add(sec.Paragraphs...)
		for _, it := range sec.Items {
			add(it.Title, it.Date, it.Subtitle, it.Location, it.Text)
			add(it.Bullets...)
		}
		for _, c := range sec.Columns {
			add(c.Title)
			add(c.Lines...)
		}
	}
	return out
}

// normalize makes resume source text and extracted PDF text comparable:
// inline markup removed, case and quote styles folded, whitespace collapsed,
// and line wraps after "-" or "/" rejoined.
func normalize(s string) string {
	s = markupLinkRe.ReplaceAllString(s, "$1")
	s = strings.ReplaceAll(s, "*", "")
	s = quotes.Replace(strings.ToLower(s))
	s = spaceRe.ReplaceAllString(s, " ")
	s = wrapRe.ReplaceAllString(s, "$1")
	return strings.TrimSpace(s)
}

// symbols returns the distinct "Symbol, other" runes (❖ ● ★ 🌎 …) and • in s.
func symbols(s string) []string {
	var out []string
	for _, r := range s {
		if (unicode.Is(unicode.So, r) || r == '•') && !slices.Contains(out, string(r)) {
			out = append(out, string(r))
		}
	}
	return out
}

func matchingLines(lines []string, match func(string) bool) []string {
	var out []string
	for _, l := range lines {
		if match(l) {
			out = append(out, strings.TrimSpace(l))
		}
	}
	return out
}

func sample(lines []string) string {
	const max = 5
	if len(lines) <= max {
		return strings.Join(lines, "\n")
	}
	return strings.Join(lines[:max], "\n") + fmt.Sprintf("\n…and %d more", len(lines)-max)
}

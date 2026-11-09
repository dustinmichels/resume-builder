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
	Basics   *DocBasics   `json:"basics"`
	Sections []DocSection `json:"sections"`
	Style    *Style       `json:"style"`
}

type DocBasics struct {
	Name     string       `json:"name"`
	Headline []string     `json:"headline"`
	Contacts []DocContact `json:"contacts"`
}

type DocContact struct {
	Text string `json:"text"`
	URL  string `json:"url"`
}

// Hidden sections, items and columns stay in the saved resume but are left
// off the page and out of every export.
type DocSection struct {
	Kind            string        `json:"kind"`
	Hidden          bool          `json:"hidden"`
	Title           string        `json:"title"`
	Paragraphs      []string      `json:"paragraphs"`
	UppercaseTitles bool          `json:"uppercaseTitles"`
	Items           []DocItem     `json:"items"`
	Variant         string        `json:"variant"`
	Columns         []DocColumn   `json:"columns"`
	Compact         bool          `json:"compact"` // Julia: each entry on one line
	Legend          *DocIconLines `json:"legend"`  // Julia: labelled icons after the title
}

type DocItem struct {
	Type     string      `json:"type"`
	Hidden   bool        `json:"hidden"`
	Title    string      `json:"title"`
	Date     string      `json:"date"`
	Subtitle string      `json:"subtitle"`
	Location string      `json:"location"`
	Text     string      `json:"text"`
	Bullets  []string    `json:"bullets"`
	Icons    []*LineIcon `json:"icons"` // Julia: after the title
}

type DocColumn struct {
	Hidden bool        `json:"hidden"`
	Title  string      `json:"title"`
	Lines  []string    `json:"lines"`
	Icons  []*LineIcon `json:"icons"` // Julia: the icon after each line, nil for none
}

// DocIconLines are lines with an optional icon beside each, nil for none.
type DocIconLines struct {
	Lines []string    `json:"lines"`
	Icons []*LineIcon `json:"icons"`
}

// iconAt returns the icon at i of icons, or nil.
func iconAt(icons []*LineIcon, i int) *LineIcon {
	if i < len(icons) {
		return icons[i]
	}
	return nil
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
	Template        string  `json:"template"` // resume template id; "" = classic
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
	l := checkList{group: groupParsers}
	lines := textLines(text)
	l.dotLeaders(lines)
	l.decorativeSymbols(text, lines)

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

	l.missingPieces(doc, text)

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

func textLines(text string) []string {
	return strings.FieldsFunc(text, func(r rune) bool { return r == '\n' || r == '\f' })
}

// dotLeaders reports dot leaders typed as characters.
func (l *checkList) dotLeaders(lines []string) {
	if hits := matchingLines(lines, dotRun.MatchString); len(hits) > 0 {
		l.add("error", "Dot leaders are in the text layer",
			"Readers see the dots as words. Draw leaders instead of typing them.\n"+sample(hits))
	} else {
		l.add("ok", "No typed dot leaders", "")
	}
}

// decorativeSymbols reports symbols (❖ ● ★ emoji …) that parsers render as noise.
func (l *checkList) decorativeSymbols(text string, lines []string) {
	if syms := symbols(text); len(syms) > 0 {
		hits := matchingLines(lines, func(s string) bool { return strings.ContainsAny(s, strings.Join(syms, "")) })
		l.add("warn", "Symbols in the text layer: "+strings.Join(syms, " "),
			"Parsers may show these as garbage or drop the line. Remove them, or draw them in the template.\n"+sample(hits))
	} else {
		l.add("ok", "No decorative symbols", "")
	}
}

// missingPieces reports content that didn't survive into the text layer.
func (l *checkList) missingPieces(doc Doc, text string) {
	groups := contentGroups(doc)
	if missing, _ := findPieces(groups, text, normalize); len(missing) > 0 {
		l.add("error", fmt.Sprintf("%d of %d text pieces not found in the text layer", len(missing), pieceCount(groups)),
			sample(missing))
	} else {
		l.add("ok", fmt.Sprintf("All %d text pieces found in the text layer", pieceCount(groups)), "")
	}
}

// findPieces looks for each group of pieces in text, in order, comparing
// their key forms. Pieces within a group may read in any order (templates
// arrange an entry's title, date, subtitle and location differently), but
// after the group before. missing pieces aren't in text at all; outOfOrder
// pieces are, but only before an earlier group.
func findPieces(groups [][]string, text string, key func(string) string) (missing, outOfOrder []string) {
	full := key(text)
	pos := 0
	for _, g := range groups {
		end := pos
		for _, p := range g {
			np := key(p)
			if i := strings.Index(full[pos:], np); i >= 0 {
				end = max(end, pos+i+len(np))
			} else if strings.Contains(full, np) {
				outOfOrder = append(outOfOrder, p)
			} else {
				missing = append(missing, p)
			}
		}
		pos = end
	}
	return missing, outOfOrder
}

// contentPieces lists the user-written strings that must appear in the PDF.
func contentPieces(doc Doc) []string { return slices.Concat(contentGroups(doc)...) }

func pieceCount(groups [][]string) int { return len(slices.Concat(groups...)) }

// contentGroups lists the user-written strings that must appear in the PDF,
// in reading order: an entry's title, date, subtitle and location as one
// group, every other string as a group of its own.
func contentGroups(doc Doc) [][]string {
	var out [][]string
	group := func(s ...string) {
		var g []string
		for _, v := range s {
			if strings.TrimSpace(v) != "" {
				g = append(g, v)
			}
		}
		if len(g) > 0 {
			out = append(out, g)
		}
	}
	each := func(s ...string) {
		for _, v := range s {
			group(v)
		}
	}
	if doc.Basics != nil {
		each(doc.Basics.Name)
		each(doc.Basics.Headline...)
		for _, c := range doc.Basics.Contacts {
			each(c.Text)
		}
	}
	for _, sec := range doc.Sections {
		each(sec.Title)
		each(sec.Paragraphs...)
		for _, it := range sec.Items {
			group(it.Title, it.Date, it.Subtitle, it.Location, it.Text)
			each(it.Bullets...)
		}
		for _, c := range sec.Columns {
			each(c.Title)
			each(c.Lines...)
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

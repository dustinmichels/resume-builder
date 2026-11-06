package main

import (
	"context"
	"errors"
	"fmt"
	"math"
	"os/exec"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

// pdfFacts is what pdfinfo reads from a rendered PDF beyond its text layer.
type pdfFacts struct {
	// TagText is the text in tag order, the order screen readers read in.
	TagText string
	// Headings holds the text of each heading tag (H, H1–H6), in order.
	Headings []string
	// URLs are the targets of the PDF's link annotations.
	URLs []string
}

var (
	headingTagRe = regexp.MustCompile(`^H[1-6]?(?:\s|$)`)
	urlLineRe    = regexp.MustCompile(`^\s*\d+\s+\S+\s+(.+)$`)
)

// inspectPDF reads the PDF's tag tree (pdfinfo -struct-text) and link
// targets (pdfinfo -url).
func inspectPDF(ctx context.Context, pdfPath string) (pdfFacts, error) {
	var f pdfFacts
	if _, err := exec.LookPath("pdfinfo"); err != nil {
		return f, errors.New("pdfinfo not found; install poppler (`brew install poppler`)")
	}
	tree, err := exec.CommandContext(ctx, "pdfinfo", "-struct-text", pdfPath).Output()
	if err != nil {
		return f, fmt.Errorf("pdfinfo -struct-text: %w", err)
	}
	urls, err := exec.CommandContext(ctx, "pdfinfo", "-url", pdfPath).Output()
	if err != nil {
		return f, fmt.Errorf("pdfinfo -url: %w", err)
	}
	f.TagText, f.Headings = parseTagTree(string(tree))
	for i, line := range strings.Split(string(urls), "\n") {
		if m := urlLineRe.FindStringSubmatch(line); i > 0 && m != nil { // line 0 is the column header
			f.URLs = append(f.URLs, strings.TrimSpace(m[1]))
		}
	}
	return f, nil
}

// parseTagTree reads pdfinfo -struct-text output: one tag per line, nested by
// indentation, with each text run on its own line in quotes. Runs join
// without a separator: most carry their own spacing, and tag text is compared
// squashed, so the forced line breaks that don't are harmless.
func parseTagTree(tree string) (text string, headings []string) {
	var b strings.Builder
	heading := -1 // indentation of the heading being read, or -1
	for _, line := range strings.Split(tree, "\n") {
		s := strings.TrimSpace(line)
		indent := len(line) - len(strings.TrimLeft(line, " "))
		if heading >= 0 && s != "" && indent <= heading {
			heading = -1
		}
		switch {
		case len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"':
			run := s[1 : len(s)-1]
			b.WriteString(run)
			if heading >= 0 {
				headings[len(headings)-1] += run
			}
		case headingTagRe.MatchString(s):
			heading = indent
			headings = append(headings, "")
		}
	}
	return b.String(), headings
}

// accessibilityChecks reports what screen readers get from the PDF: Typst's
// PDF/UA-1 validation, the tag tree (when facts is set) and text contrast.
func accessibilityChecks(doc Doc, uaErrors []string, facts *pdfFacts) []Check {
	l := checkList{group: groupScreenReaders}

	if len(uaErrors) > 0 {
		l.add("error", "Fails Typst’s PDF/UA-1 (accessible PDF) checks",
			"The PDF is still tagged, but assistive technology may stumble here. Fix the template:\n"+sample(uaErrors))
	} else {
		l.add("ok", "Passes Typst’s PDF/UA-1 (accessible PDF) checks", "Tagged, with a language and a title shown in the title bar.")
	}

	if facts != nil {
		pieces := contentPieces(doc)
		missing, outOfOrder := findPieces(pieces, facts.TagText, squash)
		if len(missing) > 0 {
			l.add("error", fmt.Sprintf("%d of %d text pieces hidden from screen readers", len(missing), len(pieces)),
				"Not in the PDF’s tags (marked as decoration, or untagged):\n"+sample(missing))
		}
		if len(outOfOrder) > 0 {
			l.add("warn", fmt.Sprintf("%d text pieces read out of order", len(outOfOrder)),
				"Screen readers reach these earlier than they appear in the resume:\n"+sample(outOfOrder))
		}
		if len(missing) == 0 && len(outOfOrder) == 0 {
			l.add("ok", fmt.Sprintf("Screen readers read all %d text pieces in order", len(pieces)), "")
		}

		titles := sectionTitles(doc)
		var plain []string
		for _, t := range titles {
			if !slices.ContainsFunc(facts.Headings, func(h string) bool { return squash(h) == squash(t) }) {
				plain = append(plain, t)
			}
		}
		if len(plain) > 0 {
			l.add("warn", "Section titles not tagged as headings",
				"Screen-reader users skim by jumping between headings; these read as plain text:\n"+sample(plain))
		} else if len(titles) > 0 {
			l.add("ok", fmt.Sprintf("All %d section titles are headings", len(titles)), "")
		}
	}

	if doc.Style != nil {
		if ratio, ok := contrastOnWhite(doc.Style.Accent); ok {
			// Names, entry titles and links use the accent color at body size,
			// so the bar is WCAG AA for normal text.
			if ratio < 4.5 {
				l.add("warn", fmt.Sprintf("Accent color contrast is %.1f:1", ratio),
					"Below 4.5:1 (WCAG AA for body-size text), so titles and links in the accent color are hard to read for people with low vision. Pick a darker accent.")
			} else {
				l.add("ok", fmt.Sprintf("Accent color contrast is %.1f:1 (WCAG AA needs 4.5:1)", ratio), "")
			}
		}
	}
	return l.checks
}

// sectionTitles lists the titles the template renders as headings: section
// titles, plus column titles of side-by-side columns sections.
func sectionTitles(doc Doc) []string {
	var out []string
	for _, sec := range doc.Sections {
		if strings.TrimSpace(sec.Title) != "" {
			out = append(out, sec.Title)
		}
		if sec.Kind == "columns" && sec.Variant == "sections" {
			for _, c := range sec.Columns {
				if strings.TrimSpace(c.Title) != "" {
					out = append(out, c.Title)
				}
			}
		}
	}
	return out
}

// squash is normalize without spaces, for text whose word spacing is unreliable.
func squash(s string) string { return strings.ReplaceAll(normalize(s), " ", "") }

// contrastOnWhite returns the WCAG contrast ratio of a #rgb or #rrggbb color
// (alpha ignored) against a white page.
func contrastOnWhite(hex string) (float64, bool) {
	h := strings.TrimPrefix(hex, "#")
	if len(h) == 3 || len(h) == 4 {
		h = string([]byte{h[0], h[0], h[1], h[1], h[2], h[2]})
	}
	if len(h) != 6 && len(h) != 8 {
		return 0, false
	}
	v, err := strconv.ParseUint(h[:6], 16, 32)
	if err != nil {
		return 0, false
	}
	channel := func(c uint64) float64 {
		s := float64(c) / 255
		if s <= 0.04045 {
			return s / 12.92
		}
		return math.Pow((s+0.055)/1.055, 2.4)
	}
	lum := 0.2126*channel(v>>16&0xff) + 0.7152*channel(v>>8&0xff) + 0.0722*channel(v&0xff)
	return 1.05 / (lum + 0.05), true
}

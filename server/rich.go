package main

import "regexp"

// richPattern is the template's inline syntax: ***bold italic***, **bold**,
// *italic*, [label](url). It must match `rich-pattern` in template/resume.typ
// (and web/src/rich.ts) so every export reads the same markup the same way.
var richPattern = regexp.MustCompile(`\*\*\*(.+?)\*\*\*|\*\*(.+?)\*\*|\*(.+?)\*|\[([^\]]+)\]\(([^)\s]+)\)`)

type inlineKind int

const (
	inlineText inlineKind = iota
	inlineBold
	inlineItalic
	inlineLink
)

// inline is one node of parsed inline text. Text holds the literal text of a
// text node or the label of a link; bold and italic nodes hold Kids.
type inline struct {
	Kind inlineKind
	Text string
	URL  string
	Kids []inline
}

// parseRich parses inline markup the way the template's `rich` function does:
// bold and italic nest, link labels are literal.
func parseRich(s string) []inline {
	var out []inline
	pos := 0
	for _, m := range richPattern.FindAllStringSubmatchIndex(s, -1) {
		if m[0] > pos {
			out = append(out, inline{Kind: inlineText, Text: s[pos:m[0]]})
		}
		switch {
		case m[2] >= 0:
			italic := inline{Kind: inlineItalic, Kids: parseRich(s[m[2]:m[3]])}
			out = append(out, inline{Kind: inlineBold, Kids: []inline{italic}})
		case m[4] >= 0:
			out = append(out, inline{Kind: inlineBold, Kids: parseRich(s[m[4]:m[5]])})
		case m[6] >= 0:
			out = append(out, inline{Kind: inlineItalic, Kids: parseRich(s[m[6]:m[7]])})
		default:
			out = append(out, inline{Kind: inlineLink, Text: s[m[8]:m[9]], URL: s[m[10]:m[11]]})
		}
		pos = m[1]
	}
	if pos < len(s) {
		out = append(out, inline{Kind: inlineText, Text: s[pos:]})
	}
	return out
}

// plain wraps literal text (fields the template never parses as markup).
func plain(s string) []inline { return []inline{{Kind: inlineText, Text: s}} }

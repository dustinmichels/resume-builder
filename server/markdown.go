package main

import (
	"regexp"
	"strings"
)

var (
	mdSpecial   = strings.NewReplacer(`\`, `\\`, "`", "\\`", `*`, `\*`, `_`, `\_`, `[`, `\[`, `]`, `\]`, `<`, `\<`, `>`, `\>`, `~`, `\~`)
	mdLineStart = regexp.MustCompile(`^[#+=-]`)
	mdOrdered   = regexp.MustCompile(`^(\d+)([.)])(\s|$)`)
)

// mdWriter builds a Markdown document block by block.
type mdWriter struct{ strings.Builder }

func (b *mdWriter) block(nodes []inline) {
	if s := mdBlock(nodes, ""); s != "" {
		b.WriteString(s + "\n\n")
	}
}

func (b *mdWriter) list(items []string) {
	var wrote bool
	for _, it := range items {
		if s := mdBlock(parseRich(it), "  "); s != "" {
			b.WriteString("- " + s + "\n")
			wrote = true
		}
	}
	if wrote {
		b.WriteString("\n")
	}
}

func (b *mdWriter) heading(level int, s string) {
	if s = mdBlock(plain(s), ""); s != "" {
		if strings.HasSuffix(s, "#") { // a trailing # would read as a closing sequence
			s = s[:len(s)-1] + `\#`
		}
		b.WriteString(strings.Repeat("#", level) + " " + s + "\n\n")
	}
}

func (b *mdWriter) bytes() []byte { return []byte(strings.TrimRight(b.String(), "\n") + "\n") }

// markdownResume renders the resume as Markdown. Inline markup is kept as
// written; everything else is escaped so it reads back as the same text.
func markdownResume(doc Doc) []byte {
	var b mdWriter
	if bs := doc.Basics; bs != nil {
		b.heading(1, bs.Name)
		b.block(plain(strings.Join(nonEmpty(bs.Headline), " // ")))
		var contacts []string
		for _, c := range bs.Contacts {
			switch {
			case strings.TrimSpace(c.Text) == "":
			case c.URL != "":
				contacts = append(contacts, "["+mdEscape(c.Text)+"]("+mdURL(c.URL)+")")
			default:
				contacts = append(contacts, mdEscape(c.Text))
			}
		}
		if len(contacts) > 0 {
			b.WriteString(strings.Join(contacts, " | ") + "\n\n")
		}
	}

	for _, sec := range doc.Sections {
		if sec.Kind != "columns" || sec.Variant != "sections" {
			b.heading(2, sec.Title)
		}
		switch sec.Kind {
		case "text":
			for _, p := range sec.Paragraphs {
				b.block(parseRich(p))
			}
		case "entries":
			for _, it := range sec.Items {
				if it.Type == "label" {
					if strings.TrimSpace(it.Text) != "" {
						b.block([]inline{{Kind: inlineBold, Kids: parseRich(it.Text)}})
					}
					continue
				}
				b.heading(3, strings.Join(nonEmpty([]string{it.Title, it.Date}), " | "))
				line := parseRich(it.Subtitle)
				if strings.TrimSpace(it.Location) != "" {
					if strings.TrimSpace(it.Subtitle) != "" {
						line = append(line, plain(" | ")...)
					}
					line = append(line, plain(it.Location)...)
				}
				b.block(line)
				b.list(it.Bullets)
			}
		case "columns":
			for _, col := range sec.Columns {
				if sec.Variant == "sections" {
					b.heading(2, col.Title)
				} else if strings.TrimSpace(col.Title) != "" {
					b.block([]inline{{Kind: inlineBold, Kids: plain(col.Title)}})
				}
				b.list(col.Lines)
			}
		}
	}
	return b.bytes()
}

// mdBlock renders inline nodes as one Markdown block. Line breaks typed in the
// editor become hard breaks, continued at indent; lines that would start a
// heading, list or rule are escaped.
func mdBlock(nodes []inline, indent string) string {
	var lines []string
	for _, l := range strings.Split(mdInline(nodes), "\n") {
		l = strings.TrimSpace(l)
		if l == "" {
			continue
		}
		if mdLineStart.MatchString(l) {
			l = `\` + l
		} else {
			l = mdOrdered.ReplaceAllString(l, `$1\$2$3`)
		}
		lines = append(lines, l)
	}
	return strings.Join(lines, "\\\n"+indent)
}

func mdInline(nodes []inline) string {
	var b strings.Builder
	for _, n := range nodes {
		switch n.Kind {
		case inlineText:
			b.WriteString(mdEscape(n.Text))
		case inlineBold:
			b.WriteString("**" + mdInline(n.Kids) + "**")
		case inlineItalic:
			b.WriteString("*" + mdInline(n.Kids) + "*")
		case inlineLink:
			b.WriteString("[" + mdEscape(n.Text) + "](" + mdURL(n.URL) + ")")
		}
	}
	return b.String()
}

func mdEscape(s string) string { return mdSpecial.Replace(s) }

// mdURL writes a link destination, using the <…> form when the URL has
// characters a bare destination can't hold.
func mdURL(u string) string {
	u = strings.TrimSpace(u)
	if !strings.ContainsAny(u, " ()<>") {
		return u
	}
	return "<" + strings.NewReplacer("<", "%3C", ">", "%3E").Replace(u) + ">"
}

func nonEmpty(ss []string) []string {
	var out []string
	for _, s := range ss {
		if strings.TrimSpace(s) != "" {
			out = append(out, s)
		}
	}
	return out
}

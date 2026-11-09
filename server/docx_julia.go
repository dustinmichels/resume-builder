package main

// Word export in the Julia template's layout: docxJuliaResume mirrors
// template/julia.typ, docxJuliaLetter template/julia-letter.typ, with the
// sizes and gaps of those templates.
//
// Where Word can't follow the PDF:
// - Fonts are named, not embedded; the light and medium weights print in the
//   font's regular weight.
// - A gap the template draws between parts of a line is one space widened
//   with character spacing, so the text reads as single-spaced words.
// - A section's legend follows its title through a style separator (a hidden
//   paragraph mark), so the heading holds only the title, as in the PDF.
// - Icons are pictures marked decorative: SVG, with a PNG copy drawn by Typst
//   for readers without SVG support.
// - In the letter, the dots and diamonds the template draws are typed
//   characters; the letterhead's hairline is its contact line's bottom
//   border, without the diamond; the title has no hairline after it.

import (
	"context"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
)

const (
	juliaInk        = "231F20"
	juliaMuted      = "828282" // luma(130)
	juliaDisplay    = "Source Sans 3"
	juliaIndent     = 2.3      // ems the content hangs in from the section titles
	juliaIcon       = 0.92     // icon size, in ems of the text beside it
	juliaRule       = "CDCDCD" // luma(205), the letter's hairlines
)

// juliaWriter writes paragraphs as Typst stacks blocks: the space between
// two is the larger of the first's space below and the second's above
// (Word would add them). Typst measures it from the last line's baseline to
// the next one's cap height, Word from line box to line box, whose exact
// pitch already holds the leading; so a paragraph's space before is the
// Typst space less the leading.
type juliaWriter struct {
	*docxWriter
	leading float64 // Typst's, in ems of the base size
	below   float64 // the last paragraph's space below; -1 at the top of the page
}

// juliaPara is a paragraph's layout, in ems of the base size.
type juliaPara struct {
	style         string // paragraph style id, "" for Normal
	keep          bool   // keep with the next paragraph, and its lines together
	above, below  float64
	left, hanging float64
	center        bool
	justify       bool
	rule          float64 // a hairline this many points below the paragraph; 0 for none
}

// newJuliaWriter starts a Julia document with lines leading ems apart: bold
// text in the display font, a touch larger, as julia.typ's `show strong` does.
func newJuliaWriter(st Style, leading float64) *juliaWriter {
	d := newDocxWriter(st)
	d.strongFont, d.strongScale = juliaDisplay, 1.08
	return &juliaWriter{docxWriter: d, leading: leading, below: -1}
}

// before is the space before a paragraph wanting above ems above it.
func (j *juliaWriter) before(above float64) int {
	if j.below < 0 {
		return 0
	}
	return j.em(max(max(j.below, above)-j.leading, 0))
}

func (j *juliaWriter) block(p juliaPara, runs string) {
	var b strings.Builder
	if p.style != "" {
		fmt.Fprintf(&b, `<w:pStyle w:val="%s"/>`, p.style)
	}
	if p.keep {
		b.WriteString("<w:keepNext/><w:keepLines/>")
	}
	// pPr's children in schema order: pBdr before spacing, ind and jc.
	if p.rule > 0 {
		fmt.Fprintf(&b, `<w:pBdr><w:bottom w:val="single" w:sz="4" w:space="%d" w:color="%s"/></w:pBdr>`, min(int(math.Round(p.rule)), 31), juliaRule)
	}
	fmt.Fprintf(&b, `<w:spacing w:before="%d" w:after="0"/>`, j.before(p.above))
	if p.left != 0 || p.hanging != 0 {
		fmt.Fprintf(&b, `<w:ind w:left="%d" w:hanging="%d"/>`, j.em(p.left), j.em(p.hanging))
	}
	if p.center {
		b.WriteString(`<w:jc w:val="center"/>`)
	} else if p.justify {
		b.WriteString(`<w:jc w:val="both"/>`)
	}
	j.para(b.String(), runs)
	j.below = p.below
}

// space adds vertical space before the next paragraph, as Typst's v() does.
func (j *juliaWriter) space(ems float64) { j.below = max(j.below, 0) + ems }

// gap is a space width ems wide (of the base size) in text of the given size.
func (d *docxWriter) gap(width, size float64) string {
	return d.run(" ", runProps{size: size, spacing: width - 0.25*size})
}

// header writes the name centered over a gray line of parts, in multiples r
// of the base size (the resume's size in the base size's ems).
func (j *juliaWriter) header(name string, parts []string, r float64) {
	if strings.TrimSpace(name) != "" {
		j.block(juliaPara{style: "Title", below: 0.45 * r, center: true}, j.runs(plain(name), runProps{size: 2.45 * r}))
	}
	if len(parts) > 0 {
		j.block(juliaPara{center: true}, strings.Join(parts, j.gap(1.4*1.05*r, 1.05*r)))
	}
}

// contacts are the header's contacts, links in the header's gray.
func (j *juliaWriter) contacts(contacts []DocContact, p runProps) []string {
	j.linkColor = juliaMuted
	defer func() { j.linkColor = "" }()
	var out []string
	for _, c := range contacts {
		if strings.TrimSpace(c.Text) == "" {
			continue
		}
		node := plain(c.Text)
		if c.URL != "" {
			node = []inline{{Kind: inlineLink, Text: c.Text, URL: c.URL}}
		}
		out = append(out, j.runs(node, p))
	}
	return out
}

// sectionTitle writes a section title in the Heading1 style, sized for a
// resume of r times the base size.
func (j *juliaWriter) sectionTitle(title string, r float64) {
	j.block(juliaPara{style: "Heading1", keep: true, above: 1.3 * r, below: 0.55 * r}, j.runs(plain(title), runProps{}))
}

// ── Icons ───────────────────────────────────────────────────────────────────

// entryIcons are an entry's icons after a thin gap, in text of the given size.
func (d *docxWriter) entryIcons(icons []*LineIcon, size float64) string {
	var out []string
	for _, ic := range icons {
		if ic != nil {
			out = append(out, d.icon(ic.SVG, size))
		}
	}
	if len(out) == 0 {
		return ""
	}
	return d.gap(0.3*size, size) + strings.Join(out, d.gap(0.12*size, size))
}

// icon draws svg inline, square, juliaIcon ems of the text size (a multiple
// of the base size) beside it, centered on the x-height as the template's
// icons are, and marked decorative so screen readers skip it. Lucide icons,
// drawn in currentColor, take the accent. Each distinct icon is stored once.
func (d *docxWriter) icon(svg string, size float64) string {
	svg = strings.ReplaceAll(svg, "currentColor", "#"+d.accent)
	i, ok := d.iconIndex[svg]
	if !ok {
		i = len(d.icons)
		d.icons = append(d.icons, svg)
		d.iconIndex[svg] = i
	}
	d.drawings++
	pt := juliaIcon * size * d.st.FontSize
	// The template hangs icons 0.125em below the baseline; Word sets them on it.
	return fmt.Sprintf(iconXML, -int(math.Round(0.125*size*d.st.FontSize*2)), int(math.Round(pt*12700)), d.drawings, i+1, d.panel())
}

// drawIcons draws each placed icon as a PNG, the copy Word shows where it
// can't draw SVG: square, the SVG fit inside as the template's icon box does.
func (d *docxWriter) drawIcons(ctx context.Context) error {
	if len(d.icons) == 0 {
		return nil
	}
	ctx, cancel := context.WithTimeout(ctx, renderTimeout)
	defer cancel()
	dir, err := os.MkdirTemp("", "resume-icons-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)
	for i, svg := range d.icons {
		if err := os.WriteFile(filepath.Join(dir, fmt.Sprintf("icon-%d.svg", i)), []byte(svg), 0o600); err != nil {
			return err
		}
	}
	main := fmt.Sprintf(`#set page(width: 24pt, height: 24pt, margin: 0pt, fill: none)
#for i in range(%d) {
  pagebreak(weak: true)
  image("icon-" + str(i) + ".svg", width: 100%%, height: 100%%, fit: "contain")
}
`, len(d.icons))
	if err := os.WriteFile(filepath.Join(dir, "icons.typ"), []byte(main), 0o600); err != nil {
		return err
	}
	if _, err := typst(ctx, dir, dir, "icons.typ", "page-{0p}.png", "--ppi", "300"); err != nil {
		return fmt.Errorf("drawing icons: %w", err)
	}
	paths, err := filepath.Glob(filepath.Join(dir, "page-*.png")) // sorted; {0p} zero-pads
	if err != nil {
		return err
	}
	if len(paths) != len(d.icons) {
		return fmt.Errorf("drawing icons: %d of %d drawn", len(paths), len(d.icons))
	}
	d.iconPNGs = make([][]byte, len(paths))
	for i, p := range paths {
		if d.iconPNGs[i], err = os.ReadFile(p); err != nil {
			return err
		}
	}
	return nil
}

// ── Resume ──────────────────────────────────────────────────────────────────

func docxJuliaResume(ctx context.Context, doc Doc) ([]byte, error) {
	j := newJuliaWriter(*doc.Style, 0.38)
	name := ""
	if bs := doc.Basics; bs != nil {
		name = bs.Name
		line := runProps{color: juliaMuted, size: 1.05}
		var parts []string
		for _, h := range nonEmpty(bs.Headline) {
			parts = append(parts, j.runs(plain(h), line))
		}
		j.header(name, append(parts, j.contacts(bs.Contacts, line)...), 1)
	}
	for _, sec := range doc.Sections {
		j.sectionHeading(sec)
		switch sec.Kind {
		case "text":
			for _, p := range nonEmpty(sec.Paragraphs) {
				j.block(juliaPara{above: 0.45, below: 0.45, left: juliaIndent}, j.runs(parseRich(p), runProps{}))
			}
		case "entries":
			for _, it := range sec.Items {
				switch {
				case it.Type == "label":
					if strings.TrimSpace(it.Text) != "" {
						bi := []inline{{Kind: inlineBold, Kids: []inline{{Kind: inlineItalic, Kids: parseRich(it.Text)}}}}
						j.block(juliaPara{keep: true, above: 1.1, left: juliaIndent}, j.runs(bi, runProps{}))
					}
				case sec.Compact:
					j.compactEntry(it)
				default:
					j.entry(it)
				}
			}
		case "columns":
			for _, col := range sec.Columns {
				j.column(col)
			}
		}
	}
	if err := j.drawIcons(ctx); err != nil {
		return nil, err
	}
	return j.pack(name+" – Resume", name, j.styles(1), j.numbering())
}

// sectionHeading writes the section title, then its legend on the same line
// (lines in their icons' colors, each icon before it) on the legend's fill
// and inside its border, padded half an em each side; a section without a
// title gets a small gap instead.
func (j *juliaWriter) sectionHeading(sec DocSection) {
	if strings.TrimSpace(sec.Title) == "" {
		j.space(0.6)
		return
	}
	const size = 1.02
	var fill, border string
	var items []string
	if lg := sec.Legend; lg != nil {
		fill, border = styleHex(lg.Fill), styleHex(lg.Border)
		j.shade, j.border = fill, border
		for i, l := range lg.Lines {
			if strings.TrimSpace(l) == "" {
				continue
			}
			color, item := juliaInk, ""
			if ic := iconAt(lg.Icons, i); ic != nil {
				if m := hexColor.FindStringSubmatch(ic.Color); m != nil {
					color = strings.ToUpper(m[1])
				}
				item = j.icon(ic.SVG, size) + j.gap(0.1*size, size)
			}
			items = append(items, item+j.runs(parseRich(l), runProps{bold: true, caps: true, color: color, size: size}))
		}
		j.shade, j.border = "", ""
	}
	if len(items) == 0 {
		j.sectionTitle(sec.Title, 1)
		return
	}
	// The hidden paragraph mark joins the title and legend paragraphs into one line.
	below := j.below
	j.para(fmt.Sprintf(`<w:pStyle w:val="Heading1"/><w:spacing w:before="%d" w:after="0"/><w:rPr><w:vanish/><w:specVanish/></w:rPr>`, j.before(1.3)),
		j.runs(plain(sec.Title), runProps{}))
	j.below = below
	// As in the template: a 1.3em gutter, half an em of it the panel's padding.
	gutter, pad := j.gap(0.9, size), ""
	if fill != "" || border != "" {
		gutter = j.gap(1.3-0.5*size, size)
		j.shade, j.border = fill, border
		pad = j.gap(0.5*size, size)
	}
	legend := pad + strings.Join(items, j.gap(0.7*size, size)) + pad
	j.shade, j.border = "", ""
	j.block(juliaPara{keep: true, above: 1.3, below: 0.55}, gutter+legend)
}

// styleHex is a color as RRGGBB, "" for none.
func styleHex(c string) string {
	if m := hexColor.FindStringSubmatch(c); m != nil {
		return strings.ToUpper(m[1])
	}
	return ""
}

// entry writes an entry as one unbreakable block: the organization line
// (subtitle in small caps, location, date), the title and its icons, then
// each bullet as an indented paragraph.
func (j *juliaWriter) entry(e DocItem) {
	type para struct {
		layout juliaPara
		runs   string
	}
	var paras []para
	org := runProps{size: 1.04}
	var parts []string
	if strings.TrimSpace(e.Subtitle) != "" {
		q := org
		q.smallCaps = true
		parts = append(parts, j.runs(parseRich(e.Subtitle), q))
	}
	for _, s := range []string{e.Location, e.Date} {
		if strings.TrimSpace(s) != "" {
			parts = append(parts, j.runs(plain(s), org))
		}
	}
	hasTitle := strings.TrimSpace(e.Title) != ""
	if len(parts) > 0 || !hasTitle {
		runs := strings.Join(parts, j.run(", ", org))
		below := 0.42
		if !hasTitle {
			runs += j.entryIcons(e.Icons, org.size)
			below = 0.32
		}
		paras = append(paras, para{juliaPara{below: below, left: juliaIndent}, runs})
	}
	if hasTitle {
		paras = append(paras, para{juliaPara{above: 0.32, left: juliaIndent}, j.entryTitle(e.Title, 1.05) + j.entryIcons(e.Icons, 1.05)})
	}
	for _, b := range nonEmpty(e.Bullets) {
		paras = append(paras, para{juliaPara{above: 0.32, left: juliaIndent + 1.9 + 1, hanging: 1}, j.runs(parseRich(b), runProps{})})
	}
	paras[0].layout.above = 0.95
	for i, p := range paras {
		p.layout.keep = i < len(paras)-1
		j.block(p.layout, p.runs)
	}
}

// entryTitle is an entry title in bold, the part after " - " (a lab or
// department) in the regular weight.
func (j *juliaWriter) entryTitle(title string, size float64) string {
	parts := strings.Split(title, " - ")
	s := j.runs([]inline{{Kind: inlineBold, Kids: plain(parts[0])}}, runProps{size: size})
	if len(parts) > 1 {
		s += j.run(" - "+strings.Join(parts[1:], " - "), runProps{size: size})
	}
	return s
}

// compactEntry writes an entry on one line: organization, location, title,
// date, icons, then the bullets in smaller type.
func (j *juliaWriter) compactEntry(e DocItem) {
	const size = 1.08
	base := runProps{size: size}
	soft := runProps{size: size, italic: true, color: juliaMuted}
	var b strings.Builder
	if strings.TrimSpace(e.Subtitle) != "" {
		q := base
		q.smallCaps = true
		b.WriteString(j.runs(parseRich(e.Subtitle), q))
	}
	if strings.TrimSpace(e.Location) != "" {
		b.WriteString(j.run(", ", base) + j.runs(plain(e.Location), soft))
	}
	hasTitle := strings.TrimSpace(e.Title) != ""
	if hasTitle {
		b.WriteString(j.run(", ", base) + j.runs([]inline{{Kind: inlineBold, Kids: plain(e.Title)}}, base))
	}
	if strings.TrimSpace(e.Date) != "" {
		sep := ", "
		if hasTitle {
			sep = " "
		}
		b.WriteString(j.run(sep, base) + j.runs(plain(e.Date), soft))
	}
	b.WriteString(j.entryIcons(e.Icons, size))
	for _, bl := range nonEmpty(e.Bullets) {
		b.WriteString(j.run(" ", base) + j.runs(parseRich(bl), runProps{size: 0.92}))
	}
	j.block(juliaPara{above: 0.42, left: 2 * juliaIndent}, b.String())
}

// column writes a column as one paragraph: its title, then its lines run
// together, each line's icon after it.
func (j *juliaWriter) column(col DocColumn) {
	var lines []string
	for i, l := range col.Lines {
		if strings.TrimSpace(l) == "" {
			continue
		}
		s := j.runs(parseRich(l), runProps{})
		if ic := iconAt(col.Icons, i); ic != nil {
			s += j.gap(0.15, 1) + j.icon(ic.SVG, 1)
		}
		lines = append(lines, s)
	}
	runs := ""
	if strings.TrimSpace(col.Title) != "" {
		runs = j.runs([]inline{{Kind: inlineBold, Kids: plain(col.Title)}}, runProps{}) + j.run(": ", runProps{})
	}
	j.block(juliaPara{above: 0.5, left: juliaIndent}, runs+strings.Join(lines, j.gap(0.6, 1)))
}

// ── Cover letter ────────────────────────────────────────────────────────────

// docxJuliaLetter mirrors template/julia-letter.typ: the letterhead and
// title at the resume's size, the body at the letter's, justified, inside a
// margin of at least 0.8in.
func docxJuliaLetter(ld LetterDoc) ([]byte, error) {
	l := ld.Letter
	j := newJuliaWriter(*ld.Style, 0.62)
	resumeFont, resumeSize := j.st.Font, j.st.FontSize
	j.st.Font, j.st.FontSize = l.Style.Font, l.Style.FontSize
	if j.st.Font == "" {
		j.st.Font = resumeFont
	}
	if j.st.FontSize <= 0 {
		j.st.FontSize = 10.5
	}
	j.st.Margin = max(j.st.Margin, 0.8)
	j.bottom = j.st.Margin
	// The letterhead and title keep the resume's size: r is it in the letter's ems.
	r := resumeSize / j.st.FontSize
	name := ld.Basics.Name

	// Letterhead: the header lines not repeating a contact, then the
	// contacts, parted by accent dots, over a hairline.
	size := 1.05 * r
	line := runProps{font: resumeFont, color: juliaMuted, size: size}
	contacts := map[string]bool{}
	for _, c := range ld.Basics.Contacts {
		contacts[c.Text] = true
	}
	var parts []string
	for _, h := range nonEmpty(l.Header) {
		if !contacts[strings.TrimSpace(h)] {
			parts = append(parts, j.runs(parseRich(h), line))
		}
	}
	parts = append(parts, j.contacts(ld.Basics.Contacts, line)...)
	if strings.TrimSpace(name) != "" {
		j.block(juliaPara{style: "Title", below: 0.6 * r, center: true}, j.runs(plain(name), runProps{size: 2.45 * r}))
	}
	sep := j.gap(0.75*size, size) + j.run("•", runProps{font: resumeFont, color: j.accent, size: 0.8 * size}) + j.gap(0.75*size, size)
	j.block(juliaPara{center: true, rule: 1.1 * resumeSize}, strings.Join(parts, sep))

	if strings.TrimSpace(l.Title) != "" {
		j.block(juliaPara{style: "Heading1", keep: true, above: 1.8 * 1.55 * r, below: 1.1 * 1.55 * r},
			j.runs(plain(l.Title), runProps{spacing: 0.04 * 1.55 * r}))
	} else {
		j.space(1.8 * 1.55 * r)
	}

	if strings.TrimSpace(l.Greeting) != "" {
		j.block(juliaPara{above: 1, below: 1.2}, j.runs(plain(l.Greeting), runProps{}))
	}
	for _, blk := range l.Blocks {
		switch blk.Kind {
		case "paragraph":
			if strings.TrimSpace(blk.Text) != "" {
				j.block(juliaPara{above: 1, below: 1, justify: true}, j.runs(parseRich(blk.Text), runProps{}))
			}
		case "heading":
			if strings.TrimSpace(blk.Text) != "" {
				j.block(juliaPara{style: "Heading2", keep: true, above: 1.7, below: 0.75}, j.runs(plain(blk.Text), runProps{}))
			}
		case "bullets":
			// Each item hangs from an accent diamond, tabbed to the text.
			marker := j.run("◆", runProps{color: j.accent, size: 0.55}) + "<w:r><w:tab/></w:r>"
			items := nonEmpty(blk.Items)
			for i, it := range items {
				above := 0.6
				if i == 0 {
					above = 1
				}
				j.block(juliaPara{above: above, below: 0.6, left: 1.1 + 0.8, hanging: 0.8, justify: true}, marker+j.runs(parseRich(it), runProps{}))
			}
			if len(items) > 0 {
				j.below = 1
			}
		}
	}
	if strings.TrimSpace(l.Closing) != "" {
		signed := strings.TrimSpace(name) != ""
		j.block(juliaPara{keep: signed, above: 2}, j.runs(plain(l.Closing), runProps{}))
		if signed {
			j.block(juliaPara{above: 1.6}, j.runs(plain(name), runProps{font: juliaDisplay, bold: true, size: 1.35}))
		}
	}

	title := l.Title
	if strings.TrimSpace(name) != "" {
		title = name + " – " + title
	}
	return j.pack(title, name, j.styles(r), j.numbering())
}

// styles are the Julia document's styles: the name and section titles sized
// for a resume of r times the base size, body lines spaced as Typst's
// leading spaces them.
func (j *juliaWriter) styles(r float64) string {
	// Typst's line pitch is the cap height (0.7em in Quicksand) plus the
	// leading. Exact, as Word's single spacing depends on the font and the
	// platform (Quicksand's is 1.25em on a Mac, 1.49em on Windows).
	return fmt.Sprintf(juliaStylesXML,
		xmlText(j.st.Font), j.halfPoints(1), juliaInk, j.em(0.7+j.leading),
		xmlText(juliaDisplay), j.halfPoints(2.45*r),
		j.em(0.02*1.55*r), j.accent, j.halfPoints(1.55*r),
		j.halfPoints(1.08))
}

// juliaStylesXML takes: body font, size, ink, line pitch (twips); display
// font, title size; heading 1 letter spacing, accent, size; heading 2 size.
// Paragraphs set their own space before and after.
const juliaStylesXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:styles xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:docDefaults><w:rPrDefault><w:rPr><w:rFonts w:ascii="%[1]s" w:hAnsi="%[1]s" w:cs="%[1]s" w:eastAsia="%[1]s"/><w:color w:val="%[3]s"/><w:sz w:val="%[2]d"/><w:szCs w:val="%[2]d"/><w:lang w:val="en-US"/></w:rPr></w:rPrDefault><w:pPrDefault><w:pPr><w:spacing w:after="0" w:line="%[4]d" w:lineRule="exact"/></w:pPr></w:pPrDefault></w:docDefaults><w:style w:type="paragraph" w:default="1" w:styleId="Normal"><w:name w:val="Normal"/><w:qFormat/></w:style><w:style w:type="paragraph" w:styleId="Title"><w:name w:val="Title"/><w:basedOn w:val="Normal"/><w:next w:val="Normal"/><w:qFormat/><w:pPr><w:spacing w:line="240" w:lineRule="auto"/><w:jc w:val="center"/></w:pPr><w:rPr><w:rFonts w:ascii="%[5]s" w:hAnsi="%[5]s" w:cs="%[5]s"/><w:b/><w:sz w:val="%[6]d"/><w:szCs w:val="%[6]d"/></w:rPr></w:style><w:style w:type="paragraph" w:styleId="Heading1"><w:name w:val="heading 1"/><w:basedOn w:val="Normal"/><w:next w:val="Normal"/><w:qFormat/><w:pPr><w:keepNext/><w:spacing w:line="240" w:lineRule="auto"/><w:outlineLvl w:val="0"/></w:pPr><w:rPr><w:rFonts w:ascii="%[5]s" w:hAnsi="%[5]s" w:cs="%[5]s"/><w:b/><w:caps/><w:color w:val="%[8]s"/><w:spacing w:val="%[7]d"/><w:sz w:val="%[9]d"/><w:szCs w:val="%[9]d"/></w:rPr></w:style><w:style w:type="paragraph" w:styleId="Heading2"><w:name w:val="heading 2"/><w:basedOn w:val="Normal"/><w:next w:val="Normal"/><w:qFormat/><w:pPr><w:keepNext/><w:outlineLvl w:val="1"/></w:pPr><w:rPr><w:rFonts w:ascii="%[5]s" w:hAnsi="%[5]s" w:cs="%[5]s"/><w:b/><w:sz w:val="%[10]d"/><w:szCs w:val="%[10]d"/></w:rPr></w:style></w:styles>`

// iconXML takes: the vertical offset (half-points), the size (EMUs), the
// drawing's id, the icon's number (its media files and relationships), the
// run's panel (border and shading).
const iconXML = `<w:r><w:rPr><w:position w:val="%[1]d"/>%[5]s</w:rPr><w:drawing><wp:inline distT="0" distB="0" distL="0" distR="0"><wp:extent cx="%[2]d" cy="%[2]d"/><wp:effectExtent l="0" t="0" r="0" b="0"/><wp:docPr id="%[3]d" name="Icon %[3]d"><a:extLst><a:ext uri="{C183D7F6-B498-43B3-948B-1728B52AA6E4}"><adec:decorative xmlns:adec="http://schemas.microsoft.com/office/drawing/2017/decorative" val="1"/></a:ext></a:extLst></wp:docPr><wp:cNvGraphicFramePr><a:graphicFrameLocks noChangeAspect="1"/></wp:cNvGraphicFramePr><a:graphic><a:graphicData uri="http://schemas.openxmlformats.org/drawingml/2006/picture"><pic:pic><pic:nvPicPr><pic:cNvPr id="%[3]d" name="icon%[4]d.png"/><pic:cNvPicPr/></pic:nvPicPr><pic:blipFill><a:blip r:embed="rIdP%[4]d"><a:extLst><a:ext uri="{96DAC541-7B7A-43D3-8B79-37D633B846F1}"><asvg:svgBlip xmlns:asvg="http://schemas.microsoft.com/office/drawing/2016/SVG/main" r:embed="rIdS%[4]d"/></a:ext></a:extLst></a:blip><a:stretch><a:fillRect/></a:stretch></pic:blipFill><pic:spPr><a:xfrm><a:off x="0" y="0"/><a:ext cx="%[2]d" cy="%[2]d"/></a:xfrm><a:prstGeom prst="rect"><a:avLst/></a:prstGeom></pic:spPr></pic:pic></a:graphicData></a:graphic></wp:inline></w:drawing></w:r>`

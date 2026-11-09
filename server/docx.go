package main

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"math"
	"regexp"
	"strings"
)

// Word export. The document mirrors the resume's template with native Word
// features, so the text stays clean for parsers: bullets are list numbering,
// dot leaders are tab leaders, capitals are the caps font effect. Columns stack
// vertically (no tables) to keep each column's words together. This file
// writes the classic template's layout; docx_julia.go the Julia template's.

const (
	docxInk   = "373737" // luma(55) in the template
	docxMuted = "5F5F5F" // luma(95)
	pageW     = 12240    // US Letter width, twips
)

var hexColor = regexp.MustCompile(`^#?([0-9a-fA-F]{6})$`)

var docxBullets = map[string]string{"diamonds": "❖", "diamond": "◆", "square": "▪", "disc": "•"}

type docxWriter struct {
	st     Style
	accent string
	body   strings.Builder
	links  []string // hyperlink targets; link i has relationship id rIdL<i+1>
	bottom float64  // bottom page margin in inches; st.Margin sets the other three

	// Bold text's font and size (a multiple of the surrounding text's);
	// "" and 0 keep them.
	strongFont  string
	strongScale float64
	linkColor   string // links' color; "" for the accent

	// Icons, each drawn once as word/media/icon<i+1>.svg with a PNG copy
	// for readers without SVG support; drawings counts the pictures placed.
	icons     []string
	iconIndex map[string]int
	iconPNGs  [][]byte
	drawings  int
}

type runProps struct {
	bold, italic, caps, smallCaps bool
	font                          string // "" inherits
	color                         string
	size                          float64 // multiple of the base font size; 0 inherits
	spacing                       float64 // extra space after each character, in ems of the base font size
}

// docxResume writes the resume as Word in its template's layout.
func docxResume(ctx context.Context, doc Doc) ([]byte, error) {
	if doc.Style.Template == "julia" {
		return docxJuliaResume(ctx, doc)
	}
	d := newDocxWriter(*doc.Style)
	d.document(doc)
	name := ""
	if doc.Basics != nil {
		name = doc.Basics.Name
	}
	return d.pack(name+" – Resume", name, d.styles(), d.numbering())
}

// newDocxWriter starts a document in style st, filling in missing values.
func newDocxWriter(st Style) *docxWriter {
	d := &docxWriter{st: st, accent: "000000", iconIndex: map[string]int{}}
	if m := hexColor.FindStringSubmatch(d.st.Accent); m != nil {
		d.accent = strings.ToUpper(m[1])
	}
	if d.st.FontSize <= 0 {
		d.st.FontSize = 10
	}
	if d.st.Margin <= 0 {
		d.st.Margin = 0.6
	}
	d.bottom = d.st.Margin
	if d.st.Font == "" {
		d.st.Font = "Helvetica"
	}
	return d
}

// pack zips the written body with the given styles and numbering parts, and
// the icons, which drawIcons must have drawn.
func (d *docxWriter) pack(title, author, styles, numbering string) ([]byte, error) {
	if len(d.iconPNGs) != len(d.icons) {
		return nil, fmt.Errorf("%d icons placed, %d drawn", len(d.icons), len(d.iconPNGs))
	}
	var buf bytes.Buffer
	z := zip.NewWriter(&buf)
	add := func(name string, body []byte) error {
		f, err := z.Create(name)
		if err == nil {
			_, err = f.Write(body)
		}
		return err
	}
	parts := []struct{ name, body string }{
		{"[Content_Types].xml", contentTypes},
		{"_rels/.rels", packageRels},
		{"docProps/core.xml", fmt.Sprintf(coreXML, xmlText(title), xmlText(author))},
		{"word/_rels/document.xml.rels", d.documentRels()},
		{"word/styles.xml", styles},
		{"word/numbering.xml", numbering},
		{"word/document.xml", fmt.Sprintf(documentXML, d.body.String(), d.twips(d.st.Margin*72), d.twips(d.bottom*72))},
	}
	for _, p := range parts {
		if err := add(p.name, []byte(p.body)); err != nil {
			return nil, err
		}
	}
	for i, svg := range d.icons {
		if err := add(fmt.Sprintf("word/media/icon%d.svg", i+1), []byte(svg)); err != nil {
			return nil, err
		}
		if err := add(fmt.Sprintf("word/media/icon%d.png", i+1), d.iconPNGs[i]); err != nil {
			return nil, err
		}
	}
	if err := z.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (d *docxWriter) document(doc Doc) {
	if bs := doc.Basics; bs != nil {
		d.para(`<w:pStyle w:val="Title"/>`, d.runs(plain(bs.Name), runProps{}))
		if hl := nonEmpty(bs.Headline); len(hl) > 0 {
			d.para(fmt.Sprintf(`<w:spacing w:after="%d"/><w:jc w:val="center"/>`, d.em(0.65)),
				d.runs(plain(strings.Join(hl, " // ")), runProps{color: docxMuted, size: 1.3}))
		}
		var contacts []string
		for _, c := range bs.Contacts {
			if strings.TrimSpace(c.Text) == "" {
				continue
			}
			node := plain(c.Text)
			if c.URL != "" {
				node = []inline{{Kind: inlineLink, Text: c.Text, URL: c.URL}}
			}
			contacts = append(contacts, d.runs(node, runProps{color: d.accent, size: 0.95}))
		}
		if len(contacts) > 0 {
			sep := d.runs(plain(" | "), runProps{color: docxMuted, size: 0.95})
			d.para(`<w:jc w:val="center"/>`, strings.Join(contacts, sep))
		}
	}

	for _, sec := range doc.Sections {
		if sec.Kind == "columns" && sec.Variant == "sections" {
			for _, col := range sec.Columns {
				d.heading1(col.Title)
				d.lines(col.Lines, 1)
			}
			continue
		}
		d.heading1(sec.Title)
		switch sec.Kind {
		case "text":
			for _, p := range nonEmpty(sec.Paragraphs) {
				if d.st.ParagraphMarker == "chevron" {
					d.para(fmt.Sprintf(`<w:numPr><w:ilvl w:val="0"/><w:numId w:val="2"/></w:numPr><w:spacing w:before="%d"/>`, d.em(0.7)),
						d.runs(parseRich(p), runProps{}))
				} else {
					d.para(d.indent(), d.runs(parseRich(p), runProps{}))
				}
			}
		case "entries":
			for _, it := range sec.Items {
				if it.Type == "label" {
					if strings.TrimSpace(it.Text) != "" {
						d.para(fmt.Sprintf(`<w:keepNext/><w:spacing w:before="%d" w:after="0"/>%s`, d.em(1.6), d.indent()),
							d.runs(parseRich(it.Text), runProps{bold: true, italic: true}))
					}
					continue
				}
				d.entry(it.Title, it.Date, it.Subtitle, it.Location, nonEmpty(it.Bullets), sec.UppercaseTitles)
			}
		case "columns":
			for i, col := range sec.Columns {
				before := 0
				if i > 0 {
					before = d.em(0.8)
				}
				if strings.TrimSpace(col.Title) != "" {
					d.para(fmt.Sprintf(`<w:keepNext/><w:pBdr><w:bottom w:val="dotted" w:sz="4" w:space="2" w:color="%s"/></w:pBdr><w:spacing w:before="%d" w:after="%d"/>%s`,
						docxMuted, before, d.em(0.6), d.indent()),
						d.runs(plain(col.Title), runProps{bold: true, color: docxMuted}))
				}
				d.lines(col.Lines, 0.9)
			}
		}
	}
}

// entry writes an entry as one unbreakable block: the title and date share a
// line (right tab with leader), then subtitle and location, then bullets.
func (d *docxWriter) entry(title, date, subtitle, location string, bullets []string, caps bool) {
	var paras []struct{ pPr, runs string }
	add := func(pPr, runs string) { paras = append(paras, struct{ pPr, runs string }{pPr, runs}) }

	head := d.runs(plain(title), runProps{caps: caps})
	if strings.TrimSpace(date) != "" {
		head += `<w:r><w:tab/></w:r>` + d.runs(plain(date), runProps{})
	}
	add(`<w:pStyle w:val="Heading2"/>`, head)
	if strings.TrimSpace(subtitle) != "" || strings.TrimSpace(location) != "" {
		line := d.runs(parseRich(subtitle), runProps{italic: true, color: d.accent})
		if strings.TrimSpace(location) != "" {
			line += `<w:r><w:tab/></w:r>` + d.runs(plain(location), runProps{italic: true, color: d.accent})
		}
		add(fmt.Sprintf(`<w:tabs><w:tab w:val="right" w:pos="%d"/></w:tabs><w:spacing w:after="%d"/>%s`, d.textWidth(), d.em(0.45), d.indent()), line)
	}
	for _, b := range bullets {
		pPr := d.indentBy(2.1)
		if _, ok := docxBullets[d.st.Bullet]; ok {
			pPr = `<w:numPr><w:ilvl w:val="0"/><w:numId w:val="1"/></w:numPr>`
		}
		add(pPr, d.runs(parseRich(b), runProps{}))
	}
	for i, p := range paras {
		if i < len(paras)-1 && !strings.HasPrefix(p.pPr, "<w:pStyle") { // Heading2 already keeps with next
			p.pPr = "<w:keepNext/>" + p.pPr
		}
		d.para(p.pPr, p.runs)
	}
}

func (d *docxWriter) heading1(title string) {
	if strings.TrimSpace(title) != "" {
		d.para(`<w:pStyle w:val="Heading1"/>`, d.runs(plain(title), runProps{}))
	}
}

func (d *docxWriter) lines(lines []string, size float64) {
	for _, l := range nonEmpty(lines) {
		d.para(d.indent(), d.runs(parseRich(l), runProps{size: size}))
	}
}

func (d *docxWriter) para(pPr, runs string) {
	d.body.WriteString("<w:p><w:pPr>" + pPr + "</w:pPr>" + runs + "</w:p>")
}

// runs renders inline nodes as Word runs; bold and italic add to props (bold
// switches to strongFont, if set), links become hyperlinks in the link color.
func (d *docxWriter) runs(nodes []inline, p runProps) string {
	var b strings.Builder
	for _, n := range nodes {
		switch n.Kind {
		case inlineText:
			b.WriteString(d.run(n.Text, p))
		case inlineBold:
			q := p
			q.bold = true
			if d.strongFont != "" {
				q.font = d.strongFont
			}
			if d.strongScale != 0 {
				q.size = max(q.size, 1) * d.strongScale
			}
			b.WriteString(d.runs(n.Kids, q))
		case inlineItalic:
			q := p
			q.italic = true
			b.WriteString(d.runs(n.Kids, q))
		case inlineLink:
			d.links = append(d.links, n.URL)
			q := p
			q.color = d.accent
			if d.linkColor != "" {
				q.color = d.linkColor
			}
			fmt.Fprintf(&b, `<w:hyperlink r:id="rIdL%d" w:history="1">%s</w:hyperlink>`, len(d.links), d.run(n.Text, q))
		}
	}
	return b.String()
}

func (d *docxWriter) run(text string, p runProps) string {
	if text == "" {
		return ""
	}
	var b strings.Builder
	b.WriteString("<w:r><w:rPr>")
	if p.font != "" {
		fmt.Fprintf(&b, `<w:rFonts w:ascii="%[1]s" w:hAnsi="%[1]s" w:cs="%[1]s"/>`, xmlText(p.font))
	}
	if p.bold {
		b.WriteString("<w:b/>")
	}
	if p.italic {
		b.WriteString("<w:i/>")
	}
	if p.caps {
		b.WriteString("<w:caps/>")
	}
	if p.smallCaps {
		b.WriteString("<w:smallCaps/>")
	}
	if p.color != "" {
		fmt.Fprintf(&b, `<w:color w:val="%s"/>`, p.color)
	}
	if p.spacing != 0 {
		fmt.Fprintf(&b, `<w:spacing w:val="%d"/>`, d.em(p.spacing))
	}
	if p.size != 0 {
		fmt.Fprintf(&b, `<w:sz w:val="%d"/>`, d.halfPoints(p.size))
	}
	b.WriteString("</w:rPr>")
	// Line breaks typed in the editor become Word line breaks, as in the PDF.
	for i, line := range strings.Split(text, "\n") {
		if i > 0 {
			b.WriteString("<w:br/>")
		}
		b.WriteString(`<w:t xml:space="preserve">` + xmlText(line) + "</w:t>")
	}
	b.WriteString("</w:r>")
	return b.String()
}

func (d *docxWriter) styles() string {
	leader := map[string]string{"dotted": "dot", "dashed": "hyphen", "solid": "underscore"}[d.st.Leader]
	if leader == "" {
		leader = "none"
	}
	return fmt.Sprintf(stylesXML,
		xmlText(d.st.Font), d.halfPoints(1), docxInk, d.em(0.4),
		d.em(0.7), d.accent, d.halfPoints(2.25),
		d.em(1.5), d.em(0.85), d.accent, d.halfPoints(1.32),
		leader, d.textWidth(), d.em(1.1), d.em(0.4), d.em(2.05), d.accent,
	)
}

func (d *docxWriter) numbering() string {
	bullet := docxBullets[d.st.Bullet]
	if bullet == "" {
		bullet = docxBullets["disc"] // unused: bullets without a marker are plain indented paragraphs
	}
	return fmt.Sprintf(numberingXML,
		xmlText(bullet), d.em(2.05+2.7), d.em(1.3), docxMuted,
		d.em(2.05+1), d.em(1), docxMuted)
}

func (d *docxWriter) documentRels() string {
	var b strings.Builder
	for i, u := range d.links {
		fmt.Fprintf(&b, `<Relationship Id="rIdL%d" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/hyperlink" Target="%s" TargetMode="External"/>`,
			i+1, xmlText(strings.ReplaceAll(strings.TrimSpace(u), " ", "%20")))
	}
	for i := range d.icons {
		for _, ext := range []string{"svg", "png"} {
			fmt.Fprintf(&b, `<Relationship Id="rId%s%d" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/image" Target="media/icon%[2]d.%[3]s"/>`,
				strings.ToUpper(ext[:1]), i+1, ext)
		}
	}
	return fmt.Sprintf(documentRelsXML, b.String())
}

func (d *docxWriter) indent() string { return d.indentBy(0) }

// indentBy indents a paragraph past the section heading's square, as the
// template's pad(left: 2.05 * base) does, plus extra ems.
func (d *docxWriter) indentBy(extra float64) string {
	return fmt.Sprintf(`<w:ind w:left="%d"/>`, d.em(2.05+extra))
}

// em converts a length in ems of the base font size to twips.
func (d *docxWriter) em(x float64) int { return d.twips(x * d.st.FontSize) }

func (d *docxWriter) twips(pt float64) int { return int(math.Round(pt * 20)) }

func (d *docxWriter) halfPoints(x float64) int { return int(math.Round(x * d.st.FontSize * 2)) }

func (d *docxWriter) textWidth() int { return pageW - 2*d.twips(d.st.Margin*72) }

func xmlText(s string) string {
	var b strings.Builder
	xml.EscapeText(&b, []byte(s))
	return b.String()
}

const contentTypes = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/><Default Extension="xml" ContentType="application/xml"/><Default Extension="png" ContentType="image/png"/><Default Extension="svg" ContentType="image/svg+xml"/><Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/><Override PartName="/word/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.styles+xml"/><Override PartName="/word/numbering.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.numbering+xml"/><Override PartName="/docProps/core.xml" ContentType="application/vnd.openxmlformats-package.core-properties+xml"/></Types>`

const packageRels = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/><Relationship Id="rId2" Type="http://schemas.openxmlformats.org/package/2006/relationships/metadata/core-properties" Target="docProps/core.xml"/></Relationships>`

const coreXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<cp:coreProperties xmlns:cp="http://schemas.openxmlformats.org/package/2006/metadata/core-properties" xmlns:dc="http://purl.org/dc/elements/1.1/"><dc:title>%s</dc:title><dc:creator>%s</dc:creator></cp:coreProperties>`

const documentRelsXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles" Target="styles.xml"/><Relationship Id="rId2" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/numbering" Target="numbering.xml"/>%s</Relationships>`

// documentXML takes the body paragraphs, the page margin and the bottom margin in twips.
const documentXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships" xmlns:wp="http://schemas.openxmlformats.org/drawingml/2006/wordprocessingDrawing" xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" xmlns:pic="http://schemas.openxmlformats.org/drawingml/2006/picture"><w:body>%s<w:sectPr><w:pgSz w:w="12240" w:h="15840"/><w:pgMar w:top="%[2]d" w:right="%[2]d" w:bottom="%[3]d" w:left="%[2]d" w:header="720" w:footer="720" w:gutter="0"/></w:sectPr></w:body></w:document>`

// stylesXML takes: font, size, ink, paragraph spacing; title spacing, accent,
// size; heading 1 before, after, accent, size; heading 2 leader, tab position,
// before, after, indent, accent.
const stylesXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:styles xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:docDefaults><w:rPrDefault><w:rPr><w:rFonts w:ascii="%[1]s" w:hAnsi="%[1]s" w:cs="%[1]s" w:eastAsia="%[1]s"/><w:color w:val="%[3]s"/><w:sz w:val="%[2]d"/><w:szCs w:val="%[2]d"/><w:lang w:val="en-US"/></w:rPr></w:rPrDefault><w:pPrDefault><w:pPr><w:spacing w:after="%[4]d" w:line="252" w:lineRule="auto"/></w:pPr></w:pPrDefault></w:docDefaults><w:style w:type="paragraph" w:default="1" w:styleId="Normal"><w:name w:val="Normal"/><w:qFormat/></w:style><w:style w:type="paragraph" w:styleId="Title"><w:name w:val="Title"/><w:basedOn w:val="Normal"/><w:next w:val="Normal"/><w:qFormat/><w:pPr><w:spacing w:after="%[5]d"/><w:jc w:val="center"/></w:pPr><w:rPr><w:b/><w:caps/><w:color w:val="%[6]s"/><w:sz w:val="%[7]d"/><w:szCs w:val="%[7]d"/></w:rPr></w:style><w:style w:type="paragraph" w:styleId="Heading1"><w:name w:val="heading 1"/><w:basedOn w:val="Normal"/><w:next w:val="Normal"/><w:qFormat/><w:pPr><w:keepNext/><w:spacing w:before="%[8]d" w:after="%[9]d"/><w:outlineLvl w:val="0"/></w:pPr><w:rPr><w:b/><w:caps/><w:color w:val="%[10]s"/><w:sz w:val="%[11]d"/><w:szCs w:val="%[11]d"/></w:rPr></w:style><w:style w:type="paragraph" w:styleId="Heading2"><w:name w:val="heading 2"/><w:basedOn w:val="Normal"/><w:next w:val="Normal"/><w:qFormat/><w:pPr><w:keepNext/><w:tabs><w:tab w:val="right" w:leader="%[12]s" w:pos="%[13]d"/></w:tabs><w:spacing w:before="%[14]d" w:after="%[15]d"/><w:ind w:left="%[16]d"/><w:outlineLvl w:val="1"/></w:pPr><w:rPr><w:b/><w:color w:val="%[17]s"/></w:rPr></w:style></w:styles>`

// numberingXML takes: bullet character, text indent, hanging, color (numId 1,
// bullets: marker at 1.4em past the body, text 0.7em past the marker); then
// text indent, hanging, color (numId 2, paragraph chevrons).
const numberingXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:numbering xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:abstractNum w:abstractNumId="0"><w:multiLevelType w:val="singleLevel"/><w:lvl w:ilvl="0"><w:start w:val="1"/><w:numFmt w:val="bullet"/><w:lvlText w:val="%[1]s"/><w:lvlJc w:val="left"/><w:pPr><w:tabs><w:tab w:val="num" w:pos="%[2]d"/></w:tabs><w:ind w:left="%[2]d" w:hanging="%[3]d"/></w:pPr><w:rPr><w:color w:val="%[4]s"/></w:rPr></w:lvl></w:abstractNum><w:abstractNum w:abstractNumId="1"><w:multiLevelType w:val="singleLevel"/><w:lvl w:ilvl="0"><w:start w:val="1"/><w:numFmt w:val="bullet"/><w:lvlText w:val="›"/><w:lvlJc w:val="left"/><w:pPr><w:tabs><w:tab w:val="num" w:pos="%[5]d"/></w:tabs><w:ind w:left="%[5]d" w:hanging="%[6]d"/></w:pPr><w:rPr><w:color w:val="%[7]s"/></w:rPr></w:lvl></w:abstractNum><w:num w:numId="1"><w:abstractNumId w:val="0"/></w:num><w:num w:numId="2"><w:abstractNumId w:val="1"/></w:num></w:numbering>`

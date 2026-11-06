package main

import (
	"fmt"
	"strings"
)

// markdownLetter renders a cover letter as Markdown, as markdownResume does a resume.
func markdownLetter(ld LetterDoc) []byte {
	var b mdWriter
	l := ld.Letter
	b.heading(1, ld.Basics.Name)
	b.block(parseRich(strings.Join(nonEmpty(l.Header), "\n")))
	b.heading(2, l.Title)
	b.block(plain(l.Greeting))
	for _, blk := range l.Blocks {
		switch blk.Kind {
		case "paragraph":
			b.block(parseRich(blk.Text))
		case "heading":
			b.heading(3, blk.Text)
		case "bullets":
			b.list(blk.Items)
		}
	}
	if strings.TrimSpace(l.Closing) != "" {
		b.block(plain(strings.TrimRight(l.Closing+"\n"+ld.Basics.Name, "\n")))
	}
	return b.bytes()
}

// docxLetter writes a cover letter as Word, mirroring template/cover-letter.typ
// as docxResume mirrors the resume template: headings are heading styles,
// bullets list numbering, capitals the caps effect.
func docxLetter(ld LetterDoc) ([]byte, error) {
	l := ld.Letter
	d := newDocxWriter(*ld.Style)
	sans := d.st.Font // name and headings use the resume's font
	d.st.Font, d.st.FontSize = l.Style.Font, l.Style.FontSize
	if d.st.Font == "" {
		d.st.Font = "Libertinus Serif"
	}
	if d.st.FontSize <= 0 {
		d.st.FontSize = 12
	}
	name := ld.Basics.Name

	right := `<w:jc w:val="right"/>`
	if strings.TrimSpace(name) != "" {
		d.para(right, d.runs(plain(name), runProps{bold: true, caps: true, font: sans, color: d.accent, size: 1.75}))
	}
	for _, h := range nonEmpty(l.Header) {
		d.para(fmt.Sprintf(`<w:spacing w:before="%d"/>%s`, d.em(0.3), right), d.runs(parseRich(h), runProps{font: sans, color: docxMuted, size: 0.95}))
	}
	if strings.TrimSpace(l.Title) != "" {
		d.para(`<w:pStyle w:val="Heading1"/>`, d.runs(plain(l.Title), runProps{}))
	} else {
		d.para(fmt.Sprintf(`<w:spacing w:after="%d"/>`, d.em(2.4)), "")
	}

	// As in the template, a paragraph right after another (or the greeting)
	// continues the run with an indented first line; space between other
	// blocks comes from the later one's spacing before (headings carry their own).
	prev := "heading"
	if strings.TrimSpace(l.Greeting) != "" {
		d.para(fmt.Sprintf(`<w:spacing w:after="%d"/>`, d.em(1.2)), d.runs(plain(l.Greeting), runProps{}))
		prev = "paragraph"
	}
	gap := func() string {
		if prev == "bullets" {
			return fmt.Sprintf(`<w:spacing w:before="%d"/>`, d.em(1.2))
		}
		return ""
	}
	for _, blk := range l.Blocks {
		switch blk.Kind {
		case "paragraph":
			if strings.TrimSpace(blk.Text) == "" {
				continue
			}
			pPr := gap()
			if prev == "paragraph" {
				pPr = fmt.Sprintf(`<w:ind w:firstLine="%d"/>`, d.em(3))
			}
			d.para(pPr, d.runs(parseRich(blk.Text), runProps{}))
			prev = "paragraph"
		case "heading":
			if strings.TrimSpace(blk.Text) == "" {
				continue
			}
			d.para(`<w:pStyle w:val="Heading2"/>`, d.runs(plain(blk.Text), runProps{}))
			prev = "heading"
		case "bullets":
			items := nonEmpty(blk.Items)
			for i, it := range items {
				before := 0
				if i == 0 && prev != "heading" {
					before = d.em(1.2)
				}
				d.para(fmt.Sprintf(`<w:numPr><w:ilvl w:val="0"/><w:numId w:val="1"/></w:numPr><w:spacing w:before="%d"/>`, before),
					d.runs(parseRich(it), runProps{}))
			}
			if len(items) > 0 {
				prev = "bullets"
			}
		}
	}
	if strings.TrimSpace(l.Closing) != "" {
		d.para(fmt.Sprintf(`<w:keepNext/><w:spacing w:before="%d"/>`, d.em(1.8)), d.runs(plain(l.Closing), runProps{}))
		if strings.TrimSpace(name) != "" {
			d.para(fmt.Sprintf(`<w:spacing w:before="%d"/>`, d.em(1.2)), d.runs(plain(name), runProps{}))
		}
	}

	title := l.Title
	if strings.TrimSpace(name) != "" {
		title = name + " – " + title
	}
	styles := fmt.Sprintf(letterStylesXML,
		xmlText(d.st.Font), d.halfPoints(1), docxInk,
		xmlText(sans), d.em(2.4), d.em(1.2), d.accent, d.halfPoints(1.2),
		d.em(2), d.em(1.2), docxMuted, d.halfPoints(1.05))
	// Bullet text lines up 3em in, the marker 1.75em in (the template's list indent).
	numbering := fmt.Sprintf(numberingXML, "•", d.em(3), d.em(1.25), docxMuted, d.em(3), d.em(1.25), docxMuted)
	return d.pack(title, name, styles, numbering)
}

// letterStylesXML takes: body font, size, ink; heading font, heading 1 before,
// after, accent, size; heading 2 before, after, color, size.
const letterStylesXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:styles xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:docDefaults><w:rPrDefault><w:rPr><w:rFonts w:ascii="%[1]s" w:hAnsi="%[1]s" w:cs="%[1]s" w:eastAsia="%[1]s"/><w:color w:val="%[3]s"/><w:sz w:val="%[2]d"/><w:szCs w:val="%[2]d"/><w:lang w:val="en-US"/></w:rPr></w:rPrDefault><w:pPrDefault><w:pPr><w:spacing w:after="0" w:line="276" w:lineRule="auto"/></w:pPr></w:pPrDefault></w:docDefaults><w:style w:type="paragraph" w:default="1" w:styleId="Normal"><w:name w:val="Normal"/><w:qFormat/></w:style><w:style w:type="paragraph" w:styleId="Heading1"><w:name w:val="heading 1"/><w:basedOn w:val="Normal"/><w:next w:val="Normal"/><w:qFormat/><w:pPr><w:keepNext/><w:spacing w:before="%[5]d" w:after="%[6]d"/><w:outlineLvl w:val="0"/></w:pPr><w:rPr><w:rFonts w:ascii="%[4]s" w:hAnsi="%[4]s" w:cs="%[4]s"/><w:b/><w:caps/><w:color w:val="%[7]s"/><w:sz w:val="%[8]d"/><w:szCs w:val="%[8]d"/></w:rPr></w:style><w:style w:type="paragraph" w:styleId="Heading2"><w:name w:val="heading 2"/><w:basedOn w:val="Normal"/><w:next w:val="Normal"/><w:qFormat/><w:pPr><w:keepNext/><w:spacing w:before="%[9]d" w:after="%[10]d"/><w:outlineLvl w:val="1"/></w:pPr><w:rPr><w:rFonts w:ascii="%[4]s" w:hAnsi="%[4]s" w:cs="%[4]s"/><w:b/><w:caps/><w:color w:val="%[11]s"/><w:sz w:val="%[12]d"/><w:szCs w:val="%[12]d"/></w:rPr></w:style></w:styles>`

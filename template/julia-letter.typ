// Julia cover letter: the letter for a resume on template/julia.typ, in its
// look, as template/cover-letter.typ is for the classic template.
//
// Reads `data.json` written next to this file by the server: the letter, plus
// its resume's basics (the name and contacts at the top) and style (accent
// color, font size, page margin), so a letter always matches its resume.
//
// Layout, after julia.typ:
// - Letterhead: the resume's header at the resume's size, the name centered
//   over a gray line of the letter's header lines and the resume's contacts.
// - The letter's title as a section title; the body indented beneath it as a
//   section's content is, in the letter's font and size. Paragraphs are
//   separated by a gap, without first-line indents; subheadings are bold like
//   entry titles; bullets are indented paragraphs, as in the resume.
//
// Self-contained, like julia.typ: the server copies only this file beside the
// data. Keep `rich-pattern` and the field marks in sync with resume.typ, and
// the letterhead and title styles with julia.typ.

#let data = json("data.json")
#let basics = data.basics
#let st = data.style
#let letter = data.letter

#let accent = rgb(st.accent)
#let ink = rgb("#231f20")
#let muted = luma(130)
// The resume's size, for the letterhead and title, so they match the resume's.
#let rbase = st.fontSize * 1pt
// The letter's size, for its words.
#let base = letter.style.fontSize * 1pt
#let display = ("Source Sans 3", st.font)
#let light = 300
#let body-weight = 500
#let bold = 600

#set page(paper: "us-letter", margin: st.margin * 1in)
// Word spaces a little wider than the font's, as in julia.typ (pdftotext
// otherwise runs some words together).
#set text(font: letter.style.font, size: base, weight: body-weight, fill: ink, hyphenate: false, spacing: 115%)
#set strong(delta: 0)
#show strong: set text(font: display, weight: bold, size: 1.08em)
#set par(leading: 0.55em, spacing: 0.9em)
#show link: set text(fill: accent)

#set document(
  title: (if basics.name != "" { basics.name + " – " } else { "" }) + if letter.title != "" { letter.title } else { "Cover letter" },
  author: if basics.name != "" { basics.name } else { () },
)

// ── Inline rich text (same syntax as resume.typ) ───────────────────────────
#let rich-pattern = regex("\*\*\*(.+?)\*\*\*|\*\*(.+?)\*\*|\*(.+?)\*|\[([^\]]+)\]\(([^)\s]+)\)")
#let rich(txt) = {
  let out = ()
  let pos = 0
  for m in txt.matches(rich-pattern) {
    out.push(txt.slice(pos, m.start))
    let c = m.captures
    if c.at(0) != none {
      out.push(strong(emph(rich(c.at(0)))))
    } else if c.at(1) != none {
      out.push(strong(rich(c.at(1))))
    } else if c.at(2) != none {
      out.push(emph(rich(c.at(2))))
    } else {
      let l = link(c.at(4), c.at(3))
      out.push(if c.at(3).contains(regex("\s")) { l } else { box(l) })
    }
    pos = m.end
  }
  out.push(txt.slice(pos))
  out.join()
}

// ── Click-to-edit field map (as in cover-letter.typ) ───────────────────────
// Paths index the unfiltered arrays in data.json ("letter.blocks.3.items.1").
// The name and contacts belong to the resume and are edited there.
#let mark(path, edge, ..extra) = context [#metadata((
  path: path,
  edge: edge,
  page: here().page(),
  x: here().position().x / 1pt,
  y: here().position().y / 1pt,
  size: text.size / 1pt,
  ..extra.named(),
))<field>]
#let field(path, body) = { box(mark(path, "start")); body; box(mark(path, "end")) }
#let block-field(path, body) = layout(size => {
  mark(path, "top", width: size.width / 1pt)
  body
  mark(path, "bottom")
})

// ── Letterhead (julia.typ's header) ────────────────────────────────────────
#align(center, {
  set text(font: st.font, size: rbase)
  if basics.name != "" {
    block(below: 0.45em, text(font: display, size: 2.45 * rbase, weight: "bold", basics.name))
  }
  let parts = (
    letter.header.enumerate().filter(((i, h)) => h != "").map(((i, h)) => field("letter.header." + str(i), rich(h)))
      + basics.contacts.filter(c => c.text != "").map(c => if c.url != "" { link(c.url, c.text) } else { c.text })
  )
  if parts.len() > 0 {
    show link: set text(fill: muted)
    text(size: 1.05 * rbase, weight: light, fill: muted, parts.join(h(1.4em)))
  }
})

// ── Title (julia.typ's section title) ──────────────────────────────────────
#show heading.where(level: 1): it => block(above: 1.3 * 1.55 * rbase, below: 0.55 * 1.55 * rbase, sticky: true,
  text(font: display, size: 1.55 * rbase, weight: "bold", fill: accent, tracking: 0.02em, upper(it.body)))
// Subheadings, bold like julia.typ's entry titles.
#show heading.where(level: 2): it => block(above: 1.6em, below: 0.7em, sticky: true,
  text(font: display, size: 1.08 * base, weight: bold, it.body))

// Bullets are indented paragraphs without a marker, as in julia.typ, still
// tagged as a list.
#set list(marker: [], indent: 1.9em, body-indent: 0pt, spacing: 0.6em)

#if letter.title != "" { heading(level: 1, field("letter.title", letter.title)) } else { v(1.3 * 1.55 * rbase) }

// ── Body, indented as a resume section's content ───────────────────────────
// Entries in julia.typ hang this far in from the section titles.
#pad(left: 2.3 * rbase, {
  if letter.greeting != "" {
    block(below: 1.2em, par(field("letter.greeting", letter.greeting)))
  }

  for (i, b) in letter.blocks.enumerate() {
    let at = "letter.blocks." + str(i)
    if b.kind == "paragraph" {
      if b.text == "" { continue }
      block-field(at + ".text", par(rich(b.text)))
    } else if b.kind == "heading" {
      if b.text == "" { continue }
      heading(level: 2, field(at + ".text", b.text))
    } else if b.kind == "bullets" {
      let items = b.items.enumerate().filter(((j, it)) => it != "")
      if items.len() == 0 { continue }
      block(above: 0.9em, below: 0.9em, list(..items.map(((j, it)) => block-field(at + ".items." + str(j), par(hanging-indent: 1em, rich(it))))))
    }
  }

  if letter.closing != "" {
    block(above: 1.8em, breakable: false, {
      par(field("letter.closing", letter.closing))
      if basics.name != "" { block(above: 1.2em, par(basics.name)) }
    })
  }
})

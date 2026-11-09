// Julia cover letter: the letter for a resume on template/julia.typ, in its
// look, as template/cover-letter.typ is for the classic template.
//
// Reads `data.json` written next to this file by the server: the letter, plus
// its resume's basics (the name and contacts at the top) and style (accent
// color, font size, page margin), so a letter always matches its resume.
//
// Layout, after julia.typ:
// - Letterhead: the resume's header at the resume's size: the name centered
//   over a gray line of the letter's header lines and the resume's contacts
//   (a header line repeating a contact is shown once), parted by small accent
//   dots, then a hairline with an accent diamond at its middle.
// - The letter's title in julia.typ's section title style, a hairline running
//   on from it to the margin.
// - The body in the letter's font and size, justified, on a letter's measure:
//   the page margin is at least 0.8in, whatever the resume's.
//   Paragraphs are separated by a gap, without first-line indents;
//   subheadings are bold like entry titles; bullets hang from small accent
//   diamonds, the letterhead's ornament.
// - The closing, then the name as a signature in the name's font.
//
// Every rule, dot and diamond is drawn, never typed, so the text layer holds
// only the letter's words; they are tagged as artifacts.
//
// Self-contained, like julia.typ: the server copies only this file beside the
// data. Keep `rich-pattern` and the field marks in sync with resume.typ, the
// letterhead and title styles with julia.typ, and the layout with
// docxJuliaLetter in server/docx_julia.go.

#let data = json("data.json")
#let basics = data.basics
#let st = data.style
#let letter = data.letter

#let accent = rgb(st.accent)
#let ink = rgb("#231f20")
#let muted = luma(130)
#let rule-color = luma(205)
// The resume's size, for the letterhead and title, so they match the resume's.
#let rbase = st.fontSize * 1pt
// The letter's size, for its words.
#let base = letter.style.fontSize * 1pt
#let display = ("Source Sans 3", st.font)
#let light = 300
#let body-weight = 500
#let bold = 600

// A resume's tight margins make lines too long to read in a letter.
#set page(paper: "us-letter", margin: calc.max(st.margin, 0.8) * 1in)
// Word spaces a little wider than the font's, as in julia.typ (pdftotext
// otherwise runs some words together).
#set text(font: letter.style.font, size: base, weight: body-weight, fill: ink, hyphenate: false, spacing: 115%)
#set strong(delta: 0)
#show strong: set text(font: display, weight: bold, size: 1.08em)
#set par(leading: 0.62em, spacing: 1em, justify: true)
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

// ── Ornaments ───────────────────────────────────────────────────────────────
// A hairline across the space it is given.
#let hairline = pdf.artifact(line(length: 100%, stroke: 0.5pt + rule-color))
// The accent diamond, s across its flat sides.
#let diamond(s) = rotate(45deg, square(size: s, fill: accent))
// A dot parting the letterhead's parts, centered on the x-height, with a
// word space either side so the text layer keeps them apart.
#let dot = [ #pdf.artifact(box(baseline: -0.3em, circle(radius: 0.12em, fill: accent))) ]

// ── Letterhead (julia.typ's header) ────────────────────────────────────────
#let contacts = basics.contacts.filter(c => c.text != "")
#align(center, {
  set text(font: st.font, size: rbase)
  set par(justify: false)
  if basics.name != "" {
    block(below: 0.6em, text(font: display, size: 2.45 * rbase, weight: "bold", basics.name))
  }
  let parts = (
    letter.header.enumerate()
      .filter(((i, h)) => h != "" and not contacts.any(c => c.text == h.trim()))
      .map(((i, h)) => field("letter.header." + str(i), rich(h)))
      + contacts.map(c => if c.url != "" { link(c.url, c.text) } else { c.text })
  )
  if parts.len() > 0 {
    show link: set text(fill: muted)
    text(size: 1.05 * rbase, weight: light, fill: muted, parts.map(box).join(h(0.5em) + dot + h(0.5em)))
  }
  // The hairline, an accent diamond at its middle.
  block(above: 1.1em, below: 0pt, pdf.artifact(grid(
    columns: (1fr, auto, 1fr),
    column-gutter: 0.6em,
    align: horizon,
    hairline,
    diamond(0.36em),
    hairline,
  )))
})

// ── Title (julia.typ's section title) ──────────────────────────────────────
#show heading.where(level: 1): it => block(above: 1.8 * 1.55 * rbase, below: 1.1 * 1.55 * rbase, sticky: true, grid(
  columns: (auto, 1fr),
  column-gutter: 0.7em,
  align: horizon,
  text(font: display, size: 1.55 * rbase, weight: "bold", fill: accent, tracking: 0.04em, upper(it.body)),
  hairline,
))
// Subheadings, bold like julia.typ's entry titles.
#show heading.where(level: 2): it => block(above: 1.7em, below: 0.75em, sticky: true,
  text(font: display, size: 1.08 * base, weight: bold, it.body))

#set list(marker: box(baseline: -0.33em, diamond(0.3em)), indent: 1.1em, body-indent: 0.8em, spacing: 0.6em)

#if letter.title != "" { heading(level: 1, field("letter.title", letter.title)) } else { v(1.8 * 1.55 * rbase) }

// ── Body ────────────────────────────────────────────────────────────────────
#if letter.greeting != "" {
  block(below: 1.2em, par(field("letter.greeting", letter.greeting)))
}

#for (i, b) in letter.blocks.enumerate() {
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
    block(above: 1em, below: 1em, list(..items.map(((j, it)) => block-field(at + ".items." + str(j), par(rich(it))))))
  }
}

// ── Closing and signature ──────────────────────────────────────────────────
#if letter.closing != "" {
  block(above: 2em, breakable: false, {
    par(field("letter.closing", letter.closing))
    if basics.name != "" {
      block(above: 1.6em, text(font: display, size: 1.35 * base, weight: bold, basics.name))
    }
  })
}

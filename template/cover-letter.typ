// Cover letter template.
//
// Reads `data.json` written next to this file by the server: the letter, plus
// its resume's basics (the name at the top) and style (accent color, heading
// font, page margin), so a letter always matches its resume.
//
// The bottom margin is half the resume's, as in samples/cover-letter, so a
// typical letter fits on one page.
//
// As in resume.typ, decorations (the heading square, bullets) are drawn as
// shapes, never typed as characters, so the PDF text layer holds only words.

#let data = json("data.json")
#let basics = data.basics
#let st = data.style
#let letter = data.letter

#let accent = rgb(st.accent)
#let ink = luma(55)
#let muted = luma(95)
#let base = letter.style.fontSize * 1pt
// Name and headings use the resume's font; the body uses the letter's.
#let sans = st.font

#set page(paper: "us-letter", margin: (x: st.margin * 1in, top: st.margin * 1in, bottom: st.margin * 0.5in))
#set text(font: letter.style.font, size: base, fill: ink, hyphenate: false)
#set par(leading: 0.65em, spacing: 0.65em)
#show link: set text(fill: accent)

#set document(
  title: (if basics.name != "" { basics.name + " – " } else { "" }) + if letter.title != "" { letter.title } else { "Cover letter" },
  author: if basics.name != "" { basics.name } else { () },
)

// ── Inline rich text ────────────────────────────────────────────────────────
// The same syntax and rendering as resume.typ (and server/rich.go,
// web/src/rich.ts); keep the patterns in sync.
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

// ── Click-to-edit field map ─────────────────────────────────────────────────
// As in resume.typ: paths index the unfiltered arrays in data.json
// ("letter.blocks.3.items.1"). The name belongs to the resume and is edited there.
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

// ── Headings ────────────────────────────────────────────────────────────────
// The letter's title, with the resume's section square hung in the margin.
#show heading.where(level: 1): it => block(above: 2.4em, below: 1.2em, sticky: true, {
  let s = 0.85 * base
  box(width: 0pt, place(bottom + left, dx: -s - 1.05 * base, rect(width: s, height: s, fill: accent)))
  text(font: sans, size: 1.2 * base, weight: "bold", fill: accent, upper(it.body))
})
// Subheadings within the letter.
#show heading.where(level: 2): it => block(above: 2em, below: 1.2em, sticky: true,
  text(font: sans, size: 1.05 * base, weight: "bold", fill: muted, upper(it.body)))

#set list(
  marker: box(baseline: -0.1em, circle(radius: 0.19em, fill: muted)),
  indent: 1.75em,
  body-indent: 0.87em,
  spacing: 0.65em,
)

// ── Header ──────────────────────────────────────────────────────────────────
#align(right, {
  set par(spacing: 0.65em)
  if basics.name != "" {
    par(text(font: sans, size: 1.75 * base, weight: "bold", fill: accent, upper(basics.name)))
  }
  for (i, h) in letter.header.enumerate().filter(((i, h)) => h != "") {
    par(text(font: sans, size: 0.95 * base, fill: muted, field("letter.header." + str(i), rich(h))))
  }
})

// ── Body ────────────────────────────────────────────────────────────────────
#if letter.title != "" { heading(level: 1, field("letter.title", letter.title)) } else { v(2.4em) }

#if letter.greeting != "" {
  block(below: 1.2em, par(field("letter.greeting", letter.greeting)))
}

// A paragraph right after another (or after the greeting) continues the run:
// no gap, first line indented. After a heading or list it starts flush.
#let after-paragraph = letter.greeting != ""
#for (i, b) in letter.blocks.enumerate() {
  let at = "letter.blocks." + str(i)
  if b.kind == "paragraph" {
    if b.text == "" { continue }
    let indent = if after-paragraph { 3em } else { 0pt }
    block(above: if after-paragraph { 0.65em } else { 1.2em }, below: 0.65em,
      block-field(at + ".text", par(first-line-indent: (amount: indent, all: true), rich(b.text))))
    after-paragraph = true
  } else if b.kind == "heading" {
    if b.text == "" { continue }
    heading(level: 2, field(at + ".text", b.text))
    after-paragraph = false
  } else if b.kind == "bullets" {
    let items = b.items.enumerate().filter(((j, it)) => it != "")
    if items.len() == 0 { continue }
    block(above: 1.2em, below: 1.2em, list(..items.map(((j, it)) => block-field(at + ".items." + str(j), rich(it)))))
    after-paragraph = false
  }
}

#if letter.closing != "" {
  block(above: 1.8em, breakable: false, {
    par(field("letter.closing", letter.closing))
    if basics.name != "" { block(above: 1.2em, par(basics.name)) }
  })
}

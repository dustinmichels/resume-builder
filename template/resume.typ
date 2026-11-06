// Resume template.
//
// Reads `data.json` (content + style) written next to this file by the server,
// and attaches `resume.json` (content only) to the PDF.
//
// Machine-readability rule: every decoration (dot leaders, bullets, separators,
// section squares) is drawn as a shape, never typed as a character, so the PDF
// text layer contains only the resume's words.

#let data = json("data.json")
#let basics = data.basics
#let st = data.style

#let accent = rgb(st.accent)
#let ink = luma(55)
#let muted = luma(95)
#let base = st.fontSize * 1pt

#set page(paper: "us-letter", margin: st.margin * 1in)
#set text(
  font: st.font,
  size: base,
  fill: ink,
  hyphenate: false,
)
#set par(leading: 0.42em, spacing: 0.55em)
#show link: set text(fill: accent)

// ── Inline rich text ────────────────────────────────────────────────────────
// Strings are rendered as literal text, never evaluated as Typst markup, so
// characters like # $ @ in resume content are safe. Supported inline syntax:
// ***bold italic***, **bold**, *italic*, [label](url). server/rich.go and
// web/src/rich.ts parse the same syntax; keep the patterns in sync.
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
      // A label without spaces (a URL) stays on one line: a break inside it
      // would split the URL in the PDF text layer.
      let l = link(c.at(4), c.at(3))
      out.push(if c.at(3).contains(regex("\s")) { l } else { box(l) })
    }
    pos = m.end
  }
  out.push(txt.slice(pos))
  out.join()
}

// The same text with markup removed, for metadata fields.
#let plain-text(txt) = txt.replace(rich-pattern, m => plain-text(m.captures.slice(0, 4).find(c => c != none)))

// ── Document metadata ───────────────────────────────────────────────────────
// What file browsers, PDF viewers and applicant-tracking systems show without
// reading the page: name as title and author, headline as subject, and the
// skills grid (table-variant columns sections) split into keywords.
// Sections, items and columns with `hidden: true` are left off the page.
#let shown(x) = not x.at("hidden", default: false)
#let keywords = (
  data.sections
    .filter(s => shown(s) and s.kind == "columns" and s.at("variant", default: "table") == "table")
    .map(s => s.columns.filter(shown).map(c => c.lines.map(l => plain-text(l).split(regex("[,;()]")))))
    .flatten()
    .map(k => k.trim())
    .filter(k => k != "")
    .dedup()
)

#set document(
  title: if basics.name != "" { basics.name + " – Resume" } else { "Resume" },
  author: if basics.name != "" { basics.name } else { () },
  description: basics.headline.filter(h => h.trim() != "").join(" · "),
  keywords: keywords,
)

#pdf.attach(
  "resume.json",
  mime-type: "application/json",
  description: "Machine-readable resume (JSON)",
)

// ── Click-to-edit field map ─────────────────────────────────────────────────
// Each editable value is wrapped so the editor can map a click on the page
// back to the data. `mark` records an invisible position; the server reads
// them with `typst eval 'query(<field>)'`. Paths index the unfiltered arrays
// in data.json (e.g. "sections.2.items.0.bullets.3").
#let mark(path, edge, ..extra) = context [#metadata((
  path: path,
  edge: edge,
  page: here().page(),
  x: here().position().x / 1pt,
  y: here().position().y / 1pt,
  size: text.size / 1pt,
  ..extra.named(),
))<field>]
// Inline value: marks sit on the baseline at the start and end of the text.
#let field(path, body) = { box(mark(path, "start")); body; box(mark(path, "end")) }
// Block value (bullet, paragraph): marks sit at the block's top and bottom,
// and the start mark carries the available width.
#let block-field(path, body) = layout(size => {
  mark(path, "top", width: size.width / 1pt)
  body
  mark(path, "bottom")
})

// ── Drawn decorations ───────────────────────────────────────────────────────
#let leader = if st.leader == "none" {
  h(1fr)
} else {
  let dash = if st.leader == "solid" { "solid" } else { st.leader }
  box(width: 1fr, inset: (x: 2pt), line(
    length: 100%,
    stroke: (paint: accent, thickness: 0.9pt, dash: dash),
  ))
}

#let diamond(size, fill) = polygon(
  fill: fill,
  (size / 2, 0pt),
  (size, size / 2),
  (size / 2, size),
  (0pt, size / 2),
)

#let marker = {
  let m = 0.62 * base // marker box size
  let lift = -0.08 * base // nudge up to sit on the x-height
  if st.bullet == "diamonds" {
    // Four small diamonds arranged like ❖, drawn as shapes.
    let d = m * 0.48
    let off = (m - d) / 2
    box(width: m, height: m, baseline: lift, {
      place(dx: off, dy: 0pt, diamond(d, muted))
      place(dx: off, dy: m - d, diamond(d, muted))
      place(dx: 0pt, dy: off, diamond(d, muted))
      place(dx: m - d, dy: off, diamond(d, muted))
    })
  } else if st.bullet == "diamond" {
    box(baseline: lift, diamond(m * 0.8, muted))
  } else if st.bullet == "square" {
    box(baseline: lift, rect(width: m * 0.55, height: m * 0.55, fill: muted))
  } else if st.bullet == "disc" {
    box(baseline: lift, circle(radius: m * 0.25, fill: muted))
  } else {
    box(width: 0pt)
  }
}

#let chevron = box(width: 0.42 * base, height: 0.62 * base, baseline: -0.05 * base, {
  let w = 0.32 * base
  let h = 0.62 * base
  let s = 0.85pt + muted
  place(line(start: (0pt, 0pt), end: (w, h / 2), stroke: s))
  place(line(start: (w, h / 2), end: (0pt, h), stroke: s))
})

#set list(marker: marker, indent: 1.4em, body-indent: 0.7em, spacing: 0.4em)

#show heading.where(level: 1): it => block(above: 1.5em, below: 0.85em, sticky: true, {
  box(rect(width: 0.95 * base, height: 0.95 * base, fill: accent))
  h(1.1 * base)
  text(size: 1.32 * base, weight: "bold", fill: accent, upper(it.body))
})

// ── Header ──────────────────────────────────────────────────────────────────
#align(center, {
  block(below: 0.7em, text(size: 2.25 * base, weight: "bold", fill: accent, field("basics.name", upper(basics.name))))
  if basics.headline.len() > 0 {
    block(below: 0.65em, text(size: 1.3 * base, fill: muted, basics.headline.enumerate().map(((i, h)) => field("basics.headline." + str(i), h)).join(" // ")))
  }
  let sep = [ #box(circle(radius: 0.17 * base, fill: muted), baseline: -0.2 * base) ]
  text(size: 0.95 * base, fill: accent, basics.contacts
    .enumerate()
    .filter(((i, c)) => c.text != "")
    .map(((i, c)) => field("basics.contacts." + str(i), if c.url != "" { link(c.url, c.text) } else { c.text }))
    .join(sep))
})

// ── Section kinds ───────────────────────────────────────────────────────────
// `at` is the section's path in data.json, for the field map.
#let text-section(sec, at) = {
  for (i, p) in sec.paragraphs.enumerate().filter(((i, p)) => p != "") {
    let at = at + ".paragraphs." + str(i)
    if st.paragraphMarker == "chevron" {
      block(above: 0.7em, below: 0pt, grid(columns: (auto, 1fr), column-gutter: 0.55em, chevron, block-field(at, rich(p))))
    } else {
      block-field(at, par(rich(p)))
    }
  }
}

#let entry(e, caps, at) = block(breakable: false, above: 1.1em, below: 0pt, {
  set par(spacing: 0.4em)
  let title = if caps { upper(e.title) } else { e.title }
  par(text(weight: "bold", fill: accent)[#field(at + ".title", title) #leader #field(at + ".date", e.date)])
  if e.subtitle != "" or e.location != "" {
    par(text(style: "italic", fill: accent)[#field(at + ".subtitle", rich(e.subtitle)) #h(1fr) #field(at + ".location", e.location)])
  }
  let bullets = e.bullets.enumerate().filter(((i, b)) => b != "")
  if bullets.len() > 0 {
    block(above: 0.45em, list(..bullets.map(((i, b)) => block-field(at + ".bullets." + str(i), rich(b)))))
  }
})

#let entries-section(sec, at) = {
  for (i, item) in sec.items.enumerate().filter(((i, item)) => shown(item)) {
    let at = at + ".items." + str(i)
    if item.type == "label" {
      block(above: 1.6em, below: 0pt, text(weight: "bold", style: "italic", field(at + ".text", rich(item.text))))
    } else {
      entry(item, sec.at("uppercaseTitles", default: false), at)
    }
  }
}

#let column-lines(c, at) = {
  for (i, l) in c.lines.enumerate().filter(((i, l)) => l != "") {
    block-field(at + ".lines." + str(i), par(rich(l)))
  }
}

// Table variant: one row per column, label beside its lines. Rows read left to
// right, so raw extraction, layout extraction and copy/paste all yield
// "Label: line" in order, and lines wrap at full width instead of in narrow
// columns. The typed colon keeps label and lines distinguishable in plain text.
// Side-by-side variant renders cell-by-cell (title, then its lines), so the
// text layer keeps each column's words together.
#let columns-section(sec, at) = {
  let cols = sec.columns.enumerate().filter(((i, c)) => shown(c)).map(((i, c)) => (c, at + ".columns." + str(i)))
  if cols.len() == 0 { return }
  if sec.at("variant", default: "table") == "table" {
    let rule = (paint: muted, thickness: 0.6pt, dash: "dotted")
    let pad-y = 0.5em
    grid(
      columns: (auto, 1fr),
      // Gap as inset, not gutter, so the row rules run unbroken.
      inset: (x, y) => (
        left: if x > 0 { 1.2em } else { 0pt },
        top: if y > 0 { pad-y } else { 0pt },
        bottom: if y < cols.len() - 1 { pad-y } else { 0pt },
      ),
      stroke: (x, y) => if y > 0 { (top: rule) },
      ..cols.map(((c, at)) => (
        if c.title != "" { text(weight: "bold", fill: muted)[#field(at + ".title", c.title):] } else { [] },
        { set par(spacing: 0.4em); column-lines(c, at) },
      )).flatten(),
    )
  } else {
    // Side-by-side sections: each column gets its own section heading.
    block(above: 1.5em, grid(
      columns: (1fr,) * cols.len(),
      column-gutter: 1.5em,
      ..cols.map(((c, at)) => {
        heading(level: 1, field(at + ".title", c.title))
        pad(left: 2.05 * base, column-lines(c, at))
      }),
    ))
  }
}

// ── Body ────────────────────────────────────────────────────────────────────
#for (i, sec) in data.sections.enumerate().filter(((i, sec)) => shown(sec)) {
  let at = "sections." + str(i)
  if sec.title != "" { heading(level: 1, field(at + ".title", sec.title)) }
  if sec.kind == "columns" and sec.at("variant", default: "table") == "sections" {
    columns-section(sec, at)
  } else {
    pad(left: 2.05 * base, {
      if sec.kind == "text" { text-section(sec, at) }
      else if sec.kind == "entries" { entries-section(sec, at) }
      else if sec.kind == "columns" { columns-section(sec, at) }
    })
  }
}

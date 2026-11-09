// Julia: a one-column resume with category icons, after
// samples/julia/Res.6.3.26-agfocus-icon2.pdf.
//
// Reads `data.json` (content + style) and attaches `resume.json` (content
// only), like template/resume.typ. Self-contained: the server copies only
// this file beside the data, so the shared helpers below are duplicated from
// resume.typ; keep `rich-pattern` and the field marks in sync with it.
//
// Layout:
// - Header: the name centered, then headline parts and contacts on one line.
// - Section titles in the accent color. An entries section's `legend` (lines
//   with icons) runs after its title, on a panel of its `fill` and inside a
//   rounded border of its `border` when set (neither by default, as in the
//   sample), each label in its icon's `color`.
// - Entries: organization (`subtitle`, small caps), location, date; then the
//   title in bold followed by the entry's `icons`; then each bullet as an
//   indented paragraph. Text after " - " in a title is set in the light
//   weight (a lab or department). A `compact` section runs each entry on one
//   line: organization, location, title, date, then the bullets. A
//   `brackets` section draws one thin bracket in the gutter beside its entries.
// - Columns sections: each column runs its lines together, each line's icon
//   after it (a list of interests).
//
// Every icon and gap is drawn, never typed, so the text layer holds only the
// resume's words; icons are tagged as artifacts.

#let data = json("data.json")
#let basics = data.basics
#let st = data.style

#let accent = rgb(st.accent)
#let ink = rgb("#231f20")
#let muted = luma(130)
#let base = st.fontSize * 1pt
// The name, section titles and bold text, after the sample's Source Sans Pro
// and Adrianna Demibold; the style's font is the body (the sample's Houschka
// Pro, for which Quicksand stands in). Source Sans 3 is bundled in
// template/fonts; without it, the style's font is used.
#let display = ("Source Sans 3", st.font)
// Weights, after the sample's Houschka Pro Light / Medium and Adrianna
// Demibold; Typst picks the nearest weight the font has.
#let light = 300
#let body-weight = 500
#let bold = 600

#set page(paper: "us-letter", margin: st.margin * 1in)
// Word spaces a little wider than the font's: at its default spacing,
// pdftotext runs some words together (after a period before a capital).
#set text(font: st.font, size: base, weight: body-weight, fill: ink, hyphenate: false, spacing: 115%)
#set strong(delta: 0)
// Source Sans has a smaller x-height than the body fonts; a touch larger, its
// bold text sits evenly beside them.
#show strong: set text(font: display, weight: bold, size: 1.08em)
#set par(leading: 0.38em, spacing: 0.45em)
#show link: set text(fill: accent)

// Small capitals drawn from capitals, so they work in fonts without real
// ones (Quicksand). Lowercase letters read as capitals in the text layer.
#let small-caps(body) = {
  show regex("\p{Ll}+"): it => text(size: 0.8em, upper(it))
  body
}

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

#let plain-text(txt) = txt.replace(rich-pattern, m => plain-text(m.captures.slice(0, 4).find(c => c != none)))

// ── Document metadata (as in resume.typ) ───────────────────────────────────
#let shown(x) = not x.at("hidden", default: false)
#let keywords = (
  data.sections
    .filter(s => shown(s) and s.kind == "columns")
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

#pdf.attach("resume.json", mime-type: "application/json", description: "Machine-readable resume (JSON)")

// ── Click-to-edit field map (as in resume.typ) ─────────────────────────────
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

// ── Icons ───────────────────────────────────────────────────────────────────
// A template icon keeps its own colors; a Lucide icon (drawn in currentColor)
// takes the accent. Placed out of flow, centered on the x-height, so icons
// never widen line spacing.
#let icon-size = 0.92em
#let icon(svg) = pdf.artifact(box(width: icon-size, place(dy: -0.335em - icon-size / 2, image(
  bytes(svg.replace("currentColor", accent.to-hex())),
  format: "svg",
  height: icon-size,
  fit: "contain",
))))

// The icon at `i` of a nullable array (`icons` absent, or null in JSON).
#let icon-at(x, i) = {
  let icons = x.at("icons", default: none)
  if icons != none { icons.at(i, default: none) }
}

// An entry's icons, each marked "<at>.icons.<k>", after a thin gap.
#let entry-icons(e, at) = {
  let icons = e.at("icons", default: none)
  if icons != none and icons.len() > 0 {
    h(0.3em)
    icons.enumerate().map(((k, ic)) => field(at + ".icons." + str(k), icon(ic.svg))).join(h(0.12em))
  }
}

// ── Header ──────────────────────────────────────────────────────────────────
#align(center, {
  block(below: 0.45em, text(font: display, size: 2.45 * base, weight: "bold", field("basics.name", basics.name)))
  let parts = (
    basics.headline.enumerate().filter(((i, h)) => h != "").map(((i, h)) => field("basics.headline." + str(i), h))
      + basics.contacts.enumerate().filter(((i, c)) => c.text != "").map(((i, c)) => field(
        "basics.contacts." + str(i),
        if c.url != "" { link(c.url, c.text) } else { c.text },
      ))
  )
  if parts.len() > 0 {
    // Contacts are links, but the line stays gray like the sample.
    show link: set text(fill: muted)
    text(size: 1.05 * base, weight: light, fill: muted, parts.join(h(1.4em)))
  }
})

// ── Sections ────────────────────────────────────────────────────────────────
// Entries hang this far in from the section titles; compact ones twice as far.
#let indent = 2.3 * base

#let section-title(title, at) = text(font: display, size: 1.55 * base, weight: "bold", fill: accent, tracking: 0.02em, upper(field(at + ".title", title)))

// A legend's color `key` ("fill" or "border"), "#rrggbb"; missing or "" for none.
#let legend-color(lg, key) = {
  let c = lg.at(key, default: "")
  if c == "" { none } else { rgb(c) }
}

// The legend: each line in bold, in its icon's color (editable per line), the
// icon before it, on a panel drawn outside the text's bounds (outset), so it
// never changes line spacing.
#let legend(lg, at) = {
  let items = lg.lines.enumerate().filter(((i, l)) => l != "").map(((i, l)) => {
    let ic = icon-at(lg, i)
    let fill = if ic != none and ic.at("color", default: none) != none { rgb(ic.color) } else { ink }
    box({
      if ic != none { field(at + ".legend.icons." + str(i), icon(ic.svg)); h(0.1em) }
      text(fill: fill, upper(field(at + ".legend.lines." + str(i), rich(l))))
    })
  })
  if items.len() > 0 {
    let border = legend-color(lg, "border")
    let stroke = if border == none { none } else { 0.75pt + border }
    text(size: 1.02 * base, weight: "bold", box(fill: legend-color(lg, "fill"), stroke: stroke, outset: (x: 0.5em, y: 0.35em), radius: 0.3em, items.join(h(0.7em))))
  }
}

#let heading-row(sec, at) = {
  let lg = sec.at("legend", default: none)
  let body = if lg == none { none } else { legend(lg, at) }
  block(above: 1.3em, below: 0.55em, sticky: true, if body == none {
    heading(level: 1, section-title(sec.title, at))
  } else {
    let panel = legend-color(lg, "fill") != none or legend-color(lg, "border") != none
    grid(columns: (auto, 1fr), column-gutter: if panel { 1.3em } else { 0.9em }, align: bottom, heading(level: 1, section-title(sec.title, at)), body)
  })
}

#show heading.where(level: 1): it => it.body

// "Research Intern - Cardozo Lab": the part after " - " in the light weight.
#let entry-title(title) = {
  let parts = title.split(" - ")
  strong(parts.first())
  if parts.len() > 1 { text(weight: light, " - " + parts.slice(1).join(" - ")) }
}

#let entry(e, at) = block(breakable: false, above: 0.95em, below: 0pt, {
  set par(spacing: 0.32em)
  let org = (
    (if e.subtitle != "" { field(at + ".subtitle", small-caps(rich(e.subtitle))) },)
      + (if e.location != "" { field(at + ".location", e.location) },)
      + (if e.date != "" { field(at + ".date", e.date) },)
  ).filter(x => x != none)
  if org.len() > 0 or e.title == "" {
    block(below: if e.title != "" { 0.42em } else { 0.32em }, par(text(size: 1.04 * base, weight: light, org.join[, ] + if e.title == "" { entry-icons(e, at) })))
  }
  if e.title != "" {
    par(text(size: 1.05 * base, field(at + ".title", entry-title(e.title)) + entry-icons(e, at)))
  }
  for (i, b) in e.bullets.enumerate().filter(((i, b)) => b != "") {
    pad(left: 1.9em, block-field(at + ".bullets." + str(i), par(hanging-indent: 1em, rich(b))))
  }
})

// One line per entry: organization, location, title, date, then bullets.
#let compact-entry(e, at) = block(above: 0.42em, below: 0pt, par(hanging-indent: 0em, {
  let soft = (body) => text(style: "italic", weight: light, fill: muted, body)
  set text(size: 1.08 * base, weight: light)
  if e.subtitle != "" { field(at + ".subtitle", small-caps(rich(e.subtitle))) }
  if e.location != "" { [, ]; soft(field(at + ".location", e.location)) }
  if e.title != "" { [, ]; strong(field(at + ".title", e.title)) }
  if e.date != "" { if e.title != "" [ ] else [, ]; soft(field(at + ".date", e.date)) }
  entry-icons(e, at)
  for (i, b) in e.bullets.enumerate().filter(((i, b)) => b != "") {
    [ ]
    text(size: 0.92 * base, field(at + ".bullets." + str(i), rich(b)))
  }
}))

// The section's content hung `left` in from its title; with `on`, a thin
// square bracket in the gutter beside it, from just above the first line's
// capitals to just below the last line's baseline. The bracket's stem is the
// block's left border, drawn outside its bounds (outset) and continued on the
// next page when the section breaks; its ticks are placed out of flow. So it
// never moves text, and draws no text.
#let bracket-stroke = 0.6pt + muted
#let bracket-gap = 0.9em
#let bracketed(left, on, body) = if not on { pad(left: left, body) } else {
  let tick(dy) = place(dx: -bracket-gap, dy: dy, pdf.artifact(line(length: 0.3em, stroke: bracket-stroke)))
  pad(left: left - bracket-gap, block(stroke: (left: bracket-stroke), outset: (y: 0.15em), inset: (left: bracket-gap), {
    tick(-0.15em)
    body
    tick(0.15em)
  }))
}

#let entries-section(sec, at) = {
  let compact = sec.at("compact", default: false)
  bracketed(if compact { 2 * indent } else { indent }, sec.at("brackets", default: false), for (i, item) in sec.items.enumerate().filter(((i, item)) => shown(item)) {
    let at = at + ".items." + str(i)
    if item.type == "label" {
      block(above: 1.1em, below: 0pt, strong(emph(field(at + ".text", rich(item.text)))))
    } else if compact {
      compact-entry(item, at)
    } else {
      entry(item, at)
    }
  })
}

#let text-section(sec, at) = pad(left: indent, for (i, p) in sec.paragraphs.enumerate().filter(((i, p)) => p != "") {
  block-field(at + ".paragraphs." + str(i), par(rich(p)))
})

// Each column is one paragraph: its title, then its lines run together,
// each line's icon after it.
#let columns-section(sec, at) = pad(left: indent, for (k, c) in sec.columns.enumerate().filter(((k, c)) => shown(c)) {
  let at = at + ".columns." + str(k)
  let lines = c.lines.enumerate().filter(((i, l)) => l != "").map(((i, l)) => {
    let ic = icon-at(c, i)
    box(field(at + ".lines." + str(i), rich(l)) + if ic != none { h(0.15em); field(at + ".icons." + str(i), icon(ic.svg)) })
  })
  block(above: 0.5em, below: 0pt, text(size: base, weight: light, {
    if c.title != "" { strong(field(at + ".title", c.title)); [: ] }
    lines.join(h(0.6em))
  }))
})

#for (i, sec) in data.sections.enumerate().filter(((i, sec)) => shown(sec)) {
  let at = "sections." + str(i)
  if sec.title != "" { heading-row(sec, at) } else { v(0.6em) }
  if sec.kind == "text" { text-section(sec, at) }
  else if sec.kind == "entries" { entries-section(sec, at) }
  else if sec.kind == "columns" { columns-section(sec, at) }
}

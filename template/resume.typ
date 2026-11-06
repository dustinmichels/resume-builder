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

#set document(
  title: basics.name + " – Resume",
  author: basics.name,
)
#set page(paper: "us-letter", margin: st.margin * 1in)
#set text(
  font: st.font,
  size: base,
  fill: ink,
  hyphenate: false,
)
#set par(leading: 0.42em, spacing: 0.55em)
#show link: set text(fill: accent)

#pdf.attach(
  "resume.json",
  mime-type: "application/json",
  description: "Machine-readable resume (JSON)",
)

// ── Inline rich text ────────────────────────────────────────────────────────
// Strings are rendered as literal text, never evaluated as Typst markup, so
// characters like # $ @ in resume content are safe. Supported inline syntax:
// **bold**, *italic*, [label](url).
#let rich-pattern = regex("\*\*(.+?)\*\*|\*(.+?)\*|\[([^\]]+)\]\(([^)\s]+)\)")
#let rich(txt) = {
  let out = ()
  let pos = 0
  for m in txt.matches(rich-pattern) {
    out.push(txt.slice(pos, m.start))
    let c = m.captures
    if c.at(0) != none {
      out.push(strong(rich(c.at(0))))
    } else if c.at(1) != none {
      out.push(emph(rich(c.at(1))))
    } else {
      out.push(link(c.at(3), c.at(2)))
    }
    pos = m.end
  }
  out.push(txt.slice(pos))
  out.join()
}

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
  block(below: 0.7em, text(size: 2.25 * base, weight: "bold", fill: accent, upper(basics.name)))
  if basics.headline.len() > 0 {
    block(below: 0.65em, text(size: 1.3 * base, fill: muted, basics.headline.join(" // ")))
  }
  let sep = [ #box(circle(radius: 0.17 * base, fill: muted), baseline: -0.2 * base) ]
  text(size: 0.95 * base, fill: accent, basics.contacts
    .filter(c => c.text != "")
    .map(c => if c.url != "" { link(c.url, c.text) } else { c.text })
    .join(sep))
})

// ── Section kinds ───────────────────────────────────────────────────────────
#let text-section(sec) = {
  for p in sec.paragraphs.filter(p => p != "") {
    if st.paragraphMarker == "chevron" {
      block(above: 0.7em, below: 0pt, grid(columns: (auto, 1fr), column-gutter: 0.55em, chevron, rich(p)))
    } else {
      par(rich(p))
    }
  }
}

#let entry(e, caps) = block(breakable: false, above: 1.1em, below: 0pt, {
  set par(spacing: 0.4em)
  let title = if caps { upper(e.title) } else { e.title }
  par(text(weight: "bold", fill: accent)[#title #leader #e.date])
  if e.subtitle != "" or e.location != "" {
    par(text(style: "italic", fill: accent)[#rich(e.subtitle) #h(1fr) #e.location])
  }
  let bullets = e.bullets.filter(b => b != "")
  if bullets.len() > 0 {
    block(above: 0.45em, list(..bullets.map(rich)))
  }
})

#let entries-section(sec) = {
  for item in sec.items {
    if item.type == "label" {
      block(above: 1.6em, below: 0pt, text(weight: "bold", style: "italic", rich(item.text)))
    } else {
      entry(item, sec.at("uppercaseTitles", default: false))
    }
  }
}

// Columns render cell-by-cell (title, then its lines), so the text layer keeps
// each column's words together instead of interleaving rows across columns.
#let columns-section(sec) = {
  let cols = sec.columns
  if cols.len() == 0 { return }
  if sec.at("variant", default: "table") == "table" {
    let rule = (paint: muted, thickness: 0.6pt, dash: "dotted")
    grid(
      columns: (1fr,) * cols.len(),
      inset: (x: 0.55em, y: 0pt),
      stroke: (x, y) => if x > 0 { (left: rule) },
      ..cols.map(c => {
        set text(size: 0.9 * base)
        block(below: 0.6em, inset: (bottom: 0.45em), stroke: (bottom: rule), width: 100%,
          text(weight: "bold", fill: muted, size: base, c.title))
        for l in c.lines.filter(l => l != "") { par(rich(l)) }
      }),
    )
  } else {
    // Side-by-side sections: each column gets its own section heading.
    block(above: 1.5em, grid(
      columns: (1fr,) * cols.len(),
      column-gutter: 1.5em,
      ..cols.map(c => {
        heading(level: 1, c.title)
        pad(left: 2.05 * base, for l in c.lines.filter(l => l != "") { par(rich(l)) })
      }),
    ))
  }
}

// ── Body ────────────────────────────────────────────────────────────────────
#for sec in data.sections {
  if sec.title != "" { heading(level: 1, sec.title) }
  if sec.kind == "columns" and sec.at("variant", default: "table") == "sections" {
    columns-section(sec)
  } else {
    pad(left: 2.05 * base, {
      if sec.kind == "text" { text-section(sec) }
      else if sec.kind == "entries" { entries-section(sec) }
      else if sec.kind == "columns" { columns-section(sec) }
    })
  }
}

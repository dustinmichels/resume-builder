# Writing templates

Templates are [Typst](https://typst.app) files in `template/`. The server re-reads them on every render, so with
`mise run dev` running, saving a template updates the preview in the editor.

| Template | Resume                | Letter                       | Icons                                                         |
| -------- | --------------------- | ---------------------------- | ------------------------------------------------------------- |
| Classic  | `template/resume.typ` | `template/cover-letter.typ`  | Lucide, beside column lines                                   |
| Julia    | `template/julia.typ`  | `template/julia-letter.typ`  | `template/icons/julia/`, beside entries, column lines and a legend |

## What a template reads

The server writes two files beside the template in a temp folder before running `typst compile`:

| File          | Contents                                                                                     |
| ------------- | -------------------------------------------------------------------------------------------- |
| `data.json`   | The full document: content and `style` (for a letter: `{basics, style, letter}`)             |
| `resume.json` | Content only (no ids, style, layout flags or blank lines), attached to the PDF with `pdf.attach` |

See [Content format](./content-format) for the schema.

## Registering a template

Templates are listed in `server/templates.go`:

```go
{ID: "julia", Name: "Julia",
  Features: []string{"entryIcons", "legend", "compactEntries", "entryBrackets"},
  file: "julia.typ", letter: "julia-letter.typ", iconSet: "julia"},
```

`Features` tell the editor which extra controls to show:

| Feature           | Shows                                                          |
| ----------------- | -------------------------------------------------------------- |
| `markers`         | Leader, bullet and paragraph-marker styles                     |
| `uppercaseTitles` | ALL-CAPS titles per entries section                            |
| `entryIcons`      | Icons on each entry                                            |
| `legend`          | A legend on entries sections                                   |
| `compactEntries`  | Compact entries sections                                       |
| `entryBrackets`   | A bracket beside an entries section                            |

A resume picks its template by `style.template` (the id; empty for Classic).

## Click-to-edit markers

For a value to be clickable on the page, the template wraps it in a marker:

```typ
// Inline value: invisible marks at the start and end of the text.
#field("basics.name", name)

// Block value (bullet, paragraph): marks at the top and bottom of the block.
#block-field("sections.2.items.0.bullets.3", bullet)
```

Paths index the arrays in `data.json`. The server reads the markers with `typst eval 'query(<field>)'` and sends their
positions with each preview. Markers change neither the layout nor the text layer. A template without them still
renders; it just isn't clickable. Side-by-side column cells add a `"cell"` mark with their size, so the editor can draw
the draggable dividers.

## Keep it readable

The checks hold templates to a few rules. Keep them and every resume passes:

- **Draw decorations, don't type them.** Leaders, bullets, dots between contacts and icons are shapes (`line`,
  `polygon`, `circle`, images), never characters like `·` or `❖`.
- **Keep an entry's title and date in one text run** so parsers read them as one line.
- **Tag headings** (`heading`) and lists, and keep PDF/UA-1 passing. A template that breaks it still renders, without
  the standard, and the Checks tab says why.

## Fonts

Every template finds the fonts in `template/fonts/` as if installed (Typst's `--font-path`), and they're offered on the
Style tab. Drop another `.ttf`/`.otf` there to use it, but only fonts whose license allows redistribution, with their
license file, as the bundled SIL Open Font License ones do.

## Icon sets

An icon set is a folder `template/icons/<set>/` of SVGs plus `icons.json`, listing them in picker order:

```json
[
  { "file": "research.svg", "name": "Research", "color": "#7fb7e6" },
  { "file": "agriculture.svg", "name": "Agriculture", "color": "#a9b854" }
]
```

`color` is the legend label color. Add an SVG and a line to `icons.json` and it's offered on the next picker load, no
restart needed.

## Using your own copy

Release builds unpack the bundled templates on every start into `<user cache dir>/resume-builder/template/`, undoing
edits there. Copy that folder somewhere else and run `resume-builder -template <copy>/resume.typ`; the other templates
are found beside it.

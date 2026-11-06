# Editing with Typst directly

`template/resume.typ` can be edited and compiled outside the resume editor, as long as two files sit next to it.

## What the template needs

The template contains no resume content. It reads and attaches two files from its own folder:

| File | Used by | Contents |
| --- | --- | --- |
| `data.json` | `#let data = json("data.json")` | Full document: content + `style` |
| `resume.json` | `#pdf.attach("resume.json", …)` | Content only (the server strips ids, `style`, layout flags and blank lines) |

Normally `render()` in `server/render.go` writes the template, `data.json` and `resume.json` into a temp folder
and runs `typst compile` there. Neither file exists in `template/`, so compiling the template on its own fails until
you provide them.

## Local (Tinymist in VS Code, or `typst watch`)

Symlink both files to the real data:

```sh
cd template
ln -s ../data/resume.json data.json
ln -s ../data/resume.json resume.json
typst watch resume.typ
```

- Symlinks stay live: autosaves from the browser editor show up in the Typst preview.
- The server re-reads the template on every render, so template edits also show up live in the browser editor.
- Difference from the server's build: the PDF's attached `resume.json` is the raw file, with ids and the `style` block.
- Add `template/data.json` and `template/resume.json` to `.gitignore` if you keep the symlinks.

Tested with typst 0.15.1: the template compiles to a tagged 2-page PDF.

## typst.app (web)

Upload `resume.typ`, then upload `data/resume.json` twice: once as `data.json` and once as `resume.json`.

- **Font:** `style.font` defaults to `"Helvetica"`, a macOS system font that typst.app doesn't have. Typst
  substitutes another font and shows a warning. Upload a font file or change `style.font`.
- **No sync:** changes made in typst.app never reach the repo. Copy template edits back by hand. Content changes
  have to be made in the JSON, and the uploaded JSON goes stale when the browser editor saves.

## Cover letter template

`template/cover-letter.typ` reads only `data.json`, shaped `{basics, style, letter}`: the resume's `basics` and
`style` plus one letter from the resume's `cover-letters/` folder. No file on disk has that shape, and the name clashes
with the resume template's `data.json`, so build it in a scratch folder:

```sh
mkdir -p /tmp/letter && cp template/cover-letter.typ /tmp/letter/
jq -n --slurpfile r ~/resume-builder/Resume/resume.json --slurpfile l "$HOME/resume-builder/Resume/cover-letters/MPO Data Analyst.json" \
  '{basics: $r[0].basics, style: $r[0].style, letter: $l[0]}' > /tmp/letter/data.json
typst watch /tmp/letter/cover-letter.typ
```

The JSON is a snapshot: rerun `jq` to pick up edits from the browser editor.

## Which tool for what

- **Template** (layout, spacing, decorations): Typst. You get compiler diagnostics and a fast preview.
- **Content**: the browser editor. Content is JSON in Typst, and the browser editor is the only place the
  readability checks and the "What machines read" tab run.

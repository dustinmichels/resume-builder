# Resume editor

A local resume editor: edit content in the browser, see the exact PDF pages as you type, and check what
machines (applicant tracking systems, parsers) will read from the PDF.

- **Frontend** (`web/`): Vue 3 + Vite, run with Bun. Forms, drag-and-drop reordering, click-to-edit on the live
  preview, rich-text formatting.
- **Backend** (`server/`): Go, standard library only. Stores resumes and their versions, renders them with Typst, extracts the
  PDF text layer with `pdftotext`, runs readability checks, and writes the Word and Markdown exports.
- **Template** (`template/resume.typ`): Typst. Re-read on every render, so template edits show up live.
- **Data** (your data folder, `~/resume-builder/` by default): one folder per resume, autosaved from the editor.
  See [Your files](#your-files).

## Run it

Prerequisites: [mise](https://mise.jdx.dev) installs `go`, `bun`, `typst` and `air` (pinned in `mise.toml`; run
`mise install`). `pdftotext` comes from Poppler: `brew install poppler`.

```sh
mise run dev     # API on 127.0.0.1:8787 (rebuilt by air on server/*.go changes) + editor on http://localhost:5180
mise run serve   # build the editor, then serve everything from http://127.0.0.1:8787
mise run test    # server tests
```

Without mise: install `go`, `bun` and `typst` (`brew install typst`) yourself, then run `go run ./server` in one
terminal and `cd web && bun install && bun run dev` in another.

The data folder you pick on first run is remembered in `<user config dir>/resume-builder/config.json` (on macOS
`~/Library/Application Support/resume-builder/config.json`); `go run ./server -config <file>` uses another settings
file. Picking a folder with no resumes in it imports `data/resume.json` from an older checkout, or starts from a
sample resume.

## Your files

Everything lives in plain files in the data folder, so it can be backed up, synced or versioned like any other folder:

```
~/resume-builder/
  Resume/
    resume.json                               the resume, autosaved as you edit
    versions/
      20261106-153000 Sent to Acme.json       a saved version: timestamp, then its label
  Designer CV/
    resume.json
  .trash/
    Old draft 20261101-090000/                deleted resumes, moved here rather than erased
```

Each folder holding a `resume.json` is a resume; its folder name is the name shown in the editor, so renaming one
renames the folder. Saving a version snapshots the resume as last saved; restoring a version first saves the current
resume as a version labelled `Before restoring …`, so a restore can always be undone. Deleting a resume moves its
folder, versions included, into `.trash`; empty that folder by hand when you no longer need them.

## Readable by people and machines

Every decoration is drawn by the template as a shape, never typed as a character:

| On the page | In the PDF text layer |
| --- | --- |
| `TUFTS UNIVERSITY ·········· Exp. May 2027` | `TUFTS UNIVERSITY Exp. May 2027` |
| `❖ Emphasis on walkable…` | `Emphasis on walkable…` |
| `+1 720… ● dustin@… ● dustinmichels.com` | `+1 720… dustin@… dustinmichels.com` |

The PDF is also tagged (headings, lists) and carries the content as an attached `resume.json`, without editor-only
fields (ids, style, layout flags) or blank lines. Its document properties, shown by viewers, file browsers and
applicant-tracking systems, come from the resume:

| Property | Source |
| --- | --- |
| Title | `<name> – Resume` |
| Author | name |
| Subject | headline lines, joined with ` · ` |
| Keywords | lines of table-style columns sections (the skills grid), split at `,` `;` `(` `)`, markup removed |

The **What machines read** tab shows the extracted text (reading order and layout). The **Checks** tab flags
typed dot leaders, symbol characters (❖ ● emoji), entry titles split from their dates, dates without a year,
and any content missing from the text layer.

## Downloads

- **PDF**: the Typst render you see in the preview.
- **Word** (`.docx`): the same content and style, built from native Word features so it stays editable and
  parser-friendly: bullets are list formatting, dot leaders are tab leaders, capitals are the caps effect.
  Column sections stack vertically instead of using tables.
- **Markdown** (`.md`): inline `**bold**`/`*italic*`/links kept as written, everything else escaped. The
  preview's **Markdown** tab shows it rendered or as source, updating as you type.
- **Text** and **JSON**: the PDF text layer and the resume content.

## Editing

Click any text on the **Page** preview to edit that value in place: a small editor opens next to it and the page
re-renders as you type. In bullets, paragraphs, column lines and headline parts, Enter adds the next item and
Backspace in an empty one removes it; Tab and Shift+Tab move between values. Esc closes the editor.

The template tags every editable value with an invisible marker (`field`/`block-field` in `template/resume.typ`);
the server reads their positions with `typst eval 'query(<field>)'` and sends them with each preview. Markers don't
change the layout or the PDF text layer. A custom template without them still renders; it just isn't clickable.

## Content format

Select text and press ⌘B, ⌘I or ⌘K (link) to format it; Shift+Enter adds a line break. Formatting is stored as
`**bold**`, `*italic*`, `***bold italic***` and `[label](https://link)` in paragraphs, subtitles, bullets,
subheadings and column lines; other fields are plain text. Content is rendered as literal text, so characters like
`#`, `$` and `@` are safe. The schema lives in `web/src/types.ts`.

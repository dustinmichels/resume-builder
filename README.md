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
    resume.md, resume.pdf                     Markdown and PDF copies, written by Sync files
    additional/                               supporting details, never printed
      contact.json                            your contact details (Contact tab)
      supervisors.json                        a supervisor per job (Supervisors tab)
    versions/
      20261106-153000 Sent to Acme.json       a saved version: timestamp, then its label
      20261106-153000 Sent to Acme.md, .pdf   its Markdown and PDF copies, written by Sync files
  Designer CV/
    resume.json
  .trash/
    Old draft 20261101-090000/                deleted resumes, moved here rather than erased
```

Each folder holding a `resume.json` is a resume; its folder name is the name shown in the editor, so renaming one
renames the folder. Saving a version snapshots the resume as last saved; restoring a version first saves the current
resume as a version labelled `Before restoring …`, so a restore can always be undone. Deleting a resume moves its
folder, versions included, into `.trash`; empty that folder by hand when you no longer need them. The folder button in
the toolbar opens the current resume's folder in Finder (Explorer on Windows, the default file manager elsewhere).

**Sync files** (next to the save status in the toolbar) writes a Markdown and a PDF copy beside `resume.json` and each
version, named after the JSON file. It saves pending edits first and skips copies already up to date: a Markdown file
is rewritten when its content changes, a PDF when it is older than its JSON or the template. A file that fails to
render is reported and the rest still sync. Deleting a version deletes its copies. The button reads **Files synced**
until the resume next changes.

`additional/contact.json` and `additional/supervisors.json` hold details for job applications that don't belong on the page: your name,
email, phone and mailing address, and for each entry a supervisor's name, position, email, phone and whether they may be contacted. They are
written only once you fill something in, autosave with the resume, and are never rendered or exported. They move,
duplicate and go to `.trash` with their resume, but versions don't include them: restoring a version leaves them as
they are, and a copy of a version gets the current ones. Supervisors are matched to entries by the entry's `id`; each
also records the entry's title and organization (`job`), so one whose entry you delete is still listed, under
**No longer in the resume**, until you remove it.

## Readable by people and machines

Every decoration is drawn by the template as a shape, never typed as a character:

| On the page                                 | In the PDF text layer                |
| ------------------------------------------- | ------------------------------------ |
| `TUFTS UNIVERSITY ·········· Exp. May 2027` | `TUFTS UNIVERSITY Exp. May 2027`     |
| `❖ Emphasis on walkable…`                   | `Emphasis on walkable…`              |
| `+1 720… ● dustin@… ● dustinmichels.com`    | `+1 720… dustin@… dustinmichels.com` |

The PDF is also tagged (headings, lists), validated as PDF/UA-1 (the accessible-PDF standard) when the template
allows it, and carries the content as an attached `resume.json`, without editor-only fields (ids, style, layout
flags) or blank lines. Its document properties, shown by viewers, file browsers and applicant-tracking systems,
come from the resume:

| Property | Source                                                                                            |
| -------- | ------------------------------------------------------------------------------------------------- |
| Title    | `<name> – Resume`                                                                                 |
| Author   | name                                                                                              |
| Subject  | headline lines, joined with `·`                                                                   |
| Keywords | lines of table-style columns sections (the skills grid), split at `,` `;` `(` `)`, markup removed |

The **What machines read** tab shows the extracted text (reading order and layout). The **Checks** tab groups its
findings by reader, plus the dates:

- **Parsers** (the `pdftotext` text layer): typed dot leaders, symbol characters (❖ ● emoji), entry titles split
  from their dates, dates without a year, and any content missing from the text layer.
- **Screen readers** (the PDF's tags, via `pdfinfo -struct-text`): Typst's PDF/UA-1 violations (a template that
  breaks them still renders, without the standard), content hidden from or read out of order by screen readers,
  section titles not tagged as headings, and accent-color contrast below WCAG AA (4.5:1).
- **Links**: links without `https://` (or `mailto:`, `tel:`), link text naming a different address than the
  target, links not clickable in the PDF (markup in a field the template prints as plain text), and web links
  that don't open. The server requests each web link at most once every 10 minutes, separately from rendering;
  sites that refuse automated requests (LinkedIn's 999) show as unconfirmed, not broken.
- **Dates**: date ranges that end before they start, and entries not listed newest first (by end date, "Present"
  being this month, then start date). Dates are read as month names, 4-digit years and "Present"; a label item
  (like "Non-degree education:") starts a new list within its section.

## Downloads

**Download PDF** is its own toolbar button; the other formats are under the **Download** menu next to it.

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
Backspace in an empty one removes it; Tab and Shift+Tab move between values. Esc closes the editor. The sidebar
follows along: it switches to **Content**, expands the section and entry holding the value, scrolls to its field
and briefly highlights it.

The template tags every editable value with an invisible marker (`field`/`block-field` in `template/resume.typ`);
the server reads their positions with `typst eval 'query(<field>)'` and sends them with each preview. Markers don't
change the layout or the PDF text layer. A custom template without them still renders; it just isn't clickable.

⌘Z / Ctrl+Z (or the arrows next to the save status) undoes resume edits, including inside text fields; ⇧⌘Z /
Ctrl+Y redoes. Typing with pauses under a second counts as one step. History covers the resume (Content and Style
tabs, the preview editor) and resets when you open another resume or restore a version; the Contact, Supervisors and
Versions fields keep the browser's own per-field undo.

The eye button on a section, entry, subheading or column hides it from the page without deleting it: it stays in
`resume.json` (as `"hidden": true`) and in the editor, dimmed, but is left out of the PDF, the Word, Markdown and text
exports, the attached JSON, the PDF keywords and the checks. Click the eye again to bring it back.

## Content format

Select text and press ⌘B, ⌘I or ⌘K (link) to format it; Shift+Enter adds a line break. Formatting is stored as
`**bold**`, `*italic*`, `***bold italic***` and `[label](https://link)` in paragraphs, subtitles, bullets,
subheadings and column lines; other fields are plain text. Content is rendered as literal text, so characters like
`#`, `$` and `@` are safe. The schema lives in `web/src/types.ts`.

Column lines can carry an icon, hung in the margin to the left of the line. Click the image button beside a line in
the sidebar, an icon on the page, or the image button on a line's page editor, then search Lucide's icon set (Enter
picks the first match). The icon's SVG is stored with the column (`icons`, parallel to `lines`), so the template
needs no icon set. Icons are drawn as decorations: left out of the text layer, the screen-reader tags, the Word and
Markdown exports and the attached JSON.

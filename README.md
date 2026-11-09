# Resume editor

A local resume editor: edit content in the browser or the Mac app, see the exact PDF pages as you type, and check what
machines (applicant tracking systems, parsers) will read from the PDF.

- **Frontend** (`web/`): Vue 3 + Vite, run with Bun. Forms, drag-and-drop reordering, click-to-edit on the live
  preview, rich-text formatting.
- **Backend** (`server/`): Go, standard library only (the Mac app adds [Wails](https://v3.wails.io)). Stores resumes,
  their versions and cover letters, renders them with Typst, extracts the PDF text layer with `pdftotext`, runs
  readability checks, and writes the Word and Markdown exports.
- **Templates** (`template/resume.typ`, `template/julia.typ`, and their cover letter templates
  `template/cover-letter.typ`, `template/julia-letter.typ`): Typst. Re-read on every render, so template edits show up
  live. See [Templates](#templates).
- **Data** (your data folder, `~/resume-builder/` by default): one folder per resume, autosaved from the editor.
  See [Your files](#your-files).

## Install

### Mac

Download [Resume-Builder.dmg](https://github.com/dustinmichels/resume-builder/releases/latest/download/Resume-Builder.dmg),
open it, and drag **Resume Builder** to **Applications**. Then open it from Applications or Launchpad like any other
app. It runs on macOS 13 (Ventura) or later, on Apple Silicon and Intel Macs.

On first run, choose where to keep your resumes. The machine view ("What machines read") and the screen-reader and
PDF-link checks need Poppler, which the app doesn't include yet; if you have Homebrew, `brew install poppler` turns
them on. The app logs to `~/Library/Logs/Resume Builder/` (Help → Show Log File).

### Command line

Every [release](https://github.com/dustinmichels/resume-builder/releases/latest) also has an archive per platform with
`resume-builder` to run from a terminal. The editor opens in your browser at http://127.0.0.1:8787; leave the terminal
open while you use it, and press Ctrl-C to stop it. Each archive carries the editor, the templates and Typst, so nothing
else is needed. Install Poppler for the machine view and the tag checks (`brew install poppler` on macOS,
`poppler-utils` on Linux).

The command-line binaries are unsigned. On macOS, download with `curl` rather than a browser, which marks the file as
quarantined and Gatekeeper then refuses to run it:

```sh
curl -fsSL https://github.com/dustinmichels/resume-builder/releases/latest/download/resume-builder-darwin-arm64.tar.gz | tar -xz
./resume-builder-darwin-arm64/resume-builder
```

Use `darwin-amd64` on an Intel Mac. The same page has `linux-amd64`, `linux-arm64` and, as `.zip`, `windows-amd64` and
`windows-arm64`. `-addr` changes the address, `-open=false` skips opening the browser, and `-config` picks another
settings file. The templates are unpacked on every start (by the app too) into `<user cache dir>/resume-builder/template/`
(on macOS `~/Library/Caches/resume-builder/template/`), undoing edits there; to change one, copy that folder and run
with `-template <copy>/resume.typ`.

## Run from source

Prerequisites: [mise](https://mise.jdx.dev) installs `go`, `bun`, `typst` and `air` (pinned in `mise.toml`; run
`mise install`). `pdftotext` comes from Poppler: `brew install poppler`.

```sh
mise run dev     # API on 127.0.0.1:8787 (rebuilt by air on server/*.go changes) + editor on http://localhost:5180
mise run serve   # build the editor, then serve everything from http://127.0.0.1:8787
mise run app     # build the editor, then run the desktop app (a native window instead of the browser)
mise run test    # server tests
```

The desktop app is the same server built with `-tags desktop` (`server/desktop.go`): a [Wails v3](https://v3.wails.io)
window shows the editor, served straight from the server's handler with no network port, and native dialogs stand in
for what a web view can't do (downloads become a Save dialog, links open in the default browser, "Choose…" picks the
data folder). It needs CGO and Xcode's command line tools; the command-line builds don't, and don't include Wails.
The editor knows it's in the app from `desktop: true` on `GET /api/settings`.

Without mise: install `go`, `bun` and `typst` (`brew install typst`) yourself, then run `go run ./server` in one
terminal and `cd web && bun install && bun run dev` in another.

The data folder you pick on first run is remembered in `<user config dir>/resume-builder/config.json` (on macOS
`~/Library/Application Support/resume-builder/config.json`); `go run ./server -config <file>` uses another settings
file. Picking a folder with no resumes in it imports `data/resume.json` from an older checkout, or starts from a
sample resume.

## Releasing

`mise run release` cuts a release from a clean `main` that's level with `origin/main`, on a Mac: it checks signing and
notarization will work, runs the server tests, builds the archives and the Mac app, signs and notarizes the app, tags
`vX.Y.Z`, pushes the tag and publishes a GitHub release with `Resume-Builder.dmg`, the archives and `checksums.txt`.
Nothing is tagged until notarization has passed. Versions come from the git tags: a patch bump by default, or
`mise run release minor|major|X.Y.Z`; the first release is `v0.1.0`. It needs the
[GitHub CLI](https://cli.github.com) logged in (`gh auth login`). `mise run package [version]` builds everything into
`build/release/` without publishing; elsewhere than macOS it skips the DMG.

Each archive (`scripts/package.sh`) holds `resume-builder` built with `-tags release`, which embeds `template/` and
`web/dist/` (`bundle.go`), plus the Typst binary of the version pinned in `mise.toml`, downloaded from Typst's
releases, and Typst's license. A release build puts its own folder first on `PATH`, so that Typst is the one it runs.

The Mac app (`scripts/package-macos.sh`) is `Resume Builder.app` with universal (arm64 + x86_64) builds of
`resume-builder` (`-tags "release desktop production"`) and Typst in `Contents/MacOS/`, and the icon
(`macos/AppIcon.icns`, made from `macos/icon.svg` by `scripts/icon.sh`). Signing needs, once per Mac:

1. A **Developer ID Application** certificate in the Keychain (developer.apple.com → Certificates, or Xcode →
   Settings → Accounts → Manage Certificates); `security find-identity -v -p codesigning` lists it. Back up its private
   key as a `.p12`.
2. A notarytool profile named `resume-builder`, with an app-specific password for the Apple ID:
   `xcrun notarytool store-credentials resume-builder --apple-id <apple id> --team-id LLM5F36KYW`.

`CODESIGN_IDENTITY` and `NOTARY_PROFILE` override either. Without the certificate, `mise run package` signs the app ad
hoc and skips notarization: it runs on the Mac that built it, but Gatekeeper blocks it elsewhere.

## Your files

Everything lives in plain files in the data folder, so it can be backed up, synced or versioned like any other folder:

```
~/resume-builder/
  Resume/
    resume.json                               the resume, autosaved as you edit
    resume.md, .docx, .pdf                    Markdown, Word and PDF copies, written by Sync files
    additional/                               supporting details, never printed
      contact.json                            your contact details (Contact tab)
      supervisors.json                        a supervisor per job (Supervisors tab)
    cover-letters/
      MPO Data Analyst.json                   a cover letter (Cover letters tab)
      MPO Data Analyst.md, .docx, .pdf        its Markdown, Word and PDF copies, written by Sync files
    versions/
      20261106-153000 Sent to Acme/           a saved version: timestamp, then its label
        resume.json, additional/, cover-letters/   the resume, its details and letters as they were saved
        resume.md, .docx, .pdf                Markdown, Word and PDF copies of the resume and each letter,
                                              written when the version is saved
  Designer CV/
    resume.json
  .trash/
    Old draft 20261101-090000/                deleted resumes, moved here rather than erased
    Resume – Acme 20261102-100000.json        deleted cover letters, named after their resume
```

Each folder holding a `resume.json` is a resume; its folder name is the name shown in the editor, so renaming one
renames the folder. Saving a version snapshots the resume, its `additional/` details and its cover letters as last
saved, and writes Markdown, Word and PDF copies of the resume and each letter inside it (a copy that fails to render
is reported, the snapshot is kept, and Sync files retries it). Restoring a version replaces only `resume.json`; it first
saves the current state as a version labelled `Before restoring …`, so a restore can always be undone. Deleting a resume moves its
folder, versions included, into `.trash`; empty that folder by hand when you no longer need them. The folder button in
the toolbar opens the current resume's folder in Finder (Explorer on Windows, the default file manager elsewhere).

**Sync files** (next to the save status in the toolbar) writes a Markdown, a Word and a PDF copy beside `resume.json`
and each cover letter, and the same inside each version, named after the JSON file. It saves pending edits first and skips copies already up
to date: a Markdown or Word file is rewritten when its content changes, a PDF when it is older than its JSON or the
template (for a letter, also the `resume.json` beside its folder, whose name and style it shows). A file that fails to render is reported
and the rest still sync. Deleting a version or letter deletes its copies. The button reads **Files synced** until the
resume next changes.

`additional/contact.json` and `additional/supervisors.json` hold details for job applications that don't belong on the page: your name,
email, phone and mailing address, and for each entry a supervisor's name, position, email, phone, mailing address and whether they may be contacted. They are
written only once you fill something in, autosave with the resume, and are never rendered or exported. They move,
duplicate, go to `.trash` and are saved in versions with their resume. Restoring a version leaves them as they are; a
copy of a version gets the version's (or, for a version with neither these nor letters, the current ones). Supervisors are matched to entries by the entry's `id`; each
also records the entry's title and organization (`job`), so one whose entry you delete is still listed, under
**No longer in the resume**, until you remove it.

## Cover letters

The **Cover letters** tab holds any number of letters for the open resume, each saved as
`cover-letters/<name>.json` in its folder; name them after the job. While that tab shows a letter, the preview,
checks, downloads and undo follow the letter; every other tab shows the resume.

A letter renders with its resume's template's letter template, so the two always match. The name, accent color, heading
font and margin come from the resume, so changing them there restyles its letters. The body font and size are the
letter's own.

`template/cover-letter.typ` (Classic) follows `samples/cover-letter`: the resume's name top right in its accent color,
lines of your choice beneath it (a new letter starts with the resume's last headline part, often the location), a
heading with the resume's section square hung in the margin, then the greeting, paragraphs, subheadings and bullet
lists, and an optional closing above your name. A paragraph right after another runs on with an indented first line;
one after a heading or list starts flush. The bottom margin is half the resume's, as in the sample. The body font
defaults to Libertinus Serif, which Typst bundles.

`template/julia-letter.typ` (Julia) repeats the Julia resume's header at the resume's size: the name centered over a
gray line of the letter's header lines and the resume's contacts. The letter's title is a section title, and the
body is indented beneath it as a section's entries are: paragraphs separated by a gap, subheadings bold like entry
titles, bullets as indented paragraphs without markers. The margin is the resume's all round. A new letter starts in
the resume's font at 10.5 pt.

Both templates read `data.json` as `{basics, style, letter}`. The Word export of a letter follows its template too
(see [Downloads](#downloads)).

Letters autosave with the resume and are clickable on the page (Enter in a paragraph starts the next one). Their
checks are the resume's Parsers, Screen readers and Links checks, without the resume-only ones (dot leaders, entry
dates, email); a letter longer than one page gets a note. They move, duplicate, go to `.trash` and are saved in
versions with their resume, like the details above. Deleting a letter moves it alone to `.trash`.

## Templates

Each resume picks its template on the **Style** tab (saved as `style.template`; missing means Classic). Every render of
that resume uses it: the preview, the PDF download, Sync files and saved versions. Its cover letters use its letter
template (see [Cover letters](#cover-letters)), styled by the resume.

| Template | File                  | Letter                       | Icons                                    |
| -------- | --------------------- | ---------------------------- | ---------------------------------------- |
| Classic  | `template/resume.typ` | `template/cover-letter.typ`  | Lucide, beside column lines              |
| Julia    | `template/julia.typ`  | `template/julia-letter.typ`  | `template/icons/julia/`, beside entries, column lines and a section legend |

**Julia** follows `samples/julia/Res.6.3.26-agfocus-icon2.pdf`: the name centered over a gray line of headline parts
and contacts, section titles in the accent color, and entries as an organization line (the entry's subtitle in small
capitals, then location and date), a bold title followed by the entry's icons, then each bullet as an indented
paragraph. The small capitals are drawn from capitals, so every font has them, and parsers read them as capitals. Text
after ` - ` in a title is set light (a lab or department). Its extra controls show only while it is selected:

- **Icons** on each entry (sidebar entry card, or the image button on a title's page editor), drawn after the title,
  or after the organization line when the entry has no title.
- **Legend** on an entries section: lines with icons, printed after the section title, each line in its icon's color.
- **Compact** on an entries section: each entry runs on one line (organization, location, title, date, then the
  bullets in smaller type), as in the sample's education.
- Columns sections run each column's lines together, each line's icon after it (the sample's interests).

The leader, bullet and paragraph-marker styles and ALL-CAPS titles are Classic's and are hidden for Julia.

**Fonts.** The sample's fonts are commercial (Houschka Pro, Adrianna Demibold, Ainslie Sans), so Julia uses free
stand-ins bundled in `template/fonts/`: the style's font for the body (Quicksand, after Houschka Pro, for Julia's
resume, at 8.8 pt to keep one page), and Source Sans 3 (the sample's Source Sans Pro, standing in for Adrianna too) for
the name, section titles and bold text. Quicksand has no italics, so the italic dates and places print upright. Every
template finds the fonts in `template/fonts/` as if installed (Typst's `--font-path`), and both are offered on the
Style tab; drop another `.ttf`/`.otf` there to use it (only fonts whose license allows copying them, as these SIL Open
Font License ones do, with their `OFL.txt`). The Word export names the font but doesn't embed it, so Word on a computer
without it substitutes another, and draws Julia's light and medium weights in the font's regular weight.

A template's icon set is a folder `template/icons/<set>/` of SVGs plus `icons.json`, listing them in picker order:
`{"file", "name", "color"}`, where `color` is the legend label color. Add an SVG and a line to `icons.json` and it is
offered on the next picker load, no restart needed. Julia's set holds the icons from
`samples/julia/iconsfor resume.ai`, cropped one per file (the design/branding triangle is redrawn from the resume PDF;
the last artboard's color studies are left out). Templates are registered in `server/templates.go`.

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
  breaks them still renders, without the standard), content hidden from or read out of order by screen readers (an
  entry's title, date, subtitle and location may read in any order among themselves, since templates arrange them
  differently), section titles not tagged as headings, and accent-color contrast below WCAG AA (4.5:1).
- **Links**: links without `https://` (or `mailto:`, `tel:`), link text naming a different address than the
  target, links not clickable in the PDF (markup in a field the template prints as plain text), and web links
  that don't open. The server requests each web link at most once every 10 minutes, separately from rendering;
  sites that refuse automated requests (LinkedIn's 999) show as unconfirmed, not broken.
- **Dates**: date ranges that end before they start, and entries not listed newest first (by end date, "Present"
  being this month, then start date). Dates are read as month names, 4-digit years and "Present"; a label item
  (like "Non-degree education:") starts a new list within its section.

## Downloads

**Download PDF** is its own toolbar button; the other formats are under the **Download** menu next to it. On the
**Cover letters** tab they export the letter, named `<name> - Cover Letter - <letter>`. In the Mac app each one asks
where to save the file.

- **PDF**: the Typst render you see in the preview.
- **Word** (`.docx`): the same content and style in the resume's template layout (its letters' in their template's),
  built from native Word features so it stays editable and parser-friendly: bullets are list formatting, dot leaders
  are tab leaders, capitals are the caps effect, Julia's small capitals Word's small caps. Column sections stack
  vertically instead of using tables. Julia's icons are pictures marked decorative, so screen readers skip them: SVG,
  with a PNG copy (drawn by Typst at export) for programs that can't show SVG, such as Google Docs. A legend shares
  its section title's line through a style separator, so the heading holds only the title. Julia's line spacing is
  exact, matching the PDF's, so a substituted font changes only how lines wrap, not their height.
- **Markdown** (`.md`): inline `**bold**`/`*italic*`/links kept as written, everything else escaped. The
  preview's **Markdown** tab shows it rendered or as source, updating as you type.
- **Text** and **JSON**: the PDF text layer and the content.

## Editing

Click any text on the **Page** preview to edit that value in place: a small editor opens next to it and the page
re-renders as you type. In bullets, paragraphs, column lines and headline parts, Enter adds the next item and
Backspace in an empty one removes it; Tab and Shift+Tab move between values. Esc closes the editor. The sidebar
follows along: it switches to **Content** (or **Cover letters**, for a letter), expands the section and entry holding
the value, collapses every other section and entry, scrolls to its field and briefly highlights it.

The template tags every editable value with an invisible marker (`field`/`block-field` in each `template/*.typ`);
the server reads their positions with `typst eval 'query(<field>)'` and sends them with each preview. Markers don't
change the layout or the PDF text layer. A custom template without them still renders; it just isn't clickable.

In a side-by-side columns section, the dashed line between two columns on the **Page** preview moves where the next
column starts: drag it, or focus it and use the arrow keys (Shift for bigger steps). The left column's share of the
line width is saved as its `width` (and the right one's, unless it's the last column, so its right edge stays put);
double-click the line to clear them, so the columns fit their content again. The template marks each of these cells
(a `"cell"` marker with its size) and the preview sends their positions as `columns`.

⌘Z / Ctrl+Z (or the arrows next to the save status) undoes resume edits, including inside text fields; ⇧⌘Z /
Ctrl+Y redoes. Typing with pauses under a second counts as one step. History covers the resume (Content and Style
tabs, the preview editor) and resets when you open another resume or restore a version; on the **Cover letters** tab
the same keys and arrows act on the open letter, which has its own history. The Contact, Supervisors and Versions
fields keep the browser's own per-field undo.

The eye button on a section, entry, subheading or column hides it from the page without deleting it: it stays in
`resume.json` (as `"hidden": true`) and in the editor, dimmed, but is left out of the PDF, the Word, Markdown and text
exports, the attached JSON, the PDF keywords and the checks. Click the eye again to bring it back.

## Content format

Select text and press ⌘B, ⌘I or ⌘K (link) to format it; Shift+Enter adds a line break. Formatting is stored as
`**bold**`, `*italic*`, `***bold italic***` and `[label](https://link)` in paragraphs, subtitles, bullets,
subheadings and column lines, and in a letter's paragraphs, bullets and lines under the name; other fields are plain
text. Content is rendered as literal text, so characters like `#`, `$` and `@` are safe. The schema lives in
`web/src/types.ts`.

Column lines can carry an icon, hung in the margin to the left of the line. Click the image button beside a line in
the sidebar, an icon on the page, or the image button on a line's page editor, then search Lucide's icon set (Enter
picks the first match); a template with its own icon set lists those first. The icon's SVG is stored with the column
(`icons`, parallel to `lines`), so the template needs no icon set. Icons are drawn as decorations: left out of the
text layer, the screen-reader tags, the Markdown export and the attached JSON, and marked decorative in the Word
export. Entry icons and legends (Julia) are stored the same way, as `icons` on the entry and `legend: {lines, icons}`
on the section.

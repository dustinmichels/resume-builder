# Resume editor

A local resume editor: edit content in the browser, see the exact PDF pages as you type, and check what
machines (applicant tracking systems, parsers) will read from the PDF.

- **Frontend** (`web/`): Vue 3 + Vite, run with Bun. Forms, drag-and-drop reordering, live preview.
- **Backend** (`server/`): Go, standard library only. Saves the resume, renders it with Typst, extracts the
  PDF text layer with `pdftotext`, and runs readability checks.
- **Template** (`template/resume.typ`): Typst. Re-read on every render, so template edits show up live.
- **Data** (`data/resume.json`): the resume itself. Autosaved from the editor.

## Run it

Prerequisites: `bun`, `go`, `typst` (`brew install typst`), `pdftotext` (`brew install poppler`).

```sh
mise run dev     # API on 127.0.0.1:8787 + editor on http://localhost:5180
mise run serve   # build the editor, then serve everything from http://127.0.0.1:8787
mise run test    # server tests
```

Without mise: `go run ./server` in one terminal and `cd web && bun install && bun run dev` in another.

## Readable by people and machines

Every decoration is drawn by the template as a shape, never typed as a character:

| On the page | In the PDF text layer |
| --- | --- |
| `TUFTS UNIVERSITY ·········· Exp. May 2027` | `TUFTS UNIVERSITY Exp. May 2027` |
| `❖ Emphasis on walkable…` | `Emphasis on walkable…` |
| `+1 720… ● dustin@… ● dustinmichels.com` | `+1 720… dustin@… dustinmichels.com` |

The PDF is also tagged (headings, lists) and carries the content as an attached `resume.json`.

The **What machines read** tab shows the extracted text (reading order and layout). The **Checks** tab flags
typed dot leaders, symbol characters (❖ ● emoji), entry titles split from their dates, dates without a year,
and any content missing from the text layer.

## Content format

Text fields accept `**bold**`, `*italic*` and `[label](https://link)`. Content is rendered as literal text,
so characters like `#`, `$` and `@` are safe. The schema lives in `web/src/types.ts`.

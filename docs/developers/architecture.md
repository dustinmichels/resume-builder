# Architecture

| Part          | Where       | Built with                                                                                   |
| ------------- | ----------- | -------------------------------------------------------------------------------------------- |
| **Editor**    | `web/`      | Vue 3 + Vite, run with Bun. Forms, drag-and-drop, click-to-edit on the preview, rich text     |
| **Server**    | `server/`   | Go, standard library only (the Mac app adds [Wails](https://v3.wails.io))                    |
| **Templates** | `template/` | [Typst](https://typst.app). Re-read on every render                                           |
| **Data**      | your folder | One folder per resume, plain JSON ([Your files](/guide/files))                               |

The server stores resumes, versions and cover letters, renders them with Typst, extracts the PDF text layer with
`pdftotext`, reads the PDF's tags with `pdfinfo -struct-text`, runs the readability checks, and writes the Word and
Markdown exports.

## A render

Every edit in the editor posts the document to `POST /api/preview` (or `/api/letter/preview`) a moment after you stop
typing:

1. `server/render.go` writes the template, `data.json` (the full document) and `resume.json` (content only, for the
   PDF attachment) into a temp folder and runs `typst compile`.
2. `typst eval 'query(<field>)'` reads the positions of the invisible field markers, so the editor knows which box on
   the page is which value.
3. `pdftotext` extracts the text layer, in reading order and layout.
4. `server/checks.go` and `server/a11y.go` compare the text layer and the tags with the document.
5. The response carries the page images, the text, the checks, the Markdown export and the field boxes.

Link checks (`POST /api/links`) run separately, so a slow website never holds up a render.

## Server files

| File                          | Does                                                                         |
| ----------------------------- | ---------------------------------------------------------------------------- |
| `main.go`                     | Flags, routes, preview and PDF handlers                                      |
| `store.go`                    | Data folder, resumes, versions, side files, trash                            |
| `letters.go`                  | Cover letters                                                                |
| `render.go`                   | Running Typst                                                                |
| `fields.go`                   | Field markers → click-to-edit boxes                                           |
| `checks.go`, `a11y.go`, `dates.go`, `links.go` | The readability checks                                      |
| `docx.go`, `docx_julia.go`, `letter_export.go`  | Word export                                                |
| `markdown.go`, `rich.go`      | Markdown export, inline formatting                                           |
| `sync.go`                     | Sync files                                                                   |
| `templates.go`                | Template registry and icon sets                                              |
| `desktop.go`, `desktop_update.go`, `shell.go` | Mac app window, updater, native dialogs                       |
| `bundle.go`, `bundle_release.go` (and `/bundle.go`) | Templates and editor: from the repo, or embedded in release builds |
| `release.go`                  | Looks for a newer GitHub release                                             |

## API

| Route                                                        | For                                     |
| ------------------------------------------------------------ | --------------------------------------- |
| `GET`/`PUT /api/settings`                                    | Data folder                             |
| `GET`/`POST /api/resumes`, `GET`/`PUT`/`PATCH`/`DELETE /api/resumes/{id}` | Resumes                    |
| `/api/resumes/{id}/versions[/{vid}[/restore]]`               | Versions                                |
| `/api/resumes/{id}/letters[/{lid}]`                          | Cover letters                           |
| `GET`/`PUT /api/resumes/{id}/contact`, `/supervisors`         | Contact and supervisors side files      |
| `POST /api/resumes/{id}/sync`, `/reveal`                     | Sync files, open folder                 |
| `GET /api/templates`                                         | Templates and their icon sets           |
| `POST /api/preview`, `/pdf`, `/docx`, `/md`, `/links`        | Resume render and exports               |
| `POST /api/letter/preview`, `/pdf`, `/docx`, `/md`, `/links` | Letter render and exports               |
| `POST /api/open`, `/choose-folder`, `/save`                  | Desktop app only: native shell          |

## Build tags

| Tag          | Adds                                                                          |
| ------------ | ----------------------------------------------------------------------------- |
| `release`    | Embeds `template/` and `web/dist/` (`bundle.go`), puts its own folder first on `PATH` for the bundled Typst |
| `desktop`    | The Wails window and native dialogs (needs CGO)                               |
| `production` | Wails production mode                                                         |

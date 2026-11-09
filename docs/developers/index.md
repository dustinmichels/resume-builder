# Run from source

Resume Builder is a Go server, a Vue editor and a set of Typst templates. Everything runs locally.

## Prerequisites

[mise](https://mise.jdx.dev) installs the pinned `go`, `bun`, `typst` and `air` from `mise.toml`:

```sh
git clone https://github.com/dustinmichels/resume-builder.git
cd resume-builder
mise install
brew install poppler   # pdftotext and pdfinfo, for the machine view and checks
```

## Tasks

```sh
mise run dev     # API on 127.0.0.1:8787 (rebuilt by air on server/*.go changes) + editor on http://localhost:5180
mise run serve   # build the editor, then serve everything from http://127.0.0.1:8787
mise run app     # build the editor, then run the desktop app (a native window instead of the browser)
mise run test    # server tests
mise run fmt     # gofmt + oxfmt
mise run docs    # this documentation site, on http://localhost:5173/resume-builder/
mise run cleanup # delete generated files: build/, tmp/, web/dist/, the docs build
```

`mise run dev` is the everyday loop: Go changes rebuild the API, Vue changes hot-reload, and template changes show up on
the next render because the server re-reads templates every time.

### Without mise

Install `go`, `bun` and `typst` (`brew install typst`) yourself, then:

```sh
go run ./server                          # terminal 1
cd web && bun install && bun run dev     # terminal 2
```

## Settings and data

The data folder you pick on first run is remembered in `<user config dir>/resume-builder/config.json` (on macOS
`~/Library/Application Support/resume-builder/config.json`). Point a dev server somewhere else with
`go run ./server -config <file>`, where the file is `{"dataDir": "/path/to/folder"}`.

Picking a folder with no resumes in it imports `data/resume.json` from an older checkout if there is one, or starts from
`server/starter.json`.

## The desktop app

The desktop app is the same server built with `-tags desktop` (`server/desktop.go`). A [Wails v3](https://v3.wails.io)
window shows the editor, served straight from the server's handler with no network port. Native dialogs stand in for what
a web view can't do: downloads become a Save dialog, links open in the default browser, and **Choose…** picks the data
folder. The editor knows it's in the app from `desktop: true` on `GET /api/settings`.

It needs CGO and Xcode's command line tools. The command-line builds don't, and use only Wails' pure-Go updater package
to look for new releases. Release builds (`buildVersion` a `vX.Y.Z` tag) look for updates; development builds never do.

## This site

The docs are a [VitePress](https://vitepress.dev) site in `docs/`, styled with the app's colors and
[Lucide](https://lucide.dev) icons:

```sh
mise run docs         # dev server with hot reload
mise run docs-build   # static build into docs/.vitepress/dist
```

`.github/workflows/docs.yml` builds and deploys it to GitHub Pages on every push to `main` that touches `docs/`.

Screenshots in `docs/public/screenshots/` are taken from the editor at 1440×900, 2× scale, with a fictional resume.
Don't use real personal details: the site is public.

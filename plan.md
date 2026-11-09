# Plan: a Mac app non-technical users can install

Goal: someone who has never opened Terminal downloads `Resume-Builder.dmg` from the latest GitHub release, drags the
app to Applications, double-clicks it, and gets a window with a Dock icon. No Gatekeeper warnings, no `brew`, no
Terminal. macOS first; Windows and Linux keep today's archives until a later phase.

Approach: a Wails v3 shell (stable since v3.0.0, 2026-10-26) around the existing Go server, packaged as a universal
`.app` with Typst (and later Poppler) inside, signed with a Developer ID certificate, notarized, and shipped in a DMG by
`mise run release`.

## Where things stand

- `mise run release` publishes six CLI archives (`scripts/package.sh`): `resume-builder` built with `-tags release`
  (templates and editor embedded via `bundle.go` / `server/bundle_release.go`), plus `typst`. The user runs it from
  Terminal, it opens the browser at `127.0.0.1:8787`, and Ctrl-C stops it.
- Unsigned: browser downloads are quarantined and Gatekeeper refuses them (README tells Mac users to use `curl`).
- Poppler (`pdftotext`, `pdfinfo`) isn't bundled; without it the "What machines read" tab and the parser,
  screen-reader and PDF-link checks fail with "install poppler".
- No Developer ID Application certificate in the Keychain. Cloud signing doesn't apply (not an Xcode archive).

## Decisions to make first

| Decision               | Recommendation                                                                           |
| ---------------------- | ---------------------------------------------------------------------------------------- |
| App name and bundle ID | "Resume Builder", `com.dustinmichels.resumebuilder`                                      |
| Architectures          | One universal DMG (arm64 + x86_64 via `lipo`): users don't know their chip               |
| Minimum macOS          | 13 (Ventura); check what Wails v3 and the Typst release require and take the higher      |
| Poppler                | Ship v1 without it (UI explains the missing tab and checks), bundle it in phase 7        |
| Project license        | Pick one before the public DMG (none today); required anyway if Poppler (GPL) is bundled |
| CLI archives           | Keep all six; they cost nothing and serve technical users                                |

## Phase 1: Prerequisites (manual, once)

1. Create a **Developer ID Application** certificate (developer.apple.com → Certificates, or Xcode → Settings →
   Accounts → Manage Certificates) and confirm it shows in
   `security find-identity -v -p codesigning` as `Developer ID Application: … (LLM5F36KYW)`. Back up the private key
   (export a `.p12`); losing it means a new certificate.
2. Create an app-specific password for the Apple ID and store a notary profile:
   `xcrun notarytool store-credentials resume-builder --apple-id <apple id> --team-id LLM5F36KYW`.
3. App icon: a 1024×1024 PNG master, turned into `AppIcon.icns` with `iconutil`. Reuse `web/public/favicon.svg` as the
   starting artwork if it holds up at Dock size.

## Phase 2: Wails shell around the existing server

Keep the server a plain HTTP handler; Wails only hosts it. No Wails bindings and no `@wailsio/runtime` in the editor,
so `mise run dev` and the CLI build don't change.

1. Split serving out of `main()` in `server/main.go`: everything up to `newMux(...)` stays shared; the final
   `net.Listen` / browser open / `http.Serve` becomes the default `serve(handler)`. Follow the `bundled` pattern: a
   package variable that a build-tagged file overrides.
2. Add `server/desktop.go` (`//go:build desktop`) whose `init` swaps in the Wails runner:
   - `application.New` with `Assets: application.AssetOptions{Handler: mux}` (the whole mux, `/api/*` included),
     `SingleInstance` keyed on the bundle ID (a second launch focuses the existing window), and
     `Mac.ApplicationShouldTerminateAfterLastWindowClosed`.
   - Menu: `AppMenu`, `EditMenu`, `WindowMenu`, `HelpMenu` roles. The Edit menu is what makes Cmd-C/V/X/A work in a
     macOS web view; check that Cmd-Z still reaches the editor's own undo (`useUndoHistory.ts`) and doesn't fight the
     web view's.
   - One window, sensible default and minimum size; title "Resume Builder".
   - First thing to prove: POST bodies, status codes and `Content-Disposition` headers from the mux reach the web view
     through Wails' asset server. If they don't, fall back to the current `127.0.0.1` listener (random port) and point
     the window at it.
3. Desktop builds need CGO (WebKit); CLI builds keep `CGO_ENABLED=0`, so the six-archive cross-build is untouched.
4. Add a `mise run app` task that builds with `-tags desktop` (non-release, reading `template/` and `web/dist/` from
   the repo) and runs it.
5. Errors that currently `log.Fatal` before serving (`checkTools` when `typst` is missing, a bad config) need a native
   error dialog in the desktop build: a log line in a terminal nobody sees is a silent crash. Write logs to
   `~/Library/Logs/Resume Builder/`.

## Phase 3: Make the editor work in a web view

Each of these is browser-only today. The editor learns it runs in the app from a `desktop: true` field on
`GET /api/settings` (no Wails runtime needed).

1. **Downloads.** `download()` in `web/src/api.ts` clicks a blob URL with `a.download`; WKWebView likely ignores it.
   Test first. If it fails: in the desktop build, the export endpoints (`/api/pdf`, `/api/docx`, `/api/md` and the
   letter ones) show a native Save dialog (`app.Dialog.SaveFile()`, default name from the existing `attachment`
   naming) and write the file themselves; the editor calls them with a `?save=1` flag and shows "Saved" instead of
   making a blob.
2. **External links.** Preview links (`PreviewPane.vue`, `target="_blank"`) and links in the Checks tab would open
   inside the app window, or nowhere. Add `POST /api/open` that opens an `https:`/`mailto:`/`tel:` URL with the
   existing `openCommand` (`server/reveal.go`); in the desktop build the editor routes external link clicks there.
3. **Data folder.** `SetupScreen.vue` asks for a typed path. Add a "Choose…" button, shown only in the app, backed by
   `POST /api/choose-folder` → `app.Dialog.OpenFile().CanChooseDirectories(true).CanChooseFiles(false)`. Keep the text
   field as the default value and fallback. Add the same button wherever the folder can be changed later.
4. **Reveal in Finder** (`/api/resumes/{id}/reveal`) already uses `open`; confirm it works unchanged.
5. Click through the whole editor in the web view (Safari's engine): rich-text editing (`RichText.vue` uses
   `document.execCommand`), menus, dialogs, the SVG preview, fonts on the Style tab. Fix CSS differences.

## Phase 4: Build the `.app`

New `scripts/package-macos.sh` (called from `scripts/package.sh` on macOS):

1. Build the desktop binary twice (`GOARCH=arm64` and `amd64`, `CGO_ENABLED=1`, `-tags "release desktop"`, same
   `-ldflags` version stamp) and `lipo -create` them. Do the same for the two Typst macOS binaries `package.sh`
   already downloads.
2. Lay out `build/release/Resume Builder.app`:
   - `Contents/MacOS/resume-builder` and `Contents/MacOS/typst`. `preferBundledTools` puts the executable's folder
     first on `PATH`, so Typst is found with no code change.
   - `Contents/Resources/AppIcon.icns`, Typst's `LICENSE`/`NOTICE`.
   - `Contents/Info.plist`: `CFBundleIdentifier`, `CFBundleName`, `CFBundleShortVersionString` and
     `CFBundleVersion` from the tag, `CFBundleIconFile`, `LSMinimumSystemVersion`, `NSHighResolutionCapable`.
3. Check `lipo -info` on both binaries reports `x86_64 arm64`.

## Phase 5: Sign, notarize, DMG

In `scripts/package-macos.sh`, after the `.app` is laid out:

1. Sign inside-out with the hardened runtime and a secure timestamp: `typst` first, then the app:
   `codesign --force --options runtime --timestamp --sign "Developer ID Application: …" <path>`. No App Sandbox (it
   would block arbitrary data folders and running `typst`). Add entitlements only if the hardened runtime breaks
   something in testing.
2. `codesign --verify --deep --strict` the app.
3. Build `Resume-Builder.dmg` with `hdiutil` (the app plus an `/Applications` symlink to drag onto), then sign the DMG.
4. `xcrun notarytool submit Resume-Builder.dmg --keychain-profile resume-builder --wait`; on failure print
   `notarytool log` and stop.
5. `xcrun stapler staple Resume-Builder.dmg`, then `spctl -a -t open --context context:primary-signature -vv` on the
   DMG and `spctl -a -t exec -vv` on the app.
6. Optional while the certificate is in hand: sign and notarize the two darwin CLI binaries too (submitted as a zip;
   bare binaries can't be stapled, Gatekeeper checks online), so the README's `curl` workaround can go.

## Phase 6: Release wiring and docs

1. `scripts/package.sh`: produce `Resume-Builder.dmg` alongside the archives, with no version in the name so
   `releases/latest/download/Resume-Builder.dmg` always works; add it to `checksums.txt`. Skip it (with a message)
   when not on macOS so `mise run package` still works elsewhere.
2. `scripts/release.sh` preflight, before anything is tagged: on macOS, require the Developer ID identity
   (`security find-identity`) and a working notary profile (`xcrun notarytool history --keychain-profile …`).
   Notarization can take minutes; it must finish before `git tag`, so a failure publishes nothing.
3. `mise.toml`: descriptions for `package`/`release` mention the DMG; add the `app` task from phase 2.
4. README: "Install" leads with the DMG (download, drag to Applications, open); the CLI archives and `curl` move under
   a "Command line" subsection. Document the `desktop` build tag and `mise run app` under development.

## Done when (macOS)

On a fresh macOS user account, on both an Apple Silicon and an Intel Mac:

- [ ] Downloading the DMG with Safari and opening it shows no Gatekeeper warning; dragging to Applications and
      double-clicking launches the app with its icon in the Dock.
- [ ] First run: "Choose…" picks a data folder; the next launch remembers it.
- [ ] Editing, preview, versions, cover letters, Sync files and Reveal in Finder work.
- [ ] PDF, Word and Markdown downloads open a Save dialog and write the file.
- [ ] Links in the preview and checks open in the default browser.
- [ ] Cmd-C/V/X/A/Z work in text fields; Cmd-Q quits and nothing keeps running (`pgrep resume-builder`, `pgrep typst`).
- [ ] Launching again while open focuses the existing window.
- [ ] `mise run release` refuses to tag when signing or notarization would fail, and otherwise publishes the DMG next
      to the six archives.

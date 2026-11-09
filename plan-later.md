## Phase 7: Bundle Poppler

Only the two tools the server runs: `pdftotext` (`render.go`) and `pdfinfo` (`a11y.go`).

1. Spike: build Poppler from source for each architecture with CMake and as few optional dependencies as
   possible (no Qt, GLib, Cairo, NSS, curl), then `lipo` the results. Homebrew bottles are single-architecture and
   pull in many dylibs, so they don't make a universal app.
2. Put the binaries in `Contents/MacOS/` and their dylibs in `Contents/Frameworks/`, rewrite load paths to
   `@executable_path/../Frameworks` (`install_name_tool`, or `dylibbundler`), and sign every dylib and binary before
   the app in phase 5.
3. GPL compliance: ship Poppler's `COPYING` in `Contents/Resources/licenses/poppler/` and state where its source is
   (the exact release tarball). Running them as separate programs keeps the app's own license independent.
4. Until this lands, the app replaces the "install poppler" errors with a plain explanation that this feature isn't
   available yet: telling a non-technical user to run `brew` doesn't help them.

If the spike shows building Poppler for both architectures is a sink, the alternative is replacing both tools with Go:
text extraction is available in Go libraries, but reading the tag tree (`pdfinfo -struct-text`) would be custom code.

## Phase 8: Later

- **Windows:** Wails app (WebView2 ships with Windows 11), installer, and code signing (Azure Artifact Signing or an
  OV certificate) so SmartScreen doesn't warn. Needs a Windows build runner (GitHub Actions), since the desktop build
  uses CGO.
- **Linux:** keep the tarball; consider an AppImage only if someone asks.
- **Updates:** at minimum, the app checks the GitHub latest release on launch and offers a download link; a real
  auto-updater (Sparkle) is a separate project.

## Done when

- [ ] After phase 7: the "What machines read" tab and all checks work without Homebrew.

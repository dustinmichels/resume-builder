//go:build release

// Package resumebuilder bundles the Typst templates and the built editor into
// release builds of the server (go build -tags release ./server). Go can only
// embed files below a package's folder, hence this package at the root.
package resumebuilder

import "embed"

// Files holds template/ and web/dist/; run `mise run build` first.
//
//go:embed template web/dist
var Files embed.FS

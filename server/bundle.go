package main

// Release builds (-tags release, see scripts/package.sh) carry the templates
// and the built editor inside the binary and ship typst beside it, so they run
// from any folder. Other builds read both from the repo.

import (
	"bytes"
	"io/fs"
	"os"
	"path"
	"path/filepath"
)

// buildVersion is set at link time by scripts/package.sh.
var buildVersion = "dev"

// bundled holds template/ and web/dist/ in release builds; nil otherwise.
var bundled fs.FS

// templateFolder returns the folder holding the templates: the bundled ones,
// unpacked into the user cache folder, in release builds; else template/ in
// the working folder.
func templateFolder() (string, error) {
	if bundled == nil {
		return "template", nil
	}
	cache, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(cache, "resume-builder", "template")
	return dir, unpack(bundled, "template", dir)
}

// unpack copies the files under root in src into dir. Unchanged files are left
// alone, so their modification times, which Sync files compares PDFs against,
// only move when a release changes a template.
func unpack(src fs.FS, root, dir string) error {
	return fs.WalkDir(src, root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, filepath.FromSlash(p))
		if err != nil {
			return err
		}
		dst := filepath.Join(dir, rel)
		if d.IsDir() {
			return os.MkdirAll(dst, 0o755)
		}
		b, err := fs.ReadFile(src, p)
		if err != nil {
			return err
		}
		if old, err := os.ReadFile(dst); err == nil && bytes.Equal(old, b) {
			return nil
		}
		return os.WriteFile(dst, b, 0o644)
	})
}

// editor returns the built editor to serve: the static folder if set, else the
// bundled editor in release builds or web/dist; nil when there is none.
func editor(static string) fs.FS {
	if static == "" && bundled != nil {
		sub, err := fs.Sub(bundled, path.Join("web", "dist"))
		if err != nil {
			panic(err) // only for an invalid path
		}
		return sub
	}
	if static == "" {
		static = filepath.Join("web", "dist")
	}
	if fi, err := os.Stat(static); err != nil || !fi.IsDir() {
		return nil
	}
	return os.DirFS(static)
}

// preferBundledTools puts the binary's folder first on PATH in release builds,
// so the typst shipped beside it is the one run.
func preferBundledTools() error {
	if bundled == nil {
		return nil
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	if exe, err = filepath.EvalSymlinks(exe); err != nil {
		return err
	}
	return os.Setenv("PATH", filepath.Dir(exe)+string(os.PathListSeparator)+os.Getenv("PATH"))
}

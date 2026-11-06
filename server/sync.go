package main

// Exported copies kept beside a resume's documents: resume.json and every
// saved version get a Markdown and a PDF file of the same name,
//
//	<dataDir>/<resume name>/resume.md, resume.pdf
//	<dataDir>/<resume name>/versions/<version id>.md, <version id>.pdf

import (
	"bytes"
	"context"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// exportExts are the files written beside each resume document.
var exportExts = []string{".md", ".pdf"}

type syncFailure struct {
	File  string `json:"file"`
	Error string `json:"error"`
}

type syncResult struct {
	// Written lists the files created or updated, relative to the resume folder.
	Written []string      `json:"written"`
	Failed  []syncFailure `json:"failed"`
}

// syncFiles brings the Markdown and PDF beside resume.json and each version
// of a resume up to date. A document that fails to render is reported and
// the rest still sync.
func (s *server) syncFiles(w http.ResponseWriter, r *http.Request) {
	dir, unlock, ok := s.lockDir(w, false)
	if !ok {
		return
	}
	defer unlock()
	rdir, ok := resumeDir(w, dir, r.PathValue("id"))
	if !ok {
		return
	}
	tpl, err := os.Stat(s.templatePath)
	if err != nil {
		httpError(w, http.StatusInternalServerError, err)
		return
	}
	versions, err := listVersions(rdir)
	if err != nil {
		httpError(w, http.StatusInternalServerError, err)
		return
	}
	docs := []string{resumeFile}
	for _, v := range versions {
		docs = append(docs, filepath.Join(versionsDir, v.ID+".json"))
	}

	res := syncResult{Written: []string{}, Failed: []syncFailure{}}
	for _, rel := range docs {
		written, err := s.syncExports(r.Context(), filepath.Join(rdir, rel), tpl.ModTime())
		base := filepath.ToSlash(strings.TrimSuffix(rel, ".json"))
		for _, ext := range written {
			res.Written = append(res.Written, base+ext)
		}
		if err != nil {
			res.Failed = append(res.Failed, syncFailure{File: filepath.ToSlash(rel), Error: err.Error()})
		}
	}
	writeJSON(w, res)
}

// syncExports updates the exports beside the JSON document at path and
// returns the extensions it wrote. The Markdown is rewritten when its content
// differs; the PDF when it is older than the document or the template.
func (s *server) syncExports(ctx context.Context, path string, tplModified time.Time) ([]string, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	doc, err := decodeDoc(raw)
	if err != nil {
		return nil, err
	}
	base := strings.TrimSuffix(path, ".json")
	var written []string

	md := markdownResume(doc)
	if old, err := os.ReadFile(base + ".md"); err != nil || !bytes.Equal(old, md) {
		if err := writeFileAtomic(base+".md", md); err != nil {
			return written, err
		}
		written = append(written, ".md")
	}

	newest := fi.ModTime()
	if tplModified.After(newest) {
		newest = tplModified
	}
	if pdf, err := os.Stat(base + ".pdf"); err == nil && !pdf.ModTime().Before(newest) {
		return written, nil
	}
	out, err := render(ctx, s.templatePath, raw, false)
	if err != nil {
		return written, err
	}
	defer out.Close()
	pdf, err := os.ReadFile(out.PDFPath)
	if err != nil {
		return written, err
	}
	if err := writeFileAtomic(base+".pdf", pdf); err != nil {
		return written, err
	}
	return append(written, ".pdf"), nil
}

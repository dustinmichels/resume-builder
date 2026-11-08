package main

// Exported copies kept beside a resume's documents: resume.json, every cover
// letter and the same in every saved version get a Markdown, a Word and a PDF
// file of the same name,
//
//	<dataDir>/<resume name>/resume.md, resume.docx, resume.pdf
//	<dataDir>/<resume name>/cover-letters/<letter name>.md, .docx, .pdf
//	<dataDir>/<resume name>/versions/<version id>/resume.md, …, cover-letters/<letter name>.md, …

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// exportExts are the files written beside each resume document.
var exportExts = []string{".md", ".docx", ".pdf"}

type syncFailure struct {
	File  string `json:"file"`
	Error string `json:"error"`
}

type syncResult struct {
	// Written lists the files created or updated, relative to the resume folder.
	Written []string      `json:"written"`
	Failed  []syncFailure `json:"failed"`
}

// syncFiles brings the Markdown, Word and PDF beside resume.json and each
// cover letter of a resume and of each of its versions up to date. A document
// that fails to render is reported and the rest still sync.
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
	versions, err := listVersions(rdir)
	if err != nil {
		httpError(w, http.StatusInternalServerError, err)
		return
	}
	res := syncResult{Written: []string{}, Failed: []syncFailure{}}
	s.exportFolder(r.Context(), rdir, "", &res)
	for _, v := range versions {
		s.exportVersion(r.Context(), rdir, v.ID, &res)
	}
	writeJSON(w, res)
}

// exportVersion brings the exports of version vid of the resume in rdir up to
// date, adding them to res.
func (s *server) exportVersion(ctx context.Context, rdir, vid string, res *syncResult) {
	rel := filepath.Join(versionsDir, vid)
	s.exportFolder(ctx, filepath.Join(rdir, rel), rel, res)
}

// exportFolder brings the exports beside resume.json and each cover letter in
// folder up to date, adding them to res. folder is a resume's, or the version
// at rel inside it; res holds paths relative to the resume's folder.
func (s *server) exportFolder(ctx context.Context, folder, rel string, res *syncResult) {
	record := func(name string, written []string, err error) {
		p := filepath.ToSlash(filepath.Join(rel, name))
		base := strings.TrimSuffix(p, ".json")
		for _, ext := range written {
			res.Written = append(res.Written, base+ext)
		}
		if err != nil {
			res.Failed = append(res.Failed, syncFailure{File: p, Error: err.Error()})
		}
	}
	resumePath := filepath.Join(folder, resumeFile)
	if tpl, err := os.Stat(s.templatePath); err != nil {
		record(resumeFile, nil, err)
	} else {
		written, err := s.syncResume(ctx, resumePath, tpl.ModTime())
		record(resumeFile, written, err)
	}

	letters, err := listLetters(folder)
	if err != nil {
		record(lettersDir, nil, err)
		return
	}
	if len(letters) == 0 {
		return
	}
	// A letter's PDF shows its resume's name and style, so it is redrawn
	// when the resume or the letter template changes too.
	resumeRaw, rerr := os.ReadFile(resumePath)
	rfi, ferr := os.Stat(resumePath)
	lfi, lerr := os.Stat(s.letterTemplatePath)
	depsErr := errors.Join(rerr, ferr, lerr)
	for _, l := range letters {
		name := filepath.Join(lettersDir, l.ID+".json")
		if depsErr != nil {
			record(name, nil, depsErr)
			continue
		}
		written, err := s.syncLetter(ctx, filepath.Join(folder, name), resumeRaw, latest(rfi.ModTime(), lfi.ModTime()))
		record(name, written, err)
	}
}

func latest(a, b time.Time) time.Time {
	if b.After(a) {
		return b
	}
	return a
}

// syncResume updates the exports beside the resume JSON at path (the
// resume or a version).
func (s *server) syncResume(ctx context.Context, path string, tplModified time.Time) ([]string, error) {
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
	docx, err := docxResume(doc)
	if err != nil {
		return nil, err
	}
	return writeExports(strings.TrimSuffix(path, ".json"), markdownResume(doc), docx, latest(fi.ModTime(), tplModified), func() (*rendered, error) {
		return renderResume(ctx, s.templatePath, raw, false)
	})
}

// syncLetter updates the exports beside the letter JSON at path; its PDF is
// also redrawn when older than depsModified.
func (s *server) syncLetter(ctx context.Context, path string, resumeRaw []byte, depsModified time.Time) ([]string, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	data, ld, err := letterData(resumeRaw, raw)
	if err != nil {
		return nil, err
	}
	docx, err := docxLetter(ld)
	if err != nil {
		return nil, err
	}
	return writeExports(strings.TrimSuffix(path, ".json"), markdownLetter(ld), docx, latest(fi.ModTime(), depsModified), func() (*rendered, error) {
		return render(ctx, s.letterTemplatePath, map[string][]byte{"data.json": data}, false)
	})
}

// writeExports writes base.md, base.docx and base.pdf and returns the
// extensions it wrote. The Markdown and Word files are rewritten when their
// content differs (both are built deterministically); the PDF when it is
// older than newest.
func writeExports(base string, md, docx []byte, newest time.Time, renderPDF func() (*rendered, error)) ([]string, error) {
	var written []string
	for _, f := range []struct {
		ext  string
		body []byte
	}{{".md", md}, {".docx", docx}} {
		if old, err := os.ReadFile(base + f.ext); err == nil && bytes.Equal(old, f.body) {
			continue
		}
		if err := writeFileAtomic(base+f.ext, f.body); err != nil {
			return written, err
		}
		written = append(written, f.ext)
	}

	if pdf, err := os.Stat(base + ".pdf"); err == nil && !pdf.ModTime().Before(newest) {
		return written, nil
	}
	out, err := renderPDF()
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

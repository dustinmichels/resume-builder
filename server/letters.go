package main

// Cover letters: any number per resume, kept in its folder.
//
//	<dataDir>/<resume name>/cover-letters/<letter name>.json
//	<dataDir>/<resume name>/cover-letters/<letter name>.md, .pdf   (written by sync)
//
// A letter holds its own words and body font. The name at its top, the accent
// color, the heading font and the page margin come from its resume, so the
// two always match.

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const lettersDir = "cover-letters"

// Letter is the cover letter JSON. The frontend owns the full schema (block
// ids pass through untouched).
type Letter struct {
	Title    string        `json:"title"`
	Header   []string      `json:"header"` // lines under the name
	Greeting string        `json:"greeting"`
	Blocks   []LetterBlock `json:"blocks"`
	Closing  string        `json:"closing"` // followed by the name
	Style    *LetterStyle  `json:"style"`
}

type LetterBlock struct {
	Kind  string   `json:"kind"` // paragraph | heading | bullets
	Text  string   `json:"text"`
	Items []string `json:"items"`
}

type LetterStyle struct {
	Font     string  `json:"font"`
	FontSize float64 `json:"fontSize"`
}

// LetterDoc is what the letter template reads as data.json: a letter with
// its resume's basics and style.
type LetterDoc struct {
	Basics *DocBasics `json:"basics"`
	Style  *Style     `json:"style"`
	Letter *Letter    `json:"letter"`
}

// decodeLetter parses a saved letter; the shape check keeps obviously wrong
// payloads from overwriting it.
func decodeLetter(raw []byte) (Letter, error) {
	var l Letter
	if err := json.Unmarshal(raw, &l); err != nil {
		return l, fmt.Errorf("invalid cover letter JSON: %w", err)
	}
	if l.Blocks == nil || l.Style == nil {
		return l, errors.New("cover letter JSON needs blocks and style")
	}
	return l, nil
}

func decodeLetterDoc(raw []byte) (LetterDoc, error) {
	var ld LetterDoc
	if err := json.Unmarshal(raw, &ld); err != nil {
		return ld, fmt.Errorf("invalid cover letter JSON: %w", err)
	}
	if ld.Basics == nil || ld.Style == nil || ld.Letter == nil || ld.Letter.Blocks == nil || ld.Letter.Style == nil {
		return ld, errors.New("cover letter JSON needs basics, style and a letter with blocks and style")
	}
	return ld, nil
}

func readLetterDoc(r *http.Request) ([]byte, LetterDoc, error) {
	raw, err := io.ReadAll(io.LimitReader(r.Body, maxBody))
	if err != nil {
		return nil, LetterDoc{}, err
	}
	ld, err := decodeLetterDoc(raw)
	return raw, ld, err
}

// letterData joins a saved letter with its saved resume into the template's data.json.
func letterData(resumeRaw, letterRaw []byte) ([]byte, LetterDoc, error) {
	var resume struct {
		Basics json.RawMessage `json:"basics"`
		Style  json.RawMessage `json:"style"`
	}
	if err := json.Unmarshal(resumeRaw, &resume); err != nil {
		return nil, LetterDoc{}, fmt.Errorf("invalid resume JSON: %w", err)
	}
	data, err := json.Marshal(map[string]json.RawMessage{"basics": resume.Basics, "style": resume.Style, "letter": letterRaw})
	if err != nil {
		return nil, LetterDoc{}, err
	}
	ld, err := decodeLetterDoc(data)
	return data, ld, err
}

// asDoc reads the letter as a resume of text sections, one per heading, in
// page order, so the resume's text-layer, screen-reader and link checks apply.
func (ld LetterDoc) asDoc() Doc {
	l := ld.Letter
	doc := Doc{Basics: &DocBasics{Name: ld.Basics.Name, Headline: l.Header}, Style: ld.Style}
	sec := DocSection{Kind: "text", Title: l.Title, Paragraphs: []string{l.Greeting}}
	for _, b := range l.Blocks {
		switch b.Kind {
		case "heading":
			doc.Sections = append(doc.Sections, sec)
			sec = DocSection{Kind: "text", Title: b.Text}
		case "paragraph":
			sec.Paragraphs = append(sec.Paragraphs, b.Text)
		case "bullets":
			sec.Paragraphs = append(sec.Paragraphs, b.Items...)
		}
	}
	if strings.TrimSpace(l.Closing) != "" {
		sec.Paragraphs = append(sec.Paragraphs, l.Closing, ld.Basics.Name)
	}
	doc.Sections = append(doc.Sections, sec)
	return doc
}

// sanitizeLetterName turns a requested letter name into a file name (without .json).
func sanitizeLetterName(s string) string {
	s = strings.TrimLeft(sanitize(s), ". ")
	if s == "" {
		return "Cover letter"
	}
	return s
}

func listLetters(rdir string) ([]summary, error) {
	out := []summary{}
	entries, err := os.ReadDir(filepath.Join(rdir, lettersDir))
	if errors.Is(err, fs.ErrNotExist) {
		return out, nil
	}
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		id, ok := strings.CutSuffix(e.Name(), ".json")
		if !ok || !validID(id) || !e.Type().IsRegular() {
			continue
		}
		fi, err := e.Info()
		if err != nil {
			return nil, err
		}
		out = append(out, summary{ID: id, Modified: fi.ModTime()})
	}
	newestFirst(out)
	return out, nil
}

// letterFile returns the file of letter lid of the resume in rdir, answering
// 400 for ids that are not a plain file name and 404 for letters that don't exist.
func letterFile(w http.ResponseWriter, rdir, lid string) (string, bool) {
	p, err := childPath(filepath.Join(rdir, lettersDir), lid, ".json")
	if err != nil {
		httpError(w, http.StatusBadRequest, err)
		return "", false
	}
	_, ok := found(w, p, "cover letter", lid)
	return p, ok
}

func indentJSON(raw []byte) ([]byte, error) {
	var pretty bytes.Buffer
	if err := json.Indent(&pretty, raw, "", "  "); err != nil {
		return nil, err
	}
	pretty.WriteByte('\n')
	return pretty.Bytes(), nil
}

func (s *server) listLetters(w http.ResponseWriter, r *http.Request) {
	dir, unlock, ok := s.lockDir(w, false)
	if !ok {
		return
	}
	defer unlock()
	rdir, ok := resumeDir(w, dir, r.PathValue("id"))
	if !ok {
		return
	}
	list, err := listLetters(rdir)
	if err != nil {
		httpError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, list)
}

// createLetter saves a new letter: the given content, or a copy of letter `from`.
func (s *server) createLetter(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name   string          `json:"name"`
		From   string          `json:"from"`
		Letter json.RawMessage `json:"letter"`
	}
	if err := decodeJSON(r, &body); err != nil {
		httpError(w, http.StatusBadRequest, err)
		return
	}
	if body.From == "" {
		if _, err := decodeLetter(body.Letter); err != nil {
			httpError(w, http.StatusBadRequest, err)
			return
		}
	}
	dir, unlock, ok := s.lockDir(w, true)
	if !ok {
		return
	}
	defer unlock()
	rdir, ok := resumeDir(w, dir, r.PathValue("id"))
	if !ok {
		return
	}
	raw := []byte(body.Letter)
	if body.From != "" {
		src, ok := letterFile(w, rdir, body.From)
		if !ok {
			return
		}
		var err error
		if raw, err = os.ReadFile(src); err != nil {
			httpError(w, http.StatusInternalServerError, err)
			return
		}
	}
	pretty, err := indentJSON(raw)
	if err != nil {
		httpError(w, http.StatusBadRequest, err)
		return
	}
	ldir := filepath.Join(rdir, lettersDir)
	if err := os.MkdirAll(ldir, 0o755); err != nil {
		httpError(w, http.StatusInternalServerError, err)
		return
	}
	id := uniqueName(ldir, sanitizeLetterName(body.Name), ".json")
	if err := writeFileAtomic(filepath.Join(ldir, id+".json"), pretty); err != nil {
		httpError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSONStatus(w, http.StatusCreated, map[string]string{"id": id})
}

func (s *server) getLetter(w http.ResponseWriter, r *http.Request) {
	dir, unlock, ok := s.lockDir(w, false)
	if !ok {
		return
	}
	defer unlock()
	rdir, ok := resumeDir(w, dir, r.PathValue("id"))
	if !ok {
		return
	}
	if p, ok := letterFile(w, rdir, r.PathValue("lid")); ok {
		serveJSONFile(w, p)
	}
}

// putLetter saves an existing letter; a deleted or renamed one is not recreated.
func (s *server) putLetter(w http.ResponseWriter, r *http.Request) {
	raw, err := io.ReadAll(io.LimitReader(r.Body, maxBody))
	if err != nil {
		httpError(w, http.StatusBadRequest, err)
		return
	}
	if _, err := decodeLetter(raw); err != nil {
		httpError(w, http.StatusBadRequest, err)
		return
	}
	pretty, err := indentJSON(raw)
	if err != nil {
		httpError(w, http.StatusBadRequest, err)
		return
	}
	dir, unlock, ok := s.lockDir(w, true)
	if !ok {
		return
	}
	defer unlock()
	rdir, ok := resumeDir(w, dir, r.PathValue("id"))
	if !ok {
		return
	}
	p, ok := letterFile(w, rdir, r.PathValue("lid"))
	if !ok {
		return
	}
	if err := writeFileAtomic(p, pretty); err != nil {
		httpError(w, http.StatusInternalServerError, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// renameLetter renames a letter and the exports sync wrote beside it.
func (s *server) renameLetter(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name string `json:"name"`
	}
	if err := decodeJSON(r, &body); err != nil {
		httpError(w, http.StatusBadRequest, err)
		return
	}
	if strings.TrimSpace(body.Name) == "" {
		httpError(w, http.StatusBadRequest, errors.New("the name is empty"))
		return
	}
	dir, unlock, ok := s.lockDir(w, true)
	if !ok {
		return
	}
	defer unlock()
	rdir, ok := resumeDir(w, dir, r.PathValue("id"))
	if !ok {
		return
	}
	lid := r.PathValue("lid")
	p, ok := letterFile(w, rdir, lid)
	if !ok {
		return
	}
	newID := sanitizeLetterName(body.Name)
	if newID != lid {
		target := filepath.Join(filepath.Dir(p), newID+".json")
		// On case-insensitive file systems a change of case finds the
		// letter itself, which is not a conflict.
		if tfi, err := os.Lstat(target); err == nil {
			if sfi, err := os.Lstat(p); err != nil || !os.SameFile(sfi, tfi) {
				httpError(w, http.StatusConflict, fmt.Errorf("a cover letter named %q already exists", newID))
				return
			}
		} else if !errors.Is(err, fs.ErrNotExist) {
			httpError(w, http.StatusInternalServerError, err)
			return
		}
		if err := os.Rename(p, target); err != nil {
			httpError(w, http.StatusInternalServerError, err)
			return
		}
		from, to := strings.TrimSuffix(p, ".json"), strings.TrimSuffix(target, ".json")
		for _, ext := range exportExts {
			if err := os.Rename(from+ext, to+ext); err != nil && !errors.Is(err, fs.ErrNotExist) {
				httpError(w, http.StatusInternalServerError, err)
				return
			}
		}
	}
	writeJSON(w, map[string]string{"id": newID})
}

// deleteLetter moves a letter to the data folder's trash, named after its
// resume, and deletes its exports.
func (s *server) deleteLetter(w http.ResponseWriter, r *http.Request) {
	dir, unlock, ok := s.lockDir(w, true)
	if !ok {
		return
	}
	defer unlock()
	id := r.PathValue("id")
	rdir, ok := resumeDir(w, dir, id)
	if !ok {
		return
	}
	lid := r.PathValue("lid")
	p, ok := letterFile(w, rdir, lid)
	if !ok {
		return
	}
	trash := filepath.Join(dir, trashDir)
	if err := os.MkdirAll(trash, 0o755); err != nil {
		httpError(w, http.StatusInternalServerError, err)
		return
	}
	name := fmt.Sprintf("%s – %s %s", id, lid, time.Now().Format(stampLayout))
	target := filepath.Join(trash, name+".json")
	for n := 2; exists(target); n++ {
		target = filepath.Join(trash, fmt.Sprintf("%s-%d.json", name, n))
	}
	if err := os.Rename(p, target); err != nil {
		httpError(w, http.StatusInternalServerError, err)
		return
	}
	base := strings.TrimSuffix(p, ".json")
	for _, ext := range exportExts {
		if err := os.Remove(base + ext); err != nil && !errors.Is(err, fs.ErrNotExist) {
			httpError(w, http.StatusInternalServerError, err)
			return
		}
	}
	w.WriteHeader(http.StatusNoContent)
}

// ── Rendering ───────────────────────────────────────────────────────────────
// These take a LetterDoc: the editor sends the open letter with the resume's
// current basics and style, whose template picks the letter template.

// renderLetter renders the LetterDoc in the request with its resume
// template's letter template, answering the request itself on failure.
func (s *server) renderLetter(w http.ResponseWriter, r *http.Request, svg bool) (*rendered, LetterDoc, bool) {
	raw, ld, err := readLetterDoc(r)
	var tpl string
	if err == nil {
		tpl, err = s.letterTemplateFile(ld.Style.Template)
	}
	if err != nil {
		httpError(w, http.StatusBadRequest, err)
		return nil, ld, false
	}
	out, err := render(r.Context(), tpl, map[string][]byte{"data.json": raw}, svg)
	if err != nil {
		renderError(w, err)
		return nil, ld, false
	}
	return out, ld, true
}

func (s *server) letterPreview(w http.ResponseWriter, r *http.Request) {
	out, ld, ok := s.renderLetter(w, r, true)
	if !ok {
		return
	}
	defer out.Close()
	doc := ld.asDoc()
	writeJSON(w, previewOf(r.Context(), out, doc, markdownLetter(ld), func(text string) []Check {
		// No dot-leader check: the template draws no leaders, and an ellipsis is prose.
		l := checkList{group: groupParsers}
		l.decorativeSymbols(text, textLines(text))
		l.missingPieces(doc, text)
		if n := len(out.SVGs); n > 1 {
			l.add("info", fmt.Sprintf("%d pages", n), "Most cover letters fit on one page.")
		}
		return l.checks
	}))
}

func (s *server) letterLinks(w http.ResponseWriter, r *http.Request) {
	_, ld, err := readLetterDoc(r)
	if err != nil {
		httpError(w, http.StatusBadRequest, err)
		return
	}
	checks, err := s.links.check(r.Context(), ld.asDoc())
	if err != nil {
		return // the editor moved on
	}
	writeJSON(w, checks)
}

func (s *server) letterPDF(w http.ResponseWriter, r *http.Request) {
	out, ld, ok := s.renderLetter(w, r, false)
	if !ok {
		return
	}
	defer out.Close()
	attachment(w, ld.Basics.Name, "Cover Letter", "application/pdf", "pdf")
	http.ServeFile(w, r, out.PDFPath)
}

func letterDocx(w http.ResponseWriter, r *http.Request) {
	_, ld, err := readLetterDoc(r)
	if err != nil {
		httpError(w, http.StatusBadRequest, err)
		return
	}
	b, err := docxLetter(ld)
	if err != nil {
		httpError(w, http.StatusInternalServerError, err)
		return
	}
	attachment(w, ld.Basics.Name, "Cover Letter", "application/vnd.openxmlformats-officedocument.wordprocessingml.document", "docx")
	w.Write(b)
}

func letterMarkdown(w http.ResponseWriter, r *http.Request) {
	_, ld, err := readLetterDoc(r)
	if err != nil {
		httpError(w, http.StatusBadRequest, err)
		return
	}
	attachment(w, ld.Basics.Name, "Cover Letter", "text/markdown; charset=utf-8", "md")
	w.Write(markdownLetter(ld))
}

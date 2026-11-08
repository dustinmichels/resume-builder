// Command server is the backend for the resume editor: it stores resumes and
// their versions in the user's data folder, renders them with Typst, and
// reports what machines read from the PDF.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sync"
	"time"
)

const maxBody = 4 << 20

type server struct {
	templatePath       string
	letterTemplatePath string // cover-letter.typ beside the resume template
	configPath         string // where the chosen data folder is remembered
	legacyPath         string // pre-library resume, imported when seeding a new data folder

	mu      sync.RWMutex
	dataDir string // "" until the user chooses a folder

	links *linkChecker
}

func main() {
	addr := flag.String("addr", "127.0.0.1:8787", "listen address")
	configPath := flag.String("config", "", "settings file (default <user config dir>/resume-builder/config.json)")
	templatePath := flag.String("template", "template/resume.typ", "Typst template")
	static := flag.String("static", "web/dist", "built frontend to serve (skipped if missing)")
	flag.Parse()

	if err := checkTools(); err != nil {
		log.Fatal(err)
	}
	if *configPath == "" {
		p, err := defaultConfigPath()
		if err != nil {
			log.Fatal(err)
		}
		*configPath = p
	}
	s, err := newServer(*templatePath, *configPath, "data/resume.json")
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("resume server on http://%s", *addr)
	log.Fatal(http.ListenAndServe(*addr, newMux(s, *static)))
}

// newServer loads the remembered data folder from configPath. The cover
// letter template is read from templatePath's folder.
func newServer(templatePath, configPath, legacyPath string) (*server, error) {
	c, err := loadConfig(configPath)
	if err != nil {
		return nil, err
	}
	return &server{templatePath: templatePath, letterTemplatePath: filepath.Join(filepath.Dir(templatePath), "cover-letter.typ"),
		configPath: configPath, legacyPath: legacyPath, dataDir: c.DataDir, links: newLinkChecker()}, nil
}

// newMux routes the API, plus the built frontend in static if it exists.
func newMux(s *server, static string) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/settings", s.getSettings)
	mux.HandleFunc("PUT /api/settings", s.putSettings)
	mux.HandleFunc("GET /api/resumes", s.listResumes)
	mux.HandleFunc("POST /api/resumes", s.createResume)
	mux.HandleFunc("GET /api/resumes/{id}", s.getResume)
	mux.HandleFunc("PUT /api/resumes/{id}", s.putResume)
	mux.HandleFunc("PATCH /api/resumes/{id}", s.renameResume)
	mux.HandleFunc("DELETE /api/resumes/{id}", s.deleteResume)
	mux.HandleFunc("GET /api/resumes/{id}/versions", s.listVersions)
	mux.HandleFunc("POST /api/resumes/{id}/versions", s.createVersion)
	mux.HandleFunc("GET /api/resumes/{id}/versions/{vid}", s.getVersion)
	mux.HandleFunc("POST /api/resumes/{id}/versions/{vid}/restore", s.restoreVersion)
	mux.HandleFunc("DELETE /api/resumes/{id}/versions/{vid}", s.deleteVersion)
	mux.HandleFunc("POST /api/resumes/{id}/sync", s.syncFiles)
	mux.HandleFunc("POST /api/resumes/{id}/reveal", s.reveal)
	for path, file := range sideFiles {
		mux.HandleFunc("GET /api/resumes/{id}/"+path, s.getSideFile(file))
		mux.HandleFunc("PUT /api/resumes/{id}/"+path, s.putSideFile(file))
	}
	mux.HandleFunc("GET /api/resumes/{id}/letters", s.listLetters)
	mux.HandleFunc("POST /api/resumes/{id}/letters", s.createLetter)
	mux.HandleFunc("GET /api/resumes/{id}/letters/{lid}", s.getLetter)
	mux.HandleFunc("PUT /api/resumes/{id}/letters/{lid}", s.putLetter)
	mux.HandleFunc("PATCH /api/resumes/{id}/letters/{lid}", s.renameLetter)
	mux.HandleFunc("DELETE /api/resumes/{id}/letters/{lid}", s.deleteLetter)
	mux.HandleFunc("GET /api/templates", s.listTemplates)
	mux.HandleFunc("POST /api/preview", s.preview)
	mux.HandleFunc("POST /api/links", s.checkLinks)
	mux.HandleFunc("POST /api/pdf", s.pdf)
	mux.HandleFunc("POST /api/docx", docx)
	mux.HandleFunc("POST /api/md", markdown)
	mux.HandleFunc("POST /api/letter/preview", s.letterPreview)
	mux.HandleFunc("POST /api/letter/links", s.letterLinks)
	mux.HandleFunc("POST /api/letter/pdf", s.letterPDF)
	mux.HandleFunc("POST /api/letter/docx", letterDocx)
	mux.HandleFunc("POST /api/letter/md", letterMarkdown)
	if fi, err := os.Stat(static); err == nil && fi.IsDir() {
		mux.Handle("GET /", http.FileServer(http.Dir(static)))
	}
	return mux
}

type previewResponse struct {
	Pages      []string `json:"pages"`
	Text       string   `json:"text"`
	LayoutText string   `json:"layoutText"`
	Checks     []Check  `json:"checks"`
	Warnings   []string `json:"warnings"`
	// Markdown is the Markdown export, for the preview's Markdown tab.
	Markdown string `json:"markdown"`
	// Fields maps each editable value to its box on the page, for click-to-edit.
	Fields []FieldBox `json:"fields"`
	// Columns are the cells of side-by-side columns sections, for dragging the
	// dividers between them.
	Columns []FieldBox `json:"columns"`
}

func (s *server) preview(w http.ResponseWriter, r *http.Request) {
	raw, doc, err := readDoc(r)
	if err != nil {
		httpError(w, http.StatusBadRequest, err)
		return
	}
	tpl, err := s.templateFile(doc.Style.Template)
	if err != nil {
		httpError(w, http.StatusBadRequest, err)
		return
	}
	out, err := renderResume(r.Context(), tpl, raw, true)
	if err != nil {
		renderError(w, err)
		return
	}
	defer out.Close()
	writeJSON(w, previewOf(r.Context(), out, doc, markdownResume(doc), func(text string) []Check {
		return parserChecks(doc, text, len(out.SVGs))
	}, dateChecks(doc, time.Now())...))
}

// previewOf describes a rendered document for the preview: its pages, text
// layer and Markdown, and the checks of its text layer (by parser), tags and
// links, then extra.
func previewOf(ctx context.Context, out *rendered, doc Doc, md []byte, parser func(text string) []Check, extra ...Check) previewResponse {
	resp := previewResponse{Pages: out.SVGs, Warnings: out.Warnings, Fields: out.Fields, Columns: out.Columns, Markdown: string(md)}
	text, layout, err := extractText(ctx, out.PDFPath)
	if err != nil {
		resp.Checks = []Check{{Group: groupParsers, Level: "warn", Title: "Text layer not checked", Detail: err.Error()}}
	} else {
		resp.Text, resp.LayoutText = text, layout
		resp.Checks = parser(text)
	}
	var facts *pdfFacts
	if f, err := inspectPDF(ctx, out.PDFPath); err != nil {
		resp.Checks = append(resp.Checks, Check{Group: groupScreenReaders, Level: "warn", Title: "Tags and links not checked", Detail: err.Error()})
	} else {
		facts = &f
	}
	resp.Checks = append(resp.Checks, accessibilityChecks(doc, out.UAErrors, facts)...)
	resp.Checks = append(resp.Checks, linkChecks(doc, facts)...)
	resp.Checks = append(resp.Checks, extra...)
	return resp
}

// checkLinks requests the resume's web links. It's separate from preview so
// slow sites never hold up rendering.
func (s *server) checkLinks(w http.ResponseWriter, r *http.Request) {
	_, doc, err := readDoc(r)
	if err != nil {
		httpError(w, http.StatusBadRequest, err)
		return
	}
	checks, err := s.links.check(r.Context(), doc)
	if err != nil {
		return // the editor moved on
	}
	writeJSON(w, checks)
}

var unsafeFilename = regexp.MustCompile(`[^\w .-]+`)

func (s *server) pdf(w http.ResponseWriter, r *http.Request) {
	raw, doc, err := readDoc(r)
	if err != nil {
		httpError(w, http.StatusBadRequest, err)
		return
	}
	tpl, err := s.templateFile(doc.Style.Template)
	if err != nil {
		httpError(w, http.StatusBadRequest, err)
		return
	}
	out, err := renderResume(r.Context(), tpl, raw, false)
	if err != nil {
		renderError(w, err)
		return
	}
	defer out.Close()

	attachment(w, docName(doc), "Resume", "application/pdf", "pdf")
	http.ServeFile(w, r, out.PDFPath)
}

func docx(w http.ResponseWriter, r *http.Request) {
	_, doc, err := readDoc(r)
	if err != nil {
		httpError(w, http.StatusBadRequest, err)
		return
	}
	b, err := docxResume(doc)
	if err != nil {
		httpError(w, http.StatusInternalServerError, err)
		return
	}
	attachment(w, docName(doc), "Resume", "application/vnd.openxmlformats-officedocument.wordprocessingml.document", "docx")
	w.Write(b)
}

func markdown(w http.ResponseWriter, r *http.Request) {
	_, doc, err := readDoc(r)
	if err != nil {
		httpError(w, http.StatusBadRequest, err)
		return
	}
	attachment(w, docName(doc), "Resume", "text/markdown; charset=utf-8", "md")
	w.Write(markdownResume(doc))
}

func docName(doc Doc) string {
	if doc.Basics == nil {
		return ""
	}
	return doc.Basics.Name
}

// attachment sets download headers naming the file "<owner> - <what>".
func attachment(w http.ResponseWriter, owner, what, contentType, ext string) {
	name := unsafeFilename.ReplaceAllString(owner, "")
	if name == "" {
		name = what
	} else {
		name += " - " + what
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.%s"`, name, ext))
}

// readDoc reads a resume document from the request body, returning the raw
// JSON and its parsed form.
func readDoc(r *http.Request) ([]byte, Doc, error) {
	raw, err := io.ReadAll(io.LimitReader(r.Body, maxBody))
	if err != nil {
		return nil, Doc{}, err
	}
	doc, err := decodeDoc(raw)
	if err != nil {
		return nil, doc, err
	}
	return raw, doc, nil
}

// decodeDoc parses a resume document, without its hidden parts. The shape
// check keeps obviously wrong payloads from overwriting the saved resume.
func decodeDoc(raw []byte) (Doc, error) {
	var doc Doc
	if err := json.Unmarshal(raw, &doc); err != nil {
		return doc, fmt.Errorf("invalid resume JSON: %w", err)
	}
	if doc.Basics == nil || doc.Sections == nil || doc.Style == nil {
		return doc, errors.New("resume JSON needs basics, sections and style")
	}
	doc.dropHidden()
	return doc, nil
}

func renderError(w http.ResponseWriter, err error) {
	var ce *compileError
	if errors.As(err, &ce) {
		writeJSONStatus(w, http.StatusUnprocessableEntity, map[string]string{"error": ce.Output})
		return
	}
	httpError(w, http.StatusInternalServerError, err)
}

func httpError(w http.ResponseWriter, status int, err error) {
	writeJSONStatus(w, status, map[string]string{"error": err.Error()})
}

func writeJSON(w http.ResponseWriter, v any) { writeJSONStatus(w, http.StatusOK, v) }

func writeJSONStatus(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

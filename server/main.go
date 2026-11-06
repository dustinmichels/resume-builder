// Command server is the backend for the resume editor: it stores the resume
// JSON, renders it with Typst, and reports what machines read from the PDF.
package main

import (
	"bytes"
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
)

const maxBody = 4 << 20

type server struct {
	dataPath     string
	templatePath string
}

func main() {
	addr := flag.String("addr", "127.0.0.1:8787", "listen address")
	dataPath := flag.String("data", "data/resume.json", "resume JSON file")
	templatePath := flag.String("template", "template/resume.typ", "Typst template")
	static := flag.String("static", "web/dist", "built frontend to serve (skipped if missing)")
	flag.Parse()

	if err := checkTools(); err != nil {
		log.Fatal(err)
	}
	s := &server{dataPath: *dataPath, templatePath: *templatePath}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/resume", s.getResume)
	mux.HandleFunc("PUT /api/resume", s.putResume)
	mux.HandleFunc("POST /api/preview", s.preview)
	mux.HandleFunc("POST /api/pdf", s.pdf)
	if fi, err := os.Stat(*static); err == nil && fi.IsDir() {
		mux.Handle("GET /", http.FileServer(http.Dir(*static)))
	}

	log.Printf("resume server on http://%s", *addr)
	log.Fatal(http.ListenAndServe(*addr, mux))
}

func (s *server) getResume(w http.ResponseWriter, r *http.Request) {
	b, err := os.ReadFile(s.dataPath)
	if err != nil {
		httpError(w, http.StatusInternalServerError, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(b)
}

func (s *server) putResume(w http.ResponseWriter, r *http.Request) {
	raw, _, err := readDoc(r)
	if err != nil {
		httpError(w, http.StatusBadRequest, err)
		return
	}
	var pretty bytes.Buffer
	if err := json.Indent(&pretty, raw, "", "  "); err != nil {
		httpError(w, http.StatusBadRequest, err)
		return
	}
	pretty.WriteByte('\n')
	// Write to a sibling temp file and rename, so a crash never leaves a
	// half-written resume behind.
	tmp, err := os.CreateTemp(filepath.Dir(s.dataPath), ".resume-*.json")
	if err != nil {
		httpError(w, http.StatusInternalServerError, err)
		return
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(pretty.Bytes()); err != nil {
		tmp.Close()
		httpError(w, http.StatusInternalServerError, err)
		return
	}
	if err := tmp.Close(); err != nil {
		httpError(w, http.StatusInternalServerError, err)
		return
	}
	if err := os.Rename(tmp.Name(), s.dataPath); err != nil {
		httpError(w, http.StatusInternalServerError, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type previewResponse struct {
	Pages      []string `json:"pages"`
	Text       string   `json:"text"`
	LayoutText string   `json:"layoutText"`
	Checks     []Check  `json:"checks"`
	Warnings   []string `json:"warnings"`
}

func (s *server) preview(w http.ResponseWriter, r *http.Request) {
	raw, doc, err := readDoc(r)
	if err != nil {
		httpError(w, http.StatusBadRequest, err)
		return
	}
	out, err := render(r.Context(), s.templatePath, raw, true)
	if err != nil {
		renderError(w, err)
		return
	}
	defer out.Close()

	resp := previewResponse{Pages: out.SVGs, Warnings: out.Warnings}
	text, layout, err := extractText(r.Context(), out.PDFPath)
	if err != nil {
		resp.Checks = []Check{{Level: "warn", Title: "Text layer not checked", Detail: err.Error()}}
	} else {
		resp.Text, resp.LayoutText = text, layout
		resp.Checks = analyze(doc, text, len(out.SVGs))
	}
	writeJSON(w, resp)
}

var unsafeFilename = regexp.MustCompile(`[^\w .-]+`)

func (s *server) pdf(w http.ResponseWriter, r *http.Request) {
	raw, doc, err := readDoc(r)
	if err != nil {
		httpError(w, http.StatusBadRequest, err)
		return
	}
	out, err := render(r.Context(), s.templatePath, raw, false)
	if err != nil {
		renderError(w, err)
		return
	}
	defer out.Close()

	name := unsafeFilename.ReplaceAllString(doc.Basics.Name, "")
	if name == "" {
		name = "Resume"
	}
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s - Resume.pdf"`, name))
	http.ServeFile(w, r, out.PDFPath)
}

// readDoc reads a resume document from the request body, returning the raw
// JSON and its parsed form. The shape check keeps obviously wrong payloads
// from overwriting the saved resume.
func readDoc(r *http.Request) ([]byte, Doc, error) {
	var doc Doc
	raw, err := io.ReadAll(io.LimitReader(r.Body, maxBody))
	if err != nil {
		return nil, doc, err
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, doc, fmt.Errorf("invalid resume JSON: %w", err)
	}
	if doc.Basics == nil || doc.Sections == nil || doc.Style == nil {
		return nil, doc, errors.New("resume JSON needs basics, sections and style")
	}
	return raw, doc, nil
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

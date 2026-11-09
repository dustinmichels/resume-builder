package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

const renderTimeout = 30 * time.Second

type rendered struct {
	dir      string
	PDFPath  string
	SVGs     []string
	Warnings []string
	Fields   []FieldBox // set with the SVGs
	Columns  []FieldBox // set with the SVGs
	// UAErrors are Typst's PDF/UA-1 (accessible PDF) violations. When there
	// are any, the PDF is compiled again without the standard so it still renders.
	UAErrors []string
}

func (r *rendered) Close() { os.RemoveAll(r.dir) }

type compileError struct{ Output string }

func (e *compileError) Error() string { return "typst: " + e.Output }

func checkTools() error {
	if _, err := exec.LookPath("typst"); err != nil {
		return errors.New("typst not found on PATH (install with `brew install typst`)")
	}
	if _, err := exec.LookPath("pdftotext"); err != nil {
		log.Printf("warning: machine view disabled: %v", popplerMissing("pdftotext"))
	}
	return nil
}

// renderResume renders a resume document with the resume template, which
// reads it as data.json and attaches its content as resume.json.
func renderResume(ctx context.Context, templatePath string, raw []byte, svg bool) (*rendered, error) {
	content, err := attachedJSON(raw)
	if err != nil {
		return nil, err
	}
	return render(ctx, templatePath, map[string][]byte{"data.json": raw, "resume.json": content}, svg)
}

// render compiles a template, with files written beside it, to PDF and, when
// svg is set, to one SVG per page plus the map of where each editable value
// sits on those pages. The template is re-read on every call so template edits
// show up live. Fonts in the fonts folder beside the template are found as if
// installed.
func render(ctx context.Context, templatePath string, files map[string][]byte, svg bool) (*rendered, error) {
	ctx, cancel := context.WithTimeout(ctx, renderTimeout)
	defer cancel()

	tpl, err := os.ReadFile(templatePath)
	if err != nil {
		return nil, err
	}
	// Absolute: typst runs in the temporary folder.
	fonts, err := filepath.Abs(filepath.Join(filepath.Dir(templatePath), "fonts"))
	if err != nil {
		return nil, err
	}
	dir, err := os.MkdirTemp("", "resume-render-*")
	if err != nil {
		return nil, err
	}
	main := filepath.Base(templatePath)
	out := &rendered{dir: dir, PDFPath: filepath.Join(dir, "output.pdf")}
	files[main] = tpl
	for name, b := range files {
		if err := os.WriteFile(filepath.Join(dir, name), b, 0o600); err != nil {
			out.Close()
			return nil, err
		}
	}

	svgErr := make(chan error, 1)
	type fieldsResult struct {
		marks []fieldMark
		err   error
	}
	fields := make(chan fieldsResult, 1)
	if svg {
		go func() {
			_, err := typst(ctx, dir, fonts, main, "page-{0p}.svg")
			svgErr <- err
		}()
		go func() {
			marks, err := queryFields(ctx, dir, fonts, main)
			fields <- fieldsResult{marks, err}
		}()
	} else {
		svgErr <- nil
		fields <- fieldsResult{}
	}
	stderr, pdfErr := typst(ctx, dir, fonts, main, "output.pdf", "--pdf-standard", "ua-1")
	if ce, ok := errors.AsType[*compileError](pdfErr); ok && strings.Contains(ce.Output, uaErrorTag) {
		out.UAErrors = slices.DeleteFunc(strings.Split(ce.Output, "\n"), func(l string) bool { return !strings.Contains(l, uaErrorTag) })
		stderr, pdfErr = typst(ctx, dir, fonts, main, "output.pdf")
	}
	f := <-fields
	if err := errors.Join(pdfErr, <-svgErr); err != nil {
		out.Close()
		return nil, err
	}
	out.Warnings = diagnostics(stderr)
	// The field map only powers click-to-edit; a failure there shouldn't hide
	// an otherwise good render.
	if f.err != nil {
		out.Warnings = append(out.Warnings, "click-to-edit map unavailable: "+f.err.Error())
	}
	out.Fields = fieldBoxes(f.marks)
	out.Columns = columnBoxes(f.marks)

	if svg {
		paths, err := filepath.Glob(filepath.Join(dir, "page-*.svg")) // sorted; {0p} zero-pads
		if err != nil {
			out.Close()
			return nil, err
		}
		for _, p := range paths {
			b, err := os.ReadFile(p)
			if err != nil {
				out.Close()
				return nil, err
			}
			out.SVGs = append(out.SVGs, string(b))
		}
	}
	return out, nil
}

// uaErrorTag starts Typst's PDF/UA-1 violation messages.
const uaErrorTag = "PDF/UA-1 error"

// typst compiles dir/main to dir/output, with the fonts in the fonts folder.
// It runs inside dir so diagnostics name files relative to the template
// ("resume.typ:14:18: …").
func typst(ctx context.Context, dir, fonts, main, output string, flags ...string) (string, error) {
	args := append([]string{"compile", "--diagnostic-format", "short", "--font-path", fonts}, flags...)
	cmd := exec.CommandContext(ctx, "typst", append(args, main, output)...)
	cmd.Dir = dir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		msg := strings.Join(diagnostics(stderr.String()), "\n")
		if msg == "" {
			msg = err.Error()
		}
		return "", &compileError{Output: msg}
	}
	return stderr.String(), nil
}

func diagnostics(stderr string) []string {
	lines := []string{}
	for _, l := range strings.Split(stderr, "\n") {
		if l = strings.TrimSpace(l); l != "" {
			lines = append(lines, l)
		}
	}
	return lines
}

// attachedJSON is the resume.json attached to the PDF: the content as it reads
// on the page, without editor-only fields (ids, style, layout flags), hidden
// sections, items and columns, or blank values the template skips.
func attachedJSON(raw []byte) ([]byte, error) {
	var doc Doc
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, err
	}
	doc.dropHidden()
	type contact struct {
		Text string `json:"text"`
		URL  string `json:"url,omitempty"`
	}
	type item struct {
		Type     string   `json:"type"`
		Title    string   `json:"title,omitempty"`
		Date     string   `json:"date,omitempty"`
		Subtitle string   `json:"subtitle,omitempty"`
		Location string   `json:"location,omitempty"`
		Text     string   `json:"text,omitempty"`
		Bullets  []string `json:"bullets,omitempty"`
	}
	type column struct {
		Title string   `json:"title,omitempty"`
		Lines []string `json:"lines,omitempty"`
	}
	type section struct {
		Kind       string   `json:"kind"`
		Title      string   `json:"title,omitempty"`
		Paragraphs []string `json:"paragraphs,omitempty"`
		Items      []item   `json:"items,omitempty"`
		Columns    []column `json:"columns,omitempty"`
	}
	type basics struct {
		Name     string    `json:"name"`
		Headline []string  `json:"headline,omitempty"`
		Contacts []contact `json:"contacts,omitempty"`
	}
	var out struct {
		Basics   *basics   `json:"basics,omitempty"`
		Sections []section `json:"sections"`
	}
	if bs := doc.Basics; bs != nil {
		out.Basics = &basics{Name: bs.Name, Headline: nonEmpty(bs.Headline)}
		for _, c := range bs.Contacts {
			if c.Text != "" {
				out.Basics.Contacts = append(out.Basics.Contacts, contact{c.Text, c.URL})
			}
		}
	}
	out.Sections = []section{}
	for _, s := range doc.Sections {
		sec := section{Kind: s.Kind, Title: s.Title, Paragraphs: nonEmpty(s.Paragraphs)}
		for _, it := range s.Items {
			sec.Items = append(sec.Items, item{it.Type, it.Title, it.Date, it.Subtitle, it.Location, it.Text, nonEmpty(it.Bullets)})
		}
		for _, c := range s.Columns {
			sec.Columns = append(sec.Columns, column{c.Title, nonEmpty(c.Lines)})
		}
		out.Sections = append(out.Sections, sec)
	}
	return json.MarshalIndent(out, "", "  ")
}

// extractText returns the PDF's text layer in content order (-raw, close to
// what most parsers see first) and in visual layout order (-layout).
// Pages are separated by form feeds.
func extractText(ctx context.Context, pdfPath string) (raw, layout string, err error) {
	if _, err := exec.LookPath("pdftotext"); err != nil {
		return "", "", popplerMissing("pdftotext")
	}
	run := func(mode string) (string, error) {
		out, err := exec.CommandContext(ctx, "pdftotext", mode, "-enc", "UTF-8", pdfPath, "-").Output()
		if err != nil {
			return "", fmt.Errorf("pdftotext %s: %w", mode, err)
		}
		return string(out), nil
	}
	if raw, err = run("-raw"); err != nil {
		return "", "", err
	}
	if layout, err = run("-layout"); err != nil {
		return "", "", err
	}
	return raw, layout, nil
}

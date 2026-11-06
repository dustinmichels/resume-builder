package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const renderTimeout = 30 * time.Second

type rendered struct {
	dir      string
	PDFPath  string
	SVGs     []string
	Warnings []string
}

func (r *rendered) Close() { os.RemoveAll(r.dir) }

type compileError struct{ Output string }

func (e *compileError) Error() string { return "typst: " + e.Output }

func checkTools() error {
	if _, err := exec.LookPath("typst"); err != nil {
		return errors.New("typst not found on PATH (install with `brew install typst`)")
	}
	if _, err := exec.LookPath("pdftotext"); err != nil {
		fmt.Fprintln(os.Stderr, "warning: pdftotext not found; machine view disabled (install with `brew install poppler`)")
	}
	return nil
}

// render compiles the resume to PDF and, when svg is set, to one SVG per page.
// The template is re-read on every call so template edits show up live.
func render(ctx context.Context, templatePath string, raw []byte, svg bool) (*rendered, error) {
	ctx, cancel := context.WithTimeout(ctx, renderTimeout)
	defer cancel()

	tpl, err := os.ReadFile(templatePath)
	if err != nil {
		return nil, err
	}
	content, err := contentOnly(raw)
	if err != nil {
		return nil, err
	}
	dir, err := os.MkdirTemp("", "resume-render-*")
	if err != nil {
		return nil, err
	}
	out := &rendered{dir: dir, PDFPath: filepath.Join(dir, "resume.pdf")}
	for name, b := range map[string][]byte{"resume.typ": tpl, "data.json": raw, "resume.json": content} {
		if err := os.WriteFile(filepath.Join(dir, name), b, 0o600); err != nil {
			out.Close()
			return nil, err
		}
	}

	svgErr := make(chan error, 1)
	if svg {
		go func() {
			_, err := typst(ctx, dir, "page-{0p}.svg")
			svgErr <- err
		}()
	} else {
		svgErr <- nil
	}
	stderr, pdfErr := typst(ctx, dir, "resume.pdf")
	if err := errors.Join(pdfErr, <-svgErr); err != nil {
		out.Close()
		return nil, err
	}
	out.Warnings = diagnostics(stderr)

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

// typst compiles dir/resume.typ to dir/output. It runs inside dir so
// diagnostics name files relative to the template ("resume.typ:14:18: …").
func typst(ctx context.Context, dir, output string) (string, error) {
	cmd := exec.CommandContext(ctx, "typst", "compile", "--diagnostic-format", "short", "resume.typ", output)
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

// contentOnly drops the style block, leaving the content that gets attached
// to the PDF as resume.json.
func contentOnly(raw []byte) ([]byte, error) {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, err
	}
	delete(m, "style")
	return json.MarshalIndent(m, "", "  ")
}

// extractText returns the PDF's text layer in content order (-raw, close to
// what most parsers see first) and in visual layout order (-layout).
// Pages are separated by form feeds.
func extractText(ctx context.Context, pdfPath string) (raw, layout string, err error) {
	if _, err := exec.LookPath("pdftotext"); err != nil {
		return "", "", errors.New("pdftotext not found; install poppler (`brew install poppler`) to see the machine view")
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

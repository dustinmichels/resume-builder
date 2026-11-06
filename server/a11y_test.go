package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// renderChecks renders doc through template (the real one, edited by edit)
// and returns its screen-reader and link checks.
func renderChecks(t *testing.T, doc string, edit func(string) string) []Check {
	t.Helper()
	for _, tool := range []string{"typst", "pdfinfo"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skip(tool + " not installed")
		}
	}
	tpl, err := os.ReadFile("../template/resume.typ")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "resume.typ")
	if err := os.WriteFile(path, []byte(edit(string(tpl))), 0o600); err != nil {
		t.Fatal(err)
	}
	out, err := renderResume(context.Background(), path, []byte(doc), false)
	if err != nil {
		t.Fatal(err)
	}
	defer out.Close()
	facts, err := inspectPDF(context.Background(), out.PDFPath)
	if err != nil {
		t.Fatal(err)
	}
	d := parseDoc(t, doc)
	return append(accessibilityChecks(d, out.UAErrors, &facts), linkChecks(d, &facts)...)
}

func TestAccessibilityChecksRendered(t *testing.T) {
	for _, c := range renderChecks(t, exportDoc, func(s string) string { return s }) {
		if c.Level != "ok" {
			t.Errorf("template as shipped: unexpected %s check %q: %s", c.Level, c.Title, c.Detail)
		}
	}

	// A template that hides the name from assistive technology and skips a
	// heading level still renders, with both problems reported.
	broken := renderChecks(t, exportDoc, func(s string) string {
		s = strings.Replace(s, `field("basics.name", upper(basics.name))`, `pdf.artifact(upper(basics.name))`, 1)
		if !strings.Contains(s, "pdf.artifact") {
			t.Fatal("template no longer renders the name as expected; update this test")
		}
		return s + "\n= Extra\n=== Skipped\n"
	})
	if got := levelOf(t, broken, "Fails Typst’s PDF/UA-1"); got != "error" {
		t.Errorf("skipped heading level: got %s, want error", got)
	}
	if got := levelOf(t, broken, "1 of"); got != "error" {
		t.Errorf("name as artifact: got %s, want error", got)
	}
}

func TestContrastOnWhite(t *testing.T) {
	for hex, want := range map[string]bool{
		"#767676": true, // 4.54:1, the lightest gray passing AA
		"#777777": false,
		"#000":    true,
		"#7a1a4a": true,
		"#a0a0d0": false,
	} {
		ratio, ok := contrastOnWhite(hex)
		if !ok {
			t.Fatalf("%s: not parsed", hex)
		}
		if got := ratio >= 4.5; got != want {
			t.Errorf("%s: ratio %.2f, passes AA = %v, want %v", hex, ratio, got, want)
		}
	}
	if _, ok := contrastOnWhite("teal"); ok {
		t.Error("named color parsed as hex")
	}
}

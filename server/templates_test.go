package main

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestTemplateFile(t *testing.T) {
	s := &server{templatePath: "../template/resume.typ"}
	for id, want := range map[string][2]string{
		"":      {"../template/resume.typ", "../template/cover-letter.typ"},
		"julia": {"../template/julia.typ", "../template/julia-letter.typ"},
	} {
		if got, err := s.templateFile(id); err != nil || got != want[0] {
			t.Errorf("templateFile(%q) = %q, %v; want %q", id, got, err, want[0])
		}
		if got, err := s.letterTemplateFile(id); err != nil || got != want[1] {
			t.Errorf("letterTemplateFile(%q) = %q, %v; want %q", id, got, err, want[1])
		}
	}
	if _, err := s.templateFile("x"); err == nil || err.Error() != `unknown template "x"` {
		t.Errorf("templateFile(x) error = %v", err)
	}
}

// A letter renders with its resume's template's letter template: Julia's
// letterhead repeats the resume's contacts, Classic's doesn't.
func TestLetterTemplate(t *testing.T) {
	l := newLibrary(t)
	doc := func(template string) string {
		return `{"basics": {"name": "Ada Lovelace", "headline": [], "contacts": [{"text": "ada@example.com", "url": "mailto:ada@example.com"}]},
			"style": {"accent": "#7a1a4a", "font": "Quicksand", "fontSize": 9, "margin": 0.5, "template": "` + template + `"},
			"letter": ` + testLetter + `}`
	}
	for template, contacts := range map[string]bool{"": false, "julia": true} {
		var got struct{ Text string }
		decodeInto(t, l.call(t, "POST", "/api/letter/preview", doc(template), 200), &got)
		if !strings.Contains(got.Text, "I would love to work at Acme.") || strings.Contains(got.Text, "ada@example.com") != contacts {
			t.Errorf("template %q letter text:\n%s", template, got.Text)
		}
	}
	var got struct{ Error string }
	decodeInto(t, l.call(t, "POST", "/api/letter/pdf", doc("x"), 400), &got)
	if got.Error != `unknown template "x"` {
		t.Fatalf("error = %q", got.Error)
	}
}

func TestPreviewUnknownTemplate(t *testing.T) {
	doc := strings.Replace(exportDoc, `"style": {`, `"style": {"template": "x", `, 1)
	var got struct{ Error string }
	decodeInto(t, newLibrary(t).call(t, "POST", "/api/preview", doc, 400), &got)
	if got.Error != `unknown template "x"` {
		t.Fatalf("error = %q", got.Error)
	}
}

func TestListTemplates(t *testing.T) {
	var list []TemplateInfo
	decodeInto(t, newLibrary(t).call(t, "GET", "/api/templates", "", 200), &list)
	if len(list) != 2 || list[0].ID != "" || list[1].ID != "julia" {
		t.Fatalf("templates = %+v", list)
	}
	if classic := list[0]; classic.Name != "Classic" || len(classic.Icons) != 0 || strings.Join(classic.Features, ",") != "markers,uppercaseTitles" {
		t.Errorf("classic = %+v", classic)
	}
	julia := list[1]
	b, err := os.ReadFile("../template/icons/julia/icons.json")
	if err != nil {
		t.Fatal(err)
	}
	var set []any
	if err := json.Unmarshal(b, &set); err != nil {
		t.Fatal(err)
	}
	if strings.Join(julia.Features, ",") != "entryIcons,legend,compactEntries,entryBrackets" || len(julia.Icons) != len(set) {
		t.Fatalf("julia = %d icons (want %d), features %q", len(julia.Icons), len(set), julia.Features)
	}
	if ic := julia.Icons[0]; ic.Name != "Food access" || ic.Color != "#d36c1c" || !strings.HasPrefix(ic.SVG, "<svg ") || !strings.HasSuffix(ic.SVG, "</svg>") {
		t.Errorf("first icon = %+v", ic)
	}
}

// The fonts in template/fonts render as if installed, in the PDF and the
// click-to-edit map alike.
func TestBundledFonts(t *testing.T) {
	doc := strings.Replace(exportDoc, `"font": "Helvetica"`, `"font": "Quicksand", "template": "julia"`, 1)
	out, err := renderResume(context.Background(), "../template/julia.typ", []byte(doc), true)
	if err != nil {
		t.Fatal(err)
	}
	defer out.Close()
	for _, w := range out.Warnings {
		if strings.Contains(w, "unknown font") {
			t.Errorf("warning: %s", w)
		}
	}
	fonts, err := exec.Command("pdffonts", out.PDFPath).Output()
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Quicksand", "SourceSans3"} {
		if !strings.Contains(string(fonts), want) {
			t.Errorf("PDF fonts lack %s:\n%s", want, fonts)
		}
	}
	if len(out.Fields) == 0 {
		t.Error("no click-to-edit fields")
	}
}

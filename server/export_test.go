package main

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"io"
	"os/exec"
	"strings"
	"testing"
)

const exportDoc = `{
  "basics": {"name": "Ada Lovelace", "headline": ["Analyst"],
             "contacts": [{"text": "ada@example.com", "url": "mailto:ada@example.com"}, {"text": "London", "url": ""}]},
  "sections": [
    {"kind": "text", "title": "Profile", "paragraphs": [
      "~8 years of **bold** and *italic* work, see [the map](https://x.y/a_b) <now>",
      "- not a list",
      "1. not ordered\n# not a heading"
    ]},
    {"kind": "entries", "title": "Work", "uppercaseTitles": true, "items": [
      {"type": "label", "text": "Earlier:"},
      {"type": "entry", "title": "Engineer", "date": "JAN 2020 – PRESENT", "subtitle": "*Engine* Co",
       "location": "London", "bullets": ["Built snake_case tools", ""]}
    ]},
    {"kind": "columns", "title": "Skills", "variant": "table",
     "columns": [{"title": "Languages", "lines": ["Go", "C#"]}]},
    {"kind": "columns", "title": "", "variant": "sections",
     "columns": [{"title": "Portfolio", "lines": ["[site](https://ada.dev)"]}]}
  ],
  "style": {"accent": "#7a1a4a", "font": "Helvetica", "fontSize": 9.5, "margin": 0.6,
            "leader": "dotted", "bullet": "diamonds", "paragraphMarker": "chevron"}
}`

func parseDoc(t *testing.T, s string) Doc {
	t.Helper()
	var doc Doc
	if err := json.Unmarshal([]byte(s), &doc); err != nil {
		t.Fatal(err)
	}
	return doc
}

func TestMarkdownResume(t *testing.T) {
	md := string(markdownResume(parseDoc(t, exportDoc)))
	for _, want := range []string{
		"# Ada Lovelace\n",
		"[ada@example.com](mailto:ada@example.com) | London\n",
		// Inline markup survives; characters Markdown would read as markup are escaped.
		`\~8 years of **bold** and *italic* work, see [the map](https://x.y/a_b) \<now\>` + "\n",
		`\- not a list` + "\n",
		// A typed line break stays a line break without starting a heading.
		`1\. not ordered\` + "\n" + `\# not a heading` + "\n",
		"**Earlier:**\n",
		"### Engineer | JAN 2020 – PRESENT\n",
		"*Engine* Co | London\n",
		"- Built snake\\_case tools\n\n",
		"**Languages**\n\n- Go\n- C#\n",
		"## Portfolio\n\n- [site](https://ada.dev)\n",
	} {
		if !strings.Contains(md, want) {
			t.Errorf("markdown missing %q in:\n%s", want, md)
		}
	}
}

func TestDocxResume(t *testing.T) {
	doc := parseDoc(t, exportDoc)
	b, err := docxResume(doc)
	if err != nil {
		t.Fatal(err)
	}
	z, err := zip.NewReader(bytes.NewReader(b), int64(len(b)))
	if err != nil {
		t.Fatal(err)
	}
	parts := map[string]string{}
	for _, f := range z.File {
		rc, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}
		body, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			t.Fatal(err)
		}
		parts[f.Name] = string(body)
		// Word refuses files with malformed XML.
		dec := xml.NewDecoder(bytes.NewReader(body))
		for {
			if _, err := dec.Token(); err == io.EOF {
				break
			} else if err != nil {
				t.Fatalf("%s: %v", f.Name, err)
			}
		}
	}
	for _, name := range []string{"[Content_Types].xml", "_rels/.rels", "word/document.xml", "word/styles.xml", "word/numbering.xml", "word/_rels/document.xml.rels"} {
		if _, ok := parts[name]; !ok {
			t.Errorf("missing part %s", name)
		}
	}

	// Every piece of content reaches the document text, markup removed.
	text := docxText(t, parts["word/document.xml"])
	for _, p := range contentPieces(doc) {
		if !strings.Contains(normalize(text), normalize(p)) {
			t.Errorf("document text missing %q", p)
		}
	}
	if strings.Contains(text, "**") || strings.Contains(text, "](") {
		t.Errorf("inline markup leaked into document text:\n%s", text)
	}
	for _, url := range []string{"mailto:ada@example.com", "https://x.y/a_b", "https://ada.dev"} {
		if !strings.Contains(parts["word/_rels/document.xml.rels"], `Target="`+url+`"`) {
			t.Errorf("no hyperlink relationship for %s", url)
		}
	}
}

// TestPDFMetadata checks what viewers and applicant-tracking systems read
// without opening the page: name as title and author, the headline as the
// subject, and the skills grid (table variant only) as keywords, markup removed.
func TestPDFMetadata(t *testing.T) {
	for _, tool := range []string{"typst", "pdfinfo"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skip(tool + " not installed")
		}
	}
	doc := strings.NewReplacer(
		`"headline": ["Analyst"]`, `"headline": ["Analyst", "", "London"]`,
		`["Go", "C#"]`, `["**Go** (*gRPC*, [Cobra](https://cobra.dev))", "C#; Go"]`,
	).Replace(exportDoc)
	out, err := render(context.Background(), "../template/resume.typ", []byte(doc), false)
	if err != nil {
		t.Fatal(err)
	}
	defer out.Close()
	b, err := exec.Command("pdfinfo", out.PDFPath).Output()
	if err != nil {
		t.Fatal(err)
	}
	info := map[string]string{}
	for _, line := range strings.Split(string(b), "\n") {
		if k, v, ok := strings.Cut(line, ":"); ok {
			info[k] = strings.TrimSpace(v)
		}
	}
	for k, want := range map[string]string{
		"Title":    "Ada Lovelace – Resume",
		"Author":   "Ada Lovelace",
		"Subject":  "Analyst · London",
		"Keywords": "Go, gRPC, Cobra, C#",
	} {
		if info[k] != want {
			t.Errorf("%s = %q, want %q", k, info[k], want)
		}
	}
}

func TestAttachedJSON(t *testing.T) {
	got, err := attachedJSON([]byte(`{
  "basics": {"name": "Ada", "headline": ["Analyst", " "],
             "contacts": [{"id": "c1", "text": "ada@example.com", "url": ""}, {"id": "c2", "text": "", "url": "https://x.y"}]},
  "sections": [
    {"id": "s1", "kind": "entries", "title": "Work", "uppercaseTitles": true, "items": [
      {"id": "e1", "type": "entry", "title": "Engineer", "date": "2020", "subtitle": "", "location": "", "bullets": ["**Built** it", ""]},
      {"id": "l1", "type": "label", "text": "Earlier:"}]},
    {"id": "s2", "kind": "columns", "title": "", "variant": "sections",
     "columns": [{"id": "k1", "title": "Skills", "lines": ["Go", ""]}]}
  ],
  "style": {"accent": "#000000"}
}`))
	if err != nil {
		t.Fatal(err)
	}
	var compact bytes.Buffer
	if err := json.Compact(&compact, got); err != nil {
		t.Fatal(err)
	}
	want := `{"basics":{"name":"Ada","headline":["Analyst"],"contacts":[{"text":"ada@example.com"}]},` +
		`"sections":[{"kind":"entries","title":"Work","items":[{"type":"entry","title":"Engineer","date":"2020","bullets":["**Built** it"]},{"type":"label","text":"Earlier:"}]},` +
		`{"kind":"columns","columns":[{"title":"Skills","lines":["Go"]}]}]}`
	if compact.String() != want {
		t.Errorf("attached JSON\n got %s\nwant %s", compact.String(), want)
	}
}

// docxText returns the text of document.xml, one line per paragraph, with
// tabs and line breaks as spaces.
func docxText(t *testing.T, documentXML string) string {
	t.Helper()
	var b strings.Builder
	dec := xml.NewDecoder(strings.NewReader(documentXML))
	inText := false
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			return b.String()
		} else if err != nil {
			t.Fatal(err)
		}
		switch tok := tok.(type) {
		case xml.StartElement:
			switch tok.Name.Local {
			case "t":
				inText = true
			case "tab", "br":
				b.WriteString(" ")
			}
		case xml.EndElement:
			switch tok.Name.Local {
			case "t":
				inText = false
			case "p":
				b.WriteString("\n")
			}
		case xml.CharData:
			if inText {
				b.Write(tok)
			}
		}
	}
}

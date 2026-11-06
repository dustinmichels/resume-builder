package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

const testLetter = `{"title":"Cover letter","header":["Somerville, MA"],"greeting":"Dear Acme,",
"blocks":[{"id":"b1","kind":"paragraph","text":"I would love to work at **Acme**."},{"id":"b2","kind":"bullets","items":["One","Two"]}],
"closing":"Sincerely,","style":{"font":"Libertinus Serif","fontSize":12}}`

func TestCoverLetters(t *testing.T) {
	if _, err := exec.LookPath("typst"); err != nil {
		t.Skip("typst not installed")
	}
	l := newLibrary(t)
	dir := l.choose(t)
	ldir := filepath.Join(dir, "Resume", lettersDir)
	create := func(body string) string {
		t.Helper()
		var got struct{ ID string }
		decodeInto(t, l.call(t, "POST", "/api/resumes/Resume/letters", body, 201), &got)
		return got.ID
	}
	sync := func() []string {
		t.Helper()
		var res syncResult
		decodeInto(t, l.call(t, "POST", "/api/resumes/Resume/sync", "", 200), &res)
		if len(res.Failed) > 0 {
			t.Fatalf("sync failed: %+v", res.Failed)
		}
		slices.Sort(res.Written)
		return res.Written
	}
	exists := func(p string) bool { _, err := os.Stat(p); return err == nil }

	if id := create(`{"name":"Acme","letter":` + testLetter + `}`); id != "Acme" {
		t.Fatalf("created %q", id)
	}
	if id := create(`{"name":"Acme","letter":` + testLetter + `}`); id != "Acme 2" {
		t.Fatalf("second letter named %q, want a unique name", id)
	}
	l.call(t, "POST", "/api/resumes/Resume/letters", `{"name":"Bad","letter":{"title":"x"}}`, 400)

	// Each letter gets Markdown, Word and PDF copies; its PDF shows the
	// resume's name, so editing the resume redraws it.
	want := []string{
		"cover-letters/Acme 2.docx", "cover-letters/Acme 2.md", "cover-letters/Acme 2.pdf",
		"cover-letters/Acme.docx", "cover-letters/Acme.md", "cover-letters/Acme.pdf",
		"resume.docx", "resume.md", "resume.pdf",
	}
	if got := sync(); !slices.Equal(got, want) {
		t.Fatalf("first sync wrote %q, want %q", got, want)
	}
	if md, _ := os.ReadFile(filepath.Join(ldir, "Acme.md")); !strings.Contains(string(md), "I would love to work at **Acme**.") {
		t.Fatalf("letter markdown:\n%s", md)
	}
	l.call(t, "PUT", "/api/resumes/Resume", exportDoc, 204)
	if got := sync(); !slices.Contains(got, "cover-letters/Acme.pdf") {
		t.Fatalf("sync after a resume edit wrote %q, without the letter PDFs", got)
	}

	// Renaming carries the copies along; a taken name is refused.
	l.call(t, "PATCH", "/api/resumes/Resume/letters/Acme", `{"name":"Acme 2"}`, 409)
	l.call(t, "PATCH", "/api/resumes/Resume/letters/Acme", `{"name":"Acme Planner"}`, 200)
	for _, ext := range []string{".json", ".md", ".docx", ".pdf"} {
		if !exists(filepath.Join(ldir, "Acme Planner"+ext)) || exists(filepath.Join(ldir, "Acme"+ext)) {
			t.Fatalf("%s not renamed with its letter", ext)
		}
	}

	// A copy of the resume gets its letters.
	copied := l.create(t, map[string]string{"name": "Copy", "from": "Resume"})
	l.call(t, "GET", "/api/resumes/"+copied+"/letters/Acme%20Planner", "", 200)

	// Deleting moves the letter to the trash and its copies away; a save
	// still pending for it in the editor can't bring it back.
	l.call(t, "DELETE", "/api/resumes/Resume/letters/Acme%20Planner", "", 204)
	if exists(filepath.Join(ldir, "Acme Planner.pdf")) || exists(filepath.Join(ldir, "Acme Planner.md")) || exists(filepath.Join(ldir, "Acme Planner.docx")) {
		t.Fatal("exports survived deleting their letter")
	}
	trashed, _ := filepath.Glob(filepath.Join(dir, trashDir, "Resume – Acme Planner *.json"))
	if len(trashed) != 1 {
		t.Fatalf("trash holds %q", trashed)
	}
	l.call(t, "PUT", "/api/resumes/Resume/letters/Acme%20Planner", testLetter, 404)
	l.call(t, "GET", "/api/resumes/Resume/letters/..%2Fresume", "", 400)
}

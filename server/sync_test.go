package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestSyncFiles(t *testing.T) {
	if _, err := exec.LookPath("typst"); err != nil {
		t.Skip("typst not installed")
	}
	l := newLibrary(t)
	rdir := filepath.Join(l.choose(t), "Resume")
	l.call(t, "POST", "/api/resumes/Resume/letters", `{"name":"Acme","letter":`+testLetter+`}`, 201)
	l.call(t, "PUT", "/api/resumes/Resume/contact", `{"email":"me@example.com"}`, 204)

	// Saving a version snapshots the resume, its side files and its letters,
	// and writes the exports of the resume and letters inside it.
	var v struct {
		version
		Failed []syncFailure
	}
	decodeInto(t, l.call(t, "POST", "/api/resumes/Resume/versions", `{"label":"Sent"}`, 201), &v)
	if len(v.Failed) != 0 {
		t.Fatalf("version exports failed: %+v", v.Failed)
	}
	vdir := filepath.Join(rdir, versionsDir, v.ID)
	for name, prefix := range map[string]string{
		resumeFile: "{", sideFiles["contact"]: "{", "cover-letters/Acme.json": "{",
		"resume.md": "# ", "resume.pdf": "%PDF-", "resume.docx": "PK",
		"cover-letters/Acme.md": "", "cover-letters/Acme.pdf": "%PDF-", "cover-letters/Acme.docx": "PK",
	} {
		if b, err := os.ReadFile(filepath.Join(vdir, name)); err != nil || !strings.HasPrefix(string(b), prefix) {
			t.Fatalf("version %s: %v", name, err)
		}
	}
	l.call(t, "PUT", "/api/resumes/Resume/letters/Acme", strings.Replace(testLetter, "Dear Acme", "Dear Beta", 1), 204)
	if md, _ := os.ReadFile(filepath.Join(vdir, "cover-letters/Acme.md")); !strings.Contains(string(md), "Dear Acme") {
		t.Fatalf("editing the letter changed its version:\n%s", md)
	}

	broken := filepath.Join(rdir, versionsDir, "broken")
	if err := os.Mkdir(broken, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(broken, resumeFile), []byte(`{"basics":{}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	sync := func() syncResult {
		t.Helper()
		var res syncResult
		decodeInto(t, l.call(t, "POST", "/api/resumes/Resume/sync", "", 200), &res)
		slices.Sort(res.Written)
		return res
	}

	res := sync()
	want := []string{"cover-letters/Acme.docx", "cover-letters/Acme.md", "cover-letters/Acme.pdf", "resume.docx", "resume.md", "resume.pdf"}
	if !slices.Equal(res.Written, want) {
		t.Fatalf("first sync wrote %q, want %q", res.Written, want)
	}
	if len(res.Failed) != 1 || res.Failed[0].File != "versions/broken/resume.json" {
		t.Fatalf("failures = %+v", res.Failed)
	}

	if res := sync(); len(res.Written) != 0 {
		t.Fatalf("up-to-date sync wrote %q", res.Written)
	}

	// A version's letters show the version's resume, so editing the current
	// resume leaves the version's exports alone.
	l.call(t, "PUT", "/api/resumes/Resume", exportDoc, 204)
	if res := sync(); !slices.Equal(res.Written, want) {
		t.Fatalf("sync after edit wrote %q, want %q", res.Written, want)
	}

	l.call(t, "DELETE", "/api/resumes/Resume/versions/"+strings.ReplaceAll(v.ID, " ", "%20"), "", 204)
	if _, err := os.Stat(filepath.Join(rdir, versionsDir, v.ID)); !os.IsNotExist(err) {
		t.Fatalf("version folder survived deleting its version: %v", err)
	}
}

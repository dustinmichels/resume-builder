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
	var v version
	decodeInto(t, l.call(t, "POST", "/api/resumes/Resume/versions", `{"label":"Sent"}`, 201), &v)
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

	vbase := "versions/" + v.ID + "/resume"
	res := sync()
	if want := []string{"resume.docx", "resume.md", "resume.pdf", vbase + ".docx", vbase + ".md", vbase + ".pdf"}; !slices.Equal(res.Written, want) {
		t.Fatalf("first sync wrote %q, want %q", res.Written, want)
	}
	if len(res.Failed) != 1 || res.Failed[0].File != "versions/broken/resume.json" {
		t.Fatalf("failures = %+v", res.Failed)
	}
	md, err := os.ReadFile(filepath.Join(rdir, versionsDir, v.ID, "resume.md"))
	if err != nil || !strings.HasPrefix(string(md), "# ") {
		t.Fatalf("version markdown: %v\n%s", err, md)
	}
	pdf, err := os.ReadFile(filepath.Join(rdir, versionsDir, v.ID, "resume.pdf"))
	if err != nil || !strings.HasPrefix(string(pdf), "%PDF-") {
		t.Fatalf("version pdf: %v", err)
	}
	docx, err := os.ReadFile(filepath.Join(rdir, versionsDir, v.ID, "resume.docx"))
	if err != nil || !strings.HasPrefix(string(docx), "PK") {
		t.Fatalf("version docx: %v", err)
	}

	if res := sync(); len(res.Written) != 0 {
		t.Fatalf("up-to-date sync wrote %q", res.Written)
	}

	l.call(t, "PUT", "/api/resumes/Resume", exportDoc, 204)
	if res := sync(); !slices.Equal(res.Written, []string{"resume.docx", "resume.md", "resume.pdf"}) {
		t.Fatalf("sync after edit wrote %q", res.Written)
	}

	l.call(t, "DELETE", "/api/resumes/Resume/versions/"+strings.ReplaceAll(v.ID, " ", "%20"), "", 204)
	if _, err := os.Stat(filepath.Join(rdir, versionsDir, v.ID)); !os.IsNotExist(err) {
		t.Fatalf("version folder survived deleting its version: %v", err)
	}
}

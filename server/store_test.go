package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type library struct {
	h          http.Handler
	configPath string
	legacyPath string
}

// newLibrary builds the real mux over a temp config file, with no data
// folder chosen and no legacy resume.
func newLibrary(t *testing.T) *library {
	t.Helper()
	tmp := t.TempDir()
	l := &library{configPath: filepath.Join(tmp, "config", "config.json"), legacyPath: filepath.Join(tmp, "legacy.json")}
	s, err := newServer("../template/resume.typ", l.configPath, l.legacyPath)
	if err != nil {
		t.Fatal(err)
	}
	l.h = newMux(s, nil)
	return l
}

// call sends a request and fails the test unless it answers with status want.
func (l *library) call(t *testing.T, method, target, body string, want int) []byte {
	t.Helper()
	var rd io.Reader
	if body != "" {
		rd = strings.NewReader(body)
	}
	rec := httptest.NewRecorder()
	l.h.ServeHTTP(rec, httptest.NewRequest(method, target, rd))
	if rec.Code != want {
		t.Fatalf("%s %s: status %d, want %d: %s", method, target, rec.Code, want, rec.Body)
	}
	return rec.Body.Bytes()
}

// choose picks a fresh data folder and returns its path.
func (l *library) choose(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "Resumes")
	var got settings
	decodeInto(t, l.call(t, "PUT", "/api/settings", jsonBody(t, map[string]string{"dataDir": dir}), 200), &got)
	if got.DataDir != dir {
		t.Fatalf("dataDir = %q, want %q", got.DataDir, dir)
	}
	return dir
}

func (l *library) ids(t *testing.T) []string {
	t.Helper()
	var list []summary
	decodeInto(t, l.call(t, "GET", "/api/resumes", "", 200), &list)
	var ids []string
	for _, r := range list {
		ids = append(ids, r.ID)
	}
	return ids
}

func (l *library) create(t *testing.T, body map[string]string) string {
	t.Helper()
	var got struct{ ID string }
	decodeInto(t, l.call(t, "POST", "/api/resumes", jsonBody(t, body), 201), &got)
	return got.ID
}

func (l *library) versions(t *testing.T, id string) []version {
	t.Helper()
	var list []version
	decodeInto(t, l.call(t, "GET", "/api/resumes/"+id+"/versions", "", 200), &list)
	return list
}

func jsonBody(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func decodeInto(t *testing.T, b []byte, v any) {
	t.Helper()
	if err := json.Unmarshal(b, v); err != nil {
		t.Fatalf("%v: %s", err, b)
	}
}

func TestLibraryNeedsFolder(t *testing.T) {
	l := newLibrary(t)
	var got settings
	decodeInto(t, l.call(t, "GET", "/api/settings", "", 200), &got)
	if got.DataDir != "" || got.DefaultDataDir != defaultDataDir() {
		t.Fatalf("settings = %+v", got)
	}
	l.call(t, "GET", "/api/resumes", "", 409)
	l.call(t, "POST", "/api/resumes", `{"name":"x"}`, 409)
	l.call(t, "GET", "/api/resumes/Resume", "", 409)
	// The single-resume routes are gone.
	l.call(t, "GET", "/api/resume", "", 404)
}

func TestSettingsSeedsStarter(t *testing.T) {
	l := newLibrary(t)
	dir := l.choose(t)

	if ids := l.ids(t); len(ids) != 1 || ids[0] != "Resume" {
		t.Fatalf("resumes = %v", ids)
	}
	got := l.call(t, "GET", "/api/resumes/Resume", "", 200)
	if !bytes.Equal(got, starterJSON) {
		t.Fatalf("seeded resume is not the starter:\n%s", got)
	}
	// The starter passes the same validation as a save.
	l.call(t, "PUT", "/api/resumes/Resume", string(starterJSON), 204)

	// The choice is remembered for the next run.
	s, err := newServer("", l.configPath, l.legacyPath)
	if err != nil || s.dataDir != dir {
		t.Fatalf("reloaded dataDir = %q, %v; want %q", s.dataDir, err, dir)
	}
	// Choosing a folder that already has resumes adds nothing.
	l.call(t, "PUT", "/api/settings", jsonBody(t, map[string]string{"dataDir": dir}), 200)
	if ids := l.ids(t); len(ids) != 1 {
		t.Fatalf("resumes after re-choosing = %v", ids)
	}

	if p, err := expandDir("~/resume-builder"); err != nil || p != defaultDataDir() {
		t.Fatalf("expandDir(~/resume-builder) = %q, %v", p, err)
	}
}

func TestSettingsImportsLegacyResume(t *testing.T) {
	l := newLibrary(t)
	if err := os.WriteFile(l.legacyPath, []byte(exportDoc), 0o644); err != nil {
		t.Fatal(err)
	}
	l.choose(t)
	if got := l.call(t, "GET", "/api/resumes/Resume", "", 200); string(got) != exportDoc {
		t.Fatalf("imported resume:\n%s", got)
	}
}

func TestCreateResumes(t *testing.T) {
	l := newLibrary(t)
	l.choose(t)

	for _, c := range []struct{ name, want string }{
		{"My CV", "My CV"},
		{"My CV", "My CV 2"},
		{"", "Resume 2"}, // the seeded resume is "Resume"
		{" ../Bad:Name\t*x ", "BadName x"},
	} {
		if got := l.create(t, map[string]string{"name": c.name}); got != c.want {
			t.Errorf("create %q = %q, want %q", c.name, got, c.want)
		}
	}
	if got := l.call(t, "GET", "/api/resumes/My%20CV", "", 200); !bytes.Equal(got, starterJSON) {
		t.Fatalf("blank resume is not the starter:\n%s", got)
	}

	l.call(t, "PUT", "/api/resumes/My%20CV", exportDoc, 204)
	saved := l.call(t, "GET", "/api/resumes/My%20CV", "", 200)
	if copyID := l.create(t, map[string]string{"name": "Copy", "from": "My CV"}); copyID != "Copy" {
		t.Fatalf("duplicate id = %q", copyID)
	}
	if got := l.call(t, "GET", "/api/resumes/Copy", "", 200); !bytes.Equal(got, saved) {
		t.Fatalf("duplicate differs:\n%s", got)
	}

	var v version
	decodeInto(t, l.call(t, "POST", "/api/resumes/My%20CV/versions", `{"label":"sent"}`, 201), &v)
	l.call(t, "PUT", "/api/resumes/My%20CV", string(starterJSON), 204)
	if id := l.create(t, map[string]string{"name": "Copy", "from": "My CV", "version": v.ID}); id != "Copy 2" {
		t.Fatalf("from-version id = %q", id)
	}
	if got := l.call(t, "GET", "/api/resumes/Copy%202", "", 200); !bytes.Equal(got, saved) {
		t.Fatalf("copy of version differs:\n%s", got)
	}

	l.call(t, "POST", "/api/resumes", `{"name":"x","from":"Nope"}`, 404)
	l.call(t, "POST", "/api/resumes", `{"name":"x","version":"`+v.ID+`"}`, 400)
}

func TestRenameAndDelete(t *testing.T) {
	l := newLibrary(t)
	dir := l.choose(t)
	l.create(t, map[string]string{"name": "A"})
	l.create(t, map[string]string{"name": "B"})

	l.call(t, "PATCH", "/api/resumes/A", `{"name":"B"}`, 409)
	l.call(t, "PATCH", "/api/resumes/A", `{"name":" "}`, 400)
	var got struct{ ID string }
	decodeInto(t, l.call(t, "PATCH", "/api/resumes/A", `{"name":"C"}`, 200), &got)
	if got.ID != "C" {
		t.Fatalf("renamed id = %q", got.ID)
	}
	l.call(t, "GET", "/api/resumes/A", "", 404)
	l.call(t, "GET", "/api/resumes/C", "", 200)

	l.call(t, "DELETE", "/api/resumes/C", "", 204)
	l.call(t, "GET", "/api/resumes/C", "", 404)
	trash, err := os.ReadDir(filepath.Join(dir, trashDir))
	if err != nil || len(trash) != 1 || !strings.HasPrefix(trash[0].Name(), "C ") {
		t.Fatalf("trash = %v, %v", trash, err)
	}
	if _, err := os.Stat(filepath.Join(dir, trashDir, trash[0].Name(), resumeFile)); err != nil {
		t.Fatal(err)
	}
	for _, id := range l.ids(t) {
		if id != "Resume" && id != "B" {
			t.Fatalf("listed %q", id)
		}
	}
}

func TestVersions(t *testing.T) {
	l := newLibrary(t)
	l.choose(t)

	var first, second version
	decodeInto(t, l.call(t, "POST", "/api/resumes/Resume/versions", `{"label":"first"}`, 201), &first)
	l.call(t, "PUT", "/api/resumes/Resume", exportDoc, 204)
	edited := l.call(t, "GET", "/api/resumes/Resume", "", 200)
	decodeInto(t, l.call(t, "POST", "/api/resumes/Resume/versions", `{"label":"a/b"}`, 201), &second)
	if first.Label != "first" || second.Label != "ab" || first.ID == second.ID {
		t.Fatalf("versions = %+v, %+v", first, second)
	}
	if list := l.versions(t, "Resume"); len(list) != 2 || list[0].ID != second.ID || list[1].ID != first.ID {
		t.Fatalf("versions not newest first: %+v", list)
	}
	vpath := "/api/resumes/Resume/versions/" + strings.ReplaceAll(first.ID, " ", "%20")
	if got := l.call(t, "GET", vpath, "", 200); !bytes.Equal(got, starterJSON) {
		t.Fatalf("first version:\n%s", got)
	}

	if got := l.call(t, "POST", vpath+"/restore", "", 200); !bytes.Equal(got, starterJSON) {
		t.Fatalf("restore returned:\n%s", got)
	}
	if got := l.call(t, "GET", "/api/resumes/Resume", "", 200); !bytes.Equal(got, starterJSON) {
		t.Fatalf("restored resume:\n%s", got)
	}
	list := l.versions(t, "Resume")
	if len(list) != 3 || list[0].Label != "Before restoring first" {
		t.Fatalf("versions after restore: %+v", list)
	}
	backup := "/api/resumes/Resume/versions/" + strings.ReplaceAll(list[0].ID, " ", "%20")
	if got := l.call(t, "GET", backup, "", 200); !bytes.Equal(got, edited) {
		t.Fatalf("pre-restore snapshot:\n%s", got)
	}

	l.call(t, "DELETE", vpath, "", 204)
	l.call(t, "GET", vpath, "", 404)
	if list := l.versions(t, "Resume"); len(list) != 2 {
		t.Fatalf("versions after delete: %+v", list)
	}
}

func TestSideFiles(t *testing.T) {
	l := newLibrary(t)
	dir := l.choose(t)

	for path := range sideFiles {
		if got := l.call(t, "GET", "/api/resumes/Resume/"+path, "", 200); string(got) != "{}\n" {
			t.Fatalf("unsaved %s = %q", path, got)
		}
		for _, bad := range []string{"", "[]", "null", `"x"`, "{"} {
			l.call(t, "PUT", "/api/resumes/Resume/"+path, bad, 400)
		}
		l.call(t, "PUT", "/api/resumes/Nope/"+path, `{}`, 404)
	}
	if _, err := os.Stat(filepath.Join(dir, "Resume", sideFiles["contact"])); err == nil {
		t.Fatal("a rejected save wrote contact.json")
	}

	contact := `{"email":"me@example.com"}`
	l.call(t, "PUT", "/api/resumes/Resume/contact", contact, 204)
	if _, err := os.Stat(filepath.Join(dir, "Resume", additionalDir, "contact.json")); err != nil {
		t.Fatal(err)
	}
	if got := l.call(t, "GET", "/api/resumes/Resume/contact", "", 200); string(got) != "{\n  \"email\": \"me@example.com\"\n}\n" {
		t.Fatalf("contact = %q", got)
	}
	if got := l.call(t, "GET", "/api/resumes/Resume", "", 200); !bytes.Equal(got, starterJSON) {
		t.Fatalf("saving contact changed the resume:\n%s", got)
	}

	// Duplicates take the side files along; a copy of a version gets the
	// version's, or the current ones for a version without any.
	var v version
	decodeInto(t, l.call(t, "POST", "/api/resumes/Resume/versions", `{"label":"v"}`, 201), &v)
	if _, err := os.Stat(filepath.Join(dir, "Resume", versionsDir, v.ID, sideFiles["contact"])); err != nil {
		t.Fatalf("version lacks contact.json: %v", err)
	}
	legacy := filepath.Join(dir, "Resume", versionsDir, "20200101-000000 legacy")
	if err := os.MkdirAll(legacy, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(legacy, resumeFile), starterJSON, 0o644); err != nil {
		t.Fatal(err)
	}
	l.call(t, "PUT", "/api/resumes/Resume/contact", `{"email":"new@example.com"}`, 204)
	for _, c := range []struct {
		body  map[string]string
		email string
	}{
		{map[string]string{"name": "Copy", "from": "Resume"}, "new@example.com"},
		{map[string]string{"name": "Old", "from": "Resume", "version": v.ID}, "me@example.com"},
		{map[string]string{"name": "Legacy", "from": "Resume", "version": "20200101-000000 legacy"}, "new@example.com"},
	} {
		id := l.create(t, c.body)
		if got := l.call(t, "GET", "/api/resumes/"+id+"/contact", "", 200); !strings.Contains(string(got), c.email) {
			t.Fatalf("%s contact = %q, want %s", id, got, c.email)
		}
		if got := l.call(t, "GET", "/api/resumes/"+id+"/supervisors", "", 200); string(got) != "{}\n" {
			t.Fatalf("%s supervisors = %q", id, got)
		}
	}
	if got := l.call(t, "GET", "/api/resumes/"+l.create(t, map[string]string{"name": "Blank"})+"/contact", "", 200); string(got) != "{}\n" {
		t.Fatalf("blank resume contact = %q", got)
	}
}

func TestRejectsBadIDs(t *testing.T) {
	l := newLibrary(t)
	dir := l.choose(t)
	l.create(t, map[string]string{"name": "Gone"})
	l.call(t, "DELETE", "/api/resumes/Gone", "", 204) // creates .trash

	for _, target := range []string{
		"/api/resumes/..%2F..",
		"/api/resumes/..%2F..%2Fetc",
		"/api/resumes/.trash",
		"/api/resumes/%2Fetc",
		"/api/resumes/a%5Cb",
		"/api/resumes/a%0Ab",
		"/api/resumes/Resume/versions/..%2F..%2Fresume",
		"/api/resumes/Resume/versions/.hidden",
	} {
		l.call(t, "GET", target, "", 400)
	}
	l.call(t, "DELETE", "/api/resumes/..%2FResumes", "", 400)
	l.call(t, "PATCH", "/api/resumes/..%2FResumes", `{"name":"x"}`, 400)
	l.call(t, "POST", "/api/resumes", `{"name":"x","from":"../Resumes"}`, 400)
	l.call(t, "GET", "/api/resumes/Nope", "", 404)
	l.call(t, "GET", "/api/resumes/Resume/versions/nope", "", 404)
	if _, err := os.Stat(filepath.Join(dir, "Resume", resumeFile)); err != nil {
		t.Fatal(err)
	}
}

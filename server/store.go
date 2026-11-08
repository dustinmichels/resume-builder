package main

// The resume library: a user-chosen data folder holding one subfolder per
// resume, each with its saved versions.
//
//	<dataDir>/<resume name>/resume.json
//	<dataDir>/<resume name>/additional/contact.json, supervisors.json   (side files, optional)
//	<dataDir>/<resume name>/cover-letters/<letter name>.json   (see letters.go)
//	<dataDir>/<resume name>/versions/<YYYYMMDD-HHMMSS>[ <label>]/resume.json, additional/…, cover-letters/…   (a snapshot of the above)
//	<dataDir>/<resume name>/[versions/<version id>/][cover-letters/]<name>.md, .docx, .pdf     (exports beside each .json, see sync.go)
//	<dataDir>/.trash/<resume name> <YYYYMMDD-HHMMSS>/
//	<dataDir>/.trash/<resume name> – <letter name> <YYYYMMDD-HHMMSS>.json

import (
	"bytes"
	"cmp"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

//go:embed starter.json
var starterJSON []byte

const (
	resumeFile    = "resume.json"
	versionsDir   = "versions"
	additionalDir = "additional"
	trashDir      = ".trash"
	stampLayout   = "20060102-150405"
	maxNameLen    = 80
)

// sideFiles are the documents kept in a resume's additional folder, by the
// URL name they are served under. They are never rendered; they move, go to
// the trash, get duplicated and are versioned with the resume.
var sideFiles = map[string]string{
	"contact":     filepath.Join(additionalDir, "contact.json"),
	"supervisors": filepath.Join(additionalDir, "supervisors.json"),
}

var errNoDataDir = errors.New("choose a data folder first")

// config is what the server remembers between runs.
type config struct {
	DataDir string `json:"dataDir"`
}

type settings struct {
	DataDir        string `json:"dataDir"`
	DefaultDataDir string `json:"defaultDataDir"`
}

// summary describes a saved resume or cover letter: its id (the name shown
// in the editor) and when it last changed.
type summary struct {
	ID       string    `json:"id"`
	Modified time.Time `json:"modified"`
}

// newestFirst sorts by modification time, newest first, then by id.
func newestFirst(list []summary) {
	slices.SortFunc(list, func(a, b summary) int {
		return cmp.Or(b.Modified.Compare(a.Modified), strings.Compare(a.ID, b.ID))
	})
}

type version struct {
	ID      string    `json:"id"`
	Label   string    `json:"label"`
	Created time.Time `json:"created"`
	seq     int       // orders versions saved within the same second
}

func defaultConfigPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "resume-builder", "config.json"), nil
}

func defaultDataDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "resume-builder"
	}
	return filepath.Join(home, "resume-builder")
}

func loadConfig(path string) (config, error) {
	var c config
	b, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return c, nil
	}
	if err != nil {
		return c, err
	}
	if err := json.Unmarshal(b, &c); err != nil {
		return c, fmt.Errorf("%s: %w", path, err)
	}
	return c, nil
}

func saveConfig(path string, c config) error {
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return writeFileAtomic(path, append(b, '\n'))
}

// expandDir resolves a user-typed folder: a leading ~ is the home folder and
// relative paths are taken from the working directory.
func expandDir(p string) (string, error) {
	p = strings.TrimSpace(p)
	if p == "" {
		return "", errors.New("the data folder is empty")
	}
	if p == "~" || strings.HasPrefix(p, "~/") || strings.HasPrefix(p, `~\`) {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		p = filepath.Join(home, p[1:])
	}
	return filepath.Abs(p)
}

// writeFileAtomic writes to a sibling temp file and renames it into place, so
// a crash never leaves a half-written file behind.
func writeFileAtomic(path string, data []byte) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+"-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}

// sanitize makes s safe as part of a file name: characters file systems
// reject are dropped, whitespace runs become one space, and the result is
// capped at maxNameLen characters.
func sanitize(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r == utf8.RuneError || strings.ContainsRune(`/\:*?"<>|`, r):
		case unicode.IsControl(r):
			b.WriteByte(' ')
		default:
			b.WriteRune(r)
		}
	}
	s = strings.Join(strings.Fields(b.String()), " ")
	if r := []rune(s); len(r) > maxNameLen {
		s = strings.TrimSpace(string(r[:maxNameLen]))
	}
	return s
}

// sanitizeName turns a requested resume name into a folder name.
func sanitizeName(s string) string {
	s = strings.TrimLeft(sanitize(s), ". ")
	if s == "" {
		return "Resume"
	}
	return s
}

// validID reports whether id, taken from a URL, names a plain visible entry
// of a folder: no separators, no control characters, no leading dot.
func validID(id string) bool {
	if id == "" || strings.HasPrefix(id, ".") || strings.ContainsAny(id, `/\`) {
		return false
	}
	return !strings.ContainsFunc(id, unicode.IsControl)
}

// childPath joins dir and the URL-supplied name+ext, refusing anything that
// is not an immediate child of dir.
func childPath(dir, name, ext string) (string, error) {
	p := filepath.Join(dir, name+ext)
	if !validID(name) || filepath.Dir(p) != filepath.Clean(dir) {
		return "", fmt.Errorf("invalid id %q", name)
	}
	return p, nil
}

func exists(path string) bool {
	_, err := os.Lstat(path)
	return !errors.Is(err, fs.ErrNotExist)
}

// uniqueName returns name, or "name 2", "name 3"… if dir already has
// name+ext.
func uniqueName(dir, name, ext string) string {
	for n, c := 2, name; ; n++ {
		if !exists(filepath.Join(dir, c+ext)) {
			return c
		}
		c = fmt.Sprintf("%s %d", name, n)
	}
}

func listResumes(dir string) ([]summary, error) {
	out := []summary{}
	entries, err := os.ReadDir(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return out, nil
	}
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".") || !e.IsDir() {
			continue
		}
		fi, err := os.Stat(filepath.Join(dir, e.Name(), resumeFile))
		if err != nil || !fi.Mode().IsRegular() {
			continue
		}
		out = append(out, summary{ID: e.Name(), Modified: fi.ModTime()})
	}
	newestFirst(out)
	return out, nil
}

// newFolder creates folder path holding files, keyed by their path in it
// (resume.json and any side files and cover letters).
func newFolder(path string, files map[string][]byte) error {
	if err := os.Mkdir(path, 0o755); err != nil {
		return err
	}
	for name, data := range files {
		p := filepath.Join(path, name)
		err := os.MkdirAll(filepath.Dir(p), 0o755)
		if err == nil {
			err = writeFileAtomic(p, data)
		}
		if err != nil {
			os.RemoveAll(path)
			return err
		}
	}
	return nil
}

// snapshot reads the documents in folder (a resume's, or one of its
// versions'): resume.json, the side files that exist and the cover letters,
// keyed by their path in folder.
func snapshot(folder string) (map[string][]byte, error) {
	letters, err := listLetters(folder)
	if err != nil {
		return nil, err
	}
	names := []string{resumeFile}
	for _, name := range sideFiles {
		names = append(names, name)
	}
	for _, l := range letters {
		names = append(names, filepath.Join(lettersDir, l.ID+".json"))
	}
	files := make(map[string][]byte, len(names))
	for _, name := range names {
		b, err := os.ReadFile(filepath.Join(folder, name))
		if errors.Is(err, fs.ErrNotExist) && name != resumeFile {
			continue // side files are written only once filled in
		}
		if err != nil {
			return nil, err
		}
		files[name] = b
	}
	return files, nil
}

// seed gives a data folder without resumes its first one: the pre-library
// data/resume.json when there is one, else the starter.
func (s *server) seed(dir string) error {
	list, err := listResumes(dir)
	if err != nil || len(list) > 0 {
		return err
	}
	data := starterJSON
	if s.legacyPath != "" {
		b, err := os.ReadFile(s.legacyPath)
		switch {
		case err == nil:
			data = b
		case !errors.Is(err, fs.ErrNotExist):
			return err
		}
	}
	return newFolder(filepath.Join(dir, uniqueName(dir, "Resume", "")), map[string][]byte{resumeFile: data})
}

// parseVersion reads a version id: "<YYYYMMDD-HHMMSS>[-N][ <label>]".
func parseVersion(id string) (version, bool) {
	stamp, label, _ := strings.Cut(id, " ")
	if len(stamp) < len(stampLayout) {
		return version{}, false
	}
	created, err := time.ParseInLocation(stampLayout, stamp[:len(stampLayout)], time.Local)
	if err != nil {
		return version{}, false
	}
	seq := 1
	if rest := stamp[len(stampLayout):]; rest != "" {
		n, err := strconv.Atoi(strings.TrimPrefix(rest, "-"))
		if !strings.HasPrefix(rest, "-") || err != nil || n < 2 {
			return version{}, false
		}
		seq = n
	}
	return version{ID: id, Label: label, Created: created, seq: seq}, true
}

// versionOf describes a version folder from its resume.json; folders not named
// by saveVersion fall back to their name as label and the file's modification time.
func versionOf(id string, fi fs.FileInfo) version {
	if v, ok := parseVersion(id); ok {
		return v
	}
	return version{ID: id, Label: id, Created: fi.ModTime(), seq: 1}
}

func listVersions(rdir string) ([]version, error) {
	out := []version{}
	entries, err := os.ReadDir(filepath.Join(rdir, versionsDir))
	if errors.Is(err, fs.ErrNotExist) {
		return out, nil
	}
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, ".") || !e.IsDir() {
			continue
		}
		fi, err := os.Stat(filepath.Join(rdir, versionsDir, name, resumeFile))
		if err != nil || !fi.Mode().IsRegular() {
			continue
		}
		out = append(out, versionOf(name, fi))
	}
	slices.SortFunc(out, func(a, b version) int {
		return cmp.Or(b.Created.Compare(a.Created), cmp.Compare(b.seq, a.seq), strings.Compare(b.ID, a.ID))
	})
	return out, nil
}

// saveVersion snapshots the saved documents in rdir (see snapshot) under
// label, in a folder of its own. It writes no exports (see exportVersion).
func saveVersion(rdir, label string, now time.Time) (version, error) {
	files, err := snapshot(rdir)
	if err != nil {
		return version{}, err
	}
	vdir := filepath.Join(rdir, versionsDir)
	if err := os.MkdirAll(vdir, 0o755); err != nil {
		return version{}, err
	}
	entries, err := os.ReadDir(vdir)
	if err != nil {
		return version{}, err
	}
	taken := func(stamp string) bool {
		return slices.ContainsFunc(entries, func(e fs.DirEntry) bool {
			return e.Name() == stamp || strings.HasPrefix(e.Name(), stamp+" ")
		})
	}
	base := now.Format(stampLayout)
	stamp := base
	for n := 2; taken(stamp); n++ {
		stamp = fmt.Sprintf("%s-%d", base, n)
	}
	id := stamp
	if label = sanitize(label); label != "" {
		id += " " + label
	}
	if err := newFolder(filepath.Join(vdir, id), files); err != nil {
		return version{}, err
	}
	v, _ := parseVersion(id)
	return v, nil
}

// lockDir locks the store, exclusively when write is set, and returns the
// data folder. With no folder chosen it answers 409 and leaves it unlocked.
func (s *server) lockDir(w http.ResponseWriter, write bool) (string, func(), bool) {
	lock, unlock := s.mu.RLock, s.mu.RUnlock
	if write {
		lock, unlock = s.mu.Lock, s.mu.Unlock
	}
	lock()
	if s.dataDir == "" {
		unlock()
		httpError(w, http.StatusConflict, errNoDataDir)
		return "", nil, false
	}
	return s.dataDir, unlock, true
}

// resumeDir returns the folder of resume id, answering 400 for ids that are
// not a plain folder name and 404 for resumes that don't exist.
func resumeDir(w http.ResponseWriter, dir, id string) (string, bool) {
	rdir, err := childPath(dir, id, "")
	if err != nil {
		httpError(w, http.StatusBadRequest, err)
		return "", false
	}
	_, ok := found(w, filepath.Join(rdir, resumeFile), "resume", id)
	return rdir, ok
}

// versionFile returns the snapshot file of version vid of the resume in rdir.
func versionFile(w http.ResponseWriter, rdir, vid string) (string, version, bool) {
	vdir, err := childPath(filepath.Join(rdir, versionsDir), vid, "")
	if err != nil {
		httpError(w, http.StatusBadRequest, err)
		return "", version{}, false
	}
	p := filepath.Join(vdir, resumeFile)
	fi, ok := found(w, p, "version", vid)
	if !ok {
		return "", version{}, false
	}
	return p, versionOf(vid, fi), true
}

// found stats the regular file at path, answering 404 (or 500 for other
// errors) when there is none.
func found(w http.ResponseWriter, path, what, id string) (fs.FileInfo, bool) {
	fi, err := os.Stat(path)
	switch {
	case err == nil && fi.Mode().IsRegular():
		return fi, true
	case err == nil || errors.Is(err, fs.ErrNotExist):
		httpError(w, http.StatusNotFound, fmt.Errorf("no %s %q", what, id))
	default:
		httpError(w, http.StatusInternalServerError, err)
	}
	return nil, false
}

func decodeJSON(r *http.Request, v any) error {
	if err := json.NewDecoder(io.LimitReader(r.Body, maxBody)).Decode(v); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}
	return nil
}

func writeRawJSON(w http.ResponseWriter, status int, data []byte) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(data)
}

func serveJSONFile(w http.ResponseWriter, path string) {
	b, err := os.ReadFile(path)
	if err != nil {
		httpError(w, http.StatusInternalServerError, err)
		return
	}
	writeRawJSON(w, http.StatusOK, b)
}

func (s *server) getSettings(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	writeJSON(w, settings{DataDir: s.dataDir, DefaultDataDir: defaultDataDir()})
}

func (s *server) putSettings(w http.ResponseWriter, r *http.Request) {
	var body struct {
		DataDir string `json:"dataDir"`
	}
	if err := decodeJSON(r, &body); err != nil {
		httpError(w, http.StatusBadRequest, err)
		return
	}
	dir, err := expandDir(body.DataDir)
	if err != nil {
		httpError(w, http.StatusBadRequest, err)
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		httpError(w, http.StatusBadRequest, err)
		return
	}
	if err := s.seed(dir); err != nil {
		httpError(w, http.StatusInternalServerError, err)
		return
	}
	if err := saveConfig(s.configPath, config{DataDir: dir}); err != nil {
		httpError(w, http.StatusInternalServerError, err)
		return
	}
	s.dataDir = dir
	writeJSON(w, settings{DataDir: dir, DefaultDataDir: defaultDataDir()})
}

func (s *server) listResumes(w http.ResponseWriter, r *http.Request) {
	dir, unlock, ok := s.lockDir(w, false)
	if !ok {
		return
	}
	defer unlock()
	list, err := listResumes(dir)
	if err != nil {
		httpError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, list)
}

func (s *server) createResume(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name    string `json:"name"`
		From    string `json:"from"`
		Version string `json:"version"`
	}
	if err := decodeJSON(r, &body); err != nil {
		httpError(w, http.StatusBadRequest, err)
		return
	}
	dir, unlock, ok := s.lockDir(w, true)
	if !ok {
		return
	}
	defer unlock()

	files := map[string][]byte{resumeFile: starterJSON}
	switch {
	case body.From != "":
		rdir, ok := resumeDir(w, dir, body.From)
		if !ok {
			return
		}
		src := rdir
		if body.Version != "" {
			p, _, ok := versionFile(w, rdir, body.Version)
			if !ok {
				return
			}
			src = filepath.Dir(p)
		}
		var err error
		if files, err = snapshot(src); err != nil {
			httpError(w, http.StatusInternalServerError, err)
			return
		}
		// A version with neither side files nor cover letters (such as one saved
		// before versions included them) gets the current ones.
		if src != rdir && !exists(filepath.Join(src, additionalDir)) && !exists(filepath.Join(src, lettersDir)) {
			current, err := snapshot(rdir)
			if err != nil {
				httpError(w, http.StatusInternalServerError, err)
				return
			}
			current[resumeFile] = files[resumeFile]
			files = current
		}
	case body.Version != "":
		httpError(w, http.StatusBadRequest, errors.New("a version needs the resume it is from"))
		return
	}
	id := uniqueName(dir, sanitizeName(body.Name), "")
	if err := newFolder(filepath.Join(dir, id), files); err != nil {
		httpError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSONStatus(w, http.StatusCreated, map[string]string{"id": id})
}

func (s *server) getResume(w http.ResponseWriter, r *http.Request) {
	dir, unlock, ok := s.lockDir(w, false)
	if !ok {
		return
	}
	defer unlock()
	if rdir, ok := resumeDir(w, dir, r.PathValue("id")); ok {
		serveJSONFile(w, filepath.Join(rdir, resumeFile))
	}
}

func (s *server) putResume(w http.ResponseWriter, r *http.Request) {
	raw, _, err := readDoc(r)
	if err != nil {
		httpError(w, http.StatusBadRequest, err)
		return
	}
	s.writeResumeFile(w, r, resumeFile, raw)
}

// writeResumeFile saves raw JSON, indented, as file name (a path relative to
// the folder, created as needed) of the resume in the URL.
func (s *server) writeResumeFile(w http.ResponseWriter, r *http.Request, name string, raw []byte) {
	var pretty bytes.Buffer
	if err := json.Indent(&pretty, raw, "", "  "); err != nil {
		httpError(w, http.StatusBadRequest, err)
		return
	}
	pretty.WriteByte('\n')

	dir, unlock, ok := s.lockDir(w, true)
	if !ok {
		return
	}
	defer unlock()
	rdir, ok := resumeDir(w, dir, r.PathValue("id"))
	if !ok {
		return
	}
	p := filepath.Join(rdir, name)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		httpError(w, http.StatusInternalServerError, err)
		return
	}
	if err := writeFileAtomic(p, pretty.Bytes()); err != nil {
		httpError(w, http.StatusInternalServerError, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// getSideFile serves side file name of a resume; one never saved reads as {}.
func (s *server) getSideFile(name string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dir, unlock, ok := s.lockDir(w, false)
		if !ok {
			return
		}
		defer unlock()
		rdir, ok := resumeDir(w, dir, r.PathValue("id"))
		if !ok {
			return
		}
		b, err := os.ReadFile(filepath.Join(rdir, name))
		switch {
		case errors.Is(err, fs.ErrNotExist):
			b = []byte("{}\n")
		case err != nil:
			httpError(w, http.StatusInternalServerError, err)
			return
		}
		writeRawJSON(w, http.StatusOK, b)
	}
}

// putSideFile saves side file name of a resume. Its content is up to the
// editor; it only has to be a JSON object.
func (s *server) putSideFile(name string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		raw, err := io.ReadAll(io.LimitReader(r.Body, maxBody))
		if err != nil {
			httpError(w, http.StatusBadRequest, err)
			return
		}
		var obj map[string]json.RawMessage
		if err := json.Unmarshal(raw, &obj); err != nil || obj == nil {
			httpError(w, http.StatusBadRequest, fmt.Errorf("%s needs a JSON object", name))
			return
		}
		s.writeResumeFile(w, r, name, raw)
	}
}

func (s *server) renameResume(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name string `json:"name"`
	}
	if err := decodeJSON(r, &body); err != nil {
		httpError(w, http.StatusBadRequest, err)
		return
	}
	if strings.TrimSpace(body.Name) == "" {
		httpError(w, http.StatusBadRequest, errors.New("the name is empty"))
		return
	}
	dir, unlock, ok := s.lockDir(w, true)
	if !ok {
		return
	}
	defer unlock()
	id := r.PathValue("id")
	rdir, ok := resumeDir(w, dir, id)
	if !ok {
		return
	}
	newID := sanitizeName(body.Name)
	if newID != id {
		target := filepath.Join(dir, newID)
		// On case-insensitive file systems a change of case finds the
		// resume itself, which is not a conflict.
		if tfi, err := os.Lstat(target); err == nil {
			if sfi, err := os.Lstat(rdir); err != nil || !os.SameFile(sfi, tfi) {
				httpError(w, http.StatusConflict, fmt.Errorf("a resume named %q already exists", newID))
				return
			}
		} else if !errors.Is(err, fs.ErrNotExist) {
			httpError(w, http.StatusInternalServerError, err)
			return
		}
		if err := os.Rename(rdir, target); err != nil {
			httpError(w, http.StatusInternalServerError, err)
			return
		}
	}
	writeJSON(w, map[string]string{"id": newID})
}

func (s *server) deleteResume(w http.ResponseWriter, r *http.Request) {
	dir, unlock, ok := s.lockDir(w, true)
	if !ok {
		return
	}
	defer unlock()
	id := r.PathValue("id")
	rdir, ok := resumeDir(w, dir, id)
	if !ok {
		return
	}
	trash := filepath.Join(dir, trashDir)
	if err := os.MkdirAll(trash, 0o755); err != nil {
		httpError(w, http.StatusInternalServerError, err)
		return
	}
	name := id + " " + time.Now().Format(stampLayout)
	target := filepath.Join(trash, name)
	for n := 2; exists(target); n++ {
		target = filepath.Join(trash, fmt.Sprintf("%s-%d", name, n))
	}
	if err := os.Rename(rdir, target); err != nil {
		httpError(w, http.StatusInternalServerError, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) listVersions(w http.ResponseWriter, r *http.Request) {
	dir, unlock, ok := s.lockDir(w, false)
	if !ok {
		return
	}
	defer unlock()
	rdir, ok := resumeDir(w, dir, r.PathValue("id"))
	if !ok {
		return
	}
	list, err := listVersions(rdir)
	if err != nil {
		httpError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, list)
}

func (s *server) createVersion(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Label string `json:"label"`
	}
	if err := decodeJSON(r, &body); err != nil {
		httpError(w, http.StatusBadRequest, err)
		return
	}
	dir, unlock, ok := s.lockDir(w, true)
	if !ok {
		return
	}
	defer unlock()
	rdir, ok := resumeDir(w, dir, r.PathValue("id"))
	if !ok {
		return
	}
	v, err := saveVersion(rdir, body.Label, time.Now())
	if err != nil {
		httpError(w, http.StatusInternalServerError, err)
		return
	}
	res := syncResult{Written: []string{}, Failed: []syncFailure{}}
	s.exportVersion(r.Context(), rdir, v.ID, &res)
	// The snapshot is kept when an export fails; Sync files retries it.
	writeJSONStatus(w, http.StatusCreated, struct {
		version
		Failed []syncFailure `json:"failed"`
	}{v, res.Failed})
}

func (s *server) getVersion(w http.ResponseWriter, r *http.Request) {
	dir, unlock, ok := s.lockDir(w, false)
	if !ok {
		return
	}
	defer unlock()
	rdir, ok := resumeDir(w, dir, r.PathValue("id"))
	if !ok {
		return
	}
	if p, _, ok := versionFile(w, rdir, r.PathValue("vid")); ok {
		serveJSONFile(w, p)
	}
}

func (s *server) restoreVersion(w http.ResponseWriter, r *http.Request) {
	dir, unlock, ok := s.lockDir(w, true)
	if !ok {
		return
	}
	defer unlock()
	rdir, ok := resumeDir(w, dir, r.PathValue("id"))
	if !ok {
		return
	}
	p, v, ok := versionFile(w, rdir, r.PathValue("vid"))
	if !ok {
		return
	}
	data, err := os.ReadFile(p)
	if err != nil {
		httpError(w, http.StatusInternalServerError, err)
		return
	}
	what := v.Label
	if what == "" {
		what = v.Created.Format("Jan 2 2006 3.04 PM")
	}
	backup, err := saveVersion(rdir, "Before restoring "+what, time.Now())
	if err != nil {
		httpError(w, http.StatusInternalServerError, err)
		return
	}
	res := syncResult{Written: []string{}, Failed: []syncFailure{}}
	s.exportVersion(r.Context(), rdir, backup.ID, &res)
	for _, f := range res.Failed {
		log.Printf("exporting %s: %s", f.File, f.Error)
	}
	if err := writeFileAtomic(filepath.Join(rdir, resumeFile), data); err != nil {
		httpError(w, http.StatusInternalServerError, err)
		return
	}
	writeRawJSON(w, http.StatusOK, data)
}

func (s *server) deleteVersion(w http.ResponseWriter, r *http.Request) {
	dir, unlock, ok := s.lockDir(w, true)
	if !ok {
		return
	}
	defer unlock()
	rdir, ok := resumeDir(w, dir, r.PathValue("id"))
	if !ok {
		return
	}
	p, _, ok := versionFile(w, rdir, r.PathValue("vid"))
	if !ok {
		return
	}
	// The version's folder holds its snapshot and the exports written by sync.
	if err := os.RemoveAll(filepath.Dir(p)); err != nil {
		httpError(w, http.StatusInternalServerError, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

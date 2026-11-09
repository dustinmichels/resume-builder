package main

// The editor runs in a browser tab, or in the desktop app's window (-tags
// desktop, see desktop.go). These hooks are what the desktop build replaces,
// and the routes the editor uses there for what a web view can't do itself:
// download files, open links elsewhere, or pick a folder.

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
)

var (
	// serve runs h until the app quits.
	serve = serveBrowser
	// fail reports an error that keeps the app from running, and exits.
	fail = func(err error) { log.Fatal(err) }
	// popplerMissing explains why something that needs a Poppler tool didn't
	// run.
	popplerMissing = func(tool string) error {
		return fmt.Errorf("%s not found; install poppler (`brew install poppler`)", tool)
	}
	// native is the desktop app's native side; nil in a browser.
	native nativeUI
)

// nativeUI shows the desktop app's native dialogs. Both return "" when the
// user cancels.
type nativeUI interface {
	// chooseFolder asks for a folder, starting in dir.
	chooseFolder(dir string) (string, error)
	// saveFile asks where to save a file, suggesting name.
	saveFile(name string) (string, error)
}

// serveBrowser serves h on addr, opening the editor in the browser if open.
func serveBrowser(h http.Handler, addr string, open bool) error {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	url := "http://" + ln.Addr().String()
	log.Printf("resume-builder %s on %s", buildVersion, url)
	if open {
		cmd := openCommand(url)
		if err := cmd.Start(); err != nil {
			log.Printf("opening the browser: %v", err)
		} else {
			go cmd.Wait()
		}
	}
	return http.Serve(ln, h)
}

// openURL opens a web, mail or phone link with the system's default handler:
// the app window must not navigate away from the editor.
func openURL(w http.ResponseWriter, r *http.Request) {
	var body struct {
		URL string `json:"url"`
	}
	if err := decodeJSON(r, &body); err != nil {
		httpError(w, http.StatusBadRequest, err)
		return
	}
	u, err := url.Parse(body.URL)
	if err != nil {
		httpError(w, http.StatusBadRequest, err)
		return
	}
	switch {
	case (u.Scheme == "https" || u.Scheme == "http") && u.Host != "":
	case (u.Scheme == "mailto" || u.Scheme == "tel") && u.Opaque != "":
	default:
		httpError(w, http.StatusBadRequest, errors.New("only web, mail and phone links open"))
		return
	}
	cmd := openCommand(u.String())
	if err := cmd.Start(); err != nil {
		httpError(w, http.StatusInternalServerError, err)
		return
	}
	go cmd.Wait()
	w.WriteHeader(http.StatusNoContent)
}

// chooseFolder asks for a data folder, starting at the current one, and
// returns {"path": …}; the editor then switches to it with PUT /api/settings.
func (s *server) chooseFolder(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	dir := s.dataDir
	s.mu.RUnlock()
	if dir == "" {
		dir = defaultDataDir()
	}
	// A folder not created yet opens the panel in its parent.
	if _, err := os.Stat(dir); err != nil {
		dir = filepath.Dir(dir)
	}
	path, err := native.chooseFolder(dir)
	if err != nil {
		httpError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, map[string]string{"path": path})
}

// maxSave caps POST /api/save bodies: exports, the text layer and JSON.
const maxSave = 64 << 20

// saveAs writes the request body where the user chooses in a Save dialog,
// suggesting the name in ?name=, and returns {"path": …}. It stands in for
// browser downloads, which a web view doesn't do.
func saveAs(w http.ResponseWriter, r *http.Request) {
	b, err := io.ReadAll(io.LimitReader(r.Body, maxSave+1))
	if err != nil {
		httpError(w, http.StatusBadRequest, err)
		return
	}
	if len(b) > maxSave {
		httpError(w, http.StatusRequestEntityTooLarge, errors.New("the file is too large"))
		return
	}
	// The dialog has the last word on the name; it only mustn't be a path.
	name := filepath.Base(r.URL.Query().Get("name"))
	if name == "." || name == string(filepath.Separator) {
		name = "Untitled"
	}
	path, err := native.saveFile(name)
	if err != nil {
		httpError(w, http.StatusInternalServerError, err)
		return
	}
	if path != "" {
		if err := os.WriteFile(path, b, 0o644); err != nil {
			httpError(w, http.StatusInternalServerError, err)
			return
		}
	}
	writeJSON(w, map[string]string{"path": path})
}

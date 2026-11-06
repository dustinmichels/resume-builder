package main

import (
	"net/http"
	"os/exec"
	"runtime"
)

// reveal opens a resume's folder in the system file manager (Finder on macOS).
func (s *server) reveal(w http.ResponseWriter, r *http.Request) {
	dir, unlock, ok := s.lockDir(w, false)
	if !ok {
		return
	}
	defer unlock()
	rdir, ok := resumeDir(w, dir, r.PathValue("id"))
	if !ok {
		return
	}
	cmd := openFolderCommand(rdir)
	if err := cmd.Start(); err != nil {
		httpError(w, http.StatusInternalServerError, err)
		return
	}
	// The file manager outlives the request; reap the launcher when it exits.
	go cmd.Wait()
	w.WriteHeader(http.StatusNoContent)
}

func openFolderCommand(path string) *exec.Cmd {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", path)
	case "windows":
		return exec.Command("explorer", path)
	default:
		return exec.Command("xdg-open", path)
	}
}

package main

// Resume templates sit beside the -template file (the classic template), each
// with the cover letter template its resumes' letters use and an optional
// icon set of its own:
//
//	template/<file>.typ, <letter>.typ
//	template/icons/<set>/icons.json, *.svg

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// TemplateInfo describes a resume template to the editor.
type TemplateInfo struct {
	ID   string `json:"id"` // "" = classic
	Name string `json:"name"`
	// Features are what the template draws beyond the common schema:
	// markers | uppercaseTitles | entryIcons | legend | compactEntries.
	Features []string `json:"features"`
	// Icons are the template's icon set, [] for none.
	Icons []LineIcon `json:"icons"`

	file    string // beside the -template file
	letter  string // the cover letter template, beside the -template file
	iconSet string // icons/<iconSet> beside the -template file; "" for none
}

// LineIcon is an icon drawn beside text; Color is a legend label's beside it.
type LineIcon struct {
	Name  string `json:"name"`
	SVG   string `json:"svg"`
	Color string `json:"color,omitempty"`
}

// templates lists the resume templates, classic first.
func (s *server) templates() []TemplateInfo {
	return []TemplateInfo{
		{ID: "", Name: "Classic", Features: []string{"markers", "uppercaseTitles"}, file: filepath.Base(s.templatePath), letter: "cover-letter.typ"},
		{ID: "julia", Name: "Julia", Features: []string{"entryIcons", "legend", "compactEntries"}, file: "julia.typ", letter: "julia-letter.typ", iconSet: "julia"},
	}
}

// templateFile returns the path of the resume template with the given id
// (a document's style.template).
func (s *server) templateFile(id string) (string, error) {
	t, err := s.template(id)
	if err != nil {
		return "", err
	}
	return filepath.Join(filepath.Dir(s.templatePath), t.file), nil
}

// letterTemplateFile returns the path of the cover letter template for
// resumes on the template with the given id.
func (s *server) letterTemplateFile(id string) (string, error) {
	t, err := s.template(id)
	if err != nil {
		return "", err
	}
	return filepath.Join(filepath.Dir(s.templatePath), t.letter), nil
}

func (s *server) template(id string) (TemplateInfo, error) {
	for _, t := range s.templates() {
		if t.ID == id {
			return t, nil
		}
	}
	return TemplateInfo{}, fmt.Errorf("unknown template %q", id)
}

// listTemplates lists the resume templates with their icon sets, read on
// every request so new icons show up without a restart.
func (s *server) listTemplates(w http.ResponseWriter, r *http.Request) {
	list := s.templates()
	for i := range list {
		list[i].Icons = []LineIcon{}
		if set := list[i].iconSet; set != "" {
			icons, err := readIconSet(filepath.Join(filepath.Dir(s.templatePath), "icons", set))
			if err != nil {
				httpError(w, http.StatusInternalServerError, err)
				return
			}
			list[i].Icons = icons
		}
	}
	writeJSON(w, list)
}

// readIconSet reads the icon set in dir: icons.json lists its SVG files in
// order, with the name and legend color of each.
func readIconSet(dir string) ([]LineIcon, error) {
	b, err := os.ReadFile(filepath.Join(dir, "icons.json"))
	if err != nil {
		return nil, err
	}
	var list []struct {
		File  string `json:"file"`
		Name  string `json:"name"`
		Color string `json:"color"`
	}
	if err := json.Unmarshal(b, &list); err != nil {
		return nil, fmt.Errorf("%s: %w", filepath.Join(dir, "icons.json"), err)
	}
	icons := make([]LineIcon, 0, len(list))
	for _, ic := range list {
		svg, err := os.ReadFile(filepath.Join(dir, ic.File))
		if err != nil {
			return nil, err
		}
		icons = append(icons, LineIcon{Name: ic.Name, SVG: strings.TrimSpace(string(svg)), Color: ic.Color})
	}
	return icons, nil
}

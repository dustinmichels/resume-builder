package main

import (
	"context"
	"os/exec"
	"slices"
	"testing"
)

func TestFieldBoxes(t *testing.T) {
	marks := []fieldMark{
		// One-line inline value: box spans the marks, ascent to descent.
		{Path: "a", Edge: "start", Page: 1, X: 100, Y: 50, Size: 10},
		{Path: "a", Edge: "end", Page: 1, X: 160, Y: 50, Size: 10},
		// Block value: box takes the available width, cap height to descent.
		{Path: "b", Edge: "top", Page: 1, X: 80, Y: 70, Size: 10, Width: 400},
		{Path: "b", Edge: "bottom", Page: 1, X: 80, Y: 100, Size: 10},
		// Renders empty: no box.
		{Path: "empty", Edge: "start", Page: 1, X: 300, Y: 50, Size: 10},
		{Path: "empty", Edge: "end", Page: 1, X: 300, Y: 50, Size: 10},
		// Split across pages: no box.
		{Path: "split", Edge: "top", Page: 1, X: 80, Y: 700, Size: 10, Width: 400},
		{Path: "split", Edge: "bottom", Page: 2, X: 80, Y: 60, Size: 10},
	}
	got := fieldBoxes(marks)
	want := []FieldBox{
		{Path: "a", Page: 1, X: 99, Y: 42, W: 62, H: 10.8},
		{Path: "b", Page: 1, X: 79, Y: 68, W: 402, H: 34.8},
	}
	if len(got) != len(want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
	for i := range want {
		g, w := got[i], want[i]
		if g.Path != w.Path || g.Page != w.Page || !near(g.X, w.X) || !near(g.Y, w.Y) || !near(g.W, w.W) || !near(g.H, w.H) {
			t.Errorf("box %d = %+v, want %+v", i, g, w)
		}
	}
}

func near(a, b float64) bool { return a-b < 1e-9 && b-a < 1e-9 }

// TestRenderFieldMap renders through the real template and checks that every
// visible value is mapped, by its index in the unfiltered arrays.
func TestRenderFieldMap(t *testing.T) {
	if _, err := exec.LookPath("typst"); err != nil {
		t.Skip("typst not installed")
	}
	out, err := render(context.Background(), "../template/resume.typ", []byte(exportDoc), true)
	if err != nil {
		t.Fatal(err)
	}
	defer out.Close()

	var paths []string
	for _, f := range out.Fields {
		paths = append(paths, f.Path)
		if f.Page != 1 || f.X < 0 || f.Y < 0 || f.X+f.W > 612 || f.Y+f.H > 792 || f.W <= 0 || f.H <= 0 {
			t.Errorf("%s: box %+v is off the page", f.Path, f)
		}
	}
	for _, want := range []string{
		"basics.name", "basics.headline.0", "basics.contacts.0", "basics.contacts.1",
		"sections.0.title", "sections.0.paragraphs.0", "sections.0.paragraphs.2",
		"sections.1.items.0.text",
		"sections.1.items.1.title", "sections.1.items.1.date", "sections.1.items.1.subtitle",
		"sections.1.items.1.location", "sections.1.items.1.bullets.0",
		"sections.2.columns.0.title", "sections.2.columns.0.lines.1",
		"sections.3.columns.0.title", "sections.3.columns.0.lines.0",
	} {
		if !slices.Contains(paths, want) {
			t.Errorf("no box for %s; got %v", want, paths)
		}
	}
	// The empty second bullet isn't drawn, so it has nothing to click.
	if slices.Contains(paths, "sections.1.items.1.bullets.1") {
		t.Errorf("empty bullet has a box")
	}
}

func TestParseRichBoldItalic(t *testing.T) {
	for in, want := range map[string]string{
		"***both***":         "***both***",
		"**b*****bi*** rest": "**b*****bi*** rest",
		"*i* and **b**":      "*i* and **b**",
	} {
		if got := mdInline(parseRich(in)); got != want {
			t.Errorf("mdInline(parseRich(%q)) = %q, want %q", in, got, want)
		}
	}
	got := parseRich("***x***")
	if len(got) != 1 || got[0].Kind != inlineBold || len(got[0].Kids) != 1 || got[0].Kids[0].Kind != inlineItalic {
		t.Errorf("***x*** parsed as %+v, want bold(italic(x))", got)
	}
}

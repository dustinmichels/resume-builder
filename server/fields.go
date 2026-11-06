package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os/exec"
)

// FieldBox is where one editable value sits on a rendered page, in points
// from the page's top-left corner. Path addresses the value in the resume
// JSON ("sections.2.items.0.bullets.3").
type FieldBox struct {
	Path string  `json:"path"`
	Page int     `json:"page"`
	X    float64 `json:"x"`
	Y    float64 `json:"y"`
	W    float64 `json:"w"`
	H    float64 `json:"h"`
}

// fieldMark is one position recorded by the template's `mark` function.
// Inline values get a "start" and "end" mark on the text baseline; block
// values get a "top" mark (with the available width) and a "bottom" mark.
// Each cell of a side-by-side columns section gets one "cell" mark at its
// top-left corner, with its width and height.
type fieldMark struct {
	Path   string  `json:"path"`
	Edge   string  `json:"edge"`
	Page   int     `json:"page"`
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Size   float64 `json:"size"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

// Line metrics as a fraction of the font size: how far glyphs reach above and
// below the baseline, and how far a block's first line reaches above the
// block's top edge (Typst puts block tops at cap height).
const (
	ascent  = 0.8
	descent = 0.28
	capPad  = 0.2
	sidePad = 1.0 // pt
)

// queryFields lists the field marks of template main in dir, in document order.
func queryFields(ctx context.Context, dir, main string) ([]fieldMark, error) {
	cmd := exec.CommandContext(ctx, "typst", "eval", "--diagnostic-format", "short",
		"query(<field>).map(it => it.value)", "--in", main)
	cmd.Dir = dir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, &compileError{Output: stderr.String()}
	}
	var marks []fieldMark
	if err := json.Unmarshal(out, &marks); err != nil {
		return nil, err
	}
	return marks, nil
}

// fieldBoxes pairs each value's marks into a box. Values that render empty,
// or whose marks land on different pages, get no box.
func fieldBoxes(marks []fieldMark) []FieldBox {
	type pair struct{ open, close *fieldMark }
	pairs := map[string]*pair{}
	var order []string
	for i := range marks {
		m := &marks[i]
		if m.Edge == "cell" {
			continue // a column's layout, not a value; see columnBoxes
		}
		p := pairs[m.Path]
		if p == nil {
			p = &pair{}
			pairs[m.Path] = p
			order = append(order, m.Path)
		}
		switch m.Edge {
		case "start", "top":
			p.open = m
		case "end", "bottom":
			p.close = m
		}
	}

	boxes := []FieldBox{}
	for _, path := range order {
		p := pairs[path]
		a, b := p.open, p.close
		if a == nil || b == nil || a.Page != b.Page {
			continue
		}
		box := FieldBox{Path: path, Page: a.Page}
		if a.Edge == "top" {
			box.X, box.W = a.X, a.Width
			box.Y = a.Y - capPad*a.Size
			box.H = b.Y - a.Y + (capPad+descent)*a.Size
		} else {
			// Inline values usually sit on one line. When one wraps, its box
			// spans from the first line to the last, between the two marks.
			box.X, box.W = min(a.X, b.X), max(a.X, b.X)-min(a.X, b.X)
			box.Y = a.Y - ascent*a.Size
			box.H = b.Y - a.Y + (ascent+descent)*a.Size
		}
		if box.W < 1 || box.H <= 0 {
			continue
		}
		box.X -= sidePad
		box.W += 2 * sidePad
		boxes = append(boxes, box)
	}
	return boxes
}

// columnBoxes lists where each column of the side-by-side columns sections
// sits, unpadded, from its "cell" mark. Path addresses the column
// ("sections.4.columns.1"); the editor draws draggable dividers between them.
func columnBoxes(marks []fieldMark) []FieldBox {
	boxes := []FieldBox{}
	for _, m := range marks {
		if m.Edge == "cell" {
			boxes = append(boxes, FieldBox{Path: m.Path, Page: m.Page, X: m.X, Y: m.Y, W: m.Width, H: m.Height})
		}
	}
	return boxes
}

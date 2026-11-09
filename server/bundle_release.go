//go:build release

package main

import resumebuilder "github.com/dustinmichels/resume-builder"

func init() { bundled = resumebuilder.Files }

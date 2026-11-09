package main

// Releases are published on GitHub by scripts/release.sh. Builds of a release
// look there for a newer one: the desktop app offers to update itself
// (desktop_update.go), the command-line build says so in its log.

import (
	"context"
	"log"
	"regexp"
	"runtime"
	"slices"
	"time"

	"github.com/wailsapp/wails/v3/pkg/updater"
	"github.com/wailsapp/wails/v3/pkg/updater/providers/github"
)

const releaseRepo = "dustinmichels/resume-builder"

var releaseTag = regexp.MustCompile(`^v[0-9]+\.[0-9]+\.[0-9]+$`)

// releaseVersion is buildVersion without its "v" (1.2.3) in builds of a
// release, and "" in development builds, which never look for updates.
func releaseVersion() string {
	if !releaseTag.MatchString(buildVersion) {
		return ""
	}
	return buildVersion[1:]
}

// releaseSource finds the latest release on GitHub newer than this one, and
// its asset named asset, checked against the release's checksums.txt.
func releaseSource(asset string) (*github.Provider, error) {
	return github.New(github.Config{
		Repository:    releaseRepo,
		ChecksumAsset: "checksums.txt",
		AssetMatcher: func(_ updater.CheckRequest, assets []github.ReleaseAsset) int {
			return slices.IndexFunc(assets, func(a github.ReleaseAsset) bool { return a.Name == asset })
		},
	})
}

// releasePage is the GitHub page of a release from releaseSource.
func releasePage(rel *updater.Release) string {
	if u, ok := rel.Metadata["github.release.htmlURL"].(string); ok {
		return u
	}
	return "https://github.com/" + releaseRepo + "/releases/latest"
}

// logUpdate logs where to get a newer release, if there is one. The
// command-line build doesn't replace itself: it runs from wherever its
// archive was unpacked, typically from a terminal.
func logUpdate() {
	version := releaseVersion()
	if version == "" {
		return
	}
	ext := ".tar.gz"
	if runtime.GOOS == "windows" {
		ext = ".zip"
	}
	src, err := releaseSource("resume-builder-" + runtime.GOOS + "-" + runtime.GOARCH + ext)
	if err != nil {
		log.Printf("checking for updates: %v", err)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	rel, err := src.Check(ctx, updater.CheckRequest{CurrentVersion: version, Platform: runtime.GOOS, Arch: runtime.GOARCH})
	if err != nil {
		log.Printf("checking for updates: %v", err)
		return
	}
	if rel != nil {
		log.Printf("resume-builder %s is available (this is %s): %s", rel.Version, version, releasePage(rel))
	}
}

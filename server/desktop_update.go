//go:build desktop

package main

// Updates in the Mac app. Builds of a release check GitHub at launch and then
// daily while open, silently unless there is a newer release; Check for
// Updates… in the app menu also says when there is none or the check failed.
// Nothing is downloaded without asking: the prompt offers Update, Later and
// Release Notes. Update opens Wails' update window, which downloads
// Resume-Builder.zip, checks it against the release's checksums.txt, and
// offers Restart & Apply: that swaps the new app in for this one and reopens
// it. A copy that can't replace itself (opened from the disk image, or in a
// folder the user can't change) offers to download the disk image instead.

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sync/atomic"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
	"github.com/wailsapp/wails/v3/pkg/updater"
)

const (
	// updateAsset holds the app alone (scripts/package-macos.sh).
	updateAsset = "Resume-Builder.zip"
	// diskImageURL is the newest release's disk image.
	diskImageURL        = "https://github.com/" + releaseRepo + "/releases/latest/download/Resume-Builder.dmg"
	updateCheckInterval = 24 * time.Hour
)

// updateBusy is set from a check until its prompt is answered, so checks
// never stack prompts.
var updateBusy atomic.Bool

// startUpdates sets up app.Updater in builds of a release on macOS, and checks
// once the app has started and daily after. It reports whether it did.
func startUpdates() bool {
	version := releaseVersion()
	if version == "" || runtime.GOOS != "darwin" {
		return false
	}
	src, err := releaseSource(updateAsset)
	if err == nil {
		err = app.Updater.Init(updater.Config{CurrentVersion: version, Providers: []updater.Provider{src}})
	}
	if err != nil {
		log.Printf("updates: %v", err)
		return false
	}
	app.Event.OnApplicationEvent(events.Common.ApplicationStarted, func(*application.ApplicationEvent) {
		go func() {
			removeAbandonedUpdates()
			for {
				checkForUpdate(false)
				time.Sleep(updateCheckInterval)
			}
		}()
	})
	return true
}

// checkForUpdate asks GitHub for a newer release and offers it. Unless asked
// (Check for Updates…), it stays silent when there is none or the check fails.
func checkForUpdate(asked bool) {
	switch app.Updater.State() {
	case updater.StateDownloading, updater.StateVerifying, updater.StateInstalling:
		return // the update window is showing it
	}
	if !updateBusy.CompareAndSwap(false, true) {
		return
	}
	defer updateBusy.Store(false)
	if asked {
		app.Updater.SkipVersion("") // the update window's Skip lasts until asked
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	rel, err := app.Updater.Check(ctx)
	cancel()
	current := app.Updater.CurrentVersion()
	switch {
	case err != nil:
		log.Printf("checking for updates: %v", err)
		if asked {
			ask(app.Dialog.Warning(), "Couldn’t check for updates", err.Error(), "OK")
		}
	case rel == nil:
		if asked {
			ask(app.Dialog.Info(), appName+" is up to date", "Version "+current+" is the newest version.", "OK")
		}
	default:
		offerUpdate(rel, current)
	}
}

// offerUpdate asks whether to update to rel.
func offerUpdate(rel *updater.Release, current string) {
	title := fmt.Sprintf("%s %s is available", appName, rel.Version)
	msg := "You have version " + current + ". "
	if why := whyNotReplaceable(); why != "" {
		msg += why + " Download gets the new version’s disk image: open it, drag " + appName +
			" onto Applications and choose Replace."
		switch ask(app.Dialog.Question(), title, msg, "Download", "Later", "Release Notes") {
		case "Download":
			start(diskImageURL)
		case "Release Notes":
			start(releasePage(rel))
		}
		return
	}
	msg += "Update downloads it, then Restart & Apply replaces this copy and reopens " + appName +
		". Your resumes aren’t touched."
	switch ask(app.Dialog.Question(), title, msg, "Update", "Later", "Release Notes") {
	case "Update":
		go func() {
			if err := app.Updater.CheckAndInstall(context.Background()); err != nil {
				log.Printf("updating: %v", err)
			}
		}()
	case "Release Notes":
		start(releasePage(rel))
	}
}

// ask shows d over the app window with buttons, the first the default and the
// second (if any) the one Escape presses, and returns the label clicked.
func ask(d *application.MessageDialog, title, msg string, buttons ...string) string {
	clicked := make(chan string, 1)
	d.SetTitle(title).SetMessage(msg).AttachToWindow(window)
	// Added left to right, so the default ends up rightmost.
	for i := len(buttons) - 1; i >= 0; i-- {
		label := buttons[i]
		b := d.AddButton(label).OnClick(func() { clicked <- label })
		switch i {
		case 0:
			b.SetAsDefault()
		case 1:
			b.SetAsCancel()
		}
	}
	d.Show()
	return <-clicked
}

// appBundle is the .app this runs from.
func appBundle() (string, error) {
	exe, err := os.Executable()
	if err == nil {
		exe, err = filepath.EvalSymlinks(exe)
	}
	if err != nil {
		return "", err
	}
	bundle := filepath.Dir(filepath.Dir(filepath.Dir(exe)))
	if filepath.Base(filepath.Dir(exe)) != "MacOS" || filepath.Ext(bundle) != ".app" {
		return "", errors.New("not in an app bundle")
	}
	return bundle, nil
}

// whyNotReplaceable says why this copy can't update in place, or "" if it can.
// The updater downloads into the app's folder and swaps the app there, so the
// folder must be one the user can change. An app opened from the disk image,
// or from Downloads before it was moved (macOS then runs it from a read-only
// copy), is in one they can't.
func whyNotReplaceable() string {
	bundle, err := appBundle()
	if err != nil {
		return "This copy can’t update itself."
	}
	probe, err := os.MkdirTemp(filepath.Dir(bundle), ".update-probe-*")
	if err != nil {
		return "This copy can’t update itself where it is; it needs to be in Applications."
	}
	os.Remove(probe)
	return ""
}

// removeAbandonedUpdates deletes updates downloaded beside the app but never
// applied (the update window closed without Restart & Apply), which would
// otherwise sit in Applications as a second copy of the app.
func removeAbandonedUpdates() {
	bundle, err := appBundle()
	if err != nil {
		return
	}
	staged, _ := filepath.Glob(filepath.Join(filepath.Dir(bundle), "wails-update-*", appName+".app"))
	for _, p := range staged {
		if err := os.RemoveAll(filepath.Dir(p)); err != nil {
			log.Printf("removing an abandoned update: %v", err)
		} else {
			log.Printf("removed an abandoned update: %s", filepath.Dir(p))
		}
	}
}

//go:build desktop

package main

// The desktop app (-tags desktop, CGO): a Wails window hosts the editor,
// served straight from the handler without a network listener, with native
// menus and dialogs. Everything else is the same server.

import (
	_ "embed"
	"errors"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

const (
	appName = "Resume Builder"
	appID   = "com.dustinmichels.resumebuilder"
	helpURL = "https://github.com/dustinmichels/resume-builder#readme"
)

var (
	app    *application.App
	window *application.WebviewWindow
	// logPath is where the app logs; "" if it can't.
	logPath string
)

// appIcon is the Dock icon (and the icon in dialogs). The .app bundle has it
// too, but setting it here also covers `mise run app`. Made by scripts/icon.sh.
//
//go:embed icon.png
var appIcon []byte

func init() {
	serve = serveApp
	fail = failApp
	native = dialogs{}
	popplerMissing = func(tool string) error {
		return errors.New("not available in this version of the app yet")
	}
	openLog()
	findHomebrew()
}

// serveApp shows the editor in the app window until the app quits.
func serveApp(h http.Handler, _ string, _ bool) error {
	log.Printf("resume-builder %s (app)", buildVersion)
	newApp(h)
	window = app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:      appName,
		Width:      1360, // size when un-maximised
		Height:     860,
		MinWidth:   900,
		MinHeight:  600,
		StartState: application.WindowStateMaximised,
		URL:        "/",
	})
	return app.Run()
}

// failApp shows err in an error dialog, then quits: the app has no terminal
// to print it to.
func failApp(err error) {
	log.Print(err)
	if app == nil {
		newApp(http.NotFoundHandler())
	}
	msg := err.Error()
	if logPath != "" {
		msg += "\n\nThe log is in " + logPath
	}
	app.Event.OnApplicationEvent(events.Common.ApplicationStarted, func(*application.ApplicationEvent) {
		go func() {
			app.Dialog.Error().SetTitle(appName + " can't start").SetMessage(msg).Show()
			app.Quit()
		}()
	})
	app.Run()
	os.Exit(1)
}

// newApp sets app, serving the window from h. A second launch focuses the
// running app's window instead.
func newApp(h http.Handler) {
	app = application.New(application.Options{
		Name:        appName,
		Description: "Version " + strings.TrimPrefix(buildVersion, "v"),
		Icon:        appIcon,
		Logger:      slog.New(slog.NewTextHandler(log.Writer(), nil)),
		Assets:      application.AssetOptions{Handler: h, DisableLogging: true},
		Mac:         application.MacOptions{ApplicationShouldTerminateAfterLastWindowClosed: true},
		SingleInstance: &application.SingleInstanceOptions{
			UniqueID: appID,
			OnSecondInstanceLaunch: func(application.SecondInstanceData) {
				if window != nil {
					window.UnMinimise() // not Restore: that un-maximises
					window.Focus()
				}
			},
		},
	})

	// The Edit menu is what makes Cmd-C/V/X/A/Z reach text fields in a web view.
	menu := app.Menu.New()
	menu.AddRole(application.AppMenu)
	menu.AddRole(application.FileMenu)
	menu.AddRole(application.EditMenu)
	menu.AddRole(application.WindowMenu)
	help := menu.AddSubmenu("Help")
	help.Add(appName + " Help").OnClick(func(*application.Context) { start(helpURL) })
	if logPath != "" {
		help.Add("Show Log File").OnClick(func(*application.Context) { start(logPath) })
	}
	app.Menu.Set(menu)
}

// start opens target (a URL or file) with its default app.
func start(target string) {
	cmd := openCommand(target)
	if err := cmd.Start(); err != nil {
		log.Printf("opening %s: %v", target, err)
		return
	}
	go cmd.Wait()
}

// dialogs shows the native dialogs over the app window.
type dialogs struct{}

func (dialogs) chooseFolder(dir string) (string, error) {
	return app.Dialog.OpenFile().
		SetTitle("Choose a data folder").
		SetMessage("Your resumes and their versions are kept in this folder.").
		SetButtonText("Choose").
		CanChooseDirectories(true).
		CanChooseFiles(false).
		CanCreateDirectories(true).
		SetDirectory(dir).
		AttachToWindow(window).
		PromptForSingleSelection()
}

func (dialogs) saveFile(name string) (string, error) {
	return app.Dialog.SaveFile().
		SetFilename(name).
		CanCreateDirectories(true).
		AttachToWindow(window).
		PromptForSingleSelection()
}

// openLog sends the log to a file as well as stderr, since an app opened from
// Finder has no terminal: ~/Library/Logs/Resume Builder on macOS.
func openLog() {
	dir, err := os.UserHomeDir()
	if err == nil && runtime.GOOS == "darwin" {
		dir = filepath.Join(dir, "Library", "Logs", appName)
	} else if dir, err = os.UserCacheDir(); err == nil {
		dir = filepath.Join(dir, "resume-builder", "logs")
	}
	if err == nil {
		err = os.MkdirAll(dir, 0o755)
	}
	if err != nil {
		log.Printf("no log file: %v", err)
		return
	}
	p := filepath.Join(dir, "resume-builder.log")
	if fi, err := os.Stat(p); err == nil && fi.Size() > 1<<20 {
		os.Remove(p) // start over rather than grow without bound
	}
	f, err := os.OpenFile(p, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		log.Printf("no log file: %v", err)
		return
	}
	logPath = p
	log.SetOutput(io.MultiWriter(os.Stderr, f))
}

// findHomebrew adds Homebrew's folders to the end of PATH: apps opened from
// Finder get a PATH without them, and Poppler installed there still works.
func findHomebrew() {
	if runtime.GOOS != "darwin" {
		return
	}
	path := filepath.SplitList(os.Getenv("PATH"))
	for _, dir := range []string{"/opt/homebrew/bin", "/usr/local/bin"} {
		if !slices.Contains(path, dir) {
			path = append(path, dir)
		}
	}
	os.Setenv("PATH", strings.Join(path, string(os.PathListSeparator)))
}

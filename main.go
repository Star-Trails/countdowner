package main

import (
	"errors"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"countdowner/internal/model"

	"github.com/egoist/mygo"
)

func main() {
	if runtime.GOOS != "windows" {
		log.Print("Countdowner is a Windows desktop application; build with go tool mygo build -platform windows/amd64")
		return
	}
	mygo.App.SetName("Countdowner")
	if !mygo.App.RequestSingleInstanceLock() {
		return
	}
	dir, err := mygo.App.Path(mygo.PathUserData)
	if err != nil {
		log.Fatal(err)
	}
	logDir, err := mygo.App.Path(mygo.PathLogs)
	if err != nil {
		log.Fatal(err)
	}
	file, err := os.OpenFile(filepath.Join(logDir, "countdowner.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	log.SetOutput(file)
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	log.Printf("Countdowner %s; %s; MyGo %s; %s/%s", mygo.App.Version(), runtime.Version(), mygo.Version, runtime.GOOS, runtime.GOARCH)

	app := &application{store: model.Store{Path: filepath.Join(dir, "config.json")}}
	app.config, err = app.store.Load()
	if err != nil {
		log.Print(err)
		app.config = model.Default()
		app.readOnly, app.statusError, app.status = true, true, err.Error()
	} else if _, statErr := os.Stat(app.store.Path); errors.Is(statErr, os.ErrNotExist) {
		app.config.Language = languageForLocale(mygo.App.Locale())
	}
	mygo.Theme.SetSource(mygo.ThemeSource(app.config.Theme))
	// Windows Settings can disable an existing startup entry. Respect the
	// actual OS state; don't silently re-enable it on the next app launch.
	if !app.readOnly && app.config.Autostart != mygo.App.OpenAtLogin() {
		app.config.Autostart = mygo.App.OpenAtLogin()
		if err := app.store.Save(app.config); err != nil {
			app.fail(err)
		}
	}
	mygo.App.OnSecondInstance(func([]string, string) { app.showSettings() })
	mygo.App.WhenReady(app.start)
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}

func languageForLocale(locale string) string {
	locale = strings.ToLower(strings.ReplaceAll(locale, "_", "-"))
	switch {
	case strings.HasPrefix(locale, "zh"):
		if strings.Contains(locale, "hant") || strings.Contains(locale, "tw") || strings.Contains(locale, "hk") || strings.Contains(locale, "mo") {
			return "zh-Hant"
		}
		return "zh-Hans"
	case strings.HasPrefix(locale, "ja"):
		return "ja"
	case strings.HasPrefix(locale, "ko"):
		return "ko"
	default:
		return "en"
	}
}

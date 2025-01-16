package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
	"slices"
	"sync"
	"time"

	"golang.org/x/sys/windows"

	"github.com/dweymouth/supersonic/backend"
	"github.com/dweymouth/supersonic/res"
	"github.com/dweymouth/supersonic/ui"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/driver"
	"fyne.io/fyne/v2/lang"
)

func main() {
	// parse cmd line flags - see backend/cmdlineoptions.go
	flag.Parse()
	if *backend.FlagVersion {
		fmt.Println(res.AppVersion)
		return
	}
	if *backend.FlagHelp {
		flag.Usage()
		return
	}
	// rest of flag actions are handled in backend.StartupApp

	myApp, err := backend.StartupApp(res.AppName, res.DisplayName, res.AppVersion, res.AppVersionTag, res.LatestReleaseURL)
	if err != nil {
		if err != backend.ErrAnotherInstance {
			log.Fatalf("fatal startup error: %v", err.Error())
		}
		return
	}

	if myApp.Config.Application.UIScaleSize == "Smaller" {
		os.Setenv("FYNE_SCALE", "0.85")
	} else if myApp.Config.Application.UIScaleSize == "Larger" {
		os.Setenv("FYNE_SCALE", "1.1")
	}

	// load configured app language, or all otherwise
	lIdx := slices.IndexFunc(res.TranslationsInfo, func(t res.TranslationInfo) bool {
		return t.Name == myApp.Config.Application.Language
	})
	success := false
	if lIdx >= 0 {
		tr := res.TranslationsInfo[lIdx]
		content, err := res.Translations.ReadFile("translations/" + tr.TranslationFileName)
		if err == nil {
			// "trick" Fyne into loading translations for configured language
			// by pretending it's the translation for the system locale
			name := lang.SystemLocale().LanguageString()
			lang.AddTranslations(fyne.NewStaticResource(name+".json", content))
			success = true
		} else {
			log.Printf("Error loading translation file %s: %s\n", tr.TranslationFileName, err.Error())
		}
	}
	if !success {
		if err := lang.AddTranslationsFS(res.Translations, "translations"); err != nil {
			log.Printf("Error loading translations: %s", err.Error())
		}
	}

	fyneApp := app.New()
	fyneApp.SetIcon(res.ResAppicon256Png)

	mainWindow := ui.NewMainWindow(fyneApp, res.AppName, res.DisplayName, res.AppVersion, myApp)
	myApp.OnReactivate = mainWindow.Show
	myApp.OnExit = mainWindow.Quit

	go func() {
		defaultServer := myApp.ServerManager.GetDefaultServer()
		if defaultServer == nil {
			mainWindow.Controller.PromptForFirstServer()
		} else {
			mainWindow.Controller.DoConnectToServerWorkflow(defaultServer)
		}
	}()

	startupOnceTasks := sync.OnceFunc(func() {
		mainWindow.Window.(driver.NativeWindow).RunNative(func(ctx any) {
			// intialize Windows SMTC
			if runtime.GOOS == "windows" {
				if maj, _, _ := windows.RtlGetNtVersionNumbers(); maj >= 10 {
					// SMTC is only available from Windows 10 (10.0.10240) onward
					hwnd := ctx.(driver.WindowsWindowContext).HWND
					myApp.SetupWindowsSMTC(hwnd)
				}
			}

			// slightly hacky workaround for https://github.com/fyne-io/fyne/issues/4964
			_, isWayland := ctx.(*driver.WaylandWindowContext)
			if runtime.GOOS == "linux" && !isWayland {
				go func() {
					time.Sleep(50 * time.Millisecond)
					s := mainWindow.DesiredSize()
					mainWindow.Window.Resize(s.Subtract(fyne.NewSize(4, 0)))
					time.Sleep(50 * time.Millisecond)
					mainWindow.Window.Resize(s) // back to desired size
				}()
			}
		})
	})
	fyneApp.Lifecycle().SetOnEnteredForeground(func() {
		startupOnceTasks()
	})

	mainWindow.ShowAndRun()

	log.Println("Running shutdown tasks...")
	myApp.Shutdown()
}

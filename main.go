package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
	"slices"
	"sync"

	"github.com/dweymouth/supersonic/backend"
	"github.com/dweymouth/supersonic/res"
	"github.com/dweymouth/supersonic/ui"
	"github.com/dweymouth/supersonic/ui/util"
	"golang.org/x/term"

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
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		if *backend.FlagPlayAlbum {
			fmt.Scanln(&backend.PlayAlbumCLIArg)
		} else if *backend.FlagPlayPlaylist {
			fmt.Scanln(&backend.PlayPlaylistCLIArg)
		} else if *backend.FlagPlayTrack {
			fmt.Scanln(&backend.PlayTrackCLIArg)
		}
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

	if myApp.Config.Application.DisableDPIDetection {
		os.Setenv("FYNE_DISABLE_DPI_DETECTION", "true")
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
	mainWindow.Window.SetMaster()
	myApp.OnReactivate = util.FyneDoFunc(mainWindow.Show)
	myApp.OnExit = util.FyneDoFunc(mainWindow.Quit)

	windowStartupTasks := sync.OnceFunc(func() {
		defaultServer := myApp.ServerManager.GetDefaultServer()
		if defaultServer == nil {
			mainWindow.Controller.PromptForFirstServer()
		} else if !*backend.FlagStartMinimized { // If the minimized start flag was passed, the connection is already established.
			mainWindow.Controller.DoConnectToServerWorkflow(defaultServer)
		}

		if myApp.Config.Application.EnableOSMediaPlayerAPIs {
			mainWindow.Window.(driver.NativeWindow).RunNative(func(ctx any) {
				// intialize Windows SMTC
				if runtime.GOOS == "windows" {
					hwnd := ctx.(driver.WindowsWindowContext).HWND
					myApp.SetupWindowsSMTC(hwnd)
				}
			})
		}
	})
	fyneApp.Lifecycle().SetOnEnteredForeground(windowStartupTasks)

	if *backend.FlagStartMinimized {
		if err = myApp.LoginToDefaultServer(); err != nil {
			log.Fatalf("failed to connect to server: %v", err.Error())
			return
		}
		fyneApp.Run()
	} else {
		mainWindow.ShowAndRun()
	}

	log.Println("Running shutdown tasks...")
	myApp.Shutdown()
}

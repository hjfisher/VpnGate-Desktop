package main

import (
	"os"
	"path/filepath"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"vpngate/internal/controller"
	"vpngate/internal/ui"
)

func main() {
	a := app.NewWithID("net.hjfisher.vpngate")
	w := a.NewWindow("VPN Gate — Desktop")
	w.Resize(fyne.NewSize(1080, 720))

	ctrl := controller.New(a, configDir())
	ctrl.ApplyTheme()

	mainUI := ui.NewMain(w, ctrl)
	w.SetContent(mainUI.Content())
	ctrl.SetOnUpdate(mainUI.Refresh)

	// Kick off an initial network refresh a moment after the window opens so
	// previously cached servers render instantly.
	go func() {
		time.Sleep(250 * time.Millisecond)
		fyne.Do(ctrl.Refresh)
	}()

	w.ShowAndRun()
}

func configDir() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "vpngate"
	}
	return filepath.Join(dir, "vpngate")
}
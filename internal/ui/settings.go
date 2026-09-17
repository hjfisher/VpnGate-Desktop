package ui

import (
	"fmt"
	"os"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"vpngate/internal/controller"
	"vpngate/internal/net"
)

// settingsWindow edits theme, sort default, auto-refresh, mirror and the
// export folder. Everything is persisted immediately through the controller.
type settingsWindow struct {
	ctrl *controller.Controller
	win  fyne.Window
}

// NewSettings creates a settings launcher bound to the app.
func NewSettings(ctrl *controller.Controller) *settingsWindow {
	return &settingsWindow{ctrl: ctrl}
}

// Open shows the settings window (single instance per main window).
func (sw *settingsWindow) Open() {
	if sw.win == nil {
		sw.win = sw.ctrl.App().NewWindow("Settings")
		sw.win.SetOnClosed(func() {
			sw.win = nil
		})
	}
	sw.win.SetContent(sw.Content())
	sw.win.Resize(fyne.NewSize(540, 500))
	sw.win.CenterOnScreen()
	sw.win.Show()
}

func (sw *settingsWindow) Content() fyne.CanvasObject {
	s := sw.ctrl.Settings()

	themeSel := widget.NewSelect([]string{"System", "Light", "Dark"}, func(v string) {
		settings := sw.ctrl.Settings()
		switch v {
		case "Light":
			settings.Theme = "light"
		case "Dark":
			settings.Theme = "dark"
		default:
			settings.Theme = "system"
		}
		sw.ctrl.UpdateSettings(settings)
		sw.ctrl.ApplyTheme()
	})
	themeSel.SetSelected(themeLabel(s.Theme))

	sortSel := widget.NewSelect([]string{"Score", "Ping", "Speed", "Sessions"}, func(v string) {
		switch v {
		case "Ping":
			sw.ctrl.SetSort(controller.SortPing)
		case "Speed":
			sw.ctrl.SetSort(controller.SortSpeed)
		case "Sessions":
			sw.ctrl.SetSort(controller.SortSessions)
		default:
			sw.ctrl.SetSort(controller.SortScore)
		}
	})
	sortSel.SetSelected(sortLabel(s.SortBy))

	refreshSel := widget.NewSelect([]string{"Off", "Every 10 min", "Every 30 min", "Every 60 min"}, func(v string) {
		minutes := 0
		switch v {
		case "Every 10 min":
			minutes = 10
		case "Every 30 min":
			minutes = 30
		case "Every 60 min":
			minutes = 60
		}
		settings := sw.ctrl.Settings()
		settings.AutoRefreshMinutes = minutes
		sw.ctrl.UpdateSettings(settings)
	})
	refreshSel.SetSelected(refreshLabel(s.AutoRefreshMinutes))

	mirrorCheck := widget.NewCheck("Use mirror endpoint (r.jina.ai) when the main site is blocked", func(on bool) {
		settings := sw.ctrl.Settings()
		settings.UseMirror = on
		sw.ctrl.UpdateSettings(settings)
	})
	mirrorCheck.Checked = s.UseMirror
	mirrorCheck.Refresh()

	folder := s.ExportFolder
	if folder == "" {
		folder = net.DefaultExportFolder()
	}
	folderLabel := widget.NewLabel(folder)
	folderLabel.Wrapping = fyne.TextWrapWord

	chooseBtn := widget.NewButton("Choose folder", func() {
		dlg := dialog.NewFolderOpen(func(uri fyne.ListableURI, err error) {
			if err != nil || uri == nil || uri.Path() == "" {
				return
			}
			if fi, err := os.Stat(uri.Path()); err != nil || !fi.IsDir() {
				return
			}
			sw.ctrl.SetExportFolder(uri.Path())
			folderLabel.SetText(uri.Path())
		}, sw.win)
		dlg.Show()
	})
	resetBtn := widget.NewButton("Use default", func() {
		sw.ctrl.SetExportFolder("")
		folderLabel.SetText(net.DefaultExportFolder())
	})

	clearBtn := widget.NewButton("Clear saved servers", func() {
		dialog.ShowConfirm(
			"Clear saved servers",
			"Remove ALL downloaded server configs from this device?",
			func(ok bool) {
				if ok {
					sw.ctrl.ClearCache()
				}
			},
			sw.win,
		)
	})
	clearBtn.Importance = widget.DangerImportance

	status := widget.NewLabel("")
	if last := sw.ctrl.LastUpdateTime(); !last.IsZero() {
		status.SetText(fmt.Sprintf("Last update: %s  •  %d servers cached",
			formatUpdateTime(last), sw.ctrl.VisibleCount()))
	}
	status.Alignment = fyne.TextAlignCenter

	form := widget.NewForm(
		widget.NewFormItem("Theme", themeSel),
		widget.NewFormItem("Default sort", sortSel),
		widget.NewFormItem("Auto refresh", refreshSel),
		widget.NewFormItem("Network", mirrorCheck),
		widget.NewFormItem("Export folder", container.NewVBox(folderLabel, container.NewHBox(chooseBtn, resetBtn))),
	)

	return container.NewBorder(
		nil, nil,
		container.NewPadded(form),
		nil,
		container.NewBorder(
			nil, container.NewPadded(container.NewCenter(status)),
			nil, nil,
			container.NewPadded(container.NewVBox(clearBtn)),
		),
	)
}

func themeLabel(s string) string {
	switch s {
	case "light":
		return "Light"
	case "dark":
		return "Dark"
	default:
		return "System"
	}
}

func sortLabel(s string) string {
	switch s {
	case "ping":
		return "Ping"
	case "speed":
		return "Speed"
	case "sessions":
		return "Sessions"
	default:
		return "Score"
	}
}

func refreshLabel(minutes int) string {
	switch minutes {
	case 10:
		return "Every 10 min"
	case 30:
		return "Every 30 min"
	case 60:
		return "Every 60 min"
	default:
		return "Off"
	}
}
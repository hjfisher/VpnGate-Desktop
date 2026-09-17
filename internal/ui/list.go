package ui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"vpngate/internal/controller"
)

var sortNames = []string{"Score", "Ping", "Speed", "Sessions"}

// mainUI wires the main window: filter toolbar, selection toolbar,
// status bar and the (scrollable) server list.
type mainUI struct {
	win  fyne.Window
	ctrl *controller.Controller

	filterRow     *fyne.Container
	selRow        *fyne.Container
	statusRow     *fyne.Container
	listBox       *fyne.Container
	scroll        *container.Scroll
	emptyContent  *fyne.Container

	refreshBtn  *widget.Button
	searchEntry *widget.Entry
	countrySel  *widget.Select
	sortSel     *widget.Select
	ascBtn      *widget.Button
	favCheck    *widget.Check
	selCheck    *widget.Check

	selCountLabel *widget.Label
	selectAllBtn  *widget.Button

	offlineLabel *widget.Label
	infoLabel    *widget.Label
	progress     *widget.ProgressBarInfinite

	detail   fyne.Window
	settings *settingsWindow
}

func NewMain(win fyne.Window, ctrl *controller.Controller) *mainUI {
	m := &mainUI{win: win, ctrl: ctrl}
	return m
}

// Content builds the window body. A container returned once; updates mutate
// the shared widgets.
func (m *mainUI) Content() fyne.CanvasObject {
	m.refreshBtn = widget.NewButtonWithIcon("Refresh", theme.ViewRefreshIcon(), m.ctrl.Refresh)
	m.searchEntry = widget.NewEntry()
	m.searchEntry.SetPlaceHolder("Search host / IP / country")
	m.searchEntry.OnChanged = m.ctrl.SetSearch

	m.countrySel = widget.NewSelect(nil, m.onCountry)
	m.countrySel.PlaceHolder = "All countries"

	m.sortSel = widget.NewSelect([]string{"Score", "Ping", "Speed", "Sessions"}, m.onSort)
	m.sortSel.PlaceHolder = "Score"

	m.ascBtn = widget.NewButton("▲", m.ctrl.ToggleAscending)
	m.ascBtn.Importance = widget.MediumImportance

	m.favCheck = widget.NewCheck("Favorites", m.onFav)
	m.selCheck = widget.NewCheck("Select", m.onSelectMode)

	m.listBox = container.NewVBox()
	m.scroll = container.NewVScroll(m.listBox)
	m.scroll.SetMinSize(fyne.NewSize(600, 300))

	// Selection toolbar (hidden by default)
	m.selCountLabel = widget.NewLabel("")
	m.selectAllBtn = widget.NewButton("Select all", m.ctrl.SelectVisible)
	exportBtn := widget.NewButton("Export .ovpn", func() { exportSelectedAction(m.ctrl, m.win) })
	deleteBtn := widget.NewButton("Delete", m.onDeleteSelected)
	deleteBtn.Importance = widget.DangerImportance
	cancelBtn := widget.NewButton("Cancel", func() { m.ctrl.SetSelectionMode(false) })
	m.selRow = container.NewVBox(
		container.NewHBox(m.selCountLabel, layout.NewSpacer(), m.selectAllBtn, exportBtn, deleteBtn, cancelBtn),
	)
	m.selRow.Hide()

	m.filterRow = container.NewHBox(
		m.refreshBtn,
		m.searchEntry,
		m.countrySel,
		m.sortSel,
		m.ascBtn,
		m.favCheck,
		m.selCheck,
		widget.NewButtonWithIcon("Settings", theme.SettingsIcon(), func() { m.openSettings() }),
	)

	// Status bar
	m.offlineLabel = widget.NewLabel("")
	m.infoLabel = widget.NewLabel("")
	m.progress = widget.NewProgressBarInfinite()
	m.progress.Hide()
	m.statusRow = container.NewBorder(
		nil, nil,
		container.NewHBox(m.offlineLabel, m.progress),
		m.infoLabel,
		container.NewCenter(widget.NewLabel("")),
	)

	m.emptyContent = m.buildEmpty()

	return container.NewBorder(
		container.NewVBox(m.filterRow, m.selRow),
		m.statusRow,
		nil, nil,
		m.scroll,
	)
}

func (m *mainUI) buildEmpty() *fyne.Container {
	title := widget.NewLabelWithStyle("No servers yet", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	hint := widget.NewLabel("The list stays saved locally — ask for the latest servers with Refresh.")
	hint.Alignment = fyne.TextAlignCenter
	btn := widget.NewButtonWithIcon("Refresh now", theme.ViewRefreshIcon(), m.ctrl.Refresh)
	box := container.NewCenter(container.NewVBox(title, hint, btn))
	return container.New(layout.NewStackLayout(), box)
}

// Refresh rebuilds the visible list and toolbar state. Runs on the main thread.
func (m *mainUI) Refresh() {
	m.countrySel.Options = m.ctrl.Countries()
	m.countrySel.Refresh()
	m.sortSel.Options = sortNames
	m.sortSel.Refresh()

	servers := m.ctrl.Servers()
	m.listBox.Objects = m.listBox.Objects[:0]
	if len(servers) == 0 {
		m.listBox.Objects = append(m.listBox.Objects, m.emptyContent)
	} else {
		for _, sv := range servers {
			if sv.IsBlank() {
				continue
			}
			sv := sv
			row := newServerRow(m.ctrl, m.win, sv, func() { m.showDetail(sv) })
			m.listBox.Objects = append(m.listBox.Objects, row)
		}
	}
	m.listBox.Refresh()

	// Selection toolbar
	selMode := m.ctrl.SelectionMode()
	if selMode {
		m.selectAllBtn.SetText("Select visible (" + itoa64(int64(m.ctrl.VisibleCount())) + ")")
		m.selRow.Show()
	} else {
		m.selRow.Hide()
	}
	m.selCheck.Checked = selMode
	m.selCountLabel.SetText(fmt.Sprintf("%d selected", m.ctrl.SelectedCount()))

	// Status bar
	if m.ctrl.IsRefreshing() {
		m.progress.Show()
		m.progress.Start()
	} else {
		m.progress.Stop()
		m.progress.Hide()
	}
	if m.ctrl.IsOffline() {
		m.offlineLabel.SetText("Offline — showing saved data")
		m.offlineLabel.Importance = widget.WarningImportance
		m.offlineLabel.Show()
	} else {
		m.offlineLabel.Hide()
	}
	m.infoLabel.SetText(fmt.Sprintf("%d servers  •  updated %s",
		m.ctrl.VisibleCount(), formatUpdateTime(m.ctrl.LastUpdateTime())))
	m.infoLabel.Alignment = fyne.TextAlignTrailing
	m.statusRow.Refresh()
}

// checkRowVisible shows/hides a toolbar row and forces a relayout.
func (m *mainUI) onCountry(country string) { m.ctrl.SetCountry(country) }
func (m *mainUI) onSort(value string) {
	switch value {
	case "Ping":
		m.ctrl.SetSort(controller.SortPing)
	case "Speed":
		m.ctrl.SetSort(controller.SortSpeed)
	case "Sessions":
		m.ctrl.SetSort(controller.SortSessions)
	default:
		m.ctrl.SetSort(controller.SortScore)
	}
}
func (m *mainUI) onFav(on bool)            { c := m.ctrl; c.SetFavoritesOnly(on) }
func (m *mainUI) onSelectMode(on bool)     { m.ctrl.SetSelectionMode(on) }
func (m *mainUI) onDeleteSelected()        { m.onDeleteConfirm(m.ctrl.SelectedCount()) }

func (m *mainUI) onDeleteConfirm(count int) {
	dialog.ShowConfirm(
		"Delete servers",
		fmt.Sprintf("Permanently remove %d server(s) from the saved list?", count),
		func(ok bool) {
			if ok {
				m.ctrl.DeleteSelected()
			}
		},
		m.win,
	)
}

// openSettings shows the settings window once and keeps it alive.
func (m *mainUI) openSettings() {
	if m.settings == nil {
		m.settings = NewSettings(m.ctrl)
	}
	m.settings.Open()
}
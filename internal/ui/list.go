package ui

import (
	"fmt"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"vpngate/internal/controller"
	"vpngate/internal/data"
)

// mainUI wires the main window: filter toolbar, selection toolbar,
// status bar and the (scrollable) server list.
type mainUI struct {
	win  fyne.Window
	ctrl *controller.Controller

	filterRow      *fyne.Container
	selRow         *fyne.Container
	statusRow      *fyne.Container
	list           *widget.List
	scroll         *container.Scroll
	emptyContent   *fyne.Container
	filteredServers []data.VpnServer

	refreshBtn  *widget.Button
	searchEntry *widget.Entry
	clearBtn    *widget.Button
	searchTimer *time.Timer
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

	detail     fyne.Window
	detailHost string
	settings   *settingsWindow
}

func NewMain(win fyne.Window, ctrl *controller.Controller) *mainUI {
	m := &mainUI{win: win, ctrl: ctrl}
	return m
}

// SetCountryFilterShown manages server list visibility based on whether a country filter is active.
// Status bar is always visible.
func (m *mainUI) SetCountryFilterShown(shown bool) {
	if shown {
		// Show the server list when a specific country is selected
	} else {
		// Show empty state when "All" is selected (no filter)
	}
	m.statusRow.Refresh()
}

// Content builds the window body. A container returned once; updates mutate
// the shared widgets.
func (m *mainUI) Content() fyne.CanvasObject {
	m.refreshBtn = widget.NewButtonWithIcon("Refresh", theme.ViewRefreshIcon(), m.ctrl.Refresh)
	m.searchEntry = widget.NewEntry()
	m.searchEntry.SetPlaceHolder("Search host / IP / country")
	m.searchEntry.OnChanged = m.onSearchChanged
	m.clearBtn = widget.NewButtonWithIcon("", theme.CancelIcon(), m.clearSearch)
	m.clearBtn.Hide()

	m.countrySel = widget.NewSelect(nil, m.onCountry)
	m.countrySel.PlaceHolder = "All countries"
	// Add invisible "All" option as first element; actual countries populated below
	m.countrySel.Options = []string{"All"}
	m.countrySel.Refresh()

	m.sortSel = widget.NewSelect(controller.SortLabels(), m.onSort)
	m.sortSel.PlaceHolder = "Score"

	m.ascBtn = widget.NewButton("▲", m.ctrl.ToggleAscending)
	m.ascBtn.Importance = widget.MediumImportance

	m.favCheck = widget.NewCheck("Favorites", m.onFav)
	m.selCheck = widget.NewCheck("Select", m.onSelectMode)

m.list = widget.NewList(
	func() int { return len(m.filteredServers) },
	func() fyne.CanvasObject {
		// Create a template row with representative data
		// so its measured height matches real rows.
		// Use a realistic server to ensure MinSize() is
		// conservative enough for all visible rows.
		return newServerRow(m.ctrl, m.win, data.VpnServer{
			CountryLong: "United States",
			CountryShort: "US",
			HostName:    "us1.vpngate.net",
			IP:          "192.168.1.1",
			Score:       1234567,
			Ping:        123,
			ProtoType:   "TCP/UDP",
		}, func() {})
	},
	func(li widget.ListItemID, o fyne.CanvasObject) {
			i := int(li)
			if i < 0 || i >= len(m.filteredServers) {
				return
			}
			row := o.(*serverRow)
			row.server = m.filteredServers[i]
			row.onOpen = func() { m.showDetail(m.filteredServers[i]) }
			row.Refresh()
		},
	)
	m.scroll = container.NewVScroll(m.list)
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

	m.emptyContent = m.buildEmpty()

	// Search box (with clear button attached) - takes remaining space on right
	searchBox := container.NewBorder(
		nil, nil,
		nil,
		m.clearBtn,  // Clear button on the right
		m.searchEntry,  // Search entry takes remaining space
	)

	// Top toolbar with filters on the left
	filterControls := container.NewHBox(
		m.refreshBtn,
		m.countrySel,
		m.sortSel,
		m.ascBtn,
		m.favCheck,
		m.selCheck,
		widget.NewButtonWithIcon("Settings", theme.SettingsIcon(), func() { m.openSettings() }),
	)

	// Combined: controls on left, search box takes remaining space on right
	m.filterRow = container.NewBorder(
		nil, nil,
		filterControls,  // All controls on left
		nil,
		searchBox,  // Search takes all remaining space on right
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
		m.list,
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
	m.countrySel.Options = append([]string{"All"}, m.ctrl.Countries()...)
	m.countrySel.Refresh()
	if country := m.ctrl.CountryFilter(); country != "" {
		m.countrySel.SetSelected(country)
	}
	m.sortSel.Options = controller.SortLabels()
	m.sortSel.SetSelected(controller.SortLabel(m.ctrl.Sort()))
	m.sortSel.Refresh()

	if m.ctrl.Ascending() {
		m.ascBtn.SetText("▲")
	} else {
		m.ascBtn.SetText("▼")
	}

	// Show only servers matching the country filter (or "All")
	servers := m.ctrl.Servers()
	filtered := servers
	if country := m.ctrl.CountryFilter(); country != "" {
		filtered = nil
		for _, sv := range servers {
			if sv.CountryLong == country {
				filtered = append(filtered, sv)
			}
		}
	}

	m.filteredServers = filtered
	m.list.Refresh()

	// Show/hide empty state
	if len(m.filteredServers) == 0 {
		m.emptyContent.Show()
	} else {
		m.emptyContent.Hide()
	}

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
func (m *mainUI) onSort(value string)      { m.ctrl.SetSort(controller.SortFromLabel(value)) }

// onSearchChanged debounces search input so the filtered list only rebuilds
// after the user pauses typing, avoiding per-keystroke work over 1000+ rows.
func (m *mainUI) onSearchChanged(_ string) {
	if text := m.searchEntry.Text; text != "" {
		m.clearBtn.Show()
	} else {
		m.clearBtn.Hide()
	}
	if m.searchTimer != nil {
		m.searchTimer.Stop()
	}
	m.searchTimer = time.AfterFunc(250*time.Millisecond, func() {
		fyne.Do(func() {
			m.ctrl.SetSearch(m.searchEntry.Text)
		})
	})
}

// clearSearch resets both the search entry text and the controller filter.
func (m *mainUI) clearSearch() {
	m.searchEntry.SetText("")
	m.clearBtn.Hide()
	if m.searchTimer != nil {
		m.searchTimer.Stop()
		m.searchTimer = nil
	}
	m.ctrl.SetSearch("")
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
		m.settings = NewSettings(m.ctrl, m.win)
	}
	m.settings.Open()
}
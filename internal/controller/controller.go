package controller

import (
	"errors"
	"sort"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
	"vpngate/internal/data"
	"vpngate/internal/net"
)

type SortBy int

const (
	SortScore SortBy = iota
	SortPing
	SortSpeed
	SortSessions
)

// Controller holds all application state and drives background work
// (fetching, ping, refresh timer). UI mutations happen on the main thread;
// goroutines post results back through fyne.Do.
type Controller struct {
	app fyne.App

	store         *data.Store
	settingsStore *data.SettingsStore
	settings      data.AppSettings

	servers    []data.VpnServer
	countries  []string
	favorites  map[string]bool
	pingResults map[string]int64

	search        string
	countryFilter string // "" = all
	sort          SortBy
	ascending     bool
	favoritesOnly bool

	selectionMode bool
	selected      map[string]bool

	refreshing bool
	offline    bool
	lastUpdate time.Time

	onUpdate func()
	stopTick chan struct{}
}

func New(app fyne.App, configDir string) *Controller {
	s := data.NewSettingsStore(configDir)
	cfg := s.Load()
	c := &Controller{
		app:           app,
		store:         data.NewStore(configDir),
		settingsStore: s,
		settings:      cfg,
		favorites:     stringSet(cfg.Favorites),
		pingResults:   map[string]int64{},
		selected:      map[string]bool{},
		sort:          sortFromString(cfg.SortBy),
		ascending:     cfg.SortAscending,
		stopTick:      make(chan struct{}),
	}
	c.servers = c.store.Load()
	c.rebuildCountries()
	if len(c.servers) > 0 {
		c.offline = true
	}
	c.restartAutoRefresh(cfg.AutoRefreshMinutes)
	return c
}

// SetOnUpdate registers the callback invoked after any state change.
// The callback must be safe to run on the main thread.
func (c *Controller) SetOnUpdate(fn func()) { c.onUpdate = fn }

func (c *Controller) update() {
	if c.app != nil {
		fyne.Do(func() {
			if c.onUpdate != nil {
				c.onUpdate()
			}
		})
	}
}

// --- App / data getters ---

func (c *Controller) App() fyne.App                  { return c.app }
func (c *Controller) Settings() data.AppSettings     { return c.settings }
func (c *Controller) LastUpdateTime() time.Time      { return c.lastUpdate }
func (c *Controller) IsRefreshing() bool             { return c.refreshing }
func (c *Controller) IsOffline() bool                { return c.offline }
func (c *Controller) SelectionMode() bool            { return c.selectionMode }
func (c *Controller) SelectedCount() int             { return len(c.selected) }
func (c *Controller) Countries() []string            { return c.countries }
func (c *Controller) IsFavorite(host string) bool    { return c.favorites[host] }
func (c *Controller) PingResult(host string) (int64, bool) {
	v, ok := c.pingResults[host]
	return v, ok
}

// Servers returns the currently visible (filtered + sorted) list.
func (c *Controller) Servers() []data.VpnServer {
	list := c.visible()
	out := make([]data.VpnServer, len(list))
	copy(out, list)
	return out
}

// FindServer returns a server by host name and whether it was found.
func (c *Controller) FindServer(host string) (data.VpnServer, bool) {
	for _, s := range c.servers {
		if s.HostName == host {
			return s, true
		}
	}
	return data.VpnServer{}, false
}

// --- Fetch / refresh ---

func (c *Controller) Refresh() {
	if c.refreshing {
		return
	}
	c.refreshing = true
	c.update()
	api := data.NewVpnGateApi(c.settings.UseMirror)
	go func() {
		defer fyne.Do(func() {
			c.refreshing = false
			c.update()
		})
		csv, err := api.FetchRawCSV()
		if err != nil {
			fyne.Do(func() {
				if len(c.servers) == 0 {
					c.offline = true
				}
				c.notifyError(err)
			})
			return
		}
// Parse incrementally and update UI every 50 servers
	batch := make([]data.VpnServer, 0, 50)
	parser := data.VpnGateParser{}
	err = parser.ParseStream(csv, func(s data.VpnServer) bool {
			batch = append(batch, s)
			if len(batch) >= 50 {
				fyne.Do(func() {
					merged, _ := c.store.Merge(batch)
					c.servers = merged
					c.rebuildCountries()
					c.update()
				})
				batch = batch[:0]
			}
			return true
		})
		// Flush remaining
		if len(batch) > 0 {
			fyne.Do(func() {
				merged, _ := c.store.Merge(batch)
				c.servers = merged
				c.lastUpdate = time.Now()
				c.offline = false
				c.rebuildCountries()
				c.update()
			})
		}
		if err != nil {
			fyne.Do(func() {
				c.notifyError(err)
			})
		}
	}()
}

func (c *Controller) ClearCache() {
	c.store.Clear()
	c.servers = nil
	c.rebuildCountries()
	c.offline = false
	c.lastUpdate = time.Time{}
	c.selectionMode = false
	c.selected = map[string]bool{}
	c.pingResults = map[string]int64{}
	c.update()
}

// --- Search / filter / sort ---

func (c *Controller) SetSearch(q string) {
	c.search = strings.TrimSpace(q)
	c.update()
}

func (c *Controller) SetCountry(country string) {
	c.countryFilter = country
	c.update()
}

func (c *Controller) SetSort(sortBy SortBy) {
	c.sort = sortBy
	c.settings.SortBy = sortString(sortBy)
	c.saveSettings()
	c.update()
}

func (c *Controller) ToggleAscending() {
	c.ascending = !c.ascending
	c.settings.SortAscending = c.ascending
	c.saveSettings()
	c.update()
}

func (c *Controller) ToggleFavoritesOnly() {
	c.favoritesOnly = !c.favoritesOnly
	c.update()
}

// SetFavoritesOnly enables/disables the favorites-only filter.
func (c *Controller) SetFavoritesOnly(on bool) {
	c.favoritesOnly = on
	c.update()
}

// --- Selection ---

func (c *Controller) SetSelectionMode(on bool) {
	c.selectionMode = on
	if !on {
		c.selected = map[string]bool{}
	}
	c.update()
}

func (c *Controller) ToggleSelect(host string) {
	if c.selected[host] {
		delete(c.selected, host)
	} else {
		c.selected[host] = true
	}
	c.update()
}

func (c *Controller) IsSelected(host string) bool { return c.selected[host] }

func (c *Controller) SelectOnly(host string) {
	c.selected = map[string]bool{host: true}
	c.update()
}

func (c *Controller) Unselect(host string) {
	delete(c.selected, host)
	c.update()
}

func (c *Controller) SelectVisible() {
	for _, s := range c.visible() {
		c.selected[s.HostName] = true
	}
	c.update()
}

func (c *Controller) ClearSelection() {
	c.selected = map[string]bool{}
	c.update()
}

func (c *Controller) DeleteSelected() {
	ips := map[string]struct{}{}
	for host := range c.selected {
		if s, ok := c.FindServer(host); ok {
			ips[s.IP] = struct{}{}
		}
	}
	if len(ips) > 0 {
		c.servers = c.store.Delete(ips)
		c.rebuildCountries()
	}
	c.selected = map[string]bool{}
	c.selectionMode = false
	c.update()
}

func (c *Controller) DeleteServer(host string) {
	ips := map[string]struct{}{}
	if s, ok := c.FindServer(host); ok {
		ips[s.IP] = struct{}{}
		c.servers = c.store.Delete(ips)
		delete(c.favorites, host)
		c.rebuildCountries()
	}
	c.selected = map[string]bool{}
	c.selectionMode = false
	c.update()
}

// --- Favorites ---

func (c *Controller) ToggleFavorite(host string) {
	if c.favorites[host] {
		delete(c.favorites, host)
	} else {
		c.favorites[host] = true
	}
	c.persistFavorites()
	c.update()
}

// --- Ping ---

// Ping measures TCP latency to a server. done (when non-nil) runs on the main
// thread with the result (negative = unreachable).
func (c *Controller) Ping(host string, done func(ms int64)) {
	s, ok := c.FindServer(host)
	if !ok {
		if done != nil {
			done(-1)
		}
		return
	}
	go func() {
		ms := net.Ping(s.IP, 443, 4*time.Second)
		fyne.Do(func() {
			c.pingResults[host] = ms
			c.update()
			if done != nil {
				done(ms)
			}
		})
	}()
}

// --- Export / connect ---

func (c *Controller) ExportSelected() net.ExportOutcome {
	var selected []data.VpnServer
	for host := range c.selected {
		if s, ok := c.FindServer(host); ok {
			selected = append(selected, s)
		}
	}
	return c.export(selected)
}

func (c *Controller) ExportServer(host string) net.ExportOutcome {
	s, ok := c.FindServer(host)
	if !ok {
		return net.ExportOutcome{Error: ErrServerNotFound}
	}
	return c.export([]data.VpnServer{s})
}

func (c *Controller) export(servers []data.VpnServer) net.ExportOutcome {
	if len(servers) == 0 {
		return net.ExportOutcome{Error: ErrNoSelection}
	}
	folder := c.settings.ExportFolder
	if folder == "" {
		folder = net.DefaultExportFolder()
	}
	return net.ExportOVPN(folder, servers)
}

// ExportFolder is the folder .ovpn files are written to ("" = default).
func (c *Controller) ExportFolder() string { return c.settings.ExportFolder }

func (c *Controller) SetExportFolder(folder string) {
	c.settings.ExportFolder = folder
	c.saveSettings()
	c.update()
}

// Connect saves the config to the export folder and hands it to the OS
// default app (an installed OpenVPN client). Returns the written path.
func (c *Controller) Connect(host string) (string, error) {
	s, ok := c.FindServer(host)
	if !ok {
		return "", ErrServerNotFound
	}
	folder := c.settings.ExportFolder
	if folder == "" {
		folder = net.DefaultExportFolder()
	}
	path, err := net.SaveConfig(folder, s)
	if err != nil {
		return "", err
	}
	if err := net.OpenFile(path); err != nil {
		return path, err
	}
	return path, nil
}

// CopyConfig returns the decoded config text for clipboard use.
func (c *Controller) CopyConfig(host string) (string, error) {
	s, ok := c.FindServer(host)
	if !ok {
		return "", ErrServerNotFound
	}
	config := s.OpenVPNConfig()
	if config == "" {
		return "", ErrNoConfig
	}
	return config, nil
}

// VisibleServersInSelection counts currently shown servers (for button labels).
func (c *Controller) VisibleCount() int { return len(c.visible()) }

// --- Settings ---

func (c *Controller) UpdateSettings(s data.AppSettings) {
	c.settings = s
	c.favorites = stringSet(s.Favorites)
	c.sort = sortFromString(s.SortBy)
	c.ascending = s.SortAscending
	c.saveSettings()
	c.restartAutoRefresh(s.AutoRefreshMinutes)
	c.update()
}

// ApplyTheme applies the persisted light/dark/system setting to the app.
func (c *Controller) ApplyTheme() {
	switch c.settings.Theme {
	case "light":
		c.app.Settings().SetTheme(theme.LightTheme())
	case "dark":
		c.app.Settings().SetTheme(theme.DarkTheme())
	default:
		c.app.Settings().SetTheme(nil) // follow system
	}
}

func (c *Controller) saveSettings() {
	c.settings.Favorites = sortedKeys(c.favorites)
	_ = c.settingsStore.Save(c.settings)
}

func (c *Controller) persistFavorites() {
	c.settings.Favorites = sortedKeys(c.favorites)
	_ = c.settingsStore.Save(c.settings)
}

func (c *Controller) restartAutoRefresh(minutes int) {
	// Stop any previous ticker goroutine by closing its channel.
	select {
	case <-c.stopTick:
		// already stopped / closed
	default:
		close(c.stopTick)
	}
	if minutes <= 0 {
		return
	}
	stop := make(chan struct{})
	c.stopTick = stop
	go func() {
		tick := time.NewTicker(time.Duration(minutes) * time.Minute)
		defer tick.Stop()
		for {
			select {
			case <-tick.C:
				fyne.Do(c.Refresh)
			case <-stop:
				return
			}
		}
	}()
}

func (c *Controller) visible() []data.VpnServer {
	list := make([]data.VpnServer, 0, len(c.servers))
	for _, s := range c.servers {
		if c.favoritesOnly && !c.favorites[s.HostName] {
			continue
		}
		if c.countryFilter != "" && s.CountryLong != c.countryFilter {
			continue
		}
		if c.search != "" && !matches(s, c.search) {
			continue
		}
		list = append(list, s)
	}
	sortServerList(list, c.sort, c.ascending)
	return list
}

func matches(s data.VpnServer, q string) bool {
	q = strings.ToLower(q)
	return strings.Contains(strings.ToLower(s.HostName), q) ||
		strings.Contains(strings.ToLower(s.IP), q) ||
		strings.Contains(strings.ToLower(s.CountryLong), q) ||
		strings.Contains(strings.ToLower(s.CountryShort), q) ||
		strings.Contains(strings.ToLower(s.Operator), q)
}

func sortServerList(list []data.VpnServer, by SortBy, ascending bool) {
	less := func(i, j int) bool {
		a, b := list[i], list[j]
		switch by {
		case SortPing:
			return a.Ping < b.Ping
		case SortSpeed:
			return a.Speed < b.Speed
		case SortSessions:
			return a.NumVpnSessions < b.NumVpnSessions
		default:
			return a.Score < b.Score
		}
	}
	if ascending {
		sort.SliceStable(list, less)
	} else {
		sort.SliceStable(list, func(i, j int) bool { return less(j, i) })
	}
}

func (c *Controller) rebuildCountries() {
	seen := map[string]struct{}{}
	for _, s := range c.servers {
		if s.CountryLong != "" {
			seen[s.CountryLong] = struct{}{}
		}
	}
	c.countries = sortedKeys(seen)
}

// --- Helpers / errors ---

var (
	ErrServerNotFound = errors.New("server not found")
	ErrNoSelection    = errors.New("no servers selected")
	ErrNoConfig       = errors.New("no readable config")
)

func (c *Controller) notifyError(err error) {
	// Surface network errors through a notification when nothing is cached.
	if len(c.servers) > 0 {
		return
	}
	if c.app != nil {
		c.app.SendNotification(&fyne.Notification{
			Title:   "VPN Gate",
			Content: "Could not reach vpngate.net — check your connection.",
		})
	}
}

func stringSet(items []string) map[string]bool {
	out := make(map[string]bool, len(items))
	for _, it := range items {
		out[it] = true
	}
	return out
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func sortFromString(s string) SortBy {
	switch s {
	case "ping":
		return SortPing
	case "speed":
		return SortSpeed
	case "sessions":
		return SortSessions
	default:
		return SortScore
	}
}

func sortString(by SortBy) string {
	switch by {
	case SortPing:
		return "ping"
	case SortSpeed:
		return "speed"
	case SortSessions:
		return "sessions"
	default:
		return "score"
	}
}
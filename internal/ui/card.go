package ui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"vpngate/internal/controller"
	"vpngate/internal/data"
)

// serverRow is one card in the server list. The whole card is tappable
// (opens the detail window); the small Connect/Favorite buttons inside
// handle their own taps first.
type serverRow struct {
	widget.BaseWidget
	ctrl   *controller.Controller
	server data.VpnServer
	win    fyne.Window
	onOpen func()

	// Persistent widget references for content updates (created once in CreateRenderer)
	countryLabel *widget.Label
	hostLabel    *widget.Label
	ipLabel      *widget.Label
	scoreText    *canvas.Text
	pingText     *canvas.Text
	protoText    *canvas.Text
	favBtn       *widget.Button
	connectBtn   *widget.Button
	badgeStack   *fyne.Container // stack containing circle + label
	badgeLabel   *canvas.Text
	selCheck     *widget.Check
}

func newServerRow(ctrl *controller.Controller, parent fyne.Window, sv data.VpnServer, onOpen func()) *serverRow {
	r := &serverRow{ctrl: ctrl, server: sv, win: parent, onOpen: onOpen}
	return r
}

func (r *serverRow) Tapped(*fyne.PointEvent) { r.onOpen() }

var _ fyne.Tappable = (*serverRow)(nil)

// CreateRenderer implements fyne.Widget.
func (r *serverRow) CreateRenderer() fyne.WidgetRenderer {
	selection := r.ctrl.SelectionMode()
	selected := selection && r.ctrl.IsSelected(r.server.HostName)

	bg := canvas.NewRectangle(theme.BackgroundColor())
	if selected {
		bg = canvas.NewRectangle(softPrimary())
	}

	objects := []fyne.CanvasObject{bg}

	// Selection checkbox - always created, visibility controlled by updateContentFromServer()
	check := widget.NewCheck("", func(on bool) {
		if on {
			r.ctrl.SelectOnly(r.server.HostName)
		} else {
			r.ctrl.Unselect(r.server.HostName)
		}
	})
	check.Checked = selected
	check.Refresh()
	objects = append(objects, check)
	r.selCheck = check

	// Build badge, center, right with persistent widget references
	badgeStack, badgeLabel := makeBadgeInternal(r.server.CountryShort)
	r.badgeStack = badgeStack
	r.badgeLabel = badgeLabel

	center := makeCenter(r.server, r)
	right := makeRight(r.ctrl, r.win, r.server, r)

	objects = append(objects, r.badgeStack, center, right)
	return &rowRenderer{row: r, objects: objects}
}

// scoreColor returns a color for the score text based on the score value.
func scoreColor(score int64) color.NRGBA {
	switch {
	case score >= 1_000_000:
		return color.NRGBA{R: 22, G: 163, B: 74, A: 255}
	case score >= 100_000:
		return color.NRGBA{R: 37, G: 99, B: 235, A: 255}
	default:
		return color.NRGBA{R: 128, G: 128, B: 128, A: 255}
	}
}

// ctrlPing returns the ping milliseconds for a server, checking the controller's
// ping results first, then falling back to the server's stored ping.
func ctrlPing(ctrl *controller.Controller, sv data.VpnServer) int64 {
	if ms, ok := ctrl.PingResult(sv.HostName); ok {
		return ms
	}
	return int64(sv.Ping)
}

// pingText returns a formatted ping string like "123 ms" or "n/a".
func pingText(ctrl *controller.Controller, sv data.VpnServer) string {
	ms := ctrlPing(ctrl, sv)
	if ms <= 0 {
		return "n/a"
	}
	return itoa64(ms) + " ms"
}

// softPrimary returns a softened version of the primary theme color.
func softPrimary() color.NRGBA {
	c := theme.PrimaryColor()
	rgba := color.NRGBAModel.Convert(c).(color.NRGBA)
	return color.NRGBA{R: rgba.R, G: rgba.G, B: rgba.B, A: 28}
}

// makeBadgeInternal creates a circle badge with a country code label.
// Returns both the container and the label for persistent reference updates.
func makeBadgeInternal(code string) (*fyne.Container, *canvas.Text) {
	label := canvas.NewText(code, theme.PrimaryColorNamed("primary"))
	label.TextStyle = fyne.TextStyle{Bold: true}
	label.Alignment = fyne.TextAlignCenter
	circle := canvas.NewCircle(color.NRGBA{A: 0}) // Transparent fill, only outline
	circle.StrokeColor = theme.ShadowColor()
	circle.StrokeWidth = 1
	stack := container.NewStack(circle, container.NewCenter(label))
	return stack, label
}

// makeBadge creates a circle badge with a country code label.
// Returns just the container for backward compatibility (e.g. detail.go).
func makeBadge(code string) *fyne.Container {
	container, _ := makeBadgeInternal(code)
	return container
}

// makeCenter creates the center section of a server row (country, host, IP labels).
func makeCenter(sv data.VpnServer, r *serverRow) fyne.CanvasObject {
	country := widget.NewLabelWithStyle(sv.CountryLong, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	host := widget.NewLabelWithStyle(sv.HostName, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	ip := widget.NewLabelWithStyle(sv.IP, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	ip.Importance = widget.MediumImportance

	// Store references for later updates
	r.countryLabel = country
	r.hostLabel = host
	r.ipLabel = ip

	return container.NewVBox(country, host, ip)
}

// makeRight creates the right section of a server row (score, ping, protocol,
// favorite, connect buttons).
func makeRight(ctrl *controller.Controller, parent fyne.Window, sv data.VpnServer, r *serverRow) fyne.CanvasObject {
	score := canvas.NewText("Score "+formatScore(sv.Score), scoreColor(sv.Score))
	score.TextStyle.Bold = true
	ping := canvas.NewText("Ping "+pingText(ctrl, sv), pingColor(ctrlPing(ctrl, sv)))
	proto := canvas.NewText(sv.ProtoLabel(), theme.PrimaryColorNamed("primary"))
	proto.TextStyle.Bold = true

	r.scoreText = score
	r.pingText = ping
	r.protoText = proto

	fav := "☆"
	if ctrl.IsFavorite(sv.HostName) {
		fav = "★"
	}
	favBtn := widget.NewButton(fav, func() { ctrl.ToggleFavorite(sv.HostName) })
	favBtn.Importance = widget.MediumImportance

	connectBtn := widget.NewButton("Connect", func() { connectAction(ctrl, parent, sv) })
	connectBtn.Importance = widget.HighImportance

	r.favBtn = favBtn
	r.connectBtn = connectBtn

	top := container.NewHBox(score, layout.NewSpacer(), favBtn)
	return container.NewVBox(top, ping, proto, connectBtn)
}

// ----- persistently update widget content from current r.server -----

func (r *serverRow) updateContentFromServer() {
	sv := r.server
	selection := r.ctrl.SelectionMode()

	// Country label
	if r.countryLabel != nil {
		r.countryLabel.SetText(sv.CountryLong)
	}
	// Host label
	if r.hostLabel != nil {
		r.hostLabel.SetText(sv.HostName)
	}
	// IP label
	if r.ipLabel != nil {
		r.ipLabel.SetText(sv.IP)
		if selection {
			r.ipLabel.Hide()
		} else {
			r.ipLabel.Show()
		}
	}

	// Score
	if r.scoreText != nil {
		r.scoreText.Text = "Score " + formatScore(sv.Score)
		r.scoreText.Color = scoreColor(sv.Score)
		r.scoreText.Refresh()
	}

	// Ping
	if r.pingText != nil {
		ms := ctrlPing(r.ctrl, sv)
		if ms <= 0 {
			r.pingText.Text = "Ping n/a"
		} else {
			r.pingText.Text = "Ping " + itoa64(ms) + " ms"
		}
		r.pingText.Color = pingColor(ctrlPing(r.ctrl, sv))
		r.pingText.Refresh()
	}

	// Protocol
	if r.protoText != nil {
		r.protoText.Text = sv.ProtoLabel()
		r.protoText.Refresh()
	}

	// Favorite button
	if r.favBtn != nil {
		fav := "☆"
		if r.ctrl.IsFavorite(sv.HostName) {
			fav = "★"
		}
		r.favBtn.SetText(fav)
		r.favBtn.OnTapped = func() { r.ctrl.ToggleFavorite(sv.HostName) }
		r.favBtn.Refresh()
	}

	// Connect button
	if r.connectBtn != nil {
		r.connectBtn.OnTapped = func() { connectAction(r.ctrl, r.win, sv) }
		r.connectBtn.Refresh()
	}

	// Badge (country code circle)
	if r.badgeStack != nil && r.badgeLabel != nil {
		r.badgeLabel.Text = sv.CountryShort
		r.badgeLabel.Refresh()
	}

	// Selection checkbox
	if r.selCheck != nil {
		if selection {
			r.selCheck.Checked = r.ctrl.IsSelected(r.server.HostName)
			r.selCheck.OnChanged = func(on bool) {
				if on {
					r.ctrl.SelectOnly(r.server.HostName)
				} else {
					r.ctrl.Unselect(r.server.HostName)
				}
			}
			r.selCheck.Show()
			r.selCheck.Refresh()
		} else {
			r.selCheck.Hide()
		}
	}
}

// rowRenderer lays the card out with a flexible middle column.
type rowRenderer struct {
	row     *serverRow
	objects []fyne.CanvasObject
}

func (rr *rowRenderer) Layout(size fyne.Size) {
	const pad = float32(10)
	o := rr.objects
	bg := o[0]
	bg.Resize(size)
	bg.Move(fyne.NewPos(0, 0))

	idx := 1
	left := pad
	if rr.row.ctrl.SelectionMode() {
		check := o[idx]
		idx++
		check.Resize(fyne.NewSize(36, 36))
		check.Move(fyne.NewPos(pad, (size.Height-36)/2))
		left += 36 + pad
	}
	badge := o[idx]
	center := o[idx+1]
	right := o[idx+2]

	rightSize := right.MinSize()
	right.Resize(rightSize)
	right.Move(fyne.NewPos(size.Width-rightSize.Width-pad, (size.Height-rightSize.Height)/2))

	badgeSize := fyne.NewSize(40, 40)
	badge.Resize(badgeSize)
	badge.Move(fyne.NewPos(left, (size.Height-badgeSize.Height)/2))

	cw := size.Width - rightSize.Width - badgeSize.Width - 3*pad - left
	if cw < 80 {
		cw = 80
	}
	center.Resize(fyne.NewSize(cw, size.Height-2*pad))
	center.Move(fyne.NewPos(left+badgeSize.Width+pad, pad))
}

func (rr *rowRenderer) MinSize() fyne.Size {
	var w, h float32
	for i, o := range rr.objects {
		if i == 0 {
			continue // background fills the row
		}
		m := o.MinSize()
		w += m.Width
		if m.Height > h {
			h = m.Height
		}
	}
	return fyne.NewSize(w+48, h+20)
}

func (rr *rowRenderer) Refresh() {
	rr.row.updateContentFromServer()
	canvas.Refresh(rr.row)
}

func (rr *rowRenderer) Objects() []fyne.CanvasObject {
	return rr.objects
}
func (rr *rowRenderer) Destroy() {}
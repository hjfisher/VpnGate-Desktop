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

// serverRow is one card in the server list.
type serverRow struct {
	widget.BaseWidget
	ctrl   *controller.Controller
	server data.VpnServer
	win    fyne.Window
	onOpen func()

	// Persistent widget references
	countryLabel *widget.Label
	hostLabel    *widget.Label
	ipLabel      *widget.Label
	scoreText    *canvas.Text
	pingText     *canvas.Text
	protoText    *canvas.Text
	favBtn       *widget.Button
	connectBtn   *widget.Button
	badgeLabel   *canvas.Text
	selCheck     *widget.Check
}

func newServerRow(ctrl *controller.Controller, parent fyne.Window, sv data.VpnServer, onOpen func()) *serverRow {
	r := &serverRow{ctrl: ctrl, server: sv, win: parent, onOpen: onOpen}
	r.ExtendBaseWidget(r)
	return r
}

var _ fyne.Tappable = (*serverRow)(nil)

func (r *serverRow) Tapped(*fyne.PointEvent) { r.onOpen() }

// makeBadge creates a circle badge with a country code label.
// Returns just the container for backward compatibility (e.g. detail.go).
func makeBadge(code string) *fyne.Container {
	label := canvas.NewText(code, theme.PrimaryColorNamed("primary"))
	label.TextStyle = fyne.TextStyle{Bold: true}
	label.Alignment = fyne.TextAlignCenter
	circle := canvas.NewCircle(color.NRGBA{A: 0})
	circle.StrokeColor = theme.ShadowColor()
	circle.StrokeWidth = 1
	stack := container.NewStack(circle, container.NewCenter(label))
	return container.New(layout.NewGridWrapLayout(fyne.NewSize(40, 40)), stack)
}

func (r *serverRow) CreateRenderer() fyne.WidgetRenderer {
	bg := canvas.NewRectangle(theme.BackgroundColor())

	// Checkbox - always created at fixed index
	selCheck := widget.NewCheck("", func(on bool) {
		if on {
			r.ctrl.SelectOnly(r.server.HostName)
		} else {
			r.ctrl.Unselect(r.server.HostName)
		}
	})
	r.selCheck = selCheck

	// Badge - country code circle
	badgeLabel := canvas.NewText(r.server.CountryShort, theme.PrimaryColorNamed("primary"))
	badgeLabel.TextStyle = fyne.TextStyle{Bold: true}
	badgeLabel.Alignment = fyne.TextAlignCenter
	circle := canvas.NewCircle(color.NRGBA{A: 0})
	circle.StrokeColor = theme.ShadowColor()
	circle.StrokeWidth = 1
	badgeStack := container.NewStack(circle, container.NewCenter(badgeLabel))
	r.badgeLabel = badgeLabel

	// Center column - country, host, IP
	country := widget.NewLabelWithStyle(r.server.CountryLong, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	host := widget.NewLabelWithStyle(r.server.HostName, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	ip := widget.NewLabelWithStyle(r.server.IP, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	ip.Importance = widget.MediumImportance
	r.countryLabel = country
	r.hostLabel = host
	r.ipLabel = ip
	center := container.NewVBox(country, host, ip)

	// Right column - score, ping, proto, favorite, connect
	score := canvas.NewText("Score "+formatScore(r.server.Score), scoreColor(r.server.Score))
	score.TextStyle.Bold = true
	ping := canvas.NewText("Ping "+pingText(r.ctrl, r.server), pingColor(ctrlPing(r.ctrl, r.server)))
	proto := canvas.NewText(r.server.ProtoLabel(), theme.PrimaryColorNamed("primary"))
	proto.TextStyle.Bold = true
	r.scoreText = score
	r.pingText = ping
	r.protoText = proto

	fav := "☆"
	if r.ctrl.IsFavorite(r.server.HostName) {
		fav = "★"
	}
	favBtn := widget.NewButton(fav, func() { r.ctrl.ToggleFavorite(r.server.HostName) })
	favBtn.Importance = widget.MediumImportance
	r.favBtn = favBtn

	connectBtn := widget.NewButton("Connect", func() { connectAction(r.ctrl, r.win, r.server) })
	connectBtn.Importance = widget.HighImportance
	r.connectBtn = connectBtn

	rightTop := container.NewHBox(score, layout.NewSpacer(), favBtn)
	right := container.NewVBox(rightTop, ping, proto, connectBtn)

	// Fixed order: bg, selCheck, badgeStack, center, right
	objects := []fyne.CanvasObject{bg, selCheck, badgeStack, center, right}
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

// ctrlPing returns the ping milliseconds for a server.
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

// updateContentFromServer refreshes all dynamic content from r.server.
func (r *serverRow) updateContentFromServer() {
	sv := r.server
	selection := r.ctrl.SelectionMode()

	// Country/Host/IP
	if r.countryLabel != nil {
		r.countryLabel.SetText(sv.CountryLong)
	}
	if r.hostLabel != nil {
		r.hostLabel.SetText(sv.HostName)
	}
	if r.ipLabel != nil {
		r.ipLabel.SetText(sv.IP)
		if selection {
			r.ipLabel.Hide()
		} else {
			r.ipLabel.Show()
		}
	}

	// Score/Ping/Proto
	if r.scoreText != nil {
		r.scoreText.Text = "Score " + formatScore(sv.Score)
		r.scoreText.Color = scoreColor(sv.Score)
		r.scoreText.Refresh()
	}
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

	// Badge
	if r.badgeLabel != nil {
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

// rowRenderer lays the card out with fixed indices.
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

	// Fixed indices: 0=bg, 1=selCheck, 2=badge, 3=center, 4=right
	selCheck := o[1]
	badge := o[2]
	center := o[3]
	right := o[4]

	left := pad
	if rr.row.ctrl.SelectionMode() {
		selCheck.Resize(fyne.NewSize(36, 36))
		selCheck.Move(fyne.NewPos(pad, (size.Height-36)/2))
		left += 36 + pad
	}

	rightSize := right.MinSize()
	rightSize.Height = size.Height
	right.Resize(rightSize)
	right.Move(fyne.NewPos(size.Width-rightSize.Width-pad, 0))

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
	return fyne.NewSize(w+48, max(h+28, 72))
}

func (rr *rowRenderer) Refresh() {
	rr.row.updateContentFromServer()
	canvas.Refresh(rr.row)
}

func (rr *rowRenderer) Objects() []fyne.CanvasObject {
	return rr.objects
}
func (rr *rowRenderer) Destroy() {}
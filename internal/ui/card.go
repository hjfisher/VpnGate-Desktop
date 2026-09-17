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
}

func newServerRow(ctrl *controller.Controller, parent fyne.Window, sv data.VpnServer, onOpen func()) *serverRow {
	r := &serverRow{ctrl: ctrl, server: sv, win: parent, onOpen: onOpen}
	r.ExtendBaseWidget(r)
	return r
}

var _ fyne.Tappable = (*serverRow)(nil)
var _ fyne.Widget = (*serverRow)(nil)

func (r *serverRow) Tapped(*fyne.PointEvent) { r.onOpen() }

func (r *serverRow) CreateRenderer() fyne.WidgetRenderer {
	selection := r.ctrl.SelectionMode()
	selected := selection && r.ctrl.IsSelected(r.server.HostName)

	bg := canvas.NewRectangle(theme.BackgroundColor())
	if selected {
		bg = canvas.NewRectangle(softPrimary())
	}

	objects := []fyne.CanvasObject{bg}
	if selection {
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
	}

	border := container.NewBorder(nil, nil, nil, nil, makeCenter(r.server, selection))
	objects = append(objects,
		makeBadge(r.server.CountryShort),
		border,
		makeRight(r.ctrl, r.win, r.server),
	)
	return &rowRenderer{row: r, objects: objects, hasCheck: selection}
}

func softPrimary() color.NRGBA {
	c := theme.PrimaryColor()
	rgba := color.NRGBAModel.Convert(c).(color.NRGBA)
	return color.NRGBA{R: rgba.R, G: rgba.G, B: rgba.B, A: 28}
}

func makeBadge(code string) fyne.CanvasObject {
	label := canvas.NewText(code, theme.PrimaryColorNamed("primary"))
	label.TextStyle = fyne.TextStyle{Bold: true}
	circle := canvas.NewCircle(color.NRGBA{R: 255, G: 0, B: 0, A: 255})  // Red circle
	circle.StrokeColor = theme.ShadowColor()
	circle.StrokeWidth = 1
	stack := container.NewStack(circle, label)
	return container.New(layout.NewGridWrapLayout(fyne.NewSize(40, 40)), stack)
}

func makeCenter(sv data.VpnServer, selection bool) fyne.CanvasObject {
	country := widget.NewLabel(sv.CountryLong)
	country.TextStyle = fyne.TextStyle{Bold: true}
	country.Truncation = fyne.TextTruncateEllipsis
	host := widget.NewLabel(sv.HostName)
	host.Truncation = fyne.TextTruncateEllipsis
	ip := widget.NewLabel(sv.IP)
	ip.Importance = widget.MediumImportance
	ip.Truncation = fyne.TextTruncateEllipsis
	if selection {
		// Keep rows compact during selection.
		ip.Hide()
	}
	return container.NewVBox(country, host, ip)
}

func makeRight(ctrl *controller.Controller, parent fyne.Window, sv data.VpnServer) fyne.CanvasObject {
	score := canvas.NewText("Score "+formatScore(sv.Score), scoreColor(sv.Score))
	score.TextStyle.Bold = true
	ping := canvas.NewText("Ping "+pingText(ctrl, sv), pingColor(ctrlPing(ctrl, sv)))
	proto := canvas.NewText(sv.ProtoLabel(), theme.PrimaryColorNamed("primary"))
	proto.TextStyle.Bold = true

	fav := "☆"
	if ctrl.IsFavorite(sv.HostName) {
		fav = "★"
	}
	favBtn := widget.NewButton(fav, func() { ctrl.ToggleFavorite(sv.HostName) })
	favBtn.Importance = widget.MediumImportance

	connectBtn := widget.NewButton("Connect", func() { connectAction(ctrl, parent, sv) })
	connectBtn.Importance = widget.HighImportance

	top := container.NewHBox(score, layout.NewSpacer(), favBtn)
	return container.NewVBox(top, ping, proto, connectBtn)
}

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

func ctrlPing(ctrl *controller.Controller, sv data.VpnServer) int64 {
	if ms, ok := ctrl.PingResult(sv.HostName); ok {
		return ms
	}
	return int64(sv.Ping)
}

func pingText(ctrl *controller.Controller, sv data.VpnServer) string {
	ms := ctrlPing(ctrl, sv)
	if ms <= 0 {
		return "n/a"
	}
	return itoa64(ms) + " ms"
}

// rowRenderer lays the card out with a flexible middle column.
type rowRenderer struct {
	row      *serverRow
	objects  []fyne.CanvasObject
	hasCheck bool
}

func (rr *rowRenderer) Layout(size fyne.Size) {
	const pad = float32(10)
	o := rr.objects
	bg := o[0]
	bg.Resize(size)
	bg.Move(fyne.NewPos(0, 0))

	idx := 1
	left := pad
	if rr.hasCheck {
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

func (rr *rowRenderer) Refresh() { canvas.Refresh(rr.row) }
func (rr *rowRenderer) Objects() []fyne.CanvasObject {
	return rr.objects
}
func (rr *rowRenderer) Destroy() {}
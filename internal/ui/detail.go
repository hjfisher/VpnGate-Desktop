package ui

import (
	"fmt"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"vpngate/internal/controller"
	"vpngate/internal/data"
)

type detailWindow struct {
	ctrl   *controller.Controller
	server data.VpnServer
	win    fyne.Window

	pingBtn  *widget.Button
	pingText *canvas.Text
	measured int64
	measuredSet bool
}

// showDetail opens (or reuses) the detail window for a server.
func (m *mainUI) showDetail(server data.VpnServer) {
	m.detailHost = server.HostName
	if m.detail == nil {
		m.detail = m.ctrl.App().NewWindow("Server")
		m.detail.SetOnClosed(func() {
			m.detail = nil
			m.detailHost = ""
		})
	}
	OpenDetail(m.ctrl, m.detail, server)
}

// OpenDetail populates an existing detail window.
func OpenDetail(ctrl *controller.Controller, win fyne.Window, server data.VpnServer) {
	d := &detailWindow{ctrl: ctrl, server: server, win: win}
	d.build()
}

func (d *detailWindow) build() {
	d.win.SetTitle("Server — " + d.server.HostName)

	badge := makeBadge(d.server.CountryShort)
	country := widget.NewLabelWithStyle(d.server.CountryLong, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	operator := widget.NewLabel(d.server.Operator)
	operator.Importance = widget.MediumImportance
	header := container.NewBorder(
		nil, nil,
		badge,
		nil,
		container.NewVBox(country, operator),
	)

	// Ping section
	d.pingText = canvas.NewText("", theme.ForegroundColor())
	d.updatePingLabel()
	d.pingBtn = widget.NewButtonWithIcon("Check connection", theme.ZoomInIcon(), d.checkPing)
	pingRow := container.NewHBox(d.pingBtn, layoutSpacer(), d.pingText)

	divider := widget.NewSeparator()

	// Info
	info := widget.NewForm(
		widget.NewFormItem("IP", valueLabel(d.server.IP)),
		widget.NewFormItem("Score", valueLabel(formatScore(d.server.Score))),
		widget.NewFormItem("Speed", valueLabel(fmt.Sprintf("%.1f Mbit/s", d.server.SpeedMbps()))),
		widget.NewFormItem("Sessions", valueLabel(itoa64(int64(d.server.NumVpnSessions)))),
		widget.NewFormItem("Uptime", valueLabel(d.server.UptimeFormatted())),
		widget.NewFormItem("Users", valueLabel(itoa64(d.server.TotalUsers))),
		widget.NewFormItem("Traffic", valueLabel(d.server.TotalTrafficFormatted())),
		widget.NewFormItem("Protocol", valueLabel(d.server.ProtoLabel())),
		widget.NewFormItem("Log type", valueLabel(d.server.LogType)),
	)
	if d.server.Message != "" {
		info.AppendItem(widget.NewFormItem("Message", valueLabel(d.server.Message)))
	}

	// Config preview
	preview := d.configPreview()

	actionsTitle := widget.NewLabelWithStyle("Actions", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	connectBtn := widget.NewButton("Connect", func() { connectAction(d.ctrl, d.win, d.server) })
	connectBtn.Importance = widget.HighImportance
	copyBtn := widget.NewButton("Copy config", func() { copyConfigAction(d.ctrl, d.win, d.server) })
	exportBtn := widget.NewButton("Export .ovpn", func() { exportServerAction(d.ctrl, d.win, d.server) })
	deleteBtn := widget.NewButton("Delete server", func() {
		dialog.ShowConfirm(
			"Delete server",
			fmt.Sprintf("Remove %s from the saved list?", d.server.HostName),
			func(ok bool) {
				if !ok {
					return
				}
				d.ctrl.DeleteServer(d.server.HostName)
				d.win.Close()
			},
			d.win,
		)
	})
	deleteBtn.Importance = widget.DangerImportance
	actions := container.NewHBox(connectBtn, copyBtn, exportBtn, deleteBtn)

	body := container.NewVBox(
		header,
		pingRow,
		divider,
		info,
		preview,
		divider,
		actionsTitle,
		actions,
	)
	scroll := container.NewVScroll(body)
	scroll.SetMinSize(fyne.NewSize(560, 420))

	content := container.NewBorder(nil, nil, layoutSpacer(), nil,
		container.NewPadded(scroll))
	d.win.SetContent(content)
	d.win.Resize(fyne.NewSize(640, 700))
	if d.win.Canvas() != nil {
		d.win.CenterOnScreen()
	}
	d.win.Show()
}

func layoutSpacer() fyne.CanvasObject {
	return widget.NewLabel("  ")
}

func valueLabel(text string) *widget.Label {
	l := widget.NewLabel(text)
	l.Importance = widget.HighImportance
	return l
}

func (d *detailWindow) checkPing() {
	d.pingBtn.Disable()
	d.pingBtn.SetText("Checking…")
	done := make(chan struct{})
	// Safety net: re-enable the button even if the ping callback never fires.
	go func() {
		select {
		case <-done:
		case <-time.After(15 * time.Second):
			fyne.Do(func() {
				d.pingBtn.Enable()
				d.pingBtn.SetText("Check connection")
			})
		}
	}()
	d.ctrl.Ping(d.server.HostName, func(ms int64) {
		close(done)
		d.measured = ms
		d.measuredSet = true
		d.pingBtn.Enable()
		d.pingBtn.SetText("Check connection")
		d.updatePingLabel()
	})
}

func (d *detailWindow) updatePingLabel() {
	if d.measuredSet {
		if d.measured < 0 {
			d.pingText.Text = "Unreachable (TCP :443)"
		} else {
			d.pingText.Text = fmt.Sprintf("Measured latency: %d ms", d.measured)
		}
	} else {
		d.pingText.Text = fmt.Sprintf("Listed ping: %d ms", d.server.Ping)
	}
	d.pingText.Color = pingColor(d.measuredPing())
	d.pingText.Refresh()
}

func (d *detailWindow) measuredPing() int64 {
	if d.measuredSet {
		return d.measured
	}
	return int64(d.server.Ping)
}

func (d *detailWindow) configPreview() fyne.CanvasObject {
	title := widget.NewLabelWithStyle("Config preview", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	config := d.server.OpenVPNConfig()
	if config == "" {
		return widget.NewLabel("No readable OpenVPN config found for this server.")
	}
	text := canvas.NewText(extractKeyLines(config), theme.ForegroundColor())
	text.TextStyle = fyne.TextStyle{Monospace: true}
	text.TextSize = 12
	box := container.NewVScroll(text)
	box.SetMinSize(fyne.NewSize(540, 150))
	return container.NewVBox(title, box)
}

// extractKeyLines mirrors the Android app's config teaser.
func extractKeyLines(config string) string {
	var out []string
	for _, line := range strings.Split(config, "\n") {
		l := strings.TrimSpace(line)
		if l == "" || strings.HasPrefix(l, "#") {
			continue
		}
		if strings.HasPrefix(l, "remote ") ||
			strings.HasPrefix(l, "proto ") ||
			strings.HasPrefix(l, "client") ||
			strings.HasPrefix(l, "dev ") ||
			strings.Contains(l, "verb ") {
			out = append(out, l)
		}
		if len(out) == 6 {
			break
		}
	}
	if len(out) == 0 {
		return "(no key lines)"
	}
	return strings.Join(out, "\n")
}
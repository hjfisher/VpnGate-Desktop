package ui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"vpngate/internal/controller"
	"vpngate/internal/data"
	"vpngate/internal/net"
)

// connectAction saves the server's config and hands the .ovpn file to the OS
// default application (an installed OpenVPN client). Errors surface in a dialog.
func connectAction(ctrl *controller.Controller, parent fyne.Window, sv data.VpnServer) {
	path, err := ctrl.Connect(sv.HostName)
	if err != nil {
		dialog.ShowError(err, parent)
		return
	}
	dialog.ShowInformation(
		"Connect",
		fmt.Sprintf("Config saved to:\n%s\n\nOpened with your default OpenVPN client.", path),
		parent,
	)
}

// copyConfigAction copies the decoded config to the clipboard.
func copyConfigAction(ctrl *controller.Controller, parent fyne.Window, sv data.VpnServer) {
	config, err := ctrl.CopyConfig(sv.HostName)
	if err != nil {
		dialog.ShowError(err, parent)
		return
	}
	parent.Clipboard().SetContent(config)
	dialog.ShowInformation("Copy", "OpenVPN config copied to clipboard.", parent)
}

// exportServerAction exports one server and reveals its target folder.
func exportServerAction(ctrl *controller.Controller, parent fyne.Window, sv data.VpnServer) {
	out := ctrl.ExportServer(sv.HostName)
	reportExport(parent, out)
}

// exportSelectedAction exports every selected server.
func exportSelectedAction(ctrl *controller.Controller, parent fyne.Window) {
	out := ctrl.ExportSelected()
	reportExport(parent, out)
}

func reportExport(parent fyne.Window, out net.ExportOutcome) {
	if out.Error != nil {
		dialog.ShowError(out.Error, parent)
		return
	}
	dialog.ShowInformation(
		"Export",
		fmt.Sprintf("Exported %d .ovpn file(s) to:\n%s", out.Count, out.Folder),
		parent,
	)
}

// openExportFolder reveals the export directory in the file manager.
func openExportFolder(ctrl *controller.Controller, parent fyne.Window) {
	folder := ctrl.ExportFolder()
	if folder == "" {
		folder = net.DefaultExportFolder()
	}
	if err := net.OpenFile(folder); err != nil {
		dialog.ShowError(err, parent)
		return
	}
}
package data

import (
	"encoding/base64"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// VpnServer mirrors one row of the VPN Gate public server list.
type VpnServer struct {
	HostName           string
	IP                 string
	Score              int64
	Ping               int
	Speed              int64
	CountryLong        string
	CountryShort       string
	NumVpnSessions     int
	Uptime             int64
	TotalUsers         int64
	TotalTraffic       int64
	LogType            string
	Operator           string
	Message            string
	OpenVPNConfigB64   string
	ProtoType          string
	ConfigDecodedCache string
	AddedAt            time.Time `json:"added_at"` // When this server was first fetched
}

// OpenVPNConfig decodes the embedded base64 config on demand.
func (s *VpnServer) OpenVPNConfig() string {
	if s.ConfigDecodedCache != "" {
		return s.ConfigDecodedCache
	}
	raw, err := decodeB64(s.OpenVPNConfigB64)
	if err != nil {
		return ""
	}
	s.ConfigDecodedCache = raw
	return s.ConfigDecodedCache
}

// decodeB64 decodes standard or raw (padding-less) base64.
func decodeB64(s string) (string, error) {
	raw, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		raw, err = base64.RawStdEncoding.DecodeString(s)
		if err != nil {
			return "", err
		}
	}
	return string(raw), nil
}

// SpeedMbps returns transfer speed in Mbit/s.
func (s *VpnServer) SpeedMbps() float64 {
	return float64(s.Speed) / 1_000_000
}

// TotalTrafficFormatted renders the cumulative traffic in TB/GB/MB.
func (s *VpnServer) TotalTrafficFormatted() string {
	const (
		tb = 1_000_000_000_000
		gb = 1_000_000_000
		mb = 1_000_000
	)
	switch {
	case s.TotalTraffic >= tb:
		return fmt.Sprintf("%.1f TB", float64(s.TotalTraffic)/tb)
	case s.TotalTraffic >= gb:
		return fmt.Sprintf("%.1f GB", float64(s.TotalTraffic)/gb)
	case s.TotalTraffic >= mb:
		return fmt.Sprintf("%.1f MB", float64(s.TotalTraffic)/mb)
	default:
		return fmt.Sprintf("%d B", s.TotalTraffic)
	}
}

// UptimeFormatted renders uptime seconds as a compact duration string.
func (s *VpnServer) UptimeFormatted() string {
	hours := s.Uptime / 3600
	days := hours / 24
	switch {
	case hours >= 720:
		return fmt.Sprintf("%.1f mo", float64(hours)/720)
	case hours >= 168:
		return fmt.Sprintf("%.1f w", float64(hours)/168)
	case hours >= 24:
		return fmt.Sprintf("%d d", days)
	default:
		return fmt.Sprintf("%d h", hours)
	}
}

// ProtoLabel returns a human protocol label.
func (s *VpnServer) ProtoLabel() string {
	switch s.ProtoType {
	case "TCP":
		return "OVPN TCP"
	case "BOTH":
		return "OVPN UDP/TCP"
	default:
		return "OVPN UDP"
	}
}

// IsBlank reports whether an entry has enough data to be useful.
func (s *VpnServer) IsBlank() bool {
	return s.HostName == "" || s.IP == ""
}

// SanitizeFileName returns a filesystem-safe name for this server.
func (s *VpnServer) SanitizeFileName() string {
	cc := strings.ToUpper(s.CountryShort)
	cc = nonAlnum.ReplaceAllString(cc, "")
	host := strings.ToLower(s.HostName)
	host = nonSafe.ReplaceAllString(host, "_")
	if len(host) > 40 {
		host = host[:40]
	}
	return fmt.Sprintf("%s-%s.ovpn", cc, host)
}

var (
	nonAlnum = regexp.MustCompile("[^A-Z0-9]")
	nonSafe  = regexp.MustCompile("[^a-z0-9._-]")
)
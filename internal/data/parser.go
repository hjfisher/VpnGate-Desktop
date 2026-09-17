package data

import (
	"bufio"
	"strings"
)

// VpnGateParser parses the CSV returned by the VPN Gate public API.
//
// Format:
//
//	*vpn_servers
//	#<generated timestamp>
//	#HostName,IP,Score,Ping,Speed(Long),...
//	public-vpn-88,219.100.37.30,2837699,14,...,<base64 ovpn config>
//
// Lines starting with '#' or '*' are metadata and are skipped. The
// OpenVPN base64 payload never contains commas, so a naive split on ','
// (with a limit of 15 columns) is safe — base64 is always the last column.
type VpnGateParser struct{}

func (VpnGateParser) Parse(body string) []VpnServer {
	var servers []VpnServer
	scanner := bufio.NewScanner(strings.NewReader(body))
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "*") {
			continue
		}
		parts := strings.SplitN(line, ",", 15)
		if len(parts) < 15 {
			continue
		}
		base64cfg := parts[14]
		s := VpnServer{
			HostName:         parts[0],
			IP:               parts[1],
			Score:            parseInt64(parts[2]),
			Ping:             int(parseInt64(parts[3])),
			Speed:            parseInt64(parts[4]),
			CountryLong:      parts[5],
			CountryShort:     parts[6],
			NumVpnSessions:   int(parseInt64(parts[7])),
			Uptime:           parseInt64(parts[8]),
			TotalUsers:       parseInt64(parts[9]),
			TotalTraffic:     parseInt64(parts[10]),
			LogType:          parts[11],
			Operator:         parts[12],
			Message:          parts[13],
			OpenVPNConfigB64: base64cfg,
			ProtoType:        detectProto(base64cfg),
		}
		servers = append(servers, s)
	}
	// Deduplicate by IP, keeping the first occurrence.
	seen := make(map[string]struct{}, len(servers))
	out := servers[:0]
	for _, s := range servers {
		if _, ok := seen[s.IP]; ok {
			continue
		}
		seen[s.IP] = struct{}{}
		out = append(out, s)
	}
	return out
}

// detectProto inspects the decoded config to find "udp"/"tcp".
func detectProto(base64cfg string) string {
	raw, err := decodeB64(base64cfg)
	if err != nil {
		return "UDP"
	}
	lower := strings.ToLower(raw)
	hasUDP := strings.Contains(lower, "proto udp")
	hasTCP := strings.Contains(lower, "proto tcp")
	switch {
	case hasUDP && hasTCP:
		return "BOTH"
	case hasTCP:
		return "TCP"
	default:
		return "UDP"
	}
}

func parseInt64(raw string) int64 {
	var out int64
	for _, r := range raw {
		if r < '0' || r > '9' {
			break
		}
		out = out*10 + int64(r-'0')
	}
	return out
}
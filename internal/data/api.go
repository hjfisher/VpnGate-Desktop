package data

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// VpnGateApi fetches the raw CSV payload from VPN Gate.
//
// VPN Gate has no stable download guarantees; in restricted networks the
// main host is often blocked. We try a list of endpoints in order and,
// when the mirror option is enabled, also try the jina.ai text proxy.
type VpnGateApi struct {
	client    *http.Client
	useMirror bool
}

func NewVpnGateApi(useMirror bool) *VpnGateApi {
	return &VpnGateApi{
		client: &http.Client{
			Timeout: 40 * time.Second,
			Transport: &http.Transport{
				IdleConnTimeout: 30 * time.Second,
				MaxIdleConns:    2,
			},
		},
		useMirror: useMirror,
	}
}

const (
	mainHTTPS  = "https://www.vpngate.net/api/iphone/"
	mainHTTP   = "http://www.vpngate.net/api/iphone/"
	bareHTTPS  = "https://vpngate.net/api/iphone/"
	mirrorJina = "https://r.jina.ai/http://www.vpngate.net/api/iphone/"
)

// FetchRawCSV returns the decoded CSV body from the first endpoint that works.
func (a *VpnGateApi) FetchRawCSV() (string, error) {
	var endpoints []string
	if a.useMirror {
		// With a mirror configured, try the mirror first — it is often the
		// only reachable one in filtered networks.
		endpoints = []string{mirrorJina, bareHTTPS, mainHTTPS, mainHTTP}
	} else {
		endpoints = []string{mainHTTPS, mainHTTP, bareHTTPS}
	}

	var lastErr error
	for _, endpoint := range endpoints {
		body, err := a.fetchOne(endpoint)
		if err != nil {
			lastErr = err
			continue
		}
		if strings.TrimSpace(body) == "" {
			lastErr = fmt.Errorf("empty body from %s", endpoint)
			continue
		}
		return body, nil
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("all endpoints unavailable")
	}
	return "", lastErr
}

func (a *VpnGateApi) fetchOne(endpoint string) (string, error) {
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 VpnGate-Desktop")
	req.Header.Set("Accept", "text/plain,*/*")

	resp, err := a.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d from %s", resp.StatusCode, endpoint)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
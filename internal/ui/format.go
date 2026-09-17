package ui

import (
	"fmt"
	"image/color"
	"time"
)

// formatScore renders 1.2M / 34.5K style scores like the mobile app.
func formatScore(score int64) string {
	switch {
	case score >= 1_000_000:
		return fmt.Sprintf("%.1fM", float64(score)/1_000_000)
	case score >= 1_000:
		return fmt.Sprintf("%.1fK", float64(score)/1_000)
	default:
		return fmt.Sprintf("%d", score)
	}
}

// formatUpdateTime renders "5m ago" style timestamps.
func formatUpdateTime(t time.Time) string {
	if t.IsZero() {
		return "never"
	}
	diff := time.Since(t)
	switch {
	case diff < time.Minute:
		return "just now"
	case diff < time.Hour:
		return fmt.Sprintf("%d min ago", int(diff.Minutes()))
	case diff < 24*time.Hour:
		return fmt.Sprintf("%dh %dm ago", int(diff.Hours()), int(diff.Minutes())%60)
	default:
		return fmt.Sprintf("%dd ago", int(diff.Hours())/24)
	}
}

// pingColor mirrors the mobile app's latency coloring.
func pingColor(ms int64) color.NRGBA {
	switch {
	case ms < 0:
		return color.NRGBA{R: 128, G: 128, B: 128, A: 255}
	case ms < 60:
		return color.NRGBA{R: 22, G: 163, B: 74, A: 255}
	case ms < 150:
		return color.NRGBA{R: 37, G: 99, B: 235, A: 255}
	default:
		return color.NRGBA{R: 202, G: 138, B: 4, A: 255}
	}
}

func itoa64(n int64) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
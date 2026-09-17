package net

import (
	"net"
	"time"
)

// Ping measures TCP connect latency to the OpenVPN endpoint (default 443).
// Returns latency in milliseconds, or -1 on failure.
func Ping(host string, port int, timeout time.Duration) int64 {
	if port <= 0 {
		port = 443
	}
	if timeout <= 0 {
		timeout = 4 * time.Second
	}
	start := time.Now()
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, itoa(port)), timeout)
	if err != nil {
		return -1
	}
	_ = conn.Close()
	return time.Since(start).Milliseconds()
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [12]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
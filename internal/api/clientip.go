package api

import (
	"net"
	"net/http"
	"net/netip"
	"strings"
)

// clientIP returns the IP address of the client that sent r. Each of the
// trustedProxies reverse proxies in front of the server appends the address it
// received the request from to X-Forwarded-For, so the client's address is
// that many entries from the end; earlier entries come from the client and can
// be forged. Without a usable entry it falls back to the connection's address.
func clientIP(r *http.Request, trustedProxies int) string {
	if trustedProxies > 0 {
		// A request can carry several X-Forwarded-For lines; in order, they form one list.
		var entries []string
		for _, line := range r.Header.Values("X-Forwarded-For") {
			entries = append(entries, strings.Split(line, ",")...)
		}
		if i := len(entries) - trustedProxies; i >= 0 {
			if addr, err := netip.ParseAddr(strings.TrimSpace(entries[i])); err == nil {
				return addr.String()
			}
		}
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

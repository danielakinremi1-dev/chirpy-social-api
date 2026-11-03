package api

import (
	"net"
	"net/http"
	"net/netip"
	"strings"
)

// clientIP returns the IP address of the client that sent r. Behind a trusted
// reverse proxy it uses the last X-Forwarded-For entry, which the proxy
// appended; earlier entries come from the client and can be forged. Without a
// usable entry it falls back to the connection's address.
func clientIP(r *http.Request, trustProxy bool) string {
	if trustProxy {
		// A request can carry several X-Forwarded-For lines; the last is the newest.
		if lines := r.Header.Values("X-Forwarded-For"); len(lines) > 0 {
			entries := strings.Split(lines[len(lines)-1], ",")
			if addr, err := netip.ParseAddr(strings.TrimSpace(entries[len(entries)-1])); err == nil {
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

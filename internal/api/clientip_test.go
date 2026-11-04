package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/danielakinremi1-dev/chirpy-social-api/internal/config"
)

func TestClientIP(t *testing.T) {
	tests := []struct {
		name       string
		remoteAddr string
		xff        []string
		proxies    int
		want       string
	}{
		{name: "No proxy", remoteAddr: "192.0.2.1:1234", want: "192.0.2.1"},
		{name: "Header ignored without TRUST_PROXY", remoteAddr: "192.0.2.1:1234", xff: []string{"203.0.113.9"}, want: "192.0.2.1"},
		{name: "One proxy", remoteAddr: "10.0.0.1:1234", xff: []string{"203.0.113.9"}, proxies: 1, want: "203.0.113.9"},
		{name: "Forged entries before the proxy's", remoteAddr: "10.0.0.1:1234", xff: []string{"1.1.1.1, 2.2.2.2,203.0.113.9"}, proxies: 1, want: "203.0.113.9"},
		{name: "Several header lines", remoteAddr: "10.0.0.1:1234", xff: []string{"1.1.1.1", "203.0.113.9"}, proxies: 1, want: "203.0.113.9"},
		{
			// A real chain from Render: Cloudflare, then Render's load balancer,
			// then a proxy inside the container each append one entry.
			name:       "Render",
			remoteAddr: "[::1]:45910",
			xff:        []string{"6.6.6.6,24.36.246.148, 104.23.213.66, 10.24.169.5"},
			proxies:    3,
			want:       "24.36.246.148",
		},
		{name: "Proxy entries split across lines", remoteAddr: "[::1]:1234", xff: []string{"6.6.6.6, 203.0.113.9", "104.23.213.66, 10.24.169.5"}, proxies: 3, want: "203.0.113.9"},
		{name: "Fewer entries than proxies falls back", remoteAddr: "[::1]:1234", xff: []string{"104.23.213.66, 10.24.169.5"}, proxies: 3, want: "::1"},
		{name: "IPv6", remoteAddr: "10.0.0.1:1234", xff: []string{"2001:db8::1"}, proxies: 1, want: "2001:db8::1"},
		{name: "Invalid entry falls back", remoteAddr: "10.0.0.1:1234", xff: []string{"203.0.113.9, not-an-ip"}, proxies: 1, want: "10.0.0.1"},
		{name: "Missing header falls back", remoteAddr: "10.0.0.1:1234", proxies: 1, want: "10.0.0.1"},
		{name: "IPv6 remote address", remoteAddr: "[2001:db8::2]:1234", want: "2001:db8::2"},
		{name: "Remote address without port", remoteAddr: "192.0.2.1", want: "192.0.2.1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest("GET", "/", nil)
			r.RemoteAddr = tt.remoteAddr
			for _, v := range tt.xff {
				r.Header.Add("X-Forwarded-For", v)
			}
			if got := clientIP(r, tt.proxies); got != tt.want {
				t.Errorf("clientIP() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestLoginRateLimitBehindProxy(t *testing.T) {
	handler := New(newFakeStore(), config.Config{JWTSecret: testSecret, TrustedProxies: 1}).Routes()
	login := func(xff string) int {
		r := httptest.NewRequest("POST", "/api/login", strings.NewReader(`{"email":"nobody@example.com","password":"password123"}`))
		r.RemoteAddr = "10.0.0.1:1234" // the proxy
		r.Header.Set("X-Forwarded-For", xff)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, r)
		return rec.Code
	}

	// Changing the forged leftmost entry doesn't reset the limit.
	for i := range 10 {
		if code := login(strings.Repeat("1.", 3) + string(rune('0'+i)) + ", 203.0.113.9"); code != http.StatusUnauthorized {
			t.Fatalf("attempt %d: status = %d, want 401", i+1, code)
		}
	}
	if code := login("9.9.9.9, 203.0.113.9"); code != http.StatusTooManyRequests {
		t.Errorf("11th attempt from the same client: status = %d, want 429", code)
	}

	// Another client behind the same proxy has its own bucket.
	if code := login("203.0.113.10"); code != http.StatusUnauthorized {
		t.Errorf("different client: status = %d, want 401", code)
	}
}

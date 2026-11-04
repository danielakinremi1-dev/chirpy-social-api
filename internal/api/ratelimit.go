package api

import (
	"fmt"
	"math"
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// ipRateLimiter is a per-client-IP token bucket.
type ipRateLimiter struct {
	every          time.Duration
	burst          int
	trustedProxies int

	mu        sync.Mutex
	clients   map[string]*clientLimiter
	lastSweep time.Time
}

type clientLimiter struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// newIPRateLimiter allows each IP a burst of requests, refilled at one token
// per every. trustedProxies is passed to clientIP.
func newIPRateLimiter(every time.Duration, burst int, trustedProxies int) *ipRateLimiter {
	return &ipRateLimiter{
		every:          every,
		burst:          burst,
		trustedProxies: trustedProxies,
		clients:        make(map[string]*clientLimiter),
	}
}

func (l *ipRateLimiter) allow(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	// A client idle long enough to refill its whole bucket is the same as a
	// new one, so drop it to keep the map from growing without bound.
	idle := l.every * time.Duration(l.burst)
	if now.Sub(l.lastSweep) > idle {
		for clientIP, c := range l.clients {
			if now.Sub(c.lastSeen) > idle {
				delete(l.clients, clientIP)
			}
		}
		l.lastSweep = now
	}

	c, ok := l.clients[ip]
	if !ok {
		c = &clientLimiter{limiter: rate.NewLimiter(rate.Every(l.every), l.burst)}
		l.clients[ip] = c
	}
	c.lastSeen = now
	return c.limiter.Allow()
}

// limit responds with 429 when the client IP has used up its requests.
func (l *ipRateLimiter) limit(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !l.allow(clientIP(r, l.trustedProxies)) {
			w.Header().Set("Retry-After", fmt.Sprint(int(math.Ceil(l.every.Seconds()))))
			respondWithError(w, r, http.StatusTooManyRequests, "Too many requests, try again later", nil)
			return
		}
		next(w, r)
	}
}

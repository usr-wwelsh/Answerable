package endpoint

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// defaultRateLimitRPS and defaultRateLimitBurst are sized for a temporary
// public demo, not production traffic: generous enough that a room full of
// people trying it at once — often sharing one NAT'd IP — all get through,
// while still capping a single runaway or abusive client instead of leaving
// the endpoint fully open.
const (
	defaultRateLimitRPS   = 15
	defaultRateLimitBurst = 120
	rateLimitVisitorTTL   = 10 * time.Minute
)

// ipRateLimiter throttles requests per client key (IP) with a token bucket
// per key, so one caller's usage never affects another's.
type ipRateLimiter struct {
	rps   rate.Limit
	burst int
	ttl   time.Duration
	now   func() time.Time

	mu        sync.Mutex
	visitors  map[string]*rlVisitor
	lastSweep time.Time
}

type rlVisitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// newIPRateLimiter builds a limiter with the given sustained rate and
// burst. rps <= 0 disables limiting entirely (unlimited rate).
func newIPRateLimiter(rps float64, burst int, ttl time.Duration) *ipRateLimiter {
	limit := rate.Limit(rps)
	if rps <= 0 {
		limit = rate.Inf
	}
	return &ipRateLimiter{
		rps:      limit,
		burst:    burst,
		ttl:      ttl,
		now:      time.Now,
		visitors: make(map[string]*rlVisitor),
	}
}

func (rl *ipRateLimiter) allow(key string) bool {
	now := rl.now()

	rl.mu.Lock()
	defer rl.mu.Unlock()

	if rl.lastSweep.IsZero() {
		rl.lastSweep = now
	}
	if now.Sub(rl.lastSweep) > rl.ttl {
		for k, v := range rl.visitors {
			if now.Sub(v.lastSeen) > rl.ttl {
				delete(rl.visitors, k)
			}
		}
		rl.lastSweep = now
	}

	v, ok := rl.visitors[key]
	if !ok {
		v = &rlVisitor{limiter: rate.NewLimiter(rl.rps, rl.burst)}
		rl.visitors[key] = v
	}
	v.lastSeen = now
	return v.limiter.AllowN(now, 1)
}

func (rl *ipRateLimiter) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !rl.allow(clientIP(r)) {
			w.Header().Set("Retry-After", "1")
			http.Error(w, "rate limit exceeded, slow down and retry shortly", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// clientIP resolves the caller's key for rate limiting. This app is meant
// to sit a single hop behind a platform edge proxy (e.g. Railway) with no
// other proxy in front, so X-Forwarded-For's first entry is the real
// caller and is trusted directly; RemoteAddr is the fallback for direct or
// local connections where no proxy is involved.
func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if i := strings.IndexByte(xff, ','); i >= 0 {
			return strings.TrimSpace(xff[:i])
		}
		return strings.TrimSpace(xff)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

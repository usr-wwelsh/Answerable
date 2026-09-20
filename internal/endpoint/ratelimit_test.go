package endpoint

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRateLimiterAllowsBurstThenBlocks(t *testing.T) {
	clock := time.Unix(0, 0)
	rl := newIPRateLimiter(1, 3, time.Minute)
	rl.now = func() time.Time { return clock }

	for i := 0; i < 3; i++ {
		if !rl.allow("1.2.3.4") {
			t.Fatalf("request %d: want allowed within burst", i)
		}
	}
	if rl.allow("1.2.3.4") {
		t.Fatal("want blocked once burst is exhausted")
	}
}

func TestRateLimiterRefillsOverTime(t *testing.T) {
	clock := time.Unix(0, 0)
	rl := newIPRateLimiter(1, 1, time.Minute)
	rl.now = func() time.Time { return clock }

	if !rl.allow("1.2.3.4") {
		t.Fatal("want first request allowed")
	}
	if rl.allow("1.2.3.4") {
		t.Fatal("want second request blocked immediately")
	}
	clock = clock.Add(time.Second)
	if !rl.allow("1.2.3.4") {
		t.Fatal("want request allowed after the bucket refills")
	}
}

func TestRateLimiterTracksKeysIndependently(t *testing.T) {
	clock := time.Unix(0, 0)
	rl := newIPRateLimiter(1, 1, time.Minute)
	rl.now = func() time.Time { return clock }

	if !rl.allow("1.2.3.4") {
		t.Fatal("want first key's first request allowed")
	}
	if !rl.allow("5.6.7.8") {
		t.Fatal("want a different key unaffected by another key's usage")
	}
}

func TestRateLimiterZeroRPSDisablesLimiting(t *testing.T) {
	clock := time.Unix(0, 0)
	rl := newIPRateLimiter(0, 1, time.Minute)
	rl.now = func() time.Time { return clock }

	for i := 0; i < 50; i++ {
		if !rl.allow("1.2.3.4") {
			t.Fatalf("request %d: want unlimited when rps <= 0", i)
		}
	}
}

func TestClientIPPrefersForwardedForOverRemoteAddr(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.1:1234"
	req.Header.Set("X-Forwarded-For", "203.0.113.9")

	if got := clientIP(req); got != "203.0.113.9" {
		t.Errorf("clientIP = %q, want 203.0.113.9", got)
	}
}

func TestClientIPTakesFirstHopFromForwardedChain(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Forwarded-For", "203.0.113.9, 198.51.100.2")

	if got := clientIP(req); got != "203.0.113.9" {
		t.Errorf("clientIP = %q, want 203.0.113.9", got)
	}
}

func TestClientIPFallsBackToRemoteAddr(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.1:1234"

	if got := clientIP(req); got != "10.0.0.1" {
		t.Errorf("clientIP = %q, want 10.0.0.1", got)
	}
}

func TestHandlerRateLimitsPerIP(t *testing.T) {
	srv := newTestServer(t)
	srv.SetRateLimit(1, 1)

	newReq := func(ip string) *http.Request {
		r := httptest.NewRequest(http.MethodGet, "/facts.jsonld", nil)
		r.RemoteAddr = ip + ":5555"
		return r
	}

	rec1 := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec1, newReq("9.9.9.9"))
	if rec1.Code != http.StatusOK {
		t.Fatalf("first request status = %d, want 200", rec1.Code)
	}

	rec2 := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec2, newReq("9.9.9.9"))
	if rec2.Code != http.StatusTooManyRequests {
		t.Fatalf("second request status = %d, want 429", rec2.Code)
	}
	if rec2.Header().Get("Retry-After") == "" {
		t.Error("429 response missing Retry-After header")
	}

	rec3 := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec3, newReq("8.8.8.8"))
	if rec3.Code != http.StatusOK {
		t.Fatalf("different IP status = %d, want 200 (unaffected by other IP's limit)", rec3.Code)
	}
}

func TestHandlerMCPRouteIsRateLimitedTooOnceBurstIsExhausted(t *testing.T) {
	srv := newTestServer(t)
	srv.SetRateLimit(1, 1)

	newReq := func() *http.Request {
		r := httptest.NewRequest(http.MethodGet, "/mcp", nil)
		r.RemoteAddr = "9.9.9.9:5555"
		return r
	}

	rec1 := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec1, newReq())
	rec2 := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec2, newReq())

	if rec2.Code != http.StatusTooManyRequests {
		t.Fatalf("second /mcp request status = %d, want 429", rec2.Code)
	}
}

package admin

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestIsLoopbackHost(t *testing.T) {
	cases := map[string]bool{
		"127.0.0.1":    true,
		"localhost":    true,
		"::1":          true,
		"0.0.0.0":      false,
		"":             false,
		"192.168.1.10": false,
		"example.com":  false,
	}
	for host, want := range cases {
		if got := IsLoopbackHost(host); got != want {
			t.Errorf("IsLoopbackHost(%q) = %v, want %v", host, got, want)
		}
	}
}

func TestRequireAuthForBindAllowsLoopbackWithNoPassword(t *testing.T) {
	if err := RequireAuthForBind("127.0.0.1", ""); err != nil {
		t.Errorf("expected loopback bind with no password to be allowed, got: %v", err)
	}
}

func TestRequireAuthForBindRefusesNonLoopbackWithoutPassword(t *testing.T) {
	if err := RequireAuthForBind("0.0.0.0", ""); err == nil {
		t.Error("expected an error for a non-loopback bind with no password set")
	}
}

func TestRequireAuthForBindAllowsNonLoopbackWithPassword(t *testing.T) {
	if err := RequireAuthForBind("0.0.0.0", "s3cret"); err != nil {
		t.Errorf("expected non-loopback bind with a password to be allowed, got: %v", err)
	}
}

func called(t *testing.T) (http.Handler, *bool) {
	t.Helper()
	hit := false
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hit = true
		w.WriteHeader(http.StatusOK)
	}), &hit
}

func TestWrapWithAuthSkipsAuthOnLoopbackBind(t *testing.T) {
	next, hit := called(t)
	handler := WrapWithAuth("127.0.0.1", "s3cret", next)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if !*hit {
		t.Error("expected wrapped handler to be called without credentials on loopback bind")
	}
}

func TestWrapWithAuthRejectsMissingCredentialsOnNonLoopbackBind(t *testing.T) {
	next, hit := called(t)
	handler := WrapWithAuth("0.0.0.0", "s3cret", next)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
	if *hit {
		t.Error("expected wrapped handler not to be called without credentials")
	}
	if rec.Header().Get("WWW-Authenticate") == "" {
		t.Error("expected a WWW-Authenticate challenge header")
	}
}

func TestWrapWithAuthRejectsWrongPasswordOnNonLoopbackBind(t *testing.T) {
	next, hit := called(t)
	handler := WrapWithAuth("0.0.0.0", "s3cret", next)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.SetBasicAuth("staff", "wrong")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
	if *hit {
		t.Error("expected wrapped handler not to be called with the wrong password")
	}
}

func TestWrapWithAuthAcceptsCorrectPasswordOnNonLoopbackBind(t *testing.T) {
	next, hit := called(t)
	handler := WrapWithAuth("0.0.0.0", "s3cret", next)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.SetBasicAuth("staff", "s3cret")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if !*hit {
		t.Error("expected wrapped handler to be called with the correct password")
	}
}

func TestWrapWithAuthAcceptsAnyUsername(t *testing.T) {
	next, hit := called(t)
	handler := WrapWithAuth("0.0.0.0", "s3cret", next)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.SetBasicAuth("anyone-at-all", "s3cret")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if !*hit {
		t.Error("expected any username to be accepted as long as the password matches")
	}
}

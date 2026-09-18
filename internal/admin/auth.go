package admin

import (
	"crypto/sha256"
	"crypto/subtle"
	"fmt"
	"net"
	"net/http"
)

// IsLoopbackHost reports whether a bind host (no port) is loopback-only —
// unreachable from anywhere but the local machine.
func IsLoopbackHost(host string) bool {
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// RequireAuthForBind fails closed: a non-loopback admin bind (reachable
// over the network) must have ANSWERABLE_ADMIN_PASSWORD set, or the admin
// listener must not start at all. A loopback bind is already
// network-unreachable, so no password is required.
func RequireAuthForBind(bindHost, password string) error {
	if IsLoopbackHost(bindHost) {
		return nil
	}
	if password == "" {
		return fmt.Errorf("admin bind %q is not loopback-only: ANSWERABLE_ADMIN_PASSWORD must be set to start the admin webui", bindHost)
	}
	return nil
}

// WrapWithAuth gates next behind HTTP Basic Auth when bindHost is not
// loopback-only. A loopback bind is skipped entirely: it's already
// unreachable off the local machine, so requiring credentials there would
// add friction without a security benefit.
func WrapWithAuth(bindHost, password string, next http.Handler) http.Handler {
	if IsLoopbackHost(bindHost) {
		return next
	}
	return requireBasicAuth(password, next)
}

func requireBasicAuth(password string, next http.Handler) http.Handler {
	want := sha256.Sum256([]byte(password))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, pass, ok := r.BasicAuth()
		got := sha256.Sum256([]byte(pass))
		if !ok || subtle.ConstantTimeCompare(want[:], got[:]) != 1 {
			w.Header().Set("WWW-Authenticate", `Basic realm="Answerable admin"`)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

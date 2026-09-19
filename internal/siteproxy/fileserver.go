package siteproxy

import (
	"bytes"
	"net/http"
	"strconv"
	"strings"
)

// FileServer serves dir as the provider's site, injecting discovery links
// into any text/html response. Range and conditional-request headers are
// stripped from incoming requests so every response is a fresh, full 200 —
// a partial (206) or cached (304) response can't be rewritten safely.
func FileServer(dir string, links []DiscoveryLink) http.Handler {
	fs := http.FileServer(http.Dir(dir))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Header.Del("Range")
		r.Header.Del("If-Range")
		r.Header.Del("If-None-Match")
		r.Header.Del("If-Modified-Since")

		bw := &bufferingResponseWriter{ResponseWriter: w}
		fs.ServeHTTP(bw, r)
		bw.flush(links)
	})
}

// bufferingResponseWriter holds back the body only for text/html
// responses (so it can be rewritten before the headers are sent);
// everything else streams straight through.
type bufferingResponseWriter struct {
	http.ResponseWriter
	buf         bytes.Buffer
	status      int
	buffering   bool
	wroteHeader bool
}

func (w *bufferingResponseWriter) WriteHeader(status int) {
	w.status = status
	w.buffering = strings.HasPrefix(w.Header().Get("Content-Type"), "text/html")
	w.wroteHeader = true
	if !w.buffering {
		w.ResponseWriter.WriteHeader(status)
	}
}

func (w *bufferingResponseWriter) Write(p []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	if w.buffering {
		return w.buf.Write(p)
	}
	return w.ResponseWriter.Write(p)
}

func (w *bufferingResponseWriter) flush(links []DiscoveryLink) {
	if !w.buffering {
		return
	}
	body := InjectLinks(w.buf.Bytes(), links)
	w.Header().Set("Content-Length", strconv.Itoa(len(body)))
	w.ResponseWriter.WriteHeader(w.status)
	w.ResponseWriter.Write(body)
}

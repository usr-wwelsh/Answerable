package siteproxy

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
)

// ReverseProxy proxies to upstream — the provider's real existing site
// (Node, WordPress, whatever's already running) — injecting discovery
// links into any text/html response on the way back. Accept-Encoding is
// stripped from the outbound request so the upstream always answers
// uncompressed, since the response body has to be rewritten in place.
func ReverseProxy(upstream *url.URL, links []DiscoveryLink) *httputil.ReverseProxy {
	proxy := httputil.NewSingleHostReverseProxy(upstream)

	baseDirector := proxy.Director
	proxy.Director = func(r *http.Request) {
		baseDirector(r)
		r.Header.Del("Accept-Encoding")
	}

	proxy.ModifyResponse = func(resp *http.Response) error {
		if !strings.HasPrefix(resp.Header.Get("Content-Type"), "text/html") {
			return nil
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		resp.Body.Close()

		injected := InjectLinks(body, links)
		resp.Body = io.NopCloser(bytes.NewReader(injected))
		resp.ContentLength = int64(len(injected))
		resp.Header.Set("Content-Length", strconv.Itoa(len(injected)))
		return nil
	}

	return proxy
}

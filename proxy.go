package apiproxy

import (
	"github.com/bcicen/apiproxy/httpcache"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"
)

// NewCachingSingleHostReverseProxy constructs a caching reverse proxy handler for
// target. If cache is nil, a volatile, in-memory cache is used.
func NewCachingSingleHostReverseProxy(target *url.URL, cache httpcache.Cache, maxTTL time.Duration) *httputil.ReverseProxy {
	proxy := NewSingleHostReverseProxy(target)
	if cache == nil {
		cache = httpcache.NewMemoryCache(maxTTL)
	}
	proxy.Transport = httpcache.NewTransport(cache)
	return proxy
}

// NewSingleHostReverseProxy wraps net/http/httputil.NewSingleHostReverseProxy
// and sets the Host header based on the target URL.
func NewSingleHostReverseProxy(url *url.URL) *httputil.ReverseProxy {
	proxy := httputil.NewSingleHostReverseProxy(url)
	oldDirector := proxy.Director
	proxy.Director = func(r *http.Request) {
		oldDirector(r)
		r.Host = url.Host
	}
	return proxy
}

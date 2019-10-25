// Package httpcache provides a http.RoundTripper implementation that works as a
// mostly RFC-compliant cache for http responses.
//
// It is only suitable for use as a 'private' cache (i.e. for a web-browser or an API-client
// and not for a shared proxy).
//
package httpcache

import (
	"bufio"
	"bytes"
	"fmt"
	"net/http"
	"net/http/httputil"
	"strings"
	"sync"
)

const (
	XFromCache = "X-Aproxy-From-Cache"
	XCacheable = "X-Aproxy-Cacheable"
)

// Transport is an implementation of http.RoundTripper that will return values from a cache
// where possible (avoiding a network request) and will additionally add validators (etag/if-modified-since)
// to repeated requests allowing servers to return 304 / Not Modified
type Transport struct {
	// The RoundTripper interface actually used to make requests
	// If nil, http.DefaultTransport is used
	Transport http.RoundTripper
	Cache     Cache
	mu        sync.RWMutex
}

// NewTransport returns a new Transport with the
// provided Cache implementation
func NewTransport(c Cache) *Transport {
	return &Transport{
		Cache: c,
	}
}

// lookup returns the cached http.Response for a given key, if present and valid
func (t *Transport) lookup(req *http.Request) *http.Response {
	cachedVal, ok := t.Cache.Get(cacheKey(req))
	if !ok {
		return nil
	}

	resp, err := bytesToResp(cachedVal, req)
	if err != nil {
		panic(err)
	}

	resp.Header.Set(XFromCache, "1")
	return resp
}

func bytesToResp(b []byte, req *http.Request) (resp *http.Response, err error) {
	reader := bufio.NewReader(bytes.NewBuffer(b))
	return http.ReadResponse(reader, req)
}

// Client returns an *http.Client that caches responses.
func (t *Transport) Client() *http.Client {
	return &http.Client{Transport: t}
}

// RoundTrip takes a Request and returns a Response
//
// If there is a fresh Response already in cache, then it will be returned without connecting to
// the server.
func (t *Transport) RoundTrip(req *http.Request) (resp *http.Response, err error) {
	cacheable := (req.Method == "GET" || req.Method == "HEAD") && req.Header.Get("range") == ""

	if cacheable {
		resp = t.lookup(req)
		if resp != nil {
			fmt.Printf("[from-cache] %s\n", req.URL)
			return
		}
	}

	transport := t.Transport
	if transport == nil {
		transport = http.DefaultTransport
	}

	req.Header.Del("If-None-Match")
	//fmt.Printf("REQUEST: %s\n", req.URL)
	//for k, v := range req.Header {
	//fmt.Println(k, v)
	//}

	resp, err = transport.RoundTrip(req)
	if err != nil || resp.StatusCode == http.StatusNotModified {
		cacheable = false
	}

	if cacheable {
		respBytes, err := httputil.DumpResponse(resp, true)
		if err == nil {
			cacheKey := cacheKey(req)
			t.Cache.Set(cacheKey, respBytes)
			//fmt.Printf("[cache-set] %s\n", cacheKey)
			return bytesToResp(respBytes, req)
		} else {
			fmt.Printf("[ERROR] %s\n", err)
		}
	}

	return
}

// cloneRequest returns a clone of the provided *http.Request.
// The clone is a shallow copy of the struct and its Header map.
// (This function copyright goauth2 authors: https://code.google.com/p/goauth2)
func cloneRequest(r *http.Request) *http.Request {
	// shallow copy of the struct
	r2 := new(http.Request)
	*r2 = *r
	// deep copy of the Header
	r2.Header = make(http.Header)
	for k, s := range r.Header {
		r2.Header[k] = s
	}
	return r2
}

// headerAllCommaSepValues returns all comma-separated values (each
// with whitespace trimmed) for header name in headers. According to
// Section 4.2 of the HTTP/1.1 spec
// (http://www.w3.org/Protocols/rfc2616/rfc2616-sec4.html#sec4.2),
// values from multiple occurrences of a header should be concatenated, if
// the header's value is a comma-separated list.
func headerAllCommaSepValues(headers http.Header, name string) []string {
	var vals []string
	for _, val := range headers[http.CanonicalHeaderKey(name)] {
		fields := strings.Split(val, ",")
		for i, f := range fields {
			fields[i] = strings.TrimSpace(f)
		}
		vals = append(vals, fields...)
	}
	return vals
}

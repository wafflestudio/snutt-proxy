package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func newTestServer(t *testing.T, upstreamHandler http.HandlerFunc) (*httptest.Server, *httptest.Server) {
	t.Helper()
	upstream := httptest.NewServer(upstreamHandler)
	t.Cleanup(upstream.Close)
	upstreamURL, err := url.Parse(upstream.URL)
	if err != nil {
		t.Fatal(err)
	}
	proxy := httptest.NewServer(newHandler(newProxy(upstreamURL)))
	t.Cleanup(proxy.Close)
	return proxy, upstream
}

func TestForwardsRefererAndStripsCookie(t *testing.T) {
	var gotReferer, gotCookie string
	proxy, upstream := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotReferer = r.Header.Get("Referer")
		gotCookie = r.Header.Get("Cookie")
		w.Write([]byte("page"))
	})

	req, _ := http.NewRequest(http.MethodGet, proxy.URL+"/sugang/cc/cc103.action?openSchyy=2026", nil)
	req.Header.Set("Cookie", "session=secret")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
	if gotReferer != upstream.URL {
		t.Errorf("referer = %q, want %q", gotReferer, upstream.URL)
	}
	if gotCookie != "" {
		t.Errorf("cookie forwarded upstream: %q", gotCookie)
	}
}

func TestStripsSetCookie(t *testing.T) {
	proxy, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Set-Cookie", "JSESSIONID=abc")
		w.Write([]byte("page"))
	})

	resp, err := http.Get(proxy.URL + "/sugang/cc/cc103.action")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if got := resp.Header.Get("Set-Cookie"); got != "" {
		t.Errorf("set-cookie passed through: %q", got)
	}
}

func TestQueryStringForwarded(t *testing.T) {
	var gotQuery string
	proxy, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
	})

	resp, err := http.Get(proxy.URL + "/sugang/cc/cc103.action?sbjtCd=M1522.007300&ltNo=001")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	if gotQuery != "sbjtCd=M1522.007300&ltNo=001" {
		t.Errorf("query = %q", gotQuery)
	}
}

func TestPostForwardsBody(t *testing.T) {
	var gotBody string
	proxy, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.WriteHeader(http.StatusCreated)
	})

	resp, err := http.Post(
		proxy.URL+"/sugang/cc/cc103ajax.action",
		"application/x-www-form-urlencoded",
		strings.NewReader("openSchyy=2026&sbjtCd=M1522.007300"),
	)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("status = %d, want 201", resp.StatusCode)
	}
	if gotBody != "openSchyy=2026&sbjtCd=M1522.007300" {
		t.Errorf("body = %q", gotBody)
	}
}

func TestCacheControlFallbackForStatic(t *testing.T) {
	proxy, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/kor/css/cached.css" {
			w.Header().Set("Cache-Control", "no-store")
		}
		w.Write([]byte("body"))
	})

	resp, err := http.Get(proxy.URL + "/kor/css/style.css")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if got := resp.Header.Get("Cache-Control"); got != "public, max-age=86400" {
		t.Errorf("static cache-control = %q", got)
	}

	resp, err = http.Get(proxy.URL + "/kor/css/cached.css")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if got := resp.Header.Get("Cache-Control"); got != "no-store" {
		t.Errorf("upstream cache-control overridden: %q", got)
	}

	resp, err = http.Get(proxy.URL + "/sugang/cc/cc103.action")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if got := resp.Header.Get("Cache-Control"); got != "" {
		t.Errorf("page got cache-control: %q", got)
	}
}

func TestPathWhitelist(t *testing.T) {
	proxy, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})

	cases := []struct {
		method string
		path   string
		want   int
	}{
		{http.MethodGet, "/sugang/cc/cc103.action", http.StatusOK},
		{http.MethodPost, "/sugang/cc/cc103ajax.action", http.StatusOK},
		{http.MethodPost, "/sugang/cc/cc101ajax.action", http.StatusOK},
		{http.MethodGet, "/kor/css/style.css", http.StatusOK},
		{http.MethodGet, "/adm/js/default.js", http.StatusOK},
		{http.MethodGet, "/sugang/co/co010.action", http.StatusNotFound},
		{http.MethodGet, "/sugang/cc/cc200.action", http.StatusNotFound},
		{http.MethodGet, "/sugang/cc/login.action", http.StatusNotFound},
		{http.MethodDelete, "/sugang/cc/cc103.action", http.StatusMethodNotAllowed},
		{http.MethodPost, "/kor/css/style.css", http.StatusMethodNotAllowed},
		{http.MethodGet, "/", http.StatusNotFound},
		{http.MethodGet, "/v1/lectures", http.StatusNotFound},
	}
	for _, c := range cases {
		req, _ := http.NewRequest(c.method, proxy.URL+c.path, nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != c.want {
			t.Errorf("%s %s = %d, want %d", c.method, c.path, resp.StatusCode, c.want)
		}
	}
}

func TestHealthz(t *testing.T) {
	proxy, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("healthz must not hit upstream")
	})

	resp, err := http.Get(proxy.URL + "/healthz")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
}

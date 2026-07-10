package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"syscall"
	"time"
)

const upstreamBase = "https://sugang.snu.ac.kr"

var allowedAction = regexp.MustCompile(`^cc1\d{2}(ajax)?\.action$`)

func newProxy(upstream *url.URL) *httputil.ReverseProxy {
	return &httputil.ReverseProxy{
		Rewrite: func(r *httputil.ProxyRequest) {
			r.SetURL(upstream)
			r.Out.Header.Set("Referer", upstream.String())
			r.Out.Header.Del("Cookie")
		},
		ModifyResponse: func(resp *http.Response) error {
			resp.Header.Del("Set-Cookie")
			path := resp.Request.URL.Path
			if (strings.HasPrefix(path, "/kor/") || strings.HasPrefix(path, "/adm/")) &&
				resp.StatusCode < 400 && resp.Header.Get("Cache-Control") == "" {
				resp.Header.Set("Cache-Control", "public, max-age=86400")
			}
			return nil
		},
		Transport: &http.Transport{
			ResponseHeaderTimeout: 15 * time.Second,
			IdleConnTimeout:       90 * time.Second,
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			log.Printf("upstream error: %s %s: %v", r.Method, r.URL.Path, err)
			http.Error(w, "upstream error", http.StatusBadGateway)
		},
	}
}

func newHandler(proxy http.Handler) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte("ok"))
	})

	proxyAction := func(w http.ResponseWriter, r *http.Request) {
		if !allowedAction.MatchString(r.PathValue("action")) {
			http.NotFound(w, r)
			return
		}
		proxy.ServeHTTP(w, r)
	}
	mux.HandleFunc("GET /sugang/cc/{action}", proxyAction)
	mux.HandleFunc("POST /sugang/cc/{action}", proxyAction)

	proxyStatic := func(w http.ResponseWriter, r *http.Request) {
		escapedPath := r.URL.EscapedPath()
		if strings.Contains(escapedPath, "..") || strings.Contains(strings.ToLower(escapedPath), "%2e") {
			http.NotFound(w, r)
			return
		}
		proxy.ServeHTTP(w, r)
	}
	mux.HandleFunc("GET /kor/", proxyStatic)
	mux.HandleFunc("GET /adm/", proxyStatic)

	return logRequests(mux)
}

func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(sw, r)
		log.Printf("%s %s %d %s", r.Method, r.URL.RequestURI(), sw.status, time.Since(start).Round(time.Millisecond))
	})
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *statusWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	upstream, err := url.Parse(upstreamBase)
	if err != nil {
		log.Fatal(err)
	}

	server := &http.Server{
		Addr:              ":" + port,
		Handler:           newHandler(newProxy(upstream)),
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.Printf("listening on :%s, proxying %s", port, upstreamBase)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("shutdown: %v", err)
	}
}

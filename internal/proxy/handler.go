// Package proxy
package proxy

import (
	"context"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

func logInfo(format string, v ...any) {
	log.Printf("[INFO] "+format, v...)
}

func logDebug(debug bool, format string, v ...any) {
	if debug {
		log.Printf("[DEBUG] "+format, v...)
	}
}

// ProxyHandler holds the configuration map and the shared HTTP client.
type ProxyHandler struct {
	// Map of addon "type" (e.g., "comet") to a list of target base URLs.
	Routes     map[string][]string
	HTTPClient *http.Client
	Debug      bool
}

// NewProxyHandler creates a new handler with an optimized HTTP client.
func NewProxyHandler(routes map[string][]string, debug bool) *ProxyHandler {
	client := &http.Client{
		Timeout: 30 * time.Second, // Max time we will wait for ANY addon to respond
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     90 * time.Second,
		},
	}

	return &ProxyHandler{
		Routes:     routes,
		HTTPClient: client,
		Debug:      debug,
	}
}

// ServeHTTP satisfies the http.Handler interface.
func (h *ProxyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	logInfo("INCOMING: %s %s (from %s)", r.Method, r.URL.Path, r.RemoteAddr)

	if h.Debug {
		go func(ctx context.Context, path string) {
			<-ctx.Done()
			err := ctx.Err()
			// We wait 10ms to see if ServeHTTP naturally finished,
			// or if this was a premature cancellation.
			time.Sleep(10 * time.Millisecond)
			logDebug(h.Debug, "Context cancelled for %s. Reason: %v (Time elapsed: %v)", path, err, time.Since(start))
		}(r.Context(), r.URL.Path)
	}

	pathParts := strings.SplitN(strings.TrimPrefix(r.URL.Path, "/"), "/", 2)
	if len(pathParts) < 1 || pathParts[0] == "" {
		http.Error(w, "Invalid path format", http.StatusBadRequest)
		return
	}

	addonType := pathParts[0]
	if len(pathParts) == 2 {
		r.URL.Path = "/" + pathParts[1]
	} else {
		r.URL.Path = "/"
	}

	baseURLs, exists := h.Routes[addonType]
	if !exists || len(baseURLs) == 0 {
		http.Error(w, "Addon type not configured", http.StatusNotFound)
		return
	}

	resp, err := Race(r.Context(), h.HTTPClient, r, baseURLs, h.Debug)
	if err != nil {
		if err == r.Context().Err() {
			logInfo("[%s] Client disconnected mid-request (Time: %v)", addonType, time.Since(start))
			return
		}
		logInfo("[%s] Race totally failed: %v", addonType, err)
		http.Error(w, "All upstream instances failed", http.StatusBadGateway)
		return
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			if h.Debug {
				log.Printf("[DEBUG] resp.Body.Close returned an error: %s", err)
			}
		}
	}()

	logDebug(h.Debug, "Winner returned to handler for %s in %v", r.URL.Path, time.Since(start))

	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
	w.WriteHeader(resp.StatusCode)

	bytesWritten, err := io.Copy(w, resp.Body)
	if err != nil {
		logInfo("[%s] Error streaming body: %v (Bytes: %d)", addonType, err, bytesWritten)
	} else {
		logInfo("COMPLETED: %s %s in %v (Bytes: %d)", r.Method, r.URL.Path, time.Since(start), bytesWritten)
	}
}

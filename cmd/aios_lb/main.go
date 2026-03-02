package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"aios_lb/internal/config"
	"aios_lb/internal/proxy"
)

func main() {
	configPath := flag.String("config", "config.yaml", "Path to the configuration file")
	listenAddr := flag.String("listen", ":7035", "Server listen address (e.g., :7035)")
	flag.Parse()

	log.Printf("Loading configuration from %s...", *configPath)
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	routes := cfg.ToRouteMap()
	log.Printf("Loaded %d addon instance groups.", len(routes))

	proxyHandler := proxy.NewProxyHandler(routes, cfg.Debug)

	mux := http.NewServeMux()
	mux.Handle("/", proxyHandler)

	srv := &http.Server{
		Addr:         *listenAddr,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 45 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("aios-lb is running and listening on %s", *listenAddr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	sig := <-quit
	log.Printf("Received signal '%v'. Shutting down server gracefully...", sig)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited cleanly.")
}

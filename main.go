package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"naevis/config"
	"naevis/infra"
	"naevis/middleware"
	"naevis/routes"

	"github.com/rs/cors"
)

func main() {

	cfg := config.InitConfig()

	app, err := infra.New(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize infrastructure: %v", err)
	}

	// =====================
	// Rate limiter
	// =====================
	rateLimiter := middleware.NewRateLimiter(
		1,
		12,
		10*time.Minute,
		10000,
	)

	// =====================
	// Router & middleware
	// =====================
	router := routes.SetupRouter(app, rateLimiter)

	innerHandler := middleware.LoggingMiddleware(
		middleware.SecurityHeaders(router),
	)

	corsHandler := cors.New(cors.Options{
		AllowedOrigins:   cfg.AllowedOrigins,
		AllowedMethods:   []string{"HEAD", "GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization", "Idempotency-Key", "X-Requested-With"},
		AllowCredentials: true,
	}).Handler(innerHandler)

	mux := http.NewServeMux()
	mux.Handle("/", corsHandler)

	// =====================
	// HTTP server
	// =====================
	server := &http.Server{
		Addr:              cfg.HTTPPort,
		Handler:           mux,
		ReadTimeout:       7 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       120 * time.Second,
		ReadHeaderTimeout: 2 * time.Second,
	}

	go func() {
		log.Printf("API server listening on %s", cfg.HTTPPort)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("ListenAndServe error: %v", err)
		}
	}()

	// go func() {
	// 	log.Printf("API server listening on %s", cfg.HTTPPort)
	// 	if err := server.ListenAndServeTLS("cert.pem", "key.pem"); err != nil && err != http.ErrServerClosed {
	// 		log.Fatalf("ListenAndServe error: %v", err)
	// 	}
	// }()

	// =====================
	// Graceful shutdown
	// =====================
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh

	log.Println("Shutting down server...")

	// Stop rate limiter
	rateLimiter.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Graceful shutdown failed: %v", err)
	}

	log.Println("Server stopped successfully")
}

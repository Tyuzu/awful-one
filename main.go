// main.go
//
// Simple user account CRUD API in Go.
// Designed for deployment on EC2 and future expansion (chat rooms, websocket hubs, etc).
//
// Features:
// - julienschmidt/httprouter
// - rs/cors
// - Graceful shutdown
// - Middleware
// - IP rate limiting
// - JSON API
// - Clean structure for future chat room support
//
// Run:
//   go mod init example/api
//   go get github.com/julienschmidt/httprouter
//   go get github.com/rs/cors
//
// Start:
//   go run main.go
//
// Example:
//   curl -X POST localhost:4000/users \
//      -H "Content-Type: application/json" \
//      -d '{"username":"alice","email":"alice@example.com"}'

package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/julienschmidt/httprouter"
	"github.com/rs/cors"
)

type User struct {
	ID        int64     `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

type App struct {
	router *httprouter.Router

	users   map[int64]User
	userMux sync.RWMutex
	nextID  int64

	limiter *RateLimiter
}

func main() {
	app := &App{
		router:  httprouter.New(),
		users:   make(map[int64]User),
		nextID:  1,
		limiter: NewRateLimiter(100, time.Minute),
	}

	app.routes()

	c := cors.New(cors.Options{
		AllowedOrigins: []string{
			"*",
		},
		AllowedMethods: []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodDelete,
			http.MethodOptions,
		},
		AllowedHeaders: []string{
			"Content-Type",
			"Authorization",
		},
		AllowCredentials: false,
	})

	handler := app.middleware(c.Handler(app.router))

	server := &http.Server{
		Addr:              ":4000",
		Handler:           handler,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Println("server running on :4000")

		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server error: %v", err)
		}
	}()

	gracefulShutdown(server)
}

func (a *App) routes() {
	// health
	a.router.GET("/health", a.wrap(a.healthHandler))

	// user CRUD
	a.router.POST("/users", a.wrap(a.createUser))
	a.router.GET("/users", a.wrap(a.listUsers))
	a.router.GET("/users/:id", a.wrap(a.getUser))
	a.router.PUT("/users/:id", a.wrap(a.updateUser))
	a.router.DELETE("/users/:id", a.wrap(a.deleteUser))

	// reserved for future chat room support
	// a.router.POST("/rooms", ...)
	// a.router.GET("/rooms/:id", ...)
	// a.router.GET("/ws", ...)
}

func (a *App) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		// security headers
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Content-Type", "application/json")

		// basic logging
		start := time.Now()

		// rate limit
		if !a.limiter.Allow(clientIP(r)) {
			writeJSON(w, http.StatusTooManyRequests, map[string]string{
				"error": "rate limit exceeded",
			})
			return
		}

		next.ServeHTTP(w, r)

		log.Printf(
			"%s %s %s %v",
			r.Method,
			r.URL.Path,
			clientIP(r),
			time.Since(start),
		)
	})
}

func (a *App) wrap(
	fn func(http.ResponseWriter, *http.Request, httprouter.Params),
) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		fn(w, r, ps)
	}
}

func (a *App) healthHandler(
	w http.ResponseWriter,
	r *http.Request,
	_ httprouter.Params,
) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status": "ok",
	})
}

func (a *App) createUser(
	w http.ResponseWriter,
	r *http.Request,
	_ httprouter.Params,
) {
	var input struct {
		Username string `json:"username"`
		Email    string `json:"email"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid json",
		})
		return
	}

	input.Username = strings.TrimSpace(input.Username)
	input.Email = strings.TrimSpace(input.Email)

	if input.Username == "" || input.Email == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "username and email required",
		})
		return
	}

	a.userMux.Lock()
	defer a.userMux.Unlock()

	user := User{
		ID:        a.nextID,
		Username:  input.Username,
		Email:     input.Email,
		CreatedAt: time.Now(),
	}

	a.users[user.ID] = user
	a.nextID++

	writeJSON(w, http.StatusCreated, user)
}

func (a *App) listUsers(
	w http.ResponseWriter,
	r *http.Request,
	_ httprouter.Params,
) {
	a.userMux.RLock()
	defer a.userMux.RUnlock()

	users := make([]User, 0, len(a.users))

	for _, u := range a.users {
		users = append(users, u)
	}

	writeJSON(w, http.StatusOK, users)
}

func (a *App) getUser(
	w http.ResponseWriter,
	r *http.Request,
	ps httprouter.Params,
) {
	id, err := parseID(ps.ByName("id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid id",
		})
		return
	}

	a.userMux.RLock()
	defer a.userMux.RUnlock()

	user, ok := a.users[id]
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{
			"error": "user not found",
		})
		return
	}

	writeJSON(w, http.StatusOK, user)
}

func (a *App) updateUser(
	w http.ResponseWriter,
	r *http.Request,
	ps httprouter.Params,
) {
	id, err := parseID(ps.ByName("id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid id",
		})
		return
	}

	var input struct {
		Username string `json:"username"`
		Email    string `json:"email"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid json",
		})
		return
	}

	a.userMux.Lock()
	defer a.userMux.Unlock()

	user, ok := a.users[id]
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{
			"error": "user not found",
		})
		return
	}

	if strings.TrimSpace(input.Username) != "" {
		user.Username = input.Username
	}

	if strings.TrimSpace(input.Email) != "" {
		user.Email = input.Email
	}

	a.users[id] = user

	writeJSON(w, http.StatusOK, user)
}

func (a *App) deleteUser(
	w http.ResponseWriter,
	r *http.Request,
	ps httprouter.Params,
) {
	id, err := parseID(ps.ByName("id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid id",
		})
		return
	}

	a.userMux.Lock()
	defer a.userMux.Unlock()

	if _, ok := a.users[id]; !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{
			"error": "user not found",
		})
		return
	}

	delete(a.users, id)

	writeJSON(w, http.StatusOK, map[string]string{
		"message": "user deleted",
	})
}

func gracefulShutdown(server *http.Server) {
	stop := make(chan os.Signal, 1)

	signal.Notify(
		stop,
		syscall.SIGINT,
		syscall.SIGTERM,
	)

	<-stop

	log.Println("shutdown signal received")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("graceful shutdown failed: %v", err)

		if err := server.Close(); err != nil {
			log.Printf("force close failed: %v", err)
		}
	}

	log.Println("server stopped")
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("json encode error: %v", err)
	}
}

func parseID(s string) (int64, error) {
	return strconv.ParseInt(s, 10, 64)
}

func clientIP(r *http.Request) string {
	xff := r.Header.Get("X-Forwarded-For")

	if xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}

	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}

	return ip
}

//
// Simple in-memory IP rate limiter
//

type visitor struct {
	count     int
	expiresAt time.Time
}

type RateLimiter struct {
	mu       sync.Mutex
	visitors map[string]*visitor

	maxRequests int
	window      time.Duration
}

func NewRateLimiter(maxRequests int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		visitors:    make(map[string]*visitor),
		maxRequests: maxRequests,
		window:      window,
	}

	go rl.cleanup()

	return rl
}

func (r *RateLimiter) Allow(ip string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()

	v, exists := r.visitors[ip]

	if !exists || now.After(v.expiresAt) {
		r.visitors[ip] = &visitor{
			count:     1,
			expiresAt: now.Add(r.window),
		}

		return true
	}

	if v.count >= r.maxRequests {
		return false
	}

	v.count++

	return true
}

func (r *RateLimiter) cleanup() {
	ticker := time.NewTicker(time.Minute)

	for range ticker.C {
		r.mu.Lock()

		now := time.Now()

		for ip, v := range r.visitors {
			if now.After(v.expiresAt) {
				delete(r.visitors, ip)
			}
		}

		r.mu.Unlock()
	}
}

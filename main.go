package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"sync"

	"github.com/julienschmidt/httprouter"
)

/*
	Simple REST API with:
	- GET
	- POST
	- PUT
	- DELETE
	- JSON responses
	- CORS support
	- Basic in-memory storage
*/

type User struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

var (
	users  = map[int]User{}
	nextID = 1
	mu     sync.Mutex
)

func main() {
	router := httprouter.New()

	// Routes
	router.GET("/", Home)

	router.GET("/api/users", GetUsers)
	router.GET("/api/users/:id", GetUser)

	router.POST("/api/users", CreateUser)

	router.PUT("/api/users/:id", UpdateUser)

	router.DELETE("/api/users/:id", DeleteUser)

	// Wrap router with CORS middleware
	handler := corsMiddleware(router)

	log.Println("Server running on :4000")
	log.Fatal(http.ListenAndServe(":4000", handler))
}

/*
	Handlers
*/

func Home(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	respondJSON(w, http.StatusOK, map[string]string{
		"message": "API is running",
	})
}

func GetUsers(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	mu.Lock()
	defer mu.Unlock()

	var result []User

	for _, user := range users {
		result = append(result, user)
	}

	respondJSON(w, http.StatusOK, result)
}

func GetUser(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	id, err := strconv.Atoi(ps.ByName("id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid user id")
		return
	}

	mu.Lock()
	defer mu.Unlock()

	user, exists := users[id]
	if !exists {
		respondError(w, http.StatusNotFound, "user not found")
		return
	}

	respondJSON(w, http.StatusOK, user)
}

func CreateUser(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	var input struct {
		Name string `json:"name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		respondError(w, http.StatusBadRequest, "invalid json body")
		return
	}

	if input.Name == "" {
		respondError(w, http.StatusBadRequest, "name is required")
		return
	}

	mu.Lock()
	defer mu.Unlock()

	user := User{
		ID:   nextID,
		Name: input.Name,
	}

	users[nextID] = user
	nextID++

	respondJSON(w, http.StatusCreated, user)
}

func UpdateUser(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	id, err := strconv.Atoi(ps.ByName("id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid user id")
		return
	}

	var input struct {
		Name string `json:"name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		respondError(w, http.StatusBadRequest, "invalid json body")
		return
	}

	mu.Lock()
	defer mu.Unlock()

	user, exists := users[id]
	if !exists {
		respondError(w, http.StatusNotFound, "user not found")
		return
	}

	if input.Name != "" {
		user.Name = input.Name
	}

	users[id] = user

	respondJSON(w, http.StatusOK, user)
}

func DeleteUser(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	id, err := strconv.Atoi(ps.ByName("id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid user id")
		return
	}

	mu.Lock()
	defer mu.Unlock()

	_, exists := users[id]
	if !exists {
		respondError(w, http.StatusNotFound, "user not found")
		return
	}

	delete(users, id)

	respondJSON(w, http.StatusOK, map[string]string{
		"message": "user deleted",
	})
}

/*
	Response Helpers
*/

func respondJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	json.NewEncoder(w).Encode(payload)
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{
		"error": message,
	})
}

/*
	CORS Middleware
*/

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Allow frontend access
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		// Handle preflight requests
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

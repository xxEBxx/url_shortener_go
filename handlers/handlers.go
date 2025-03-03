package handlers

import (
	"URL_Shortener/auth"
	"URL_Shortener/utils"
	"context"
	"encoding/json"
	"fmt"
	"github.com/redis/go-redis/v9"
	"log"
	"net/http"
)

var rdb *redis.Client
var ctx = context.Background()

func InitHandlers(redisClient *redis.Client) {
	rdb = redisClient
}

// RedirectHandler: Redirects short URL and tracks clicks + IPs
func RedirectHandler(w http.ResponseWriter, r *http.Request) {
	utils.EnableCORS(w)
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	if rdb == nil {
		http.Error(w, "Redis is not initialized", http.StatusInternalServerError)
		log.Println("❌ Redis client is nil, cannot process request")
		return
	}

	shortKey := r.URL.Path[1:]
	longURL, err := rdb.HGet(ctx, "url:"+shortKey, "originalURL").Result()

	if err == redis.Nil {
		http.NotFound(w, r)
		return
	} else if err != nil {
		http.Error(w, "Error fetching URL", http.StatusInternalServerError)
		log.Printf("❌ Redis GET error: %v", err)
		return
	}

	userIP := utils.GetIP(r)

	_, err = rdb.HIncrBy(ctx, "clicks", shortKey, 1).Result()
	if err != nil {
		log.Printf("❌ Failed to track click for %s: %v", shortKey, err)
	}

	_, err = rdb.SAdd(ctx, "ip:"+shortKey, userIP).Result()
	if err != nil {
		log.Printf("❌ Failed to store IP for %s: %v", shortKey, err)
	}
	http.Redirect(w, r, longURL, http.StatusFound)
}

func ShortenHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf(r.Method)
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	username := r.Header.Get("X-User")
	if username == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Extract URL and optional slug from query parameters
	originalURL := r.URL.Query().Get("url")
	shortKey := r.URL.Query().Get("slug") // Custom slug (if provided)

	if originalURL == "" {
		http.Error(w, "Missing URL parameter", http.StatusBadRequest)
		return
	}
	if shortKey == "" {
		shortKey = utils.GetShortCode() // Generates a 6-character short key
	}

	_, err := rdb.HSet(ctx, "url:"+shortKey, map[string]interface{}{
		"originalURL": originalURL,
		"creator":     username, // Store creator's username
	}).Result()
	if err != nil {
		http.Error(w, "Failed to create short URL", http.StatusInternalServerError)
		return
	}

	_, err = rdb.SAdd(ctx, "user:"+username+":urls", shortKey).Result()
	if err != nil {
		http.Error(w, "Failed to link URL to user", http.StatusInternalServerError)
		return
	}

	response := fmt.Sprintf("Short URL created: http://localhost:8080/%s", shortKey)
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(response))
}
func SignupHandler(w http.ResponseWriter, r *http.Request) {
	utils.EnableCORS(w)
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	var user auth.User
	err := json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	err = auth.RegisterUser(rdb, user)
	if err != nil {
		http.Error(w, "Failed to register user", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Write([]byte("✅ User registered successfully"))
}

// LoginHandler: Authenticates a user and returns a JWT
func LoginHandler(w http.ResponseWriter, r *http.Request) {
	utils.EnableCORS(w)
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	var user auth.User
	err := json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	token, err := auth.AuthenticateUser(rdb, user)
	if err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"token": token})
}

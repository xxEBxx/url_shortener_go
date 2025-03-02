package handlers

import (
	"URL_Shortener/auth"
	"URL_Shortener/utils"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
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

	longURL, err := rdb.Get(ctx, shortKey).Result()
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

	longURL := r.URL.Query().Get("url")
	if longURL == "" {
		http.Error(w, "Missing URL", http.StatusBadRequest)
		return
	}

	shortKey := utils.GetShortCode()

	customSlug := r.URL.Query().Get("slug")
	if customSlug != "" {
		shortKey = customSlug
	}

	err := rdb.Set(ctx, shortKey, longURL, time.Hour*24*7).Err()
	//works for a week
	if err != nil {
		http.Error(w, "Failed to store URL in Redis", http.StatusInternalServerError)
		log.Printf("❌ Redis SET error: %v", err)
		return
	}

	fmt.Fprintf(w, "Short URL: http://localhost:8080/%s\n", shortKey)
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

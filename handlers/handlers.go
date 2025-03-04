package handlers

import (
	"URL_Shortener/auth"
	"URL_Shortener/utils"
	"context"
	"encoding/json"
	"errors"
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

// URLData holds the data for each short URL entry.
type URLData struct {
	ShortKey    string   `json:"shortKey"`
	OriginalURL string   `json:"originalURL"`
	Clicks      int      `json:"clicks"`
	IPs         []string `json:"ips"`
}

// Server holds shared components like the Redis client.
type Server struct {
	redisClient *redis.Client
}

// In this example, assume your JWT middleware adds user claims to the context.
func GetUserURLsHandler(w http.ResponseWriter, r *http.Request) {

	utils.EnableCORS(w)
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	ctx := context.Background()

	// 1. Extract the username from your JWT claims.
	//    (You'd adjust this based on how your middleware sets context.)
	username := r.Header.Get("X-User")
	if username == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// 2. Determine the Redis set key that contains all the short URLs for this user.
	userURLsKey := fmt.Sprintf("user:%s:urls", username)

	// 3. Get all short keys for the user from Redis.
	shortKeys, err := rdb.SMembers(ctx, userURLsKey).Result()
	if err != nil {
		http.Error(w, "Error fetching user URLs", http.StatusInternalServerError)
		return
	}

	// 4. Gather URL data.
	var results []URLData
	for _, shortKey := range shortKeys {

		// Retrieve the original URL (string key "url:<shortKey>")
		urlKey := fmt.Sprintf("url:%s", shortKey)
		originalURL, err := rdb.HGet(ctx, urlKey, "originalURL").Result()
		if errors.Is(err, redis.Nil) {
			// No data found for this shortKey; skip it
			continue
		} else if err != nil {
			http.Error(w, "Error retrieving original URL", http.StatusInternalServerError)
			return
		}

		clicksVal, err := rdb.HGet(ctx, "clicks", shortKey).Result()
		clicks := 0

		switch {
		case err == redis.Nil:
			// Not found => means no clicks yet, so clicks stays 0
		case err != nil:
			// Some unexpected error
			http.Error(w, "Error retrieving clicks count", http.StatusInternalServerError)
			return
		default:
			// Convert clicksVal to int
			fmt.Sscanf(clicksVal, "%d", &clicks)
		}

		// Retrieve the IP addresses (set "ip:<shortKey>")
		ipsKey := fmt.Sprintf("ip:%s", shortKey)
		ipList, err := rdb.SMembers(ctx, ipsKey).Result()
		if err != nil && err != redis.Nil {
			http.Error(w, "Error retrieving IP addresses", http.StatusInternalServerError)
			return
		}

		// Accumulate the data
		results = append(results, URLData{
			ShortKey:    shortKey,
			OriginalURL: originalURL,
			Clicks:      clicks,
			IPs:         ipList,
		})
	}

	// 5. Return the data as JSON
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

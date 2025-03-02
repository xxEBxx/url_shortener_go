package handlers

import (
	"URL_Shortener/utils"
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/redis/go-redis/v9"
)

var rdb *redis.Client
var ctx = context.Background()

func InitHandlers(redisClient *redis.Client) {
	rdb = redisClient
}

func ShortenHandler(w http.ResponseWriter, r *http.Request) {
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

	err := rdb.Set(ctx, shortKey, longURL, 0).Err()
	if err != nil {
		http.Error(w, "Failed to store URL in Redis", http.StatusInternalServerError)
		log.Printf("❌ Redis SET error: %v", err)
		return
	}

	fmt.Fprintf(w, "Short URL: http://localhost:8080/%s\n", shortKey)
}

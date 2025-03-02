package main

import (
	"URL_Shortener/handlers"
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
)

var (
	ctx = context.Background()
	rdb *redis.Client // Global Redis client
)

// Load environment variables from .env file
func loadEnv() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("❌ Error loading .env file")
	}
}

// Initialize Redis connection
func initRedis() {
	redisHost := os.Getenv("REDIS_HOST")
	redisPassword := os.Getenv("REDIS_PASSWORD")

	if redisHost == "" {
		log.Fatal("❌ REDIS_HOST environment variable is not set")
	}

	fmt.Println("🔹 Connecting to Redis server on:", redisHost)

	rdb = redis.NewClient(&redis.Options{
		Addr:     redisHost, // e.g., "your-redis-host:14092"
		Username: "default", // If your Redis requires a username
		Password: redisPassword,
		DB:       0,
	})

	// 🔹 Test Redis Connection
	_, err := rdb.Ping(ctx).Result()
	if err != nil {
		log.Fatalf("❌ Redis connection failed: %v", err)
	} else {
		fmt.Println("✅ Successfully connected to Redis!")
	}
}

// Fetch all keys and display their values from Redis
func getAllKeys() {
	if rdb == nil {
		log.Println("❌ Redis client is not initialized")
		return
	}

	// 🔹 Retrieve all keys
	keys, err := rdb.Keys(ctx, "*").Result()
	if err != nil {
		log.Fatalf("❌ Failed to retrieve keys: %v", err)
		return
	}

	if len(keys) == 0 {
		fmt.Println("⚠️ No keys found in Redis.")
		return
	}

	// 🔹 Fetch values efficiently using MGET (batch retrieval)
	values, err := rdb.MGet(ctx, keys...).Result()
	if err != nil {
		log.Fatalf("❌ Failed to retrieve values: %v", err)
		return
	}

	// 🔹 Display keys and their values
	fmt.Println("🔹 Redis Database Contents:")
	for i, key := range keys {
		fmt.Printf("📌 %s: %v\n", key, values[i])
	}
}

// Ensure Redis is initialized before handling requests
func main() {
	loadEnv()    // Load .env variables
	initRedis()  // Initialize Redis
	getAllKeys() // Fetch all Redis keys

	handlers.InitHandlers(rdb)
	http.HandleFunc("/shorten", handlers.ShortenHandler)

	fmt.Println("🚀 URL Shortener running on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

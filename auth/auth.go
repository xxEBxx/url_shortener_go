package auth

import (
	"context"
	"errors"
	"log"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
)

var ctx = context.Background()

type User struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// Hash password before storing
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// Check hashed password
func CheckPassword(hashedPassword, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	return err == nil
}

// Generate JWT token for a user
func GenerateJWT(username string) (string, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		log.Fatal("❌ JWT_SECRET is not set")
	}

	claims := jwt.MapClaims{
		"username": username,
		"exp":      time.Now().Add(time.Hour * 24).Unix(), // Token expires in 24 hours
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// Validate JWT token
func ValidateJWT(tokenString string) (string, error) {
	secret := os.Getenv("JWT_SECRET")
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})

	if err != nil || !token.Valid {
		return "", errors.New("invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", errors.New("cannot parse claims")
	}

	username, exists := claims["username"].(string)
	if !exists {
		return "", errors.New("invalid token structure")
	}

	return username, nil
}

// Store user in Redis
func RegisterUser(rdb *redis.Client, user User) error {
	hashedPassword, err := HashPassword(user.Password)
	if err != nil {
		return err
	}

	// Store user in Redis
	_, err = rdb.HSet(ctx, "user:"+user.Username, "password", hashedPassword).Result()
	return err
}

// Authenticate user and return JWT
func AuthenticateUser(rdb *redis.Client, user User) (string, error) {
	storedPassword, err := rdb.HGet(ctx, "user:"+user.Username, "password").Result()
	if err != nil {
		return "", errors.New("user not found")
	}

	if !CheckPassword(storedPassword, user.Password) {
		return "", errors.New("incorrect password")
	}

	return GenerateJWT(user.Username)
}

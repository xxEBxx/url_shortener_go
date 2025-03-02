package utils

import "net/http"

func GetIP(r *http.Request) string {
	// Check if behind a proxy (e.g., Cloudflare, Nginx)
	realIP := r.Header.Get("X-Forwarded-For")
	if realIP != "" {
		return realIP
	}
	return r.RemoteAddr // Default IP retrieval
}

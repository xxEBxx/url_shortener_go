package main

import (
	"URL_Shortener/utils"
	"fmt"
	"net/http"
)

func main() {
	http.HandleFunc("/", func(writer http.ResponseWriter, req *http.Request) {
		// Serve index page
	})

	http.HandleFunc("/shorten", func(writer http.ResponseWriter, req *http.Request) {
		// Get the URL to shorten from the request
		url := req.FormValue("url")
		// Close the body when done
		fmt.Println("Payload: ", url)
		// Shorten the URL
		shortURL := utils.GetShortCode()
		//fullShortURL := fmt.Sprintf("http://localhost:8080/r/%s", shortURL)
		// Generated short URL
		fmt.Printf("Generated short URL: %s\n", shortURL) // Log to console
		// @TODO: Store {shortcode: url} in Redis
		// @TODO return the shortened URL in the UI
	})

	fmt.Println("Server is running on http://localhost:8080")
	http.ListenAndServe(":8080", nil)
}

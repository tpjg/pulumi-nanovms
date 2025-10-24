package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

func main() {
	fmt.Println("Printing all environment variables.")
	env := os.Environ()
	for _, v := range env {
		fmt.Println(v)
	}

	// Set up HTTP handlers
	http.HandleFunc("/", envHandler)
	http.HandleFunc("/metadata/v1/", metadataHandler)

	// Start the server
	port := ":8888"
	fmt.Printf("\nStarting web server on port %s...\n", port)
	fmt.Printf("- Environment variables: http://localhost%s/\n", port)
	fmt.Printf("- Metadata endpoint: http://localhost%s/metadata/v1/...\n", port)

	log.Fatal(http.ListenAndServe(port, nil))
}

// envHandler serves the environment variables on the root path
func envHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	fmt.Fprintf(w, "<html><head><title>Environment Variables</title></head><body>")
	fmt.Fprintf(w, "<h1>Environment Variables</h1>")
	fmt.Fprintf(w, "<pre>")

	env := os.Environ()
	for _, v := range env {
		fmt.Fprintf(w, "%s\n", v)
	}

	fmt.Fprintf(w, "</pre>")
	fmt.Fprintf(w, "<hr>")
	fmt.Fprintf(w, "<p>Access metadata at <a href=\"/metadata/v1/\">/metadata/v1/</a></p>")
	fmt.Fprintf(w, "</body></html>")
}

// metadataHandler proxies requests to the cloud metadata endpoint
func metadataHandler(w http.ResponseWriter, r *http.Request) {
	// Build the metadata URL by using the incoming path
	metadataURL := "http://169.254.169.254" + r.URL.Path
	if r.URL.RawQuery != "" {
		metadataURL += "?" + r.URL.RawQuery
	}

	// Create a new request to the metadata endpoint
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	req, err := http.NewRequest(r.Method, metadataURL, r.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error creating request: %v", err), http.StatusInternalServerError)
		return
	}

	// Copy headers from the original request
	for key, values := range r.Header {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	// Make the request to the metadata endpoint
	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error fetching metadata: %v", err), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Copy response headers
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	// Copy status code
	w.WriteHeader(resp.StatusCode)

	// Copy response body
	io.Copy(w, resp.Body)
}

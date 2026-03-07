package main

import (
	"fmt"
	"net/http"
	"time"
)

func main() {
	// CHANGE THIS VERSION TO TEST HOT RELOAD
	version := "1.0"

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "🚀 Hot Reload Successful!\n")
		fmt.Fprintf(w, "Current Version: %s\n", version)
		fmt.Fprintf(w, "Server Time: %s\n", time.Now().Format(time.Kitchen))
	})

	fmt.Printf("🚀 Test Server starting on http://localhost:8080 (Version %s)\n", version)
	fmt.Println("Try changing the version in main.go to see the tool work!")

	err := http.ListenAndServe(":8080", mux)
	if err != nil {
		fmt.Printf("Error starting server: %s\n", err)
	}
}

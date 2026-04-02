// ABOUTME: Minimal HTTP server for benchmarking ganda throughput.
// ABOUTME: Returns 200 OK with no body, no framework, no middleware.
package main

import (
	"fmt"
	"net/http"
	"os"
)

func main() {
	port := "9876"
	if len(os.Args) > 1 {
		port = os.Args[1]
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	})

	fmt.Fprintf(os.Stderr, "bench server listening on :%s\n", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

// List robots and tasks using the SAI AUROSY API.
// Usage: go run list-robots-and-tasks.go [base_url] [api_key]
// Example: go run list-robots-and-tasks.go http://localhost:8080/api/v1 sk-integration-abc123
package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

func main() {
	baseURL := "http://localhost:8080/api/v1"
	apiKey := ""
	if len(os.Args) >= 2 {
		baseURL = os.Args[1]
	}
	if len(os.Args) >= 3 {
		apiKey = os.Args[2]
	}
	if apiKey == "" {
		fmt.Println("Usage: go run list-robots-and-tasks.go [base_url] [api_key]")
		os.Exit(1)
	}

	client := &http.Client{}
	req, _ := http.NewRequest("GET", baseURL+"/robots", nil)
	req.Header.Set("X-API-Key", apiKey)

	resp, err := client.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "robots: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	var robots []any
	if err := json.NewDecoder(resp.Body).Decode(&robots); err != nil {
		fmt.Fprintf(os.Stderr, "decode robots: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("=== Listing robots ===")
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(robots)

	req, _ = http.NewRequest("GET", baseURL+"/tasks", nil)
	req.Header.Set("X-API-Key", apiKey)
	resp, err = client.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "tasks: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	var tasks []any
	if err := json.NewDecoder(resp.Body).Decode(&tasks); err != nil {
		fmt.Fprintf(os.Stderr, "decode tasks: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("\n=== Listing tasks ===")
	enc.Encode(tasks)
}

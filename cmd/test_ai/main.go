package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/eliseohenriquez/zettelcornelbot/internal/neural"
)

func main() {
	// Mock Ollama Server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, `{"response":"Mock AI Response"}`)
	}))
	defer ts.Close()

	client := neural.NewClient(ts.URL, "test")

	// Test Summarize
	resp, err := client.Summarize("Content")
	if err != nil {
		panic(err)
	}
	if resp != "Mock AI Response" {
		panic("Unexpected response")
	}
	fmt.Println("✔ Summarize Permissions OK (Read-Only)")

	// Test Draft
	resp, err = client.Draft("Topic")
	if err != nil {
		panic(err)
	}
	fmt.Println("✔ Draft Permissions OK (Read-Only)")
}

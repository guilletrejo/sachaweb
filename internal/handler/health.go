// Package handler contains HTTP handlers — functions that receive an
// HTTP request and write an HTTP response. Each handler is responsible
// for ONE endpoint (or a small group of related endpoints).
//
// Handlers are the "edge" of your backend — they translate between
// HTTP (the transport protocol) and your business logic.
package handler

import (
	"encoding/json"
	"net/http"
)

// HealthResponse is the JSON structure returned by the health endpoint.
// Having a named struct (instead of map[string]string) makes the response
// shape explicit and documented in your code.
type HealthResponse struct {
	Status string `json:"status"`
}

// HandleHealth returns an http.HandlerFunc for the GET /health endpoint.
//
// WHY A FUNCTION THAT RETURNS A FUNCTION?
// This is a factory pattern. Right now HandleHealth takes no parameters,
// but in Phase 8, it will take a database connection and Redis client
// to check if they're alive. The factory pattern lets you inject
// dependencies without global variables.
//
// WHAT IS http.HandlerFunc?
// It's a function type defined as: type HandlerFunc func(ResponseWriter, *Request)
// Any function with that exact signature automatically satisfies the
// http.Handler interface. This is Go's way of turning a plain function
// into an interface implementation — no "implements" keyword needed.
func HandleHealth() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Set the Content-Type header BEFORE writing the body.
		// This tells the client "the response body is JSON."
		// If you forget this, browsers might treat it as plain text.
		w.Header().Set("Content-Type", "application/json")

		// WriteHeader sets the HTTP status code. 200 = OK.
		// If you don't call WriteHeader, Go defaults to 200,
		// but being explicit is better for readability.
		w.WriteHeader(http.StatusOK)

		// json.NewEncoder(w) creates a JSON encoder that writes directly
		// to the ResponseWriter (which is an io.Writer).
		// .Encode(value) serializes the struct to JSON and writes it.
		//
		// This is more efficient than json.Marshal() + w.Write() because
		// it streams the JSON directly to the response without creating
		// an intermediate []byte in memory. At scale, this matters.
		json.NewEncoder(w).Encode(HealthResponse{Status: "ok"})
	}
}

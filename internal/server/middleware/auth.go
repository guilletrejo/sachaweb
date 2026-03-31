// Package middleware contains HTTP middleware functions.
//
// WHAT IS MIDDLEWARE?
// Middleware is code that runs BEFORE (or after) your handler.
// It's like a checkpoint that every request passes through.
//
// Without middleware:
//   Request → Handler
//
// With middleware:
//   Request → Auth Middleware → Handler
//   Request → Auth Middleware → ✗ (401 Unauthorized, handler never runs)
//
// THE Go MIDDLEWARE PATTERN:
// A middleware is a function that takes an http.Handler and returns an http.Handler.
// Signature: func(next http.Handler) http.Handler
//
// It wraps the next handler: do something before, call next, do something after.
// This is the "decorator pattern" — same interface in and out, with added behavior.
//
// You can STACK middleware:
//   Request → Logging → Auth → RateLimit → Handler
// Each one calls the next. If any middleware rejects the request, the chain stops.
package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/guilletrejo/sachaweb/internal/service"
)

// contextKey is a custom type for context keys.
//
// WHY A CUSTOM TYPE?
// context.WithValue takes an any key, which means collisions are possible.
// If two packages both use the string "userID" as a key, they'd overwrite
// each other. Using a custom unexported type makes collisions impossible —
// only this package can create values of type contextKey.
type contextKey string

const userIDKey contextKey = "userID"

// UserIDFromContext extracts the authenticated user's ID from the context.
// Handlers call this to find out WHO is making the request.
//
// This is exported so handlers can use it:
//   userID := middleware.UserIDFromContext(r.Context())
func UserIDFromContext(ctx context.Context) string {
	userID, _ := ctx.Value(userIDKey).(string)
	return userID
}

// Auth returns a middleware that validates JWT tokens.
//
// HOW IT WORKS:
// 1. Extract the token from the "Authorization: Bearer <token>" header
// 2. Validate the token using the UserService
// 3. If valid: store the user ID in the context, call the next handler
// 4. If invalid: return 401 Unauthorized, stop the chain
//
// WHY DOES IT TAKE *service.UserService?
// The middleware needs to validate JWT tokens. The UserService has the
// ValidateToken method and knows the JWT secret. Rather than passing
// the secret directly, we pass the service — this keeps the JWT logic
// in one place and makes the middleware thinner.
func Auth(userService *service.UserService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Step 1: Extract the Authorization header.
			// Expected format: "Bearer eyJhbGciOiJIUzI1NiIs..."
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				writeAuthError(w, "missing Authorization header")
				return
			}

			// Step 2: Parse the "Bearer <token>" format.
			// strings.TrimPrefix removes "Bearer " from the front.
			// If the header doesn't start with "Bearer ", the token will
			// be invalid and ValidateToken will reject it.
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || parts[0] != "Bearer" {
				writeAuthError(w, "Authorization header must be: Bearer <token>")
				return
			}
			tokenString := parts[1]

			// Step 3: Validate the token and extract the user ID.
			userID, err := userService.ValidateToken(tokenString)
			if err != nil {
				writeAuthError(w, err.Error())
				return
			}

			// Step 4: Store the user ID in the request context.
			//
			// context.WithValue creates a NEW context with the added value.
			// r.WithContext creates a NEW request with the new context.
			// Neither modifies the original — they're immutable.
			//
			// The handler (next in the chain) can then read the user ID:
			//   userID := middleware.UserIDFromContext(r.Context())
			ctx := context.WithValue(r.Context(), userIDKey, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// writeAuthError writes a 401 Unauthorized JSON response.
func writeAuthError(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	json.NewEncoder(w).Encode(map[string]string{
		"error": message,
		"code":  "UNAUTHORIZED",
	})
}

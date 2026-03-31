package model

import (
	"net/mail"
	"strings"
)

// User represents a registered user in the system.
//
// IMPORTANT: PasswordHash has the json tag "-" which means it is NEVER
// included in JSON responses. If you accidentally sent the hash to a client,
// it wouldn't be the end of the world (bcrypt hashes are designed to be
// safe even if leaked), but it's a bad practice. Defense in depth:
// don't expose what you don't need to.
type User struct {
	ID           string `json:"id"`
	Email        string `json:"email"`
	PasswordHash string `json:"-"` // NEVER serialize to JSON
}

// RegisterRequest is the JSON body for POST /register.
//
// WHY SEPARATE FROM User?
// Same reason as CreateProductRequest — the client sends email + password,
// but the server generates the ID and hashes the password. The client
// should never send an ID or a pre-hashed password.
type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Validate checks that registration data is valid.
func (r *RegisterRequest) Validate() error {
	if strings.TrimSpace(r.Email) == "" {
		return &ValidationError{Field: "email", Message: "cannot be empty"}
	}
	// net/mail.ParseAddress is Go's stdlib email parser. It checks for
	// a valid format (something@something.something). It's not perfect
	// (no email regex is), but it catches obvious mistakes.
	if _, err := mail.ParseAddress(r.Email); err != nil {
		return &ValidationError{Field: "email", Message: "must be a valid email address"}
	}
	// Minimum 8 characters is a common baseline. In production, you might
	// also check for complexity (uppercase, numbers, symbols), but that's
	// increasingly considered bad UX — length matters more than complexity.
	if len(r.Password) < 8 {
		return &ValidationError{Field: "password", Message: "must be at least 8 characters"}
	}
	return nil
}

// LoginRequest is the JSON body for POST /login.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Validate checks that login data is present.
func (r *LoginRequest) Validate() error {
	if strings.TrimSpace(r.Email) == "" {
		return &ValidationError{Field: "email", Message: "cannot be empty"}
	}
	if r.Password == "" {
		return &ValidationError{Field: "password", Message: "cannot be empty"}
	}
	return nil
}

// AuthResponse is returned on successful login or registration.
// It contains the JWT token the client must include in future requests.
//
// HOW THE CLIENT USES THIS:
// 1. Client sends POST /login with email + password
// 2. Server returns { "token": "eyJhbG..." }
// 3. Client stores the token (localStorage, cookie, mobile keychain)
// 4. On every subsequent request, client sends:
//      Authorization: Bearer eyJhbG...
// 5. Server middleware validates the token and extracts the user ID
type AuthResponse struct {
	Token string `json:"token"`
}

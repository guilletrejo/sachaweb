package service

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/guilletrejo/sachaweb/internal/model"
	"github.com/guilletrejo/sachaweb/internal/repository"
)

// userIDCounter generates unique user IDs. Same approach as products.
var userIDCounter atomic.Int64

// UserService handles authentication business logic:
// registering new users, logging in, and generating JWT tokens.
type UserService struct {
	repo      repository.UserRepository
	jwtSecret []byte // the secret key used to sign JWT tokens
}

// NewUserService creates a new user service.
//
// The jwtSecret is a shared secret between this server and itself.
// It's used to SIGN tokens (prove they came from us) and VERIFY tokens
// (prove they haven't been tampered with). Anyone with this secret
// can forge tokens, so it MUST be kept private — never commit it to git,
// always load it from an environment variable.
func NewUserService(repo repository.UserRepository, jwtSecret string) *UserService {
	return &UserService{
		repo:      repo,
		jwtSecret: []byte(jwtSecret),
	}
}

// Register creates a new user account.
//
// THE PASSWORD FLOW:
// 1. Client sends plain text password over HTTPS (encrypted in transit)
// 2. Server hashes it with bcrypt (one-way transformation)
// 3. Server stores ONLY the hash, discards the plain text
// 4. On login: hash the submitted password, compare hashes
//
// WHY bcrypt?
// bcrypt includes a "cost" parameter that controls how slow hashing is.
// bcrypt.DefaultCost is 10, meaning 2^10 = 1024 rounds of hashing.
// One hash takes ~100ms — fast enough for login, but an attacker trying
// billions of passwords would need millions of years.
//
// bcrypt also generates a random "salt" per password. Two users with
// the same password get DIFFERENT hashes. This prevents "rainbow table"
// attacks (precomputed hash dictionaries).
func (s *UserService) Register(ctx context.Context, req model.RegisterRequest) (model.AuthResponse, error) {
	if err := req.Validate(); err != nil {
		return model.AuthResponse{}, err
	}

	// Hash the password. bcrypt.GenerateFromPassword:
	// 1. Generates a random 16-byte salt
	// 2. Runs the bcrypt algorithm with the password + salt + cost
	// 3. Returns a 60-character string like: "$2a$10$N9qo8uLO..."
	//    $2a = bcrypt version, $10 = cost, rest = salt + hash
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return model.AuthResponse{}, fmt.Errorf("hashing password: %w", err)
	}

	id := fmt.Sprintf("user_%d", userIDCounter.Add(1))

	user := model.User{
		ID:           id,
		Email:        req.Email,
		PasswordHash: string(hash),
	}

	// Store the user. If the email already exists, the repository
	// returns a ConflictError (from the UNIQUE constraint).
	if err := s.repo.Create(ctx, user); err != nil {
		return model.AuthResponse{}, err
	}

	// Generate a JWT token so the user is immediately logged in
	// after registration (no need to call /login separately).
	token, err := s.generateToken(user.ID)
	if err != nil {
		return model.AuthResponse{}, err
	}

	return model.AuthResponse{Token: token}, nil
}

// Login authenticates a user and returns a JWT token.
//
// THE LOGIN FLOW:
// 1. Find the user by email
// 2. Compare the submitted password against the stored hash
// 3. If they match, generate and return a JWT token
// 4. If they don't match, return a generic "invalid credentials" error
//
// WHY "invalid credentials" AND NOT "wrong password"?
// If you say "wrong password", an attacker knows the EMAIL exists
// and only needs to guess the password. "Invalid credentials" doesn't
// reveal whether the email or password was wrong. This is called
// "constant-time error messages" and it's standard security practice.
func (s *UserService) Login(ctx context.Context, req model.LoginRequest) (model.AuthResponse, error) {
	if err := req.Validate(); err != nil {
		return model.AuthResponse{}, err
	}

	// Look up the user by email.
	user, err := s.repo.FindByEmail(ctx, req.Email)
	if err != nil {
		// User not found — but don't reveal that. Return generic error.
		if _, ok := err.(*model.NotFoundError); ok {
			return model.AuthResponse{}, &model.AuthError{Message: "invalid credentials"}
		}
		return model.AuthResponse{}, err
	}

	// Compare the submitted password with the stored hash.
	// bcrypt.CompareHashAndPassword:
	// 1. Extracts the salt from the stored hash
	// 2. Hashes the submitted password with that same salt
	// 3. Compares the result — if they match, the password is correct
	//
	// This comparison is done in CONSTANT TIME to prevent timing attacks
	// (where an attacker measures how long the comparison takes to guess
	// how many characters they got right).
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password))
	if err != nil {
		return model.AuthResponse{}, &model.AuthError{Message: "invalid credentials"}
	}

	token, err := s.generateToken(user.ID)
	if err != nil {
		return model.AuthResponse{}, err
	}

	return model.AuthResponse{Token: token}, nil
}

// generateToken creates a JWT token containing the user's ID.
//
// WHAT IS A JWT (JSON Web Token)?
// A JWT is a signed JSON payload that the server gives to the client.
// The client sends it back on every request to prove who they are.
//
// A JWT has 3 parts separated by dots: xxxxx.yyyyy.zzzzz
//   Header:  {"alg": "HS256", "typ": "JWT"}     → which algorithm was used
//   Payload: {"user_id": "user_1", "exp": 1234}  → the actual data (claims)
//   Signature: HMAC-SHA256(header + payload, secret) → proves it wasn't tampered with
//
// The payload is NOT encrypted — anyone can decode it (it's just base64).
// But nobody can MODIFY it without the secret, because the signature
// would no longer match. That's the security model: integrity, not secrecy.
//
// WHY NOT SESSIONS?
// Sessions store state on the server (in memory or database).
// JWTs are STATELESS — the server doesn't store anything. The token
// itself contains everything needed to identify the user. This is
// important for scalability: if you have 10 servers behind a load
// balancer, they all just verify the signature — no shared session store.
func (s *UserService) generateToken(userID string) (string, error) {
	// Claims are the data stored in the JWT payload.
	// RegisteredClaims are standard fields defined by the JWT spec.
	claims := jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(24 * time.Hour).Unix(), // expires in 24 hours
		"iat":     time.Now().Unix(),                     // issued at
	}

	// Create the token with the HS256 signing method.
	// HS256 = HMAC-SHA256 — a symmetric algorithm (same key signs and verifies).
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Sign the token with our secret key.
	return token.SignedString(s.jwtSecret)
}

// ValidateToken parses and validates a JWT token string.
// Returns the user ID from the token's claims.
//
// This is called by the auth middleware on every protected request.
func (s *UserService) ValidateToken(tokenString string) (string, error) {
	// jwt.Parse decodes the token and verifies the signature.
	// The callback function provides the secret key for verification.
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
		// Verify the signing method is what we expect.
		// This prevents an attack where someone changes the algorithm
		// to "none" (no signature) and forges tokens.
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.jwtSecret, nil
	})
	if err != nil {
		return "", &model.AuthError{Message: "invalid or expired token"}
	}

	// Extract the user_id claim from the payload.
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", &model.AuthError{Message: "invalid token claims"}
	}

	userID, ok := claims["user_id"].(string)
	if !ok {
		return "", &model.AuthError{Message: "invalid token: missing user_id"}
	}

	return userID, nil
}

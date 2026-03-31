package handler

import (
	"net/http"

	"github.com/guilletrejo/sachaweb/internal/model"
	"github.com/guilletrejo/sachaweb/internal/service"
)

// UserHandler holds HTTP handlers for authentication endpoints.
type UserHandler struct {
	service *service.UserService
}

// NewUserHandler creates a new handler with the given user service.
func NewUserHandler(svc *service.UserService) *UserHandler {
	return &UserHandler{service: svc}
}

// HandleRegister handles POST /register — creates a new user account.
//
// REQUEST:
//   POST /register
//   {"email": "user@example.com", "password": "mysecretpassword"}
//
// RESPONSE (201 Created):
//   {"token": "eyJhbGciOiJIUzI1NiIs..."}
//
// The token is returned immediately so the user is logged in after
// registration — no need to call /login separately.
func (h *UserHandler) HandleRegister() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req model.RegisterRequest
		if err := decodeJSON(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, "INVALID_JSON", "Request body must be valid JSON")
			return
		}

		resp, err := h.service.Register(r.Context(), req)
		if err != nil {
			handleServiceError(w, err)
			return
		}

		writeJSON(w, http.StatusCreated, resp)
	}
}

// HandleLogin handles POST /login — authenticates a user.
//
// REQUEST:
//   POST /login
//   {"email": "user@example.com", "password": "mysecretpassword"}
//
// RESPONSE (200 OK):
//   {"token": "eyJhbGciOiJIUzI1NiIs..."}
//
// ERROR (401 Unauthorized):
//   {"error": "invalid credentials", "code": "AUTH_ERROR"}
func (h *UserHandler) HandleLogin() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req model.LoginRequest
		if err := decodeJSON(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, "INVALID_JSON", "Request body must be valid JSON")
			return
		}

		resp, err := h.service.Login(r.Context(), req)
		if err != nil {
			handleServiceError(w, err)
			return
		}

		writeJSON(w, http.StatusOK, resp)
	}
}

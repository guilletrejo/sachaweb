package handler

import (
	"encoding/json"
	"net/http"

	"github.com/guilletrejo/sachaweb/internal/model"
	"github.com/guilletrejo/sachaweb/internal/service"
)

// ErrorResponse is the standard JSON error format for ALL error responses.
// Having a consistent error format is critical for API consumers —
// they can always expect {"error": "...", "code": "..."} on errors.
type ErrorResponse struct {
	Error string `json:"error"` // Human-readable error message
	Code  string `json:"code"`  // Machine-readable error code (e.g., "NOT_FOUND")
}

// ProductHandler holds all HTTP handlers related to products.
// It depends on ProductService, not the repository directly.
// Handlers talk to services, services talk to repositories.
type ProductHandler struct {
	service *service.ProductService
}

// NewProductHandler creates a new handler with the given service.
func NewProductHandler(svc *service.ProductService) *ProductHandler {
	return &ProductHandler{service: svc}
}

// HandleList handles GET /products — returns all products.
func (h *ProductHandler) HandleList() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		products, err := h.service.ListProducts()
		if err != nil {
			writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
			return
		}

		writeJSON(w, http.StatusOK, products)
	}
}

// HandleGet handles GET /products/{id} — returns a single product.
//
// {id} is a path parameter. In the URL "/products/prod_1", the {id}
// is "prod_1". Go 1.22's enhanced ServeMux extracts this automatically
// via r.PathValue("id").
func (h *ProductHandler) HandleGet() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// r.PathValue("id") extracts the {id} from the URL pattern.
		// This was added in Go 1.22 — before that, you had to parse
		// the URL manually or use a third-party router.
		id := r.PathValue("id")

		product, err := h.service.GetProduct(id)
		if err != nil {
			handleServiceError(w, err)
			return
		}

		writeJSON(w, http.StatusOK, product)
	}
}

// HandleCreate handles POST /products — creates a new product.
//
// HTTP METHOD MEANINGS (REST conventions):
//   GET    = Read (safe, no side effects)
//   POST   = Create (creates a new resource)
//   PUT    = Update/Replace (replaces the entire resource)
//   DELETE = Delete (removes the resource)
//
// STATUS CODE CONVENTIONS:
//   200 OK         = Success (for GET, PUT, DELETE)
//   201 Created    = Success (specifically for POST — a new resource was created)
//   204 No Content = Success (for DELETE — nothing to return)
//   400 Bad Request  = Client sent invalid data
//   404 Not Found    = Resource doesn't exist
//   409 Conflict     = Operation conflicts with existing state
//   500 Internal Server Error = Unexpected server failure
func (h *ProductHandler) HandleCreate() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Step 1: Decode the JSON request body into our DTO.
		var req model.CreateProductRequest
		if err := decodeJSON(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, "INVALID_JSON", "Request body must be valid JSON")
			return
		}

		// Step 2: Call the service (which validates and stores).
		product, err := h.service.CreateProduct(req)
		if err != nil {
			handleServiceError(w, err)
			return
		}

		// Step 3: Return 201 Created with the new product (including its ID).
		// 201 (not 200) because a new resource was created.
		writeJSON(w, http.StatusCreated, product)
	}
}

// HandleUpdate handles PUT /products/{id} — updates an existing product.
func (h *ProductHandler) HandleUpdate() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")

		var req model.CreateProductRequest
		if err := decodeJSON(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, "INVALID_JSON", "Request body must be valid JSON")
			return
		}

		product, err := h.service.UpdateProduct(id, req)
		if err != nil {
			handleServiceError(w, err)
			return
		}

		writeJSON(w, http.StatusOK, product)
	}
}

// HandleDelete handles DELETE /products/{id} — removes a product.
func (h *ProductHandler) HandleDelete() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")

		if err := h.service.DeleteProduct(id); err != nil {
			handleServiceError(w, err)
			return
		}

		// 204 No Content — the delete succeeded, but there's nothing to return.
		// This is the standard response for successful DELETE operations.
		w.WriteHeader(http.StatusNoContent)
	}
}

// --- Helper functions ---
// These are private (lowercase) functions used only within this package.
// They reduce repetition across handlers.

// writeJSON encodes a value as JSON and writes it to the response.
// Used by every successful response.
func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// writeError writes a standardized error response.
func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, ErrorResponse{Error: message, Code: code})
}

// decodeJSON reads the request body and decodes it into the given value.
//
// DisallowUnknownFields() is a security/correctness measure: if a client
// sends {"name": "Laptop", "proce": 999} (typo in "price"), without this
// setting, the typo would be silently ignored and price would default to 0.
// With DisallowUnknownFields, the decode fails and the client gets a clear
// 400 error — much easier to debug.
func decodeJSON(r *http.Request, v any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(v)
}

// handleServiceError maps domain error types to HTTP status codes.
//
// THIS IS WHERE THE CUSTOM ERROR TYPES PAY OFF.
// The service returns domain errors (NotFoundError, ValidationError).
// The handler translates them to HTTP status codes. The service never
// knows about HTTP, and the handler never knows about business rules.
func handleServiceError(w http.ResponseWriter, err error) {
	switch e := err.(type) {
	case *model.NotFoundError:
		writeError(w, http.StatusNotFound, "NOT_FOUND", e.Error())
	case *model.ValidationError:
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", e.Error())
	case *model.ConflictError:
		writeError(w, http.StatusConflict, "CONFLICT", e.Error())
	default:
		// Any error we don't recognize is a 500 — something unexpected happened.
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected error occurred")
	}
}

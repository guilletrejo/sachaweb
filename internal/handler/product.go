package handler

import (
	"encoding/json"
	"net/http"

	"github.com/guilletrejo/sachaweb/internal/model"
	"github.com/guilletrejo/sachaweb/internal/service"
)

// ErrorResponse is the standard JSON error format for ALL error responses.
type ErrorResponse struct {
	Error string `json:"error"`
	Code  string `json:"code"`
}

// ProductHandler holds all HTTP handlers related to products.
type ProductHandler struct {
	service *service.ProductService
}

// NewProductHandler creates a new handler with the given service.
func NewProductHandler(svc *service.ProductService) *ProductHandler {
	return &ProductHandler{service: svc}
}

// HandleList handles GET /products — returns all products.
//
// WHAT CHANGED IN PHASE 3?
// We now pass r.Context() to the service. r.Context() returns the
// context associated with this HTTP request. If the client disconnects,
// this context gets cancelled, which propagates all the way down to
// the database query through: handler → service → repository → pgx → PostgreSQL.
func (h *ProductHandler) HandleList() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		products, err := h.service.ListProducts(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
			return
		}

		writeJSON(w, http.StatusOK, products)
	}
}

// HandleGet handles GET /products/{id} — returns a single product.
func (h *ProductHandler) HandleGet() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")

		product, err := h.service.GetProduct(r.Context(), id)
		if err != nil {
			handleServiceError(w, err)
			return
		}

		writeJSON(w, http.StatusOK, product)
	}
}

// HandleCreate handles POST /products — creates a new product.
func (h *ProductHandler) HandleCreate() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req model.CreateProductRequest
		if err := decodeJSON(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, "INVALID_JSON", "Request body must be valid JSON")
			return
		}

		product, err := h.service.CreateProduct(r.Context(), req)
		if err != nil {
			handleServiceError(w, err)
			return
		}

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

		product, err := h.service.UpdateProduct(r.Context(), id, req)
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

		if err := h.service.DeleteProduct(r.Context(), id); err != nil {
			handleServiceError(w, err)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

// --- Helper functions ---

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, ErrorResponse{Error: message, Code: code})
}

func decodeJSON(r *http.Request, v any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(v)
}

func handleServiceError(w http.ResponseWriter, err error) {
	switch e := err.(type) {
	case *model.NotFoundError:
		writeError(w, http.StatusNotFound, "NOT_FOUND", e.Error())
	case *model.ValidationError:
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", e.Error())
	case *model.ConflictError:
		writeError(w, http.StatusConflict, "CONFLICT", e.Error())
	default:
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected error occurred")
	}
}

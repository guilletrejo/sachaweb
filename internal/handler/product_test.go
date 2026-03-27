package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/guilletrejo/sachaweb/internal/model"
	"github.com/guilletrejo/sachaweb/internal/repository"
	"github.com/guilletrejo/sachaweb/internal/service"
)

// newTestHandler creates a ProductHandler wired to an in-memory repository.
// This is the same wiring as main.go, but with test data.
//
// WHY NOT MOCK THE SERVICE?
// In Phase 6, we'll write unit tests with mocks for isolated testing.
// For now, using the real service + in-memory repo is simpler and
// tests the full handler→service→repo chain (a "thin integration test").
func newTestHandler() *ProductHandler {
	repo := repository.NewMemoryProductRepo([]model.Product{
		{ID: "1", Name: "Test Product", Description: "A test product", Price: 1000, Category: "test"},
		{ID: "2", Name: "Another Product", Description: "Another test product", Price: 2000, Category: "test"},
	})
	svc := service.NewProductService(repo)
	return NewProductHandler(svc)
}

func TestHandleList(t *testing.T) {
	h := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/products", nil)
	rec := httptest.NewRecorder()

	h.HandleList()(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var products []model.Product
	json.NewDecoder(rec.Body).Decode(&products)
	if len(products) != 2 {
		t.Errorf("expected 2 products, got %d", len(products))
	}
}

func TestHandleGet_Found(t *testing.T) {
	h := newTestHandler()

	// To test path parameters with Go 1.22's ServeMux, we need to use
	// a real ServeMux and send the request through it. This is because
	// r.PathValue("id") only works when the request was routed by a mux
	// that defined the {id} pattern.
	mux := http.NewServeMux()
	mux.HandleFunc("GET /products/{id}", h.HandleGet())

	req := httptest.NewRequest(http.MethodGet, "/products/1", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var product model.Product
	json.NewDecoder(rec.Body).Decode(&product)
	if product.Name != "Test Product" {
		t.Errorf("expected 'Test Product', got '%s'", product.Name)
	}
}

func TestHandleGet_NotFound(t *testing.T) {
	h := newTestHandler()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /products/{id}", h.HandleGet())

	req := httptest.NewRequest(http.MethodGet, "/products/nonexistent", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rec.Code)
	}

	var errResp ErrorResponse
	json.NewDecoder(rec.Body).Decode(&errResp)
	if errResp.Code != "NOT_FOUND" {
		t.Errorf("expected code 'NOT_FOUND', got '%s'", errResp.Code)
	}
}

func TestHandleCreate_Success(t *testing.T) {
	h := newTestHandler()

	// Create the JSON request body.
	// bytes.NewBuffer creates an io.Reader from a []byte — needed for
	// httptest.NewRequest's body parameter.
	body, _ := json.Marshal(model.CreateProductRequest{
		Name:     "New Product",
		Price:    5000,
		Category: "new",
	})
	req := httptest.NewRequest(http.MethodPost, "/products", bytes.NewBuffer(body))
	rec := httptest.NewRecorder()

	h.HandleCreate()(rec, req)

	// POST should return 201 Created, not 200 OK.
	if rec.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d", rec.Code)
	}

	var product model.Product
	json.NewDecoder(rec.Body).Decode(&product)
	if product.Name != "New Product" {
		t.Errorf("expected 'New Product', got '%s'", product.Name)
	}
	// The service should have generated an ID.
	if product.ID == "" {
		t.Error("expected product to have a generated ID")
	}
}

func TestHandleCreate_ValidationError(t *testing.T) {
	h := newTestHandler()

	// Send a product with empty name — should fail validation.
	body, _ := json.Marshal(model.CreateProductRequest{
		Name:     "",
		Price:    5000,
		Category: "test",
	})
	req := httptest.NewRequest(http.MethodPost, "/products", bytes.NewBuffer(body))
	rec := httptest.NewRecorder()

	h.HandleCreate()(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}
}

func TestHandleCreate_InvalidJSON(t *testing.T) {
	h := newTestHandler()

	// Send invalid JSON.
	body := bytes.NewBufferString("not json at all")
	req := httptest.NewRequest(http.MethodPost, "/products", body)
	rec := httptest.NewRecorder()

	h.HandleCreate()(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}
}

func TestHandleUpdate_Success(t *testing.T) {
	h := newTestHandler()
	mux := http.NewServeMux()
	mux.HandleFunc("PUT /products/{id}", h.HandleUpdate())

	body, _ := json.Marshal(model.CreateProductRequest{
		Name:     "Updated Product",
		Price:    9999,
		Category: "updated",
	})
	req := httptest.NewRequest(http.MethodPut, "/products/1", bytes.NewBuffer(body))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var product model.Product
	json.NewDecoder(rec.Body).Decode(&product)
	if product.Name != "Updated Product" {
		t.Errorf("expected 'Updated Product', got '%s'", product.Name)
	}
	// ID should remain the same after update.
	if product.ID != "1" {
		t.Errorf("expected ID '1', got '%s'", product.ID)
	}
}

func TestHandleUpdate_NotFound(t *testing.T) {
	h := newTestHandler()
	mux := http.NewServeMux()
	mux.HandleFunc("PUT /products/{id}", h.HandleUpdate())

	body, _ := json.Marshal(model.CreateProductRequest{
		Name:     "Doesn't Matter",
		Price:    1000,
		Category: "test",
	})
	req := httptest.NewRequest(http.MethodPut, "/products/nonexistent", bytes.NewBuffer(body))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rec.Code)
	}
}

func TestHandleDelete_Success(t *testing.T) {
	h := newTestHandler()
	mux := http.NewServeMux()
	mux.HandleFunc("DELETE /products/{id}", h.HandleDelete())

	req := httptest.NewRequest(http.MethodDelete, "/products/1", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	// DELETE returns 204 No Content on success.
	if rec.Code != http.StatusNoContent {
		t.Errorf("expected status 204, got %d", rec.Code)
	}
}

func TestHandleDelete_NotFound(t *testing.T) {
	h := newTestHandler()
	mux := http.NewServeMux()
	mux.HandleFunc("DELETE /products/{id}", h.HandleDelete())

	req := httptest.NewRequest(http.MethodDelete, "/products/nonexistent", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rec.Code)
	}
}

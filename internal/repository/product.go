// Package repository is the DATA ACCESS layer. It's the ONLY layer
// that knows HOW data is stored (in-memory map, PostgreSQL, file, etc.).
//
// The rest of the application talks to repositories through INTERFACES,
// not concrete types. This means you can swap the storage implementation
// without changing any other code — which is exactly what we're doing
// NOW in Phase 3 by switching from an in-memory map to PostgreSQL.
//
// This is the "D" in SOLID: Dependency Inversion Principle.
// High-level code (handlers, services) depends on abstractions (interfaces),
// not on low-level details (which database engine you use).
package repository

import (
	"context"
	"sync"

	"github.com/guilletrejo/sachaweb/internal/model"
)

// ProductRepository defines the contract for product data access.
//
// WHAT CHANGED FROM PHASE 2?
// Every method now takes a context.Context as its first parameter.
//
// WHAT IS context.Context?
// Context carries deadlines, cancellation signals, and request-scoped values
// across API boundaries and between goroutines. In backend development,
// it's the #1 most important Go pattern to understand. Here's why:
//
// Imagine a user sends an HTTP request, but then closes their browser.
// The HTTP server detects this and CANCELS the request's context.
// If your database query is still running, context lets PostgreSQL know
// "stop working, nobody is waiting for this result anymore."
//
// Without context:
//   User closes browser → server keeps running the query → wastes CPU/memory
//
// With context:
//   User closes browser → context is cancelled → query stops → resources freed
//
// At MercadoLibre scale (millions of requests), this saves enormous resources.
//
// CONVENTION: context.Context is ALWAYS the first parameter, named "ctx".
// This is a universal Go convention — every Go developer expects it.
type ProductRepository interface {
	FindAll(ctx context.Context) ([]model.Product, error)
	FindByID(ctx context.Context, id string) (model.Product, error)
	Create(ctx context.Context, product model.Product) error
	Update(ctx context.Context, product model.Product) error
	Delete(ctx context.Context, id string) error
}

// MemoryProductRepo stores products in an in-memory map.
// This was our Phase 1-2 implementation. It still works (and is useful
// for tests), but the main app now uses PostgresProductRepo.
//
// Note: the context parameter is accepted but not used here —
// there's no database connection to cancel. The interface requires it
// because the PostgreSQL implementation needs it.
type MemoryProductRepo struct {
	mu       sync.RWMutex
	products map[string]model.Product
}

// NewMemoryProductRepo creates a new in-memory repository.
func NewMemoryProductRepo(initial []model.Product) *MemoryProductRepo {
	products := make(map[string]model.Product, len(initial))
	for _, p := range initial {
		products[p.ID] = p
	}
	return &MemoryProductRepo{
		products: products,
	}
}

// Compile-time check: ensure MemoryProductRepo implements ProductRepository.
var _ ProductRepository = (*MemoryProductRepo)(nil)

// FindAll returns all products. Thread-safe for concurrent reads.
// The _ before ctx means "I receive this parameter but don't use it."
// The in-memory implementation has nothing to cancel.
func (r *MemoryProductRepo) FindAll(_ context.Context) ([]model.Product, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]model.Product, 0, len(r.products))
	for _, p := range r.products {
		result = append(result, p)
	}
	return result, nil
}

// FindByID returns a single product by its ID.
func (r *MemoryProductRepo) FindByID(_ context.Context, id string) (model.Product, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	product, ok := r.products[id]
	if !ok {
		return model.Product{}, &model.NotFoundError{Resource: "product", ID: id}
	}
	return product, nil
}

// Create adds a new product.
func (r *MemoryProductRepo) Create(_ context.Context, product model.Product) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.products[product.ID]; exists {
		return &model.ConflictError{Resource: "product", ID: product.ID}
	}
	r.products[product.ID] = product
	return nil
}

// Update replaces an existing product.
func (r *MemoryProductRepo) Update(_ context.Context, product model.Product) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.products[product.ID]; !exists {
		return &model.NotFoundError{Resource: "product", ID: product.ID}
	}
	r.products[product.ID] = product
	return nil
}

// Delete removes a product by ID.
func (r *MemoryProductRepo) Delete(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.products[id]; !exists {
		return &model.NotFoundError{Resource: "product", ID: id}
	}
	delete(r.products, id)
	return nil
}

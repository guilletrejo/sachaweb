// Package repository is the DATA ACCESS layer. It's the ONLY layer
// that knows HOW data is stored (in-memory map, PostgreSQL, file, etc.).
//
// The rest of the application talks to repositories through INTERFACES,
// not concrete types. This means you can swap the storage implementation
// without changing any other code — which is exactly what we'll do
// in Phase 3 when we switch to PostgreSQL.
//
// This is the "D" in SOLID: Dependency Inversion Principle.
// High-level code (handlers, services) depends on abstractions (interfaces),
// not on low-level details (which database engine you use).
package repository

import (
	"sync"

	"github.com/guilletrejo/sachaweb/internal/model"
)

// ProductRepository defines the contract for product data access.
//
// WHAT IS AN INTERFACE IN THIS CONTEXT?
// It's a contract that says "any type that has these methods can be used
// as a ProductRepository." The handler doesn't care if products are stored
// in memory, PostgreSQL, or a text file — as long as the storage
// implements these 5 methods.
//
// WHY IS THIS POWERFUL?
// 1. Swappability: Switch from memory to PostgreSQL without changing handlers.
// 2. Testability: In tests, use a simple mock that returns hardcoded data.
// 3. Decoupling: Each layer only knows about the interface, not the implementation.
type ProductRepository interface {
	FindAll() ([]model.Product, error)
	FindByID(id string) (model.Product, error)
	Create(product model.Product) error
	Update(product model.Product) error
	Delete(id string) error
}

// MemoryProductRepo stores products in an in-memory map.
// This is our Phase 1-2 implementation. Phase 3 replaces it with PostgreSQL.
//
// WHAT IS sync.RWMutex?
//
// Go's HTTP server handles each request in a separate goroutine (lightweight
// thread). If two requests arrive at the same time — say one is reading all
// products while another is creating a new product — they both access the
// same map concurrently. In Go, concurrent reads are fine, but a concurrent
// read + write to a map causes a PANIC (crash).
//
// sync.RWMutex (Read-Write Mutex) solves this:
//   - mu.RLock()  → "I'm reading. Other readers can proceed, but writers must wait."
//   - mu.Lock()   → "I'm writing. EVERYONE must wait (readers AND writers)."
//   - mu.RUnlock() / mu.Unlock() → "I'm done."
//
// This allows many simultaneous readers (fast) but only one writer at a time (safe).
// At MercadoLibre, products are read millions of times but written rarely,
// so RWMutex is the perfect fit — reads don't block each other.
type MemoryProductRepo struct {
	mu       sync.RWMutex       // protects the products map from concurrent access
	products map[string]model.Product // key = product ID, value = product
}

// NewMemoryProductRepo creates a new in-memory repository.
//
// This is a CONSTRUCTOR function — Go doesn't have constructors like
// Java/Python, so the convention is a function named New<Type> that
// returns an initialized instance.
//
// It accepts initial products so we can seed it with sample data.
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
//
// This line doesn't execute at runtime — it's a compile-time assertion.
// If MemoryProductRepo is missing any method from ProductRepository,
// the code won't compile. This catches mistakes immediately, not when
// a user hits the endpoint.
//
// The _ means "discard the value" — we don't need the variable,
// we just need the compiler to check the type assertion.
var _ ProductRepository = (*MemoryProductRepo)(nil)

// FindAll returns all products. Thread-safe for concurrent reads.
func (r *MemoryProductRepo) FindAll() ([]model.Product, error) {
	r.mu.RLock()         // acquire read lock (multiple readers allowed)
	defer r.mu.RUnlock() // release read lock when function returns

	// Build a slice from the map values.
	// We pre-allocate the slice with make([]T, 0, len) for efficiency —
	// this tells Go "I'll need space for this many elements" so it
	// doesn't have to resize the underlying array as we append.
	result := make([]model.Product, 0, len(r.products))
	for _, p := range r.products {
		result = append(result, p)
	}
	return result, nil
}

// FindByID returns a single product by its ID.
// Returns a NotFoundError if the product doesn't exist.
func (r *MemoryProductRepo) FindByID(id string) (model.Product, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Go maps return a second value (ok) that tells you if the key exists.
	// This is called the "comma ok" idiom — you'll see it everywhere in Go.
	product, ok := r.products[id]
	if !ok {
		return model.Product{}, &model.NotFoundError{Resource: "product", ID: id}
	}
	return product, nil
}

// Create adds a new product. Returns ConflictError if the ID already exists.
func (r *MemoryProductRepo) Create(product model.Product) error {
	r.mu.Lock()         // acquire WRITE lock (exclusive access)
	defer r.mu.Unlock() // release write lock when function returns

	if _, exists := r.products[product.ID]; exists {
		return &model.ConflictError{Resource: "product", ID: product.ID}
	}
	r.products[product.ID] = product
	return nil
}

// Update replaces an existing product. Returns NotFoundError if it doesn't exist.
func (r *MemoryProductRepo) Update(product model.Product) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.products[product.ID]; !exists {
		return &model.NotFoundError{Resource: "product", ID: product.ID}
	}
	r.products[product.ID] = product
	return nil
}

// Delete removes a product by ID. Returns NotFoundError if it doesn't exist.
func (r *MemoryProductRepo) Delete(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.products[id]; !exists {
		return &model.NotFoundError{Resource: "product", ID: id}
	}
	// delete() is a built-in Go function that removes a key from a map.
	delete(r.products, id)
	return nil
}

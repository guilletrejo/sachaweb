// Package service contains the BUSINESS LOGIC layer.
//
// WHERE DOES EACH LAYER'S RESPONSIBILITY END?
//
//   Handler:    Translates HTTP ↔ Go types. Reads request body, writes response.
//               Does NOT contain business rules.
//   Service:    Contains business rules. Validates data, generates IDs,
//               orchestrates operations. Does NOT know about HTTP or databases.
//   Repository: Reads/writes data. Does NOT contain business rules.
//
// This is the Single Responsibility Principle (the "S" in SOLID):
// each layer has ONE reason to change.
//
// Example: If you change the database from PostgreSQL to MongoDB,
// only the repository changes. If you change a business rule (e.g.,
// "product names must be unique"), only the service changes.
// If you change the API format, only the handler changes.
package service

import (
	"fmt"
	"sync/atomic"

	"github.com/guilletrejo/sachaweb/internal/model"
	"github.com/guilletrejo/sachaweb/internal/repository"
)

// idCounter is an atomic counter for generating unique product IDs.
//
// WHAT IS atomic.Int64?
// It's a thread-safe integer. Multiple goroutines can increment it
// simultaneously without a mutex. The Add method returns the new value
// atomically (as a single indivisible operation).
//
// In Phase 3, the database will generate IDs (auto-increment or UUID).
// For now, a simple counter works.
var idCounter atomic.Int64

// ProductService handles business logic for products.
// It depends on the ProductRepository INTERFACE, not a concrete type.
//
// This means:
// - In production: the service talks to PostgreSQL (via PostgresProductRepo)
// - In tests: the service talks to a mock (via a test double)
// - Right now: the service talks to an in-memory map (via MemoryProductRepo)
//
// The service doesn't know or care which one it's using.
type ProductService struct {
	repo repository.ProductRepository
}

// NewProductService creates a new service with the given repository.
// This is DEPENDENCY INJECTION — the caller decides which repository
// implementation to use, not the service itself.
//
// No framework needed. Just pass the dependency through the constructor.
// This is how most Go projects do dependency injection — simple functions.
func NewProductService(repo repository.ProductRepository) *ProductService {
	return &ProductService{repo: repo}
}

// ListProducts returns all products.
// Right now it just delegates to the repo, but this is where you'd add:
// - Filtering by category
// - Sorting
// - Pagination
// - Caching logic
func (s *ProductService) ListProducts() ([]model.Product, error) {
	return s.repo.FindAll()
}

// GetProduct returns a single product by ID.
func (s *ProductService) GetProduct(id string) (model.Product, error) {
	return s.repo.FindByID(id)
}

// CreateProduct validates the input, generates an ID, and stores the product.
//
// THIS IS WHY THE SERVICE LAYER EXISTS:
// The handler just parses JSON. The repository just stores data.
// But WHO validates the data? WHO generates the ID? WHO applies business rules?
// That's the service's job.
func (s *ProductService) CreateProduct(req model.CreateProductRequest) (model.Product, error) {
	// Step 1: Validate the input.
	if err := req.Validate(); err != nil {
		return model.Product{}, err
	}

	// Step 2: Generate a unique ID.
	id := fmt.Sprintf("prod_%d", idCounter.Add(1))

	// Step 3: Create the domain model from the request.
	product := model.Product{
		ID:          id,
		Name:        req.Name,
		Description: req.Description,
		Price:       req.Price,
		Category:    req.Category,
	}

	// Step 4: Store it.
	if err := s.repo.Create(product); err != nil {
		return model.Product{}, err
	}

	// Step 5: Return the created product (with its new ID).
	// The client needs the ID to reference this product later.
	return product, nil
}

// UpdateProduct validates input and updates an existing product.
func (s *ProductService) UpdateProduct(id string, req model.CreateProductRequest) (model.Product, error) {
	// Validate the input.
	if err := req.Validate(); err != nil {
		return model.Product{}, err
	}

	// Check that the product exists (FindByID returns NotFoundError if not).
	if _, err := s.repo.FindByID(id); err != nil {
		return model.Product{}, err
	}

	// Build the updated product, keeping the original ID.
	product := model.Product{
		ID:          id,
		Name:        req.Name,
		Description: req.Description,
		Price:       req.Price,
		Category:    req.Category,
	}

	if err := s.repo.Update(product); err != nil {
		return model.Product{}, err
	}

	return product, nil
}

// DeleteProduct removes a product by ID.
func (s *ProductService) DeleteProduct(id string) error {
	return s.repo.Delete(id)
}

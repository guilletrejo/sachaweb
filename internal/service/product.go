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
// WHAT CHANGED IN PHASE 3?
// Every method now accepts context.Context and passes it to the repository.
// The service doesn't USE the context directly — it just passes it through.
// This is called "threading context through the call chain":
//
//   HTTP request arrives
//     → handler extracts r.Context()
//       → service receives ctx, passes to repo
//         → repo passes ctx to pgxpool.Query(ctx, ...)
//           → pgx passes ctx to PostgreSQL
//
// If the HTTP request is cancelled at any point, the cancellation
// propagates all the way down to the database query.
package service

import (
	"context"
	"fmt"
	"sync/atomic"

	"github.com/guilletrejo/sachaweb/internal/model"
	"github.com/guilletrejo/sachaweb/internal/repository"
)

// idCounter is an atomic counter for generating unique product IDs.
// In Phase 3, we still generate IDs in the service (not auto-increment in DB).
// The database stores whatever ID we give it.
var idCounter atomic.Int64

// ProductService handles business logic for products.
// It depends on the ProductRepository INTERFACE, not a concrete type.
type ProductService struct {
	repo repository.ProductRepository
}

// NewProductService creates a new service with the given repository.
func NewProductService(repo repository.ProductRepository) *ProductService {
	return &ProductService{repo: repo}
}

// ListProducts returns all products.
func (s *ProductService) ListProducts(ctx context.Context) ([]model.Product, error) {
	return s.repo.FindAll(ctx)
}

// GetProduct returns a single product by ID.
func (s *ProductService) GetProduct(ctx context.Context, id string) (model.Product, error) {
	return s.repo.FindByID(ctx, id)
}

// CreateProduct validates the input, generates an ID, and stores the product.
func (s *ProductService) CreateProduct(ctx context.Context, req model.CreateProductRequest) (model.Product, error) {
	if err := req.Validate(); err != nil {
		return model.Product{}, err
	}

	id := fmt.Sprintf("prod_%d", idCounter.Add(1))

	product := model.Product{
		ID:          id,
		Name:        req.Name,
		Description: req.Description,
		Price:       req.Price,
		Category:    req.Category,
	}

	if err := s.repo.Create(ctx, product); err != nil {
		return model.Product{}, err
	}

	return product, nil
}

// UpdateProduct validates input and updates an existing product.
func (s *ProductService) UpdateProduct(ctx context.Context, id string, req model.CreateProductRequest) (model.Product, error) {
	if err := req.Validate(); err != nil {
		return model.Product{}, err
	}

	if _, err := s.repo.FindByID(ctx, id); err != nil {
		return model.Product{}, err
	}

	product := model.Product{
		ID:          id,
		Name:        req.Name,
		Description: req.Description,
		Price:       req.Price,
		Category:    req.Category,
	}

	if err := s.repo.Update(ctx, product); err != nil {
		return model.Product{}, err
	}

	return product, nil
}

// DeleteProduct removes a product by ID.
func (s *ProductService) DeleteProduct(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

// Package model contains the domain types — the core data structures
// that represent your business entities. These types are used by ALL
// layers (handlers, services, repositories). They have NO dependencies
// on HTTP, databases, or any infrastructure — they are pure data.
package model

import "strings"

// Product represents an item for sale in the store.
//
// The `json:"..."` tags control how this struct is serialized to JSON.
// When you send a Product as an HTTP response, Go's encoding/json package
// reads these tags to decide the JSON field names.
//
// Without tags: {"ID": "1", "Name": "Laptop", ...}   (ugly, Go-style caps)
// With tags:    {"id": "1", "name": "Laptop", ...}    (clean, API-style lowercase)
type Product struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Price       int64  `json:"price"` // Price in cents. $19.99 = 1999. NEVER use float64 for money.
	Category    string `json:"category"`
}

// Why int64 for price instead of float64?
//
// float64 has precision issues: 0.1 + 0.2 = 0.30000000000000004 in IEEE 754.
// If you charge a customer $0.30 but your system calculates $0.30000000000000004,
// you have a bug that's nearly impossible to debug. Every payment system in the
// world (Stripe, MercadoLibre, PayPal) stores money as integer cents.
//
// $19.99 → store as 1999 (int64)
// To display: fmt.Sprintf("$%.2f", float64(price)/100)

// CreateProductRequest is the JSON body for creating or updating a product.
//
// WHY A SEPARATE STRUCT FROM Product?
// The Product struct has an ID (assigned by the server, not the client).
// When a client creates a product, they send name/description/price/category
// but NOT the id — the server generates it. Having a separate request struct
// prevents clients from setting their own IDs (which would be a security issue).
//
// This pattern is called "input DTO" (Data Transfer Object) — a struct
// specifically designed for what the client sends, separate from the
// domain model.
type CreateProductRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Price       int64  `json:"price"`
	Category    string `json:"category"`
}

// Validate checks that the request data is valid.
// Returns a *ValidationError if something is wrong, nil if everything is OK.
//
// WHY VALIDATE ON THE SERVER?
// Never trust client data. Even if the frontend has validation,
// anyone can send a raw HTTP request with curl or Postman and bypass
// the frontend entirely. Server-side validation is mandatory.
func (r *CreateProductRequest) Validate() error {
	if strings.TrimSpace(r.Name) == "" {
		return &ValidationError{Field: "name", Message: "cannot be empty"}
	}
	if r.Price <= 0 {
		return &ValidationError{Field: "price", Message: "must be greater than 0"}
	}
	if strings.TrimSpace(r.Category) == "" {
		return &ValidationError{Field: "category", Message: "cannot be empty"}
	}
	return nil
}

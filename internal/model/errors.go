package model

import "fmt"

// WHY CUSTOM ERROR TYPES?
//
// Go's built-in error interface is just: type error interface { Error() string }
// A plain error like fmt.Errorf("product not found") tells you WHAT happened
// but not WHICH KIND of problem it is. The handler needs to know the KIND
// to pick the right HTTP status code:
//
//   NotFoundError    → 404 Not Found
//   ValidationError  → 400 Bad Request
//   ConflictError    → 409 Conflict
//
// With custom error types, the handler can use a "type switch" to check
// what kind of error it received and respond accordingly.
// This is Go's idiomatic way to handle error categories — no exceptions,
// no error codes, just types.

// NotFoundError is returned when a requested resource doesn't exist.
type NotFoundError struct {
	Resource string // e.g., "product"
	ID       string // e.g., "abc-123"
}

// Error implements the error interface.
// Any type with an Error() string method IS an error in Go.
func (e *NotFoundError) Error() string {
	return fmt.Sprintf("%s with id '%s' not found", e.Resource, e.ID)
}

// ValidationError is returned when input data is invalid.
type ValidationError struct {
	Field   string // which field is invalid, e.g., "name"
	Message string // what's wrong, e.g., "cannot be empty"
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error: %s %s", e.Field, e.Message)
}

// ConflictError is returned when an operation conflicts with existing state.
// For example, trying to create a product with an ID that already exists.
type ConflictError struct {
	Resource string
	ID       string
}

func (e *ConflictError) Error() string {
	return fmt.Sprintf("%s with id '%s' already exists", e.Resource, e.ID)
}

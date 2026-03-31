package repository

import (
	"context"
	"sync"

	"github.com/guilletrejo/sachaweb/internal/model"
)

// UserRepository defines the contract for user data access.
//
// It has only two methods because users are simple right now:
// - FindByEmail: used during login (look up user by email to verify password)
// - Create: used during registration
//
// No Update, Delete, or FindAll — we don't need them yet.
// In Go, you define the SMALLEST interface that covers your needs.
// This is the "I" in SOLID: Interface Segregation Principle —
// don't force implementations to provide methods they don't use.
type UserRepository interface {
	FindByEmail(ctx context.Context, email string) (model.User, error)
	Create(ctx context.Context, user model.User) error
}

// MemoryUserRepo stores users in an in-memory map. Used for tests.
type MemoryUserRepo struct {
	mu    sync.RWMutex
	users map[string]model.User // key = email
}

// NewMemoryUserRepo creates a new in-memory user repository.
func NewMemoryUserRepo() *MemoryUserRepo {
	return &MemoryUserRepo{
		users: make(map[string]model.User),
	}
}

var _ UserRepository = (*MemoryUserRepo)(nil)

func (r *MemoryUserRepo) FindByEmail(_ context.Context, email string) (model.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	user, ok := r.users[email]
	if !ok {
		return model.User{}, &model.NotFoundError{Resource: "user", ID: email}
	}
	return user, nil
}

func (r *MemoryUserRepo) Create(_ context.Context, user model.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.users[user.Email]; exists {
		return &model.ConflictError{Resource: "user", ID: user.Email}
	}
	r.users[user.Email] = user
	return nil
}

package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/guilletrejo/sachaweb/internal/model"
)

// PostgresUserRepo implements UserRepository using PostgreSQL.
type PostgresUserRepo struct {
	pool *pgxpool.Pool
}

// NewPostgresUserRepo creates a new PostgreSQL-backed user repository.
func NewPostgresUserRepo(pool *pgxpool.Pool) *PostgresUserRepo {
	return &PostgresUserRepo{pool: pool}
}

var _ UserRepository = (*PostgresUserRepo)(nil)

// FindByEmail looks up a user by their email address.
// Used during login to retrieve the password hash for verification.
func (r *PostgresUserRepo) FindByEmail(ctx context.Context, email string) (model.User, error) {
	var u model.User
	err := r.pool.QueryRow(ctx,
		"SELECT id, email, password_hash FROM users WHERE email = $1", email).
		Scan(&u.ID, &u.Email, &u.PasswordHash)

	if errors.Is(err, pgx.ErrNoRows) {
		return model.User{}, &model.NotFoundError{Resource: "user", ID: email}
	}
	return u, err
}

// Create inserts a new user into the database.
func (r *PostgresUserRepo) Create(ctx context.Context, user model.User) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO users (id, email, password_hash) VALUES ($1, $2, $3)`,
		user.ID, user.Email, user.PasswordHash)

	if err != nil {
		// Email has a UNIQUE constraint — duplicate email → 23505 error.
		if isPgUniqueViolation(err) {
			return &model.ConflictError{Resource: "user", ID: user.Email}
		}
		return err
	}
	return nil
}

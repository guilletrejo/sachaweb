package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/guilletrejo/sachaweb/internal/model"
)

// PostgresProductRepo implements ProductRepository using PostgreSQL.
//
// THIS IS THE PAYOFF OF THE REPOSITORY PATTERN.
// In Phase 2, we built the entire app against the ProductRepository interface.
// Now we swap the implementation from in-memory to PostgreSQL, and NOTHING
// else changes — the handler and service don't even know the switch happened.
//
// WHAT IS pgxpool.Pool?
// It's a CONNECTION POOL — a collection of reusable database connections.
//
// Why not one connection?
// Your HTTP server handles many requests concurrently (one goroutine each).
// If they all share one connection, they'd have to wait in line.
// A pool maintains multiple connections and lends them out as needed:
//
//   Request 1 → borrows connection A → runs query → returns connection A
//   Request 2 → borrows connection B → runs query → returns connection B
//   Request 3 → borrows connection A (recycled!) → runs query → returns it
//
// pgxpool handles all of this automatically. You just call pool.Query()
// and it picks an available connection for you.
//
// At MercadoLibre, a single service might maintain a pool of 20-50
// connections to handle thousands of concurrent requests.
type PostgresProductRepo struct {
	pool *pgxpool.Pool
}

// NewPostgresProductRepo creates a new PostgreSQL-backed repository.
func NewPostgresProductRepo(pool *pgxpool.Pool) *PostgresProductRepo {
	return &PostgresProductRepo{pool: pool}
}

// Compile-time check: PostgresProductRepo must implement ProductRepository.
var _ ProductRepository = (*PostgresProductRepo)(nil)

// FindAll returns all products from the database.
//
// HOW pool.Query() WORKS:
// 1. Borrows a connection from the pool
// 2. Sends the SQL query to PostgreSQL
// 3. Returns a Rows iterator for reading results one at a time
// 4. When you call rows.Close(), the connection goes back to the pool
//
// The ORDER BY id ensures consistent ordering. Without it, PostgreSQL
// returns rows in whatever order is fastest (usually insertion order,
// but not guaranteed).
func (r *PostgresProductRepo) FindAll(ctx context.Context) ([]model.Product, error) {
	rows, err := r.pool.Query(ctx,
		"SELECT id, name, description, price, category FROM products ORDER BY id")
	if err != nil {
		return nil, err
	}
	// defer rows.Close() ensures the connection is returned to the pool
	// even if we return early due to an error. Always defer this.
	defer rows.Close()

	var products []model.Product
	// rows.Next() advances to the next row. Returns false when done.
	for rows.Next() {
		var p model.Product
		// Scan reads column values into Go variables, in the same order
		// as the SELECT clause. Types must be compatible:
		//   TEXT   → string
		//   BIGINT → int64
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.Price, &p.Category); err != nil {
			return nil, err
		}
		products = append(products, p)
	}
	// rows.Err() returns any error that occurred during iteration.
	// Always check this — an error mid-iteration wouldn't be caught
	// by the individual Scan calls.
	return products, rows.Err()
}

// FindByID returns a single product by its ID.
//
// WHAT IS $1?
// It's a PARAMETERIZED QUERY placeholder. Instead of building the SQL
// string with fmt.Sprintf (DANGEROUS — SQL injection!), you pass
// values separately and PostgreSQL substitutes them safely.
//
// DANGEROUS (SQL injection):
//   fmt.Sprintf("SELECT * FROM products WHERE id = '%s'", id)
//   → If id is "'; DROP TABLE products; --", your table is gone.
//
// SAFE (parameterized):
//   pool.QueryRow(ctx, "SELECT * FROM products WHERE id = $1", id)
//   → PostgreSQL treats id as a VALUE, never as SQL code.
//   → Even "'; DROP TABLE products; --" is just a harmless string.
//
// $1, $2, $3... refer to the 1st, 2nd, 3rd argument after the SQL string.
// This is PostgreSQL syntax. MySQL uses ? instead.
func (r *PostgresProductRepo) FindByID(ctx context.Context, id string) (model.Product, error) {
	var p model.Product
	// QueryRow is like Query but returns exactly one row.
	// If no row matches, Scan returns pgx.ErrNoRows.
	err := r.pool.QueryRow(ctx,
		"SELECT id, name, description, price, category FROM products WHERE id = $1", id).
		Scan(&p.ID, &p.Name, &p.Description, &p.Price, &p.Category)

	if errors.Is(err, pgx.ErrNoRows) {
		// Translate the pgx-specific error into our domain error.
		// The handler knows how to handle NotFoundError (→ 404),
		// but it should never know about pgx.ErrNoRows.
		return model.Product{}, &model.NotFoundError{Resource: "product", ID: id}
	}
	return p, err
}

// Create inserts a new product into the database.
//
// Exec is used for queries that don't return rows (INSERT, UPDATE, DELETE).
// It returns a CommandTag that tells you how many rows were affected.
func (r *PostgresProductRepo) Create(ctx context.Context, product model.Product) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO products (id, name, description, price, category)
		 VALUES ($1, $2, $3, $4, $5)`,
		product.ID, product.Name, product.Description, product.Price, product.Category)

	if err != nil {
		// Check if PostgreSQL rejected the INSERT because the ID already exists.
		// PostgreSQL error code 23505 = unique_violation.
		if isPgUniqueViolation(err) {
			return &model.ConflictError{Resource: "product", ID: product.ID}
		}
		return err
	}
	return nil
}

// Update modifies an existing product in the database.
//
// RowsAffected() tells us if the UPDATE actually changed a row.
// If it's 0, the product didn't exist — return NotFoundError.
// This is a common PostgreSQL pattern: use RowsAffected instead of
// doing a separate SELECT to check existence first (one query vs two).
func (r *PostgresProductRepo) Update(ctx context.Context, product model.Product) error {
	result, err := r.pool.Exec(ctx,
		`UPDATE products
		 SET name = $2, description = $3, price = $4, category = $5, updated_at = NOW()
		 WHERE id = $1`,
		product.ID, product.Name, product.Description, product.Price, product.Category)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return &model.NotFoundError{Resource: "product", ID: product.ID}
	}
	return nil
}

// Delete removes a product from the database.
func (r *PostgresProductRepo) Delete(ctx context.Context, id string) error {
	result, err := r.pool.Exec(ctx,
		"DELETE FROM products WHERE id = $1", id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return &model.NotFoundError{Resource: "product", ID: id}
	}
	return nil
}

// isPgUniqueViolation checks if a PostgreSQL error is a unique constraint
// violation (error code 23505).
//
// WHAT IS errors.As?
// errors.As unwraps the error chain and checks if any error in the chain
// matches the target type. It's like a type assertion but works through
// wrapped errors. This is how you inspect specific database errors in Go.
//
// PostgreSQL error codes: https://www.postgresql.org/docs/current/errcodes-appendix.html
// 23505 = unique_violation (e.g., duplicate primary key)
func isPgUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

-- Migration 000001: Create the products table.
--
-- NAMING CONVENTION: 000001_description.up.sql / 000001_description.down.sql
-- The number is the version. "up" applies the change, "down" reverts it.
-- This lets you move forward and backward through database schema changes,
-- like git commits but for your database structure.
--
-- WHY MIGRATIONS?
-- You can't just "edit" a database table the way you edit Go code.
-- If you add a column in dev, how does the staging/production database know?
-- Migrations are versioned SQL scripts that evolve the schema step by step.
-- Every environment runs the same migrations in order, guaranteeing
-- identical database structures everywhere.
--
-- At MercadoLibre, hundreds of developers change the database schema.
-- Migrations ensure those changes are applied consistently across
-- dev laptops, CI servers, staging, and production.

CREATE TABLE IF NOT EXISTS products (
    -- TEXT PRIMARY KEY: we use application-generated string IDs (e.g., "prod_1").
    -- In production systems you'd typically use UUID or BIGSERIAL (auto-increment).
    -- TEXT is fine for learning — it matches our current Go model.
    id          TEXT PRIMARY KEY,

    -- NOT NULL means the column MUST have a value — the database rejects
    -- any INSERT that tries to leave it empty. This is a "database constraint"
    -- and it's your last line of defense against bad data.
    -- Even if the Go application has a bug that skips validation,
    -- the database catches it.
    name        TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',

    -- BIGINT matches Go's int64 (both are 64-bit integers).
    -- CHECK (price > 0) is another database constraint — PostgreSQL will
    -- reject any INSERT or UPDATE that tries to set price ≤ 0.
    -- Double validation (app layer + DB layer) is standard practice.
    price       BIGINT NOT NULL CHECK (price > 0),

    category    TEXT NOT NULL,

    -- Timestamps track when rows were created and last modified.
    -- TIMESTAMPTZ = timestamp WITH time zone. Always use this, never plain
    -- TIMESTAMP, because plain TIMESTAMP silently drops timezone info.
    -- DEFAULT NOW() means PostgreSQL fills these automatically on INSERT.
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Index on category for fast filtering.
-- Without an index, PostgreSQL does a "sequential scan" — reads every single
-- row to find matches. With an index, it jumps directly to matching rows.
-- Think of it like a book's index vs. reading every page to find a topic.
--
-- We'll use this in Phase 5 when we add "list products by category."
CREATE INDEX IF NOT EXISTS idx_products_category ON products(category);

-- Migration 000002: Create the users table for authentication.
--
-- Every authenticated system needs a users table. This stores:
-- - WHO the user is (id, email)
-- - HOW they prove their identity (password_hash)
--
-- CRITICAL: We store a HASH of the password, NEVER the plain text password.
-- Even if the database is leaked, attackers can't recover the original passwords.
-- This is standard practice everywhere — Google, MercadoLibre, every bank.

CREATE TABLE IF NOT EXISTS users (
    id            TEXT PRIMARY KEY,

    -- UNIQUE constraint means no two users can have the same email.
    -- PostgreSQL enforces this on INSERT and UPDATE — if you try to
    -- register with an existing email, the INSERT fails with error 23505
    -- (unique_violation), same as duplicate product IDs.
    email         TEXT NOT NULL UNIQUE,

    -- The bcrypt hash of the password. Bcrypt hashes are always 60 characters.
    -- Example: "$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy"
    --
    -- WHY bcrypt?
    -- Regular hashes (SHA-256, MD5) are fast — that's BAD for passwords.
    -- An attacker with a GPU can try billions of SHA-256 hashes per second.
    -- bcrypt is intentionally SLOW (configurable via "cost" parameter).
    -- At cost 10 (our default), one hash takes ~100ms. That's instant for
    -- a single login, but makes brute-force attacks take years.
    password_hash TEXT NOT NULL,

    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

# Phase 3 — Questions & Answers

## 1. What if the password in a DATABASE_URL has an `@` character?

Special characters in the password must be **percent-encoded** (URL-encoded):

| Character | Encoded as |
|-----------|-----------|
| `@` | `%40` |
| `:` | `%3A` |
| `/` | `%2F` |
| `#` | `%23` |

So if your password is `s3cur3@p@ss`, the URL would be:
```
postgres://app_user:s3cur3%40p%40ss@db.neon.tech:5432/sachaweb_prod
```

The parser reads left to right: everything between `://` and the **last unencoded `@`** is `user:password`. Everything after that `@` is `host:port/db`. The `%40` inside the password is treated as data, not a separator.

Go's `url.Parse()` and pgx both handle this automatically. In practice, cloud providers (Neon, Supabase, AWS RDS) give you the pre-encoded URL — you copy-paste it into your `DATABASE_URL` env var.

---

## 2. In docker-compose.yml, why is the `pgdata` volume empty after the colon?

It's not empty — the `volumes:` section at the bottom of docker-compose.yml **declares** the volume, and the line under `postgres:` **mounts** it:

```yaml
# Under the postgres service — this MOUNTS the volume:
volumes:
  - pgdata:/var/lib/postgresql/data
#   ^^^^^^ ^^^^^^^^^^^^^^^^^^^^^^^^^^^
#   name   container path (where Postgres stores its files)

# At the bottom of the file — this DECLARES the volume:
volumes:
  pgdata:     # ← not empty, just no extra options needed
```

The bottom `pgdata:` with nothing after the colon means "create a named volume called `pgdata` with default settings." You *could* add options:

```yaml
volumes:
  pgdata:
    driver: local
    driver_opts:
      type: none
      device: /my/custom/path
      o: bind
```

But the defaults are fine 99% of the time. The `key:` with nothing after it is standard YAML — it means the value is `null`/empty, which Docker interprets as "use defaults."

---

## 3. What is MongoDB? Could it replace PostgreSQL here?

MongoDB is a **document database** (NoSQL). Instead of tables with rows and columns, it stores **JSON-like documents** in collections:

**PostgreSQL (relational):**
```sql
-- Rigid schema: every row has the same columns
SELECT * FROM products WHERE category = 'electronics';

| id | name     | price | category    |
|----|----------|-------|-------------|
| 1  | Keyboard | 8999  | electronics |
```

**MongoDB (document):**
```javascript
// Flexible schema: each document can have different fields
db.products.find({ category: "electronics" })

{
  "_id": "abc123",
  "name": "Keyboard",
  "price": 8999,
  "category": "electronics",
  "specs": { "switches": "Cherry MX", "rgb": true }  // ← this field is optional
}
```

**Could it replace PostgreSQL here?** Yes, technically. Our product CRUD would work fine in MongoDB. But for an e-commerce system, **no** — here's why:

An order involves multiple related entities: a user, multiple products, a payment, a shipping address. In a relational DB, these are separate tables linked by foreign keys, and you can update them atomically in a **transaction** (either everything succeeds or nothing changes). MongoDB can do this now (since v4.0), but it was designed around the opposite philosophy — denormalize and embed data within documents.

MercadoLibre, Amazon, Stripe, Shopify — all use relational databases for their core transactional data. MongoDB is great for things like product catalogs, content management, logs, or any data where the schema varies a lot.

---

## 4. What alternatives are there for PostgreSQL?

| Database | Type | When to use |
|----------|------|-------------|
| **PostgreSQL** | Relational | Default choice. Most features, best ecosystem |
| **MySQL/MariaDB** | Relational | Simpler than Postgres, huge adoption (WordPress, many startups) |
| **SQLite** | Relational (embedded) | Single-file DB, no server needed. Great for mobile apps, CLI tools, small projects |
| **MongoDB** | Document (NoSQL) | Flexible schemas, rapid prototyping, content management |
| **Redis** | Key-value (in-memory) | Caching, sessions, rate limiting. Ultra fast but data lives in RAM |
| **DynamoDB** | Key-value (AWS) | Massive scale, pay-per-request. AWS-locked |
| **CockroachDB** | Relational (distributed) | PostgreSQL-compatible but scales horizontally across regions |
| **Cassandra** | Wide-column (NoSQL) | Write-heavy at massive scale (Netflix, Discord) |

For 90% of backend projects, the real choice is **PostgreSQL vs MySQL**. Both work. PostgreSQL has more features (JSON support, array types, full-text search, advanced indexing). MySQL is simpler and slightly faster for simple queries.

---

## 5. Why PostgreSQL specifically?

Three reasons:

1. **Industry standard for Go backends.** pgx is the best database driver in the Go ecosystem. MercadoLibre, Uber, and most companies running Go use PostgreSQL.

2. **Features you'll use later.** JSON columns (store product metadata), array types, full-text search (search products by name), advanced constraints — PostgreSQL has all of this built in. MySQL would require workarounds.

3. **Free cloud hosting.** Neon gives you a free PostgreSQL database. For Phase 10 (deployment), this matters.

---

## 6. Do I need to know all SQL syntax?

No — you need to know **5 operations fluently** and know where to look up the rest.

**Must know by heart:**
```sql
SELECT columns FROM table WHERE condition ORDER BY column LIMIT n;
INSERT INTO table (columns) VALUES (values);
UPDATE table SET column = value WHERE condition;
DELETE FROM table WHERE condition;
CREATE TABLE table_name (column type constraints);
```

That covers 90% of day-to-day backend work, and it's exactly what's in `product_postgres.go`.

**Should understand when you encounter them:**
- `JOIN` (combine tables — Phase 5 when orders reference products)
- `GROUP BY` / `HAVING` (aggregations — "total sales per category")
- `CREATE INDEX` (performance)
- `BEGIN` / `COMMIT` / `ROLLBACK` (transactions — Phase 5)
- Subqueries

**Don't memorize, just look up:**
- Window functions, CTEs, recursive queries, EXPLAIN ANALYZE
- You learn these when you need them

At MercadoLibre, you'd write SQL daily but 80% of it is those 5 core operations. The complex stuff you write once, review carefully, and it sits in a repository query for months.

---

## 7. No new tests — is that because the handler layer doesn't care about the repo change?

**Exactly right.** The existing tests still pass because they use `MemoryProductRepo`, which implements the same `ProductRepository` interface. The handler and service don't know the repo changed.

**New tests that would be valuable with PostgreSQL (Phase 6):**

Integration tests that hit the real database:

- Does `Create` actually persist a product that survives a server restart?
- Does the `CHECK (price > 0)` constraint reject invalid data at the DB level?
- Does `FindAll` return products in the expected order?
- Does creating a duplicate ID return a `ConflictError`?
- Does `Update` actually change `updated_at`?

The in-memory repo tests can't catch bugs like: wrong column order in `Scan`, typo in SQL, wrong `$1/$2` parameter mapping, constraint violations. Those only show up when you hit a real database.

These integration tests come in **Phase 6 (Testing)**.

---

## 8. What exactly is a migration?

A migration is a **versioned change to your database schema** — like a git commit, but for your database structure.

**The problem migrations solve:**

Your Go code is in git. You can see every change ever made, revert to any version, and every developer gets the same code. But what about your database? If you add a column to `products`, how does:
- Your coworker's local database get that column?
- The staging database get it?
- The production database get it?

You can't `git pull` a database.

**The solution: numbered SQL scripts.**

```
migrations/
  000001_create_products_table.up.sql    ← creates the table
  000001_create_products_table.down.sql  ← drops the table (undo)
  000002_add_stock_column.up.sql         ← adds a "stock" column
  000002_add_stock_column.down.sql       ← removes the "stock" column
  000003_add_users_table.up.sql          ← creates users table
  000003_add_users_table.down.sql        ← drops users table
```

Every environment runs migrations **in order**. A migration tool (like golang-migrate) keeps a table in the database that records which migrations have already been applied:

```
schema_migrations
| version | dirty |
|---------|-------|
| 1       | false |   ← migration 1 applied
| 2       | false |   ← migration 2 applied
                       ← migration 3 NOT applied yet
```

When you deploy, the migration tool sees "I'm on version 2, but version 3 exists" and runs `000003_add_users_table.up.sql`. If something goes wrong, you run `down` to revert.

Right now our approach is simpler — we run the SQL directly with `IF NOT EXISTS`, which is safe to run repeatedly. When we have multiple migrations in later phases, we'll upgrade to a proper migration tool that tracks versions.

**The key insight:** code and database schema must evolve together. Migrations are how you version-control your database.

---

## 9. So where is the database? In the .sql files?

No — the `.sql` files are just **instructions** (a recipe). The database itself is a running process inside the Docker container.

```
.sql files = blueprint        ("build a table with these columns")
PostgreSQL = the actual engine ("I store rows on disk, answer queries, enforce constraints")
pgdata volume = the disk       (where the actual rows live as binary files)
```

When you run `make docker-up`:

1. Docker starts a PostgreSQL **server** (a process listening on port 5432)
2. That server stores its data in the `pgdata` volume (a folder Docker manages on your machine)
3. When your Go app starts, it connects to that server and runs the `.sql` migration — PostgreSQL reads the instructions and creates the table
4. From then on, every `INSERT`/`SELECT`/`UPDATE`/`DELETE` goes over the network to that PostgreSQL server, which reads/writes the actual data on disk

If you delete the `.sql` files after the migration ran, the table still exists — the instructions were already executed. It's like deleting a recipe after you already baked the cake.

If you run `make docker-reset`, the volume is deleted — the actual data is gone, and next startup the migration runs again to recreate the empty table.

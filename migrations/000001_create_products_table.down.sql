-- Down migration: undo what the "up" migration did.
-- This lets you rollback if something goes wrong after deploying.
--
-- Every "up" migration MUST have a corresponding "down" migration.
-- "up" builds, "down" tears down — they must be exact inverses.
DROP TABLE IF EXISTS products;

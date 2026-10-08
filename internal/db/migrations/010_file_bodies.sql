-- Each distinct body of a published file is stored once, keyed by its
-- integrity (the SHA-384 component_files already holds): a new version of a
-- component shares the bodies of the files that didn't change, like a
-- vendored library. A rowid table, since its rows are far bigger than the
-- twentieth of a page where WITHOUT ROWID pays off.
CREATE TABLE file_bodies (
	integrity TEXT NOT NULL PRIMARY KEY, -- "sha384-…" of body
	body      BLOB NOT NULL
) STRICT;

-- Row by row: OR IGNORE skips a body already stored, where DISTINCT would
-- sort every body through a temporary index.
INSERT OR IGNORE INTO file_bodies (integrity, body) SELECT integrity, body FROM component_files;

-- component_files keeps its rows and refers to the bodies. Its body column
-- stays, always NULL, so a binary from before this migration still starts
-- after a rollback (its SyncCatalog names the column; it serves the current
-- version from memory). A later migration can drop it.
CREATE TABLE component_files_new (
	slug       TEXT    NOT NULL,
	hash       TEXT    NOT NULL,          -- the component's content hash
	path       TEXT    NOT NULL,          -- inside the folder, e.g. "vendor/lib.js"
	integrity  TEXT    NOT NULL REFERENCES file_bodies (integrity) DEFERRABLE INITIALLY DEFERRED,
	created_at INTEGER NOT NULL,
	body       BLOB,                      -- NULL: the bytes are in file_bodies
	PRIMARY KEY (slug, hash, path)
) STRICT, WITHOUT ROWID;
INSERT INTO component_files_new (slug, hash, path, integrity, created_at)
	SELECT slug, hash, path, integrity, created_at FROM component_files;
DROP TABLE component_files;
ALTER TABLE component_files_new RENAME TO component_files;
CREATE INDEX component_files_integrity ON component_files (integrity);

-- The old table's pages are free now (on production about 12 MB of 34), and
-- a file only gives them back with a VACUUM, which can't run in this
-- transaction: migrate runs it once, after this migration.
-- then: VACUUM

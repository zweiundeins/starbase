-- Every published version of every component's public files, so pinned
-- URLs (/c/<slug>@<hash>/…) keep working after later deploys.
CREATE TABLE component_files (
	slug       TEXT    NOT NULL,
	hash       TEXT    NOT NULL,          -- the component's content hash
	path       TEXT    NOT NULL,          -- inside the folder, e.g. "vendor/lib.js"
	body       BLOB    NOT NULL,
	integrity  TEXT    NOT NULL,          -- "sha384-…"
	created_at INTEGER NOT NULL,
	PRIMARY KEY (slug, hash, path)
) STRICT, WITHOUT ROWID;

-- Snapshots of the whole catalog (/c/@<hash>/autoloader.js): the frozen
-- autoloader and which component versions it loads.
CREATE TABLE catalog_snapshots (
	hash       TEXT    PRIMARY KEY,
	autoloader TEXT    NOT NULL,
	integrity  TEXT    NOT NULL,
	created_at INTEGER NOT NULL
) STRICT, WITHOUT ROWID;

CREATE TABLE catalog_snapshot_components (
	snapshot TEXT NOT NULL REFERENCES catalog_snapshots (hash),
	slug     TEXT NOT NULL,
	hash     TEXT NOT NULL,
	PRIMARY KEY (snapshot, slug)
) STRICT, WITHOUT ROWID;

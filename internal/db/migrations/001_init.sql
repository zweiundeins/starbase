CREATE TABLE users (
	id         INTEGER PRIMARY KEY,
	github_id  INTEGER NOT NULL UNIQUE,
	login      TEXT    NOT NULL,
	name       TEXT    NOT NULL DEFAULT '',
	avatar_url TEXT    NOT NULL DEFAULT '',
	created_at INTEGER NOT NULL
) STRICT;

-- A session row exists only once a browser has signed in.
CREATE TABLE sessions (
	sid        TEXT    PRIMARY KEY,
	user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	created_at INTEGER NOT NULL
) STRICT, WITHOUT ROWID;

-- Per-tab UI state (filters, sort, preview theme) owned by the server.
CREATE TABLE tab_state (
	sid        TEXT    NOT NULL,
	tab_id     TEXT    NOT NULL,
	data       TEXT    NOT NULL,
	updated_at INTEGER NOT NULL,
	PRIMARY KEY (sid, tab_id)
) STRICT, WITHOUT ROWID;

CREATE TABLE components (
	slug         TEXT    PRIMARY KEY,
	tag          TEXT    NOT NULL UNIQUE,
	name         TEXT    NOT NULL,
	category     TEXT    NOT NULL,
	summary      TEXT    NOT NULL,
	author       TEXT    NOT NULL,
	tags         TEXT    NOT NULL DEFAULT '[]',
	since        TEXT    NOT NULL,
	content_hash TEXT    NOT NULL,
	stars        INTEGER NOT NULL DEFAULT 0,
	active       INTEGER NOT NULL DEFAULT 1,
	updated_at   INTEGER NOT NULL
) STRICT, WITHOUT ROWID;

CREATE INDEX components_category ON components(category) WHERE active;

CREATE VIRTUAL TABLE components_fts USING fts5(
	slug UNINDEXED, name, summary, tags,
	tokenize = 'unicode61 remove_diacritics 2',
	prefix = '1 2 3'
);

CREATE TABLE stars (
	slug       TEXT    NOT NULL REFERENCES components(slug) ON DELETE CASCADE,
	user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	created_at INTEGER NOT NULL,
	PRIMARY KEY (slug, user_id)
) STRICT, WITHOUT ROWID;

CREATE INDEX stars_user ON stars(user_id);

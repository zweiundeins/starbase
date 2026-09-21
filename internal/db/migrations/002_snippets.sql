-- Saved playground code. Immutable: every save is a new row and link.
CREATE TABLE snippets (
	id         TEXT    PRIMARY KEY,
	files      TEXT    NOT NULL,          -- JSON {"component.js": "…", …}
	component  TEXT,                      -- catalog slug it started from, if any
	user_id    INTEGER REFERENCES users(id) ON DELETE SET NULL,
	created_at INTEGER NOT NULL
) STRICT, WITHOUT ROWID;

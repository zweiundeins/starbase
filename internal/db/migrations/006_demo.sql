-- The example dataset behind /demo/data/… (internal/demo): read-only
-- reference data, seeded at startup and replaced when the dataset changes.
CREATE TABLE demo_bodies (
	id     TEXT    PRIMARY KEY,
	parent TEXT    NOT NULL DEFAULT '',
	kind   TEXT    NOT NULL,
	name   TEXT    NOT NULL,
	detail TEXT    NOT NULL DEFAULT '',
	ord    INTEGER NOT NULL,
	folded TEXT    NOT NULL           -- lower case, no accents: name and detail, for search
) STRICT, WITHOUT ROWID;
CREATE INDEX demo_bodies_parent ON demo_bodies (parent, ord);

CREATE TABLE demo_meta (
	id      INTEGER PRIMARY KEY CHECK (id = 1),
	version TEXT    NOT NULL
) STRICT;

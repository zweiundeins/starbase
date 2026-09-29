-- The first 100,000 stars of the demo catalog (internal/demo), behind
-- /demo/data/rows: a table that loads only the rows in view, sorted by SQLite.
-- Seeded by SeedStars, which keeps its version next to the bodies' one. An index per
-- sortable column: it holds the rowid too, so ties keep the id order both ways.
CREATE TABLE demo_stars (
	id            INTEGER PRIMARY KEY,
	name          TEXT    NOT NULL,
	class         TEXT    NOT NULL,
	temp          INTEGER NOT NULL,
	constellation TEXT    NOT NULL,
	distance      REAL    NOT NULL,
	magnitude     REAL    NOT NULL,
	planets       INTEGER NOT NULL
) STRICT;
CREATE INDEX demo_stars_name ON demo_stars (name);
CREATE INDEX demo_stars_temp ON demo_stars (temp);
CREATE INDEX demo_stars_constellation ON demo_stars (constellation);
CREATE INDEX demo_stars_distance ON demo_stars (distance);
CREATE INDEX demo_stars_magnitude ON demo_stars (magnitude);

ALTER TABLE demo_meta ADD COLUMN stars TEXT NOT NULL DEFAULT '';

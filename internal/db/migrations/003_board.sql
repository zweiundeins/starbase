-- The shared pixel board on the Showcase page.
CREATE TABLE boards (
	board   TEXT    PRIMARY KEY,
	version INTEGER NOT NULL DEFAULT 0,  -- bumped by every paint batch
	pixels  INTEGER NOT NULL DEFAULT 0   -- total pixels ever painted
) STRICT, WITHOUT ROWID;

CREATE TABLE board_cells (
	board      TEXT    NOT NULL,
	idx        INTEGER NOT NULL,
	color      INTEGER NOT NULL,
	painted_at INTEGER NOT NULL,
	PRIMARY KEY (board, idx)
) STRICT, WITHOUT ROWID;

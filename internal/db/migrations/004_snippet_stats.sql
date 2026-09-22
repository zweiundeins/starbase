-- Running total of snippet bytes, so saves can enforce a storage cap
-- without scanning the table.
CREATE TABLE snippet_stats (
	id    INTEGER PRIMARY KEY CHECK (id = 1),
	bytes INTEGER NOT NULL
) STRICT;
INSERT INTO snippet_stats (id, bytes) SELECT 1, COALESCE(SUM(length(files)), 0) FROM snippets;

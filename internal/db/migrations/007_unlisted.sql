-- Components that belong to the site rather than the gallery (sb-code-playground
-- powers /playground). They stay served, versioned and documented; they are just
-- not browsed, searched or counted as community components.
ALTER TABLE components ADD COLUMN listed INTEGER NOT NULL DEFAULT 1;

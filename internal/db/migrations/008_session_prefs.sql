-- Preferences that outlive a page: they carry across navigations and browser
-- tabs of one session (the sid cookie). JSON, like tab_state
-- (model.SessionPrefs); tab_state lives for one loaded page.
CREATE TABLE session_prefs (
	sid        TEXT    PRIMARY KEY,
	data       TEXT    NOT NULL,
	updated_at INTEGER NOT NULL
) STRICT, WITHOUT ROWID;

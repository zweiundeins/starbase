-- The hourly sweep deletes the state of tabs untouched for a day: by this
-- index, rather than a scan of the table inside the writer's transaction.
CREATE INDEX tab_state_updated ON tab_state (updated_at);

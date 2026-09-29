-- online_counts.is_canonical flagged a canonical probe's samples for a
-- preference rule that never engaged; nothing has read it since.
ALTER TABLE online_counts DROP COLUMN IF EXISTS is_canonical;

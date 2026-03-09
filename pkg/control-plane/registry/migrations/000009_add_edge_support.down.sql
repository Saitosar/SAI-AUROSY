DROP INDEX IF EXISTS idx_edge_commands_edge_pending;
DROP TABLE IF EXISTS edge_commands;
DROP TABLE IF EXISTS edges;

-- SQLite doesn't support DROP COLUMN easily; leave edge_id for simplicity
-- ALTER TABLE robots DROP COLUMN edge_id;

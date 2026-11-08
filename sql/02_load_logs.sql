-- Place logs.parquet at ./data/logs.parquet (mounted as user_files).
-- This loads a complete snapshot; truncating first makes reruns idempotent.
TRUNCATE TABLE search.logs;

INSERT INTO search.logs (timestamp, user_id, search_query)
SELECT timestamp, user_id, search_query
FROM file('logs.parquet', Parquet);

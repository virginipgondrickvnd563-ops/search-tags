-- Place logs.parquet at ./data/logs.parquet (mounted as user_files).
-- Run after 01_schema.sql:
INSERT INTO search.logs (timestamp, user_id, search_query)
SELECT timestamp, user_id, search_query
FROM file('logs.parquet', Parquet);

CREATE TABLE IF NOT EXISTS search.logs
(
    timestamp DateTime64(3, 'UTC'),
    user_id UInt64,
    search_query String
)
ENGINE = MergeTree
PARTITION BY toYYYYMM(timestamp)
ORDER BY (user_id, timestamp);

CREATE TABLE IF NOT EXISTS search.tags
(
    query String,
    tag String,
    users UInt64,
    rank UInt8
)
ENGINE = MergeTree
ORDER BY (query, rank);

-- Staging table for atomic tag refreshes. It has the same schema as search.tags.
CREATE TABLE IF NOT EXISTS search.tags_next
(
    query String,
    tag String,
    users UInt64,
    rank UInt8
)
ENGINE = MergeTree
ORDER BY (query, rank);

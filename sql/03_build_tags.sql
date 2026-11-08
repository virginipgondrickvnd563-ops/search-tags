TRUNCATE TABLE search.tags;

INSERT INTO search.tags (query, tag, users, rank)
WITH normalized AS
(
    SELECT
        timestamp,
        user_id,
        arrayStringConcat(arrayFilter(x -> x != '', splitByChar(' ', lowerUTF8(trimBoth(replaceRegexpAll(search_query, '\\s+', ' '))))), ' ') AS q
    FROM search.logs
), pairs AS
(
    SELECT
        user_id,
        q AS query,
        leadInFrame(q, 1, '') OVER (PARTITION BY user_id ORDER BY timestamp ROWS BETWEEN CURRENT ROW AND 1 FOLLOWING) AS tag,
        timestamp AS ts,
        leadInFrame(timestamp, 1, timestamp) OVER (PARTITION BY user_id ORDER BY timestamp ROWS BETWEEN CURRENT ROW AND 1 FOLLOWING) AS next_ts
    FROM normalized
)
SELECT query, tag, users, toUInt8(rn)
FROM
(
    SELECT
        query,
        tag,
        users,
        row_number() OVER (PARTITION BY query ORDER BY users DESC, tag ASC) AS rn
    FROM
    (
        SELECT query, tag, uniqExact(user_id) AS users
        FROM pairs
        WHERE query != '' AND tag != '' AND tag != query AND next_ts >= ts
          AND dateDiff('second', ts, next_ts) <= 1800
        GROUP BY query, tag
        HAVING users >= 5
    )
)
WHERE rn <= 10;

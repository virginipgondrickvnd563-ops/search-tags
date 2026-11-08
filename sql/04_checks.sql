-- Each query should return zero (or no rows where indicated).
SELECT count() AS self_pairs FROM search.tags WHERE query = tag;
SELECT count() AS below_threshold FROM search.tags WHERE users < 5;
SELECT count() AS over_ten FROM (SELECT query FROM search.tags GROUP BY query HAVING count() > 10);

-- Compare normalized API lookups: both parameter values should return identical rows.
SELECT tag, users, rank FROM search.tags
WHERE query = arrayStringConcat(arrayFilter(x -> x != '', splitByChar(' ', lowerUTF8(trimBoth(replaceRegexpAll('Платье ', '\\s+', ' '))))), ' ')
ORDER BY rank;
SELECT tag, users, rank FROM search.tags
WHERE query = arrayStringConcat(arrayFilter(x -> x != '', splitByChar(' ', lowerUTF8(trimBoth(replaceRegexpAll('платье', '\\s+', ' '))))), ' ')
ORDER BY rank;

# Поисковые теги

Проект строит для каждого нормализованного запроса до 10 следующих запросов, которые встречались у не менее пяти разных пользователей в соседних поисках с интервалом до 30 минут.

## Запуск

1. Положите исходный `logs.parquet` в `data/logs.parquet`.
2. Запустите `docker compose up -d clickhouse`.
3. Инициализируйте таблицы и загрузите данные (команды выполняются из корня проекта):

```sh
docker compose exec clickhouse clickhouse-client --user app --password app --multiquery --queries-file /docker-entrypoint-initdb.d/01_schema.sql
docker compose exec clickhouse clickhouse-client --user app --password app --multiquery --queries-file /docker-entrypoint-initdb.d/02_load_logs.sql
docker compose exec clickhouse clickhouse-client --user app --password app --multiquery --queries-file /docker-entrypoint-initdb.d/03_build_tags.sql
```

Для PowerShell выполните те же команды с `docker compose exec`; путь `data/logs.parquet` доступен серверу как `logs.parquet` в `user_files`.

Затем выполните `go test ./...` и `docker compose up --build web`. Страница доступна на `http://localhost:8080`, API — `/api/tags?q=платье`. Для API можно задать `CLICKHOUSE_ADDR`, `CLICKHOUSE_DATABASE`, `CLICKHOUSE_USER`, `CLICKHOUSE_PASSWORD`, `HTTP_ADDR`.

## Схема и допущения

Логи используют `MergeTree`, месячную партицию по времени и `ORDER BY (user_id, timestamp)`: основные оконные вычисления группируют события пользователя и сортируют по времени, а партиции упрощают обслуживание истории. `DateTime64(3, 'UTC')` сохраняет миллисекунды, `UInt64` подходит для числового идентификатора, исходный запрос хранится как `String`. Результат упорядочен по `(query, rank)` для быстрого поиска и выдачи по рангу.

Перед расчётом пустые запросы исключаются, регистр приводится к нижнему, пробелы по краям удаляются, последовательности whitespace сворачиваются. Пары при одинаковом timestamp упорядочены нестабильно: если в источнике возможны одинаковые метки у одного пользователя, для строгой воспроизводимости нужен уникальный event id. Число пользователей считается через `uniqExact`. API возвращает пустой JSON-массив при отсутствии тегов.

## Проверки

Проверки SQL находятся в `sql/04_checks.sql`: self-pairs, порог, максимум 10 тегов и сравнение нормализованных запросов `Платье ` / `платье`. Бэкенд-тесты: `go test ./...` — нормализация и HTTP 400 для пустого `q`.

## Ежедневное обновление

Для инкрементального пересчёта сохранял бы соседние события каждого пользователя на границе обработанного диапазона, добавлял новые события, пересчитывал переходы затронутых пользователей и обновлял агрегаты пар по пользователям. Далее пересчитывал бы топ-10 только для запросов, чьи агрегаты изменились, с идемпотентным окном и контрольной датой загрузки.

package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"html/template"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	_ "github.com/ClickHouse/clickhouse-go/v2"
)

const defaultDBTimeout = 3 * time.Second

var whitespaceRE = regexp.MustCompile(`[\t\n\v\f\r \x{00a0}]+`)

type Tag struct {
	Tag   string `json:"tag"`
	Users uint64 `json:"users"`
}

type tagStore interface {
	GetTags(context.Context, string) ([]Tag, error)
}

type clickHouseStore struct {
	db      *sql.DB
	timeout time.Duration
}

func (s clickHouseStore) GetTags(ctx context.Context, query string) ([]Tag, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	rows, err := s.db.QueryContext(ctx,
		"SELECT tag, users FROM tags WHERE query = ? ORDER BY rank LIMIT 10",
		query,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tags := make([]Tag, 0)
	for rows.Next() {
		var tag Tag
		if err := rows.Scan(&tag.Tag, &tag.Users); err != nil {
			return nil, err
		}
		tags = append(tags, tag)
	}
	return tags, rows.Err()
}

type App struct {
	store tagStore
	page  *template.Template
}

func normalize(value string) string {
	value = strings.ToLower(value)
	value = whitespaceRE.ReplaceAllString(value, " ")
	return strings.Trim(value, " ")
}

func isTimeout(err error) bool {
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
}

func writeAPIError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(struct {
		Error string `json:"error"`
	}{Error: message})
}

func (a *App) api(w http.ResponseWriter, r *http.Request) {
	query := normalize(r.URL.Query().Get("q"))
	if query == "" {
		writeAPIError(w, http.StatusBadRequest, "q is required")
		return
	}

	tags, err := a.store.GetTags(r.Context(), query)
	if err != nil {
		log.Printf("tags lookup failed: %v", err)
		if isTimeout(err) {
			writeAPIError(w, http.StatusGatewayTimeout, "database request timed out")
			return
		}
		writeAPIError(w, http.StatusInternalServerError, "database unavailable")
		return
	}
	if tags == nil {
		tags = make([]Tag, 0)
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(tags)
}

func (a *App) home(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	query := r.URL.Query().Get("q")
	normalized := normalize(query)
	tags := make([]Tag, 0)
	if normalized != "" {
		var err error
		tags, err = a.store.GetTags(r.Context(), normalized)
		if err != nil {
			log.Printf("page tags lookup failed: %v", err)
			http.Error(w, "database unavailable", http.StatusInternalServerError)
			return
		}
	}

	data := struct {
		Query    string
		Tags     []Tag
		Searched bool
	}{Query: query, Tags: tags, Searched: normalized != ""}
	if err := a.page.Execute(w, data); err != nil {
		log.Printf("render page failed: %v", err)
	}
}

func timeoutFromEnv() time.Duration {
	value := os.Getenv("DB_TIMEOUT")
	if value == "" {
		return defaultDBTimeout
	}

	timeout, err := time.ParseDuration(value)
	if err != nil || timeout <= 0 {
		log.Printf("invalid DB_TIMEOUT %q; using %s", value, defaultDBTimeout)
		return defaultDBTimeout
	}
	return timeout
}

func clickHouseDSN() string {
	addr := env("CLICKHOUSE_ADDR", "localhost:9000")
	database := env("CLICKHOUSE_DATABASE", "search")
	user := env("CLICKHOUSE_USER", "app")
	password := os.Getenv("CLICKHOUSE_PASSWORD")

	connectionURL := url.URL{
		Scheme: "clickhouse",
		Host:   addr,
		Path:   "/" + database,
		User:   url.UserPassword(user, password),
	}
	return connectionURL.String()
}

func main() {
	db, err := sql.Open("clickhouse", clickHouseDSN())
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	page := template.Must(template.ParseFiles("templates/index.html"))
	app := &App{
		store: clickHouseStore{db: db, timeout: timeoutFromEnv()},
		page:  page,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/tags", app.api)
	mux.HandleFunc("GET /", app.home)

	address := env("HTTP_ADDR", ":8080")
	log.Printf("listening on %s", address)
	log.Fatal(http.ListenAndServe(address, mux))
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

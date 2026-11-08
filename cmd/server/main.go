package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	_ "github.com/ClickHouse/clickhouse-go/v2"
)

type Tag struct {
	Tag   string `json:"tag"`
	Users uint64 `json:"users"`
}

type App struct { db *sql.DB; page *template.Template }

func normalize(s string) string { return strings.Join(strings.Fields(strings.ToLower(strings.TrimSpace(s))), " ") }

func (a *App) tags(ctx context.Context, q string) ([]Tag, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second); defer cancel()
	rows, err := a.db.QueryContext(ctx, "SELECT tag, users FROM tags WHERE query = ? ORDER BY rank LIMIT 10", q)
	if err != nil { return nil, err }; defer rows.Close()
	result := make([]Tag, 0)
	for rows.Next() { var t Tag; if err := rows.Scan(&t.Tag, &t.Users); err != nil { return nil, err }; result = append(result, t) }
	return result, rows.Err()
}

func (a *App) api(w http.ResponseWriter, r *http.Request) {
	q := normalize(r.URL.Query().Get("q")); if q == "" { http.Error(w, "q is required", http.StatusBadRequest); return }
	tags, err := a.tags(r.Context(), q); if err != nil { log.Printf("tags lookup: %v", err); http.Error(w, "database unavailable", http.StatusInternalServerError); return }
	w.Header().Set("Content-Type", "application/json; charset=utf-8"); _ = json.NewEncoder(w).Encode(tags)
}

func (a *App) home(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" { http.NotFound(w, r); return }
	q := r.URL.Query().Get("q"); var tags []Tag
	if strings.TrimSpace(q) != "" { var err error; tags, err = a.tags(r.Context(), normalize(q)); if err != nil { log.Printf("page lookup: %v", err); http.Error(w, "database unavailable", http.StatusInternalServerError); return } }
	if err := a.page.Execute(w, struct { Query string; Tags []Tag; Searched bool }{q, tags, strings.TrimSpace(q) != ""}); err != nil { log.Printf("render page: %v", err) }
}

func main() {
	addr := os.Getenv("CLICKHOUSE_ADDR"); if addr == "" { addr = "localhost:9000" }
	db, err := sql.Open("clickhouse", "clickhouse://"+addr+"/"+env("CLICKHOUSE_DATABASE", "search")+"?username="+env("CLICKHOUSE_USER", "app")+"&password="+env("CLICKHOUSE_PASSWORD", "app")+"&read_timeout=5s&write_timeout=5s")
	if err != nil { log.Fatal(err) }; defer db.Close()
	page := template.Must(template.ParseFiles("templates/index.html")); app := &App{db: db, page: page}
	mux := http.NewServeMux(); mux.HandleFunc("GET /api/tags", app.api); mux.HandleFunc("GET /", app.home)
	httpAddr := env("HTTP_ADDR", ":8080"); log.Printf("listening on %s", httpAddr); log.Fatal(http.ListenAndServe(httpAddr, mux))
}
func env(k, fallback string) string { if v := os.Getenv(k); v != "" { return v }; return fallback }

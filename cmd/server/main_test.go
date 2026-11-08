package main

import (
	"net/http/httptest"
	"testing"
)

func TestNormalize(t *testing.T) {
	if got := normalize("  ПЛАТЬЕ   ЛЕТНЕЕ "); got != "платье летнее" { t.Fatalf("normalize() = %q", got) }
}

func TestAPIBadEmptyQuery(t *testing.T) {
	a := &App{}
	r := httptest.NewRequest("GET", "/api/tags?q=%20%20", nil)
	w := httptest.NewRecorder()
	a.api(w, r)
	if w.Code != 400 { t.Fatalf("status = %d, want 400", w.Code) }
}

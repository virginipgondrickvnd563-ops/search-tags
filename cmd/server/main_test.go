package main

import (
	"context"
	"errors"
	"net/http/httptest"
	"testing"
)

type fakeTagStore struct {
	tags  []Tag
	err   error
	query string
}

func (f *fakeTagStore) GetTags(_ context.Context, query string) ([]Tag, error) {
	f.query = query
	return f.tags, f.err
}

func TestNormalize(t *testing.T) {
	got := normalize("  ПЛАТЬЕ\u00a0\tЛЕТНЕЕ  ")
	if want := "платье летнее"; got != want {
		t.Fatalf("normalize() = %q, want %q", got, want)
	}
}

func TestAPIBadEmptyQueryReturnsJSON400(t *testing.T) {
	app := &App{}
	request := httptest.NewRequest("GET", "/api/tags?q=%20%20", nil)
	response := httptest.NewRecorder()

	app.api(response, request)

	if response.Code != 400 {
		t.Fatalf("status = %d, want 400", response.Code)
	}
	if got := response.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
		t.Fatalf("Content-Type = %q, want JSON", got)
	}
	if got := response.Body.String(); got != "{\"error\":\"q is required\"}\n" {
		t.Fatalf("body = %q", got)
	}
}

func TestAPIUsesNormalizedQueryAndReturnsTags(t *testing.T) {
	store := &fakeTagStore{tags: []Tag{{Tag: "красное платье", Users: 7}}}
	app := &App{store: store}
	request := httptest.NewRequest("GET", "/api/tags?q=%20ПЛАТЬЕ%20%20", nil)
	response := httptest.NewRecorder()

	app.api(response, request)

	if response.Code != 200 {
		t.Fatalf("status = %d, want 200", response.Code)
	}
	if store.query != "платье" {
		t.Fatalf("database query = %q, want %q", store.query, "платье")
	}
	if got := response.Body.String(); got != "[{\"tag\":\"красное платье\",\"users\":7}]\n" {
		t.Fatalf("body = %q", got)
	}
}

func TestAPIWithoutTagsReturnsEmptyJSONList(t *testing.T) {
	app := &App{store: &fakeTagStore{}}
	request := httptest.NewRequest("GET", "/api/tags?q=платье", nil)
	response := httptest.NewRecorder()

	app.api(response, request)

	if response.Code != 200 {
		t.Fatalf("status = %d, want 200", response.Code)
	}
	if got := response.Body.String(); got != "[]\n" {
		t.Fatalf("body = %q, want an empty JSON list", got)
	}
}

func TestAPITimeoutReturns504JSON(t *testing.T) {
	app := &App{store: &fakeTagStore{err: context.DeadlineExceeded}}
	request := httptest.NewRequest("GET", "/api/tags?q=платье", nil)
	response := httptest.NewRecorder()

	app.api(response, request)

	if response.Code != 504 {
		t.Fatalf("status = %d, want 504", response.Code)
	}
	if got := response.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
		t.Fatalf("Content-Type = %q, want JSON", got)
	}
}

func TestAPIDatabaseErrorReturns500JSON(t *testing.T) {
	app := &App{store: &fakeTagStore{err: errors.New("connection failed")}}
	request := httptest.NewRequest("GET", "/api/tags?q=платье", nil)
	response := httptest.NewRecorder()

	app.api(response, request)

	if response.Code != 500 {
		t.Fatalf("status = %d, want 500", response.Code)
	}
	if got := response.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
		t.Fatalf("Content-Type = %q, want JSON", got)
	}
}

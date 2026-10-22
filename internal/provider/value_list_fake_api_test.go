package provider

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

// fakeValueListAPI keeps value lists in memory and answers the way the Management API does: create
// requires itemType and defaults the items to [], an update replaces the items and rejects null for
// them, and the item type never changes after create.
type fakeValueListAPI struct {
	mu    sync.Mutex
	lists map[string]map[string]any
}

func newFakeValueListAPI(t *testing.T) *fakeValueListAPI {
	t.Helper()

	fake := &fakeValueListAPI{lists: map[string]map[string]any{}}

	server := httptest.NewServer(http.HandlerFunc(fake.serve))
	t.Cleanup(server.Close)
	t.Setenv("AUTHSIGNAL_HOST", server.URL)

	return fake
}

func (f *fakeValueListAPI) serve(w http.ResponseWriter, r *http.Request) {
	var body map[string]any
	_ = json.NewDecoder(r.Body).Decode(&body)

	f.mu.Lock()
	defer f.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")

	alias := strings.TrimPrefix(r.URL.Path, "/value-lists/")

	switch {
	case r.Method == http.MethodPost && r.URL.Path == "/value-lists":
		f.create(w, body)
	case r.Method == http.MethodGet:
		f.respond(w, alias)
	case r.Method == http.MethodPatch:
		f.update(w, alias, body)
	case r.Method == http.MethodDelete:
		delete(f.lists, alias)
		_, _ = w.Write([]byte(`{"success":true}`))
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func (f *fakeValueListAPI) create(w http.ResponseWriter, body map[string]any) {
	if _, ok := body["itemType"].(string); !ok {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"itemType is required"}`))
		return
	}

	items, ok := itemsOrEmpty(body)
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"valueListItems must be an array"}`))
		return
	}

	alias := strings.ReplaceAll(strings.ToLower(body["name"].(string)), " ", "-")
	f.lists[alias] = map[string]any{
		"name":           body["name"],
		"alias":          alias,
		"itemType":       body["itemType"],
		"isActive":       body["isActive"],
		"valueListItems": items,
	}

	f.respond(w, alias)
}

func (f *fakeValueListAPI) update(w http.ResponseWriter, alias string, body map[string]any) {
	list, exists := f.lists[alias]
	if !exists {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	items, ok := itemsOrEmpty(body)
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"valueListItems must be an array"}`))
		return
	}

	list["valueListItems"] = items
	if isActive, set := body["isActive"]; set {
		list["isActive"] = isActive
	}

	f.respond(w, alias)
}

func (f *fakeValueListAPI) respond(w http.ResponseWriter, alias string) {
	list, exists := f.lists[alias]
	if !exists {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":"not_found"}`))
		return
	}

	encoded, _ := json.Marshal(list)
	_, _ = w.Write(encoded)
}

// itemsOrEmpty mirrors the API's request schema: valueListItems is an optional array, and null is
// not an array.
func itemsOrEmpty(body map[string]any) ([]any, bool) {
	value, present := body["valueListItems"]
	if !present {
		return []any{}, true
	}

	items, isArray := value.([]any)

	return items, isArray
}

func (f *fakeValueListAPI) expectList(alias string, itemType string, itemCount int) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	list, exists := f.lists[alias]
	if !exists {
		return fmt.Errorf("value list %q doesn't exist", alias)
	}

	items, _ := list["valueListItems"].([]any)
	if list["itemType"] != itemType || len(items) != itemCount {
		return fmt.Errorf("value list %q is a %v list of %d items, want a %s list of %d", alias, list["itemType"], len(items), itemType, itemCount)
	}

	return nil
}

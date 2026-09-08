package provider

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/authsignal/authsignal-management-go/v6"
)

const (
	actionGetRoute    = "GET /action-configurations/sign-in"
	actionCreateRoute = "POST /action-configurations"
	actionPatchRoute  = "PATCH /action-configurations/sign-in"
	flowPublishRoute  = "PUT /action-configurations/sign-in/flow"
	rulesListRoute    = "GET /action-configurations/sign-in/rules"
	actionDeleteRoute = "DELETE /action-configurations/sign-in"

	testFlow  = `{"actionNodes":[{"nodeId":"c","nodeType":"COMPLETE"}],"rules":[]}`
	testNodes = `[{"nodeId":"c","nodeType":"COMPLETE"}]`
)

// The resources are exercised against a stub Management API so that a test can assert
// which endpoints were called, in what order, and with what body. A route is keyed by
// "METHOD /path" and may hold several responses, which are returned in order; the last
// one repeats. An unrouted request is recorded and answered 404, so a test proves an
// endpoint was not called by asserting on the recorded calls rather than on a response.
type stubResponse struct {
	status int
	body   string
}

type recordedCall struct {
	method string
	path   string
	body   string
}

func (c recordedCall) route() string {
	return c.method + " " + c.path
}

type stubAPI struct {
	t         *testing.T
	mu        sync.Mutex
	calls     []recordedCall
	routes    map[string][]stubResponse
	responded map[string]int
}

func newStubAPI(t *testing.T, routes map[string][]stubResponse) (*authsignal.Client, *stubAPI) {
	t.Helper()

	stub := &stubAPI{t: t, routes: routes, responded: map[string]int{}}

	server := httptest.NewServer(http.HandlerFunc(stub.serve))
	t.Cleanup(server.Close)

	client := authsignal.NewClient(server.URL, "tenant", "secret")

	return &client, stub
}

func (s *stubAPI) serve(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)

	s.mu.Lock()
	call := recordedCall{method: r.Method, path: r.URL.Path, body: string(body)}
	s.calls = append(s.calls, call)

	responses, routed := s.routes[call.route()]
	index := s.responded[call.route()]
	if routed {
		s.responded[call.route()] = index + 1
	}
	s.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")

	if !routed {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":"not_found"}`))
		return
	}

	if index >= len(responses) {
		index = len(responses) - 1
	}

	w.WriteHeader(responses[index].status)
	_, _ = w.Write([]byte(responses[index].body))
}

func (s *stubAPI) calledRoutes() []string {
	s.mu.Lock()
	defer s.mu.Unlock()

	routes := make([]string, 0, len(s.calls))
	for _, call := range s.calls {
		routes = append(routes, call.route())
	}

	return routes
}

func (s *stubAPI) assertRoutes(t *testing.T, expected ...string) {
	t.Helper()

	actual := s.calledRoutes()
	if strings.Join(actual, "\n") != strings.Join(expected, "\n") {
		t.Fatalf("expected the calls\n  %s\ngot\n  %s", strings.Join(expected, "\n  "), strings.Join(actual, "\n  "))
	}
}

func (s *stubAPI) assertNotCalled(t *testing.T, route string) {
	t.Helper()

	for _, called := range s.calledRoutes() {
		if called == route {
			t.Fatalf("%s must not be called, the calls were %v", route, s.calledRoutes())
		}
	}
}

// bodyOf returns the body of the first request to a route.
func (s *stubAPI) bodyOf(t *testing.T, route string) string {
	t.Helper()

	s.mu.Lock()
	for _, call := range s.calls {
		if call.route() == route {
			s.mu.Unlock()
			return call.body
		}
	}
	s.mu.Unlock()

	t.Fatalf("%s was never called, the calls were %v", route, s.calledRoutes())

	return ""
}

func okResponse(body string) []stubResponse {
	return []stubResponse{{status: http.StatusOK, body: body}}
}

func notFoundResponse() stubResponse {
	return stubResponse{status: http.StatusNotFound, body: `{"error":"not_found"}`}
}

// actionConfigurationJson builds the response the API gives for an action configuration.
// nodes and flowVersion are rendered as-is, so a caller passes "null" for an unpublished
// flow and a JSON array for a published one.
func actionConfigurationJson(actionType string, nodes string, flowVersion string) string {
	return `{"actionCode":"sign-in","actionType":"` + actionType + `","tenantId":"tenant",` +
		`"lastActionCreatedAt":"2026-09-07T00:00:00.000Z","defaultUserActionResult":"ALLOW",` +
		`"actionNodes":` + nodes + `,"flowVersion":` + flowVersion + `}`
}

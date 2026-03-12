package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRouteRegistry_AddRoute(t *testing.T) {
	registry := NewRouteRegistry()

	route := CreateRoute("/test", GET, func(ctx context.Context, w http.ResponseWriter, r *http.Request) {})
	registry.AddRoute(route)

	if !registry.ExistRoute(route) {
		t.Error("expected route to exist")
	}
}

func TestRouteRegistry_RemoveRoute(t *testing.T) {
	registry := NewRouteRegistry()

	route := CreateRoute("/test", GET, func(ctx context.Context, w http.ResponseWriter, r *http.Request) {})
	registry.AddRoute(route)
	registry.RemoveRoute(route)

	if registry.ExistRoute(route) {
		t.Error("expected route to be removed")
	}
}

func TestRouteRegistry_AddHandler(t *testing.T) {
	registry := NewRouteRegistry()

	handler := func(ctx context.Context, w http.ResponseWriter, r *http.Request) {}
	registry.AddHandler("/api/test", GET, handler)

	if !registry.ExistHandler("/api/test", GET) {
		t.Error("expected handler to exist")
	}
}

func TestRouteRegistry_ApiVersion(t *testing.T) {
	registry := NewRouteRegistry()

	registry.SetApiVersion("v1")
	if registry.GetApiVersion() != "v1" {
		t.Errorf("expected v1, got %s", registry.GetApiVersion())
	}

	registry.SetApiVersion("v2")
	if registry.GetApiVersion() != "v2" {
		t.Errorf("expected v2, got %s", registry.GetApiVersion())
	}
}

func TestRouteRegistry_MultipleMethods(t *testing.T) {
	registry := NewRouteRegistry()

	getRoute := CreateRoute("/test", GET, func(ctx context.Context, w http.ResponseWriter, r *http.Request) {})
	postRoute := CreateRoute("/test", POST, func(ctx context.Context, w http.ResponseWriter, r *http.Request) {})

	registry.AddRoute(getRoute)
	registry.AddRoute(postRoute)

	if !registry.ExistHandler("/test", GET) {
		t.Error("expected GET route to exist")
	}
	if !registry.ExistHandler("/test", POST) {
		t.Error("expected POST route to exist")
	}
}

func TestRouteRegistry_Handle(t *testing.T) {
	registry := NewRouteRegistry()

	handlerCalled := false
	handler := func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
	}
	registry.AddHandler("/test", GET, handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	registry.Handle(context.Background(), NewResponseWriter(w), req)

	if !handlerCalled {
		t.Error("expected handler to be called")
	}
}

func TestPatternFilter_SimplePath(t *testing.T) {
	filter := NewPatternFilter("/api/v1/users")

	tests := []struct {
		path     string
		expected bool
	}{
		{"/api/v1/users", true},
		{"/api/v1/users/", true},
		{"/api/v1/users/123", false},
		{"/api/v2/users", false},
	}

	for _, tt := range tests {
		result := filter.Match(tt.path)
		if result != tt.expected {
			t.Errorf("expected %v for path %s, got %v", tt.expected, tt.path, result)
		}
	}
}

func TestPatternFilter_Wildcard(t *testing.T) {
	filter := NewPatternFilter("/api/v1/**")

	tests := []struct {
		path     string
		expected bool
	}{
		{"/api/v1/", true},
		{"/api/v1/users", true},
		{"/api/v1/users/123", true},
		{"/api/v1/users/123/profile", true},
		{"/api/v2/users", false},
	}

	for _, tt := range tests {
		result := filter.Match(tt.path)
		if result != tt.expected {
			t.Errorf("expected %v for path %s, got %v", tt.expected, tt.path, result)
		}
	}
}

func TestPatternFilter_Param(t *testing.T) {
	filter := NewPatternFilter("/api/users/:id")

	tests := []struct {
		path     string
		expected bool
	}{
		{"/api/users/123", true},
		{"/api/users/abc", true},
		{"/api/users/", false},
		{"/api/users", false},
	}

	for _, tt := range tests {
		result := filter.Match(tt.path)
		if result != tt.expected {
			t.Errorf("expected %v for path %s, got %v", tt.expected, tt.path, result)
		}
	}
}

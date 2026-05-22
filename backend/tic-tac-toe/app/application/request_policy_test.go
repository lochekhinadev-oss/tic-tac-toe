package application

import (
	"net/http"
	"testing"
)

func TestRoutePermissionPolicyResolveRequestPermission(t *testing.T) {
	policy := NewRoutePermissionPolicy()

	cases := []struct {
		name     string
		method   string
		path     string
		resource string
		action   string
		ok       bool
	}{
		{name: "create game", method: http.MethodPost, path: "/games", resource: "games", action: "create", ok: true},
		{name: "list games", method: http.MethodGet, path: "/games", resource: "games", action: "list", ok: true},
		{name: "game history", method: http.MethodGet, path: "/games/history", resource: "games", action: "history", ok: true},
		{name: "leaderboard", method: http.MethodGet, path: "/games/leaderboard", resource: "games", action: "leaderboard", ok: true},
		{name: "read game", method: http.MethodGet, path: "/games/123e4567-e89b-42d3-a456-426614174000", resource: "games", action: "read", ok: true},
		{name: "join game", method: http.MethodPost, path: "/games/123e4567-e89b-42d3-a456-426614174000/join", resource: "games", action: "join", ok: true},
		{name: "move game", method: http.MethodPost, path: "/games/123e4567-e89b-42d3-a456-426614174000/move", resource: "games", action: "move", ok: true},
		{name: "read self", method: http.MethodGet, path: "/users/me", resource: "users", action: "read_self", ok: true},
		{name: "delete self", method: http.MethodDelete, path: "/users/me", resource: "users", action: "delete_self", ok: true},
		{name: "read any user", method: http.MethodGet, path: "/users/123e4567-e89b-42d3-a456-426614174000", resource: "users", action: "read_any", ok: true},
		{name: "unknown route", method: http.MethodGet, path: "/metrics", ok: false},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			permission, ok := policy.ResolveRequestPermission(tt.method, tt.path)
			if ok != tt.ok {
				t.Fatalf("expected ok=%t, got %t", tt.ok, ok)
			}
			if !tt.ok {
				return
			}
			if permission.Resource != tt.resource || permission.Action != tt.action {
				t.Fatalf("unexpected permission: %#v", permission)
			}
		})
	}
}

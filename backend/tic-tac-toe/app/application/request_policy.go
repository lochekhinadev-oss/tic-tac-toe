package application

import (
	"net/http"
	"strings"

	"tic-tac-toe/app/domain"
)

type RequestPermissionResolver interface {
	ResolveRequestPermission(method string, path string) (domain.Permission, bool)
}

type RoutePermissionPolicy struct{}

func NewRoutePermissionPolicy() *RoutePermissionPolicy {
	return &RoutePermissionPolicy{}
}

func (p *RoutePermissionPolicy) ResolveRequestPermission(method string, path string) (domain.Permission, bool) {
	normalizedPath := strings.TrimRight(path, "/")
	if normalizedPath == "" {
		normalizedPath = "/"
	}

	switch {
	case method == http.MethodPost && normalizedPath == "/games":
		return domain.Permission{Resource: "games", Action: "create"}, true
	case method == http.MethodGet && normalizedPath == "/games":
		return domain.Permission{Resource: "games", Action: "list"}, true
	case method == http.MethodGet && normalizedPath == "/games/history":
		return domain.Permission{Resource: "games", Action: "history"}, true
	case method == http.MethodGet && normalizedPath == "/games/leaderboard":
		return domain.Permission{Resource: "games", Action: "leaderboard"}, true
	case method == http.MethodGet && isUUIDPath(normalizedPath, "games"):
		return domain.Permission{Resource: "games", Action: "read"}, true
	case method == http.MethodPost && isUUIDActionPath(normalizedPath, "games", "join"):
		return domain.Permission{Resource: "games", Action: "join"}, true
	case method == http.MethodPost && isUUIDActionPath(normalizedPath, "games", "move"):
		return domain.Permission{Resource: "games", Action: "move"}, true
	case method == http.MethodGet && normalizedPath == "/users/me":
		return domain.Permission{Resource: "users", Action: "read_self"}, true
	case method == http.MethodDelete && normalizedPath == "/users/me":
		return domain.Permission{Resource: "users", Action: "delete_self"}, true
	case method == http.MethodGet && isUUIDPath(normalizedPath, "users"):
		return domain.Permission{Resource: "users", Action: "read_any"}, true
	default:
		return domain.Permission{}, false
	}
}

func isUUIDPath(path string, resource string) bool {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 2 || parts[0] != resource {
		return false
	}
	return parts[1] != ""
}

func isUUIDActionPath(path string, resource string, action string) bool {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 3 || parts[0] != resource || parts[2] != action {
		return false
	}
	return parts[1] != ""
}

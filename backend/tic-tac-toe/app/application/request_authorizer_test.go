package application

import (
	"context"
	"errors"
	"testing"

	"tic-tac-toe/app/domain"
)

type requestAuthorizationStub struct {
	allowed bool
	err     error
	called  bool
	perm    domain.Permission
	version int64
}

func (s *requestAuthorizationStub) GrantDefaultRole(context.Context, string) error { return nil }

func (s *requestAuthorizationStub) GrantRoleToUser(context.Context, string, string) error { return nil }

func (s *requestAuthorizationStub) RevokeRoleFromUser(context.Context, string, string) error {
	return nil
}

func (s *requestAuthorizationStub) LoadPrincipalVersion(context.Context, string) (int64, error) {
	return s.version, nil
}

func (s *requestAuthorizationStub) LoadPrincipal(context.Context, string) (domain.Principal, error) {
	return domain.Principal{AuthzVersion: s.version}, nil
}

func (s *requestAuthorizationStub) Can(_ context.Context, _ string, permission domain.Permission) (bool, error) {
	s.called = true
	s.perm = permission
	return s.allowed, s.err
}

func TestRequestAuthorizationServiceAuthorizeRequest(t *testing.T) {
	authz := &requestAuthorizationStub{allowed: true}
	service := NewRequestAuthorizationService(authz, NewRoutePermissionPolicy())

	allowed, err := service.AuthorizeRequest(context.Background(), "user-1", "GET", "/games")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !allowed {
		t.Fatal("expected request to be allowed")
	}
	if !authz.called || authz.perm != (domain.Permission{Resource: "games", Action: "list"}) {
		t.Fatalf("unexpected permission call: %#v", authz.perm)
	}
}

func TestRequestAuthorizationServiceRejectsUnknownRoute(t *testing.T) {
	authz := &requestAuthorizationStub{allowed: true}
	service := NewRequestAuthorizationService(authz, NewRoutePermissionPolicy())

	allowed, err := service.AuthorizeRequest(context.Background(), "user-1", "GET", "/metrics")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if allowed {
		t.Fatal("expected request to be denied")
	}
	if authz.called {
		t.Fatal("expected authorization backend not to be called")
	}
}

func TestRequestAuthorizationServicePropagatesErrors(t *testing.T) {
	authz := &requestAuthorizationStub{allowed: false, err: errors.New("db failed")}
	service := NewRequestAuthorizationService(authz, NewRoutePermissionPolicy())

	allowed, err := service.AuthorizeRequest(context.Background(), "user-1", "GET", "/games")
	if err == nil {
		t.Fatal("expected error")
	}
	if allowed {
		t.Fatal("expected request to be denied on error")
	}
}

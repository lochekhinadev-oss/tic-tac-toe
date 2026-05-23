package application

import (
	"context"
	"errors"
	"testing"

	googleuuid "github.com/google/uuid"

	"tic-tac-toe/app/domain"
)

type requestAuthorizationStub struct {
	allowed bool
	err     error
	called  bool
	perm    domain.Permission
	version int64
}

func (s *requestAuthorizationStub) GrantDefaultRole(context.Context, googleuuid.UUID) error {
	return nil
}

func (s *requestAuthorizationStub) GrantRoleToUser(context.Context, googleuuid.UUID, string) error {
	return nil
}

func (s *requestAuthorizationStub) RevokeRoleFromUser(context.Context, googleuuid.UUID, string) error {
	return nil
}

func (s *requestAuthorizationStub) LoadPrincipalVersion(context.Context, googleuuid.UUID) (int64, error) {
	return s.version, nil
}

func (s *requestAuthorizationStub) LoadPrincipal(context.Context, googleuuid.UUID) (domain.Principal, error) {
	return domain.Principal{AuthzVersion: s.version}, nil
}

func (s *requestAuthorizationStub) Can(_ context.Context, _ googleuuid.UUID, permission domain.Permission) (bool, error) {
	s.called = true
	s.perm = permission
	return s.allowed, s.err
}

func TestRequestAuthorizationServiceAuthorizeRequest(t *testing.T) {
	authz := &requestAuthorizationStub{allowed: true}
	service := NewRequestAuthorizationService(authz, NewRoutePermissionPolicy())
	userUUID := googleuuid.MustParse("123e4567-e89b-42d3-a456-426614174001")

	allowed, err := service.AuthorizeRequest(context.Background(), userUUID, "GET", "/games")
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
	userUUID := googleuuid.MustParse("123e4567-e89b-42d3-a456-426614174001")

	allowed, err := service.AuthorizeRequest(context.Background(), userUUID, "GET", "/metrics")
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
	userUUID := googleuuid.MustParse("123e4567-e89b-42d3-a456-426614174001")

	allowed, err := service.AuthorizeRequest(context.Background(), userUUID, "GET", "/games")
	if err == nil {
		t.Fatal("expected error")
	}
	if allowed {
		t.Fatal("expected request to be denied on error")
	}
}

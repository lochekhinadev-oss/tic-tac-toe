package application

import (
	"context"
	"errors"
	"testing"

	"tic-tac-toe/app/domain"
)

type authorizationRepositoryStub struct {
	assignUserUUID string
	assignRoleName string
	assignErr      error
	principal      domain.Principal
	version        int64
	versionErr     error
	versionCalls   int
	loadErr        error
}

func (s *authorizationRepositoryStub) AssignRoleToUser(_ context.Context, userUUID string, roleName string) error {
	s.assignUserUUID = userUUID
	s.assignRoleName = roleName
	return s.assignErr
}

func (s *authorizationRepositoryStub) RevokeRoleFromUser(_ context.Context, userUUID string, roleName string) error {
	s.assignUserUUID = userUUID
	s.assignRoleName = roleName
	return s.assignErr
}

func (s *authorizationRepositoryStub) LoadPrincipalVersion(context.Context, string) (int64, error) {
	s.versionCalls++
	return s.version, s.versionErr
}

func (s *authorizationRepositoryStub) LoadPrincipal(context.Context, string) (domain.Principal, error) {
	return s.principal, s.loadErr
}

func TestAuthorizationServiceGrantDefaultRole(t *testing.T) {
	repo := &authorizationRepositoryStub{}
	service := NewAuthorizationService(repo)

	if err := service.GrantDefaultRole(context.Background(), "user-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.assignUserUUID != "user-1" || repo.assignRoleName != domain.DefaultPlayerRole {
		t.Fatalf("unexpected assign call: %#v", repo)
	}
}

func TestAuthorizationServiceRevokeRoleFromUser(t *testing.T) {
	repo := &authorizationRepositoryStub{}
	service := NewAuthorizationService(repo)

	if err := service.RevokeRoleFromUser(context.Background(), "user-1", "admin"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.assignUserUUID != "user-1" || repo.assignRoleName != "admin" {
		t.Fatalf("unexpected revoke call: %#v", repo)
	}
}

func TestAuthorizationServiceCan(t *testing.T) {
	repo := &authorizationRepositoryStub{
		version: 7,
		principal: domain.Principal{
			UserUUID:     "user-1",
			AuthzVersion: 7,
			Roles:        []string{domain.DefaultPlayerRole},
			Permissions: []domain.Permission{
				{Resource: "games", Action: "create"},
				{Resource: "users", Action: "delete_self"},
			},
		},
	}
	service := NewAuthorizationService(repo)

	allowed, err := service.Can(context.Background(), "user-1", domain.Permission{Resource: "games", Action: "create"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !allowed {
		t.Fatal("expected permission to be allowed")
	}

	allowed, err = service.Can(context.Background(), "user-1", domain.Permission{Resource: "games", Action: "move"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if allowed {
		t.Fatal("expected permission to be denied")
	}
	if repo.versionCalls != 1 {
		t.Fatalf("expected one version check after cache warmup, got %d", repo.versionCalls)
	}
}

func TestAuthorizationServiceUsesCachedPrincipalUntilVersionChanges(t *testing.T) {
	repo := &authorizationRepositoryStub{
		version: 1,
		principal: domain.Principal{
			UserUUID:     "user-1",
			AuthzVersion: 1,
			Roles:        []string{domain.DefaultPlayerRole},
			Permissions:  []domain.Permission{{Resource: "games", Action: "create"}},
		},
	}
	service := NewAuthorizationService(repo)

	allowed, err := service.Can(context.Background(), "user-1", domain.Permission{Resource: "games", Action: "create"})
	if err != nil || !allowed {
		t.Fatalf("unexpected first authorize result: allowed=%v err=%v", allowed, err)
	}
	if repo.versionCalls != 0 {
		t.Fatalf("expected no version check on first cache fill, got %d", repo.versionCalls)
	}

	allowed, err = service.Can(context.Background(), "user-1", domain.Permission{Resource: "games", Action: "create"})
	if err != nil || !allowed {
		t.Fatalf("unexpected second authorize result: allowed=%v err=%v", allowed, err)
	}
	if repo.versionCalls != 1 {
		t.Fatalf("expected one version check on cache hit, got %d", repo.versionCalls)
	}

	repo.version = 2
	repo.principal.AuthzVersion = 2
	repo.principal.Permissions = []domain.Permission{{Resource: "games", Action: "move"}}

	allowed, err = service.Can(context.Background(), "user-1", domain.Permission{Resource: "games", Action: "create"})
	if err != nil {
		t.Fatalf("unexpected third authorize error: %v", err)
	}
	if allowed {
		t.Fatal("expected stale permission to be denied after version bump")
	}
	if repo.versionCalls != 2 {
		t.Fatalf("expected version recheck after bump, got %d", repo.versionCalls)
	}
}

func TestAuthorizationServicePropagatesErrors(t *testing.T) {
	service := NewAuthorizationService(&authorizationRepositoryStub{loadErr: errors.New("db failed")})

	if _, err := service.LoadPrincipal(context.Background(), "user-1"); err == nil {
		t.Fatal("expected error")
	}
	if _, err := service.Can(context.Background(), "user-1", domain.Permission{Resource: "games", Action: "create"}); err == nil {
		t.Fatal("expected error")
	}
}

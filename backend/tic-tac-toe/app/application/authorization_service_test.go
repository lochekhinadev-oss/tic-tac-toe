package application

import (
	"context"
	"errors"
	"testing"

	googleuuid "github.com/google/uuid"

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

func (s *authorizationRepositoryStub) AssignRoleToUser(_ context.Context, userUUID googleuuid.UUID, roleName string) error {
	s.assignUserUUID = userUUID.String()
	s.assignRoleName = roleName
	return s.assignErr
}

func (s *authorizationRepositoryStub) RevokeRoleFromUser(_ context.Context, userUUID googleuuid.UUID, roleName string) error {
	s.assignUserUUID = userUUID.String()
	s.assignRoleName = roleName
	return s.assignErr
}

func (s *authorizationRepositoryStub) LoadPrincipalVersion(context.Context, googleuuid.UUID) (int64, error) {
	s.versionCalls++
	return s.version, s.versionErr
}

func (s *authorizationRepositoryStub) LoadPrincipal(context.Context, googleuuid.UUID) (domain.Principal, error) {
	return s.principal, s.loadErr
}

func TestAuthorizationServiceGrantDefaultRole(t *testing.T) {
	repo := &authorizationRepositoryStub{}
	service := NewAuthorizationService(repo)

	userUUID := googleuuid.MustParse("123e4567-e89b-42d3-a456-426614174001")
	if err := service.GrantDefaultRole(context.Background(), userUUID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.assignUserUUID != userUUID.String() || repo.assignRoleName != domain.DefaultPlayerRole {
		t.Fatalf("unexpected assign call: %#v", repo)
	}
}

func TestAuthorizationServiceRevokeRoleFromUser(t *testing.T) {
	repo := &authorizationRepositoryStub{}
	service := NewAuthorizationService(repo)

	userUUID := googleuuid.MustParse("123e4567-e89b-42d3-a456-426614174001")
	if err := service.RevokeRoleFromUser(context.Background(), userUUID, "admin"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.assignUserUUID != userUUID.String() || repo.assignRoleName != "admin" {
		t.Fatalf("unexpected revoke call: %#v", repo)
	}
}

func TestAuthorizationServiceCan(t *testing.T) {
	userUUID := googleuuid.MustParse("123e4567-e89b-42d3-a456-426614174001")
	repo := &authorizationRepositoryStub{
		version: 7,
		principal: domain.Principal{
			UserUUID:     userUUID.String(),
			AuthzVersion: 7,
			Roles:        []string{domain.DefaultPlayerRole},
			Permissions: []domain.Permission{
				{Resource: "games", Action: "create"},
				{Resource: "users", Action: "delete_self"},
			},
		},
	}
	service := NewAuthorizationService(repo)
	allowed, err := service.Can(context.Background(), userUUID, domain.Permission{Resource: "games", Action: "create"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !allowed {
		t.Fatal("expected permission to be allowed")
	}

	allowed, err = service.Can(context.Background(), userUUID, domain.Permission{Resource: "games", Action: "move"})
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
	userUUID := googleuuid.MustParse("123e4567-e89b-42d3-a456-426614174001")
	repo := &authorizationRepositoryStub{
		version: 1,
		principal: domain.Principal{
			UserUUID:     userUUID.String(),
			AuthzVersion: 1,
			Roles:        []string{domain.DefaultPlayerRole},
			Permissions:  []domain.Permission{{Resource: "games", Action: "create"}},
		},
	}
	service := NewAuthorizationService(repo)
	allowed, err := service.Can(context.Background(), userUUID, domain.Permission{Resource: "games", Action: "create"})
	if err != nil || !allowed {
		t.Fatalf("unexpected first authorize result: allowed=%v err=%v", allowed, err)
	}
	if repo.versionCalls != 0 {
		t.Fatalf("expected no version check on first cache fill, got %d", repo.versionCalls)
	}

	allowed, err = service.Can(context.Background(), userUUID, domain.Permission{Resource: "games", Action: "create"})
	if err != nil || !allowed {
		t.Fatalf("unexpected second authorize result: allowed=%v err=%v", allowed, err)
	}
	if repo.versionCalls != 1 {
		t.Fatalf("expected one version check on cache hit, got %d", repo.versionCalls)
	}

	repo.version = 2
	repo.principal.AuthzVersion = 2
	repo.principal.Permissions = []domain.Permission{{Resource: "games", Action: "move"}}

	allowed, err = service.Can(context.Background(), userUUID, domain.Permission{Resource: "games", Action: "create"})
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

	userUUID := googleuuid.MustParse("123e4567-e89b-42d3-a456-426614174001")
	if _, err := service.LoadPrincipal(context.Background(), userUUID); err == nil {
		t.Fatal("expected error")
	}
	if _, err := service.Can(context.Background(), userUUID, domain.Permission{Resource: "games", Action: "create"}); err == nil {
		t.Fatal("expected error")
	}
}

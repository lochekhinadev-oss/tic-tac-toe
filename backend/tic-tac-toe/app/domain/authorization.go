package domain

import (
	"context"

	googleuuid "github.com/google/uuid"
)

const DefaultPlayerRole = "player"

type Permission struct {
	Resource string
	Action   string
}

func (p Permission) IsZero() bool {
	return p.Resource == "" && p.Action == ""
}

type Principal struct {
	UserUUID     string
	AuthzVersion int64
	Roles        []string
	Permissions  []Permission
}

type AuthorizationRepository interface {
	AssignRoleToUser(ctx context.Context, userUUID googleuuid.UUID, roleName string) error
	RevokeRoleFromUser(ctx context.Context, userUUID googleuuid.UUID, roleName string) error
	LoadPrincipalVersion(ctx context.Context, userUUID googleuuid.UUID) (int64, error)
	LoadPrincipal(ctx context.Context, userUUID googleuuid.UUID) (Principal, error)
}

type AuthorizationService interface {
	GrantDefaultRole(ctx context.Context, userUUID googleuuid.UUID) error
	GrantRoleToUser(ctx context.Context, userUUID googleuuid.UUID, roleName string) error
	RevokeRoleFromUser(ctx context.Context, userUUID googleuuid.UUID, roleName string) error
	LoadPrincipal(ctx context.Context, userUUID googleuuid.UUID) (Principal, error)
	Can(ctx context.Context, userUUID googleuuid.UUID, permission Permission) (bool, error)
}

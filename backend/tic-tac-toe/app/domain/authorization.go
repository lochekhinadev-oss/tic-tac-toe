package domain

import "context"

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
	AssignRoleToUser(ctx context.Context, userUUID string, roleName string) error
	RevokeRoleFromUser(ctx context.Context, userUUID string, roleName string) error
	LoadPrincipalVersion(ctx context.Context, userUUID string) (int64, error)
	LoadPrincipal(ctx context.Context, userUUID string) (Principal, error)
}

type AuthorizationService interface {
	GrantDefaultRole(ctx context.Context, userUUID string) error
	GrantRoleToUser(ctx context.Context, userUUID string, roleName string) error
	RevokeRoleFromUser(ctx context.Context, userUUID string, roleName string) error
	LoadPrincipal(ctx context.Context, userUUID string) (Principal, error)
	Can(ctx context.Context, userUUID string, permission Permission) (bool, error)
}

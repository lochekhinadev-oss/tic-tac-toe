package application

import (
	"context"
	"sync"
	"time"

	googleuuid "github.com/google/uuid"

	"tic-tac-toe/app/domain"
)

const defaultPrincipalCacheTTL = 30 * time.Second

type AuthorizationService struct {
	repository     domain.AuthorizationRepository
	cacheTTL       time.Duration
	cacheMu        sync.Mutex
	principalCache map[string]cachedPrincipal
}

func NewAuthorizationService(repository domain.AuthorizationRepository) *AuthorizationService {
	return &AuthorizationService{
		repository:     repository,
		cacheTTL:       defaultPrincipalCacheTTL,
		principalCache: make(map[string]cachedPrincipal),
	}
}

func (s *AuthorizationService) GrantDefaultRole(ctx context.Context, userUUID googleuuid.UUID) error {
	logApplication("grant default role", "user_uuid", userUUID, "role", domain.DefaultPlayerRole)
	if err := s.GrantRoleToUser(ctx, userUUID, domain.DefaultPlayerRole); err != nil {
		logApplication("grant default role failed", "user_uuid", userUUID, "role", domain.DefaultPlayerRole, "error", err)
		return err
	}
	logApplication("grant default role ok", "user_uuid", userUUID, "role", domain.DefaultPlayerRole)
	return nil
}

func (s *AuthorizationService) GrantRoleToUser(ctx context.Context, userUUID googleuuid.UUID, roleName string) error {
	logApplication("grant role", "user_uuid", userUUID, "role", roleName)
	if err := s.repository.AssignRoleToUser(ctx, userUUID, roleName); err != nil {
		logApplication("grant role failed", "user_uuid", userUUID, "role", roleName, "error", err)
		return err
	}
	s.invalidatePrincipalCache(userUUID)
	logApplication("grant role ok", "user_uuid", userUUID, "role", roleName)
	return nil
}

func (s *AuthorizationService) RevokeRoleFromUser(ctx context.Context, userUUID googleuuid.UUID, roleName string) error {
	logApplication("revoke role", "user_uuid", userUUID, "role", roleName)
	if err := s.repository.RevokeRoleFromUser(ctx, userUUID, roleName); err != nil {
		logApplication("revoke role failed", "user_uuid", userUUID, "role", roleName, "error", err)
		return err
	}
	s.invalidatePrincipalCache(userUUID)
	logApplication("revoke role ok", "user_uuid", userUUID, "role", roleName)
	return nil
}

func (s *AuthorizationService) LoadPrincipal(ctx context.Context, userUUID googleuuid.UUID) (domain.Principal, error) {
	logApplication("load principal", "user_uuid", userUUID)
	if principal, ok, err := s.cachedPrincipal(ctx, userUUID); err != nil {
		logApplication("load principal cache check failed", "user_uuid", userUUID, "error", err)
		return domain.Principal{}, err
	} else if ok {
		logApplication("load principal cache hit", "user_uuid", userUUID, "version", principal.AuthzVersion, "roles", len(principal.Roles), "permissions", len(principal.Permissions))
		return principal, nil
	}

	principal, err := s.repository.LoadPrincipal(ctx, userUUID)
	if err != nil {
		logApplication("load principal failed", "user_uuid", userUUID, "error", err)
		return domain.Principal{}, err
	}
	s.storePrincipalCache(principal)
	logApplication("load principal ok", "user_uuid", userUUID, "version", principal.AuthzVersion, "roles", len(principal.Roles), "permissions", len(principal.Permissions))
	return principal, nil
}

func (s *AuthorizationService) Can(ctx context.Context, userUUID googleuuid.UUID, permission domain.Permission) (bool, error) {
	logApplication("authorize", "user_uuid", userUUID, "resource", permission.Resource, "action", permission.Action)
	principal, err := s.LoadPrincipal(ctx, userUUID)
	if err != nil {
		return false, err
	}
	for _, granted := range principal.Permissions {
		if granted == permission {
			logApplication("authorize ok", "user_uuid", userUUID, "resource", permission.Resource, "action", permission.Action)
			return true, nil
		}
	}
	logApplication("authorize denied", "user_uuid", userUUID, "resource", permission.Resource, "action", permission.Action)
	return false, nil
}

type cachedPrincipal struct {
	principal domain.Principal
	version   int64
	expiresAt time.Time
}

func (s *AuthorizationService) cachedPrincipal(ctx context.Context, userUUID googleuuid.UUID) (domain.Principal, bool, error) {
	s.cacheMu.Lock()
	entry, ok := s.principalCache[userUUID.String()]
	s.cacheMu.Unlock()
	if !ok {
		return domain.Principal{}, false, nil
	}
	if time.Now().After(entry.expiresAt) {
		s.invalidatePrincipalCache(userUUID)
		return domain.Principal{}, false, nil
	}

	version, err := s.repository.LoadPrincipalVersion(ctx, userUUID)
	if err != nil {
		return domain.Principal{}, false, err
	}
	if version != entry.version {
		s.invalidatePrincipalCache(userUUID)
		return domain.Principal{}, false, nil
	}

	return clonePrincipal(entry.principal), true, nil
}

func (s *AuthorizationService) storePrincipalCache(principal domain.Principal) {
	s.cacheMu.Lock()
	s.principalCache[principal.UserUUID] = cachedPrincipal{
		principal: clonePrincipal(principal),
		version:   principal.AuthzVersion,
		expiresAt: time.Now().Add(s.cacheTTL),
	}
	s.cacheMu.Unlock()
}

func (s *AuthorizationService) invalidatePrincipalCache(userUUID googleuuid.UUID) {
	s.cacheMu.Lock()
	delete(s.principalCache, userUUID.String())
	s.cacheMu.Unlock()
}

func clonePrincipal(principal domain.Principal) domain.Principal {
	return domain.Principal{
		UserUUID:     principal.UserUUID,
		AuthzVersion: principal.AuthzVersion,
		Roles:        append([]string(nil), principal.Roles...),
		Permissions:  append([]domain.Permission(nil), principal.Permissions...),
	}
}

package application

import (
	"context"

	"tic-tac-toe/app/domain"
)

type RequestAuthorizer interface {
	AuthorizeRequest(ctx context.Context, userUUID string, method string, path string) (bool, error)
}

type RequestAuthorizationService struct {
	authorization domain.AuthorizationService
	policy        RequestPermissionResolver
}

func NewRequestAuthorizationService(authorization domain.AuthorizationService, policy RequestPermissionResolver) *RequestAuthorizationService {
	return &RequestAuthorizationService{
		authorization: authorization,
		policy:        policy,
	}
}

func (s *RequestAuthorizationService) AuthorizeRequest(ctx context.Context, userUUID string, method string, path string) (bool, error) {
	permission, ok := s.policy.ResolveRequestPermission(method, path)
	if !ok {
		return false, nil
	}
	return s.authorization.Can(ctx, userUUID, permission)
}

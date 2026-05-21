package mapper

import (
	"testing"

	"tic-tac-toe/app/domain"
	"tic-tac-toe/infrastructure/postgres/datasource"
)

func TestUserMapper(t *testing.T) {
	domainUser := domain.User{UUID: "user-1", Login: "player", Password: "secret"}
	dsUser := ToDatasourceUser(domainUser)
	if dsUser.UUID != domainUser.UUID || dsUser.Login != domainUser.Login || dsUser.Password != domainUser.Password {
		t.Fatalf("unexpected datasource user: %#v", dsUser)
	}

	got := ToDomainUser(datasource.User{UUID: "user-2", Login: "other", Password: "hash"})
	if got.UUID != "user-2" || got.Login != "other" || got.Password != "hash" {
		t.Fatalf("unexpected domain user: %#v", got)
	}
}

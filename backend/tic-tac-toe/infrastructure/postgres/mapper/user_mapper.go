package mapper

import (
	"tic-tac-toe/app/domain"
	"tic-tac-toe/infrastructure/postgres/datasource"
)

func ToDatasourceUser(user domain.User) datasource.User {
	return datasource.User{
		UUID:     user.UUID,
		Login:    user.Login,
		Password: user.Password,
	}
}

func ToDomainUser(user datasource.User) domain.User {
	return domain.User{
		UUID:     user.UUID,
		Login:    user.Login,
		Password: user.Password,
	}
}

package main

import "tic-tac-toe/internal/di"

// @title Tic-Tac-Toe API
// @version 1.0
// @description HTTP API for user authentication and tic-tac-toe games.
// @BasePath /tic-tac-toe
// @schemes https
// @securityDefinitions.apikey SessionCookieAuth
// @in cookie
// @name tic-tac-toe.session
func main() {
	di.NewApp().Run()
}

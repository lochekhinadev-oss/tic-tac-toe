package main

import "tic-tac-toe/internal/di"

// @title Tic-Tac-Toe API
// @version 1.0
// @description HTTP API for user authentication and tic-tac-toe games.
// @BasePath /
// @schemes http
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
func main() {
	di.NewApp().Run()
}

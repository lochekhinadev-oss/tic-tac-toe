package main

import "tic-tac-toe/internal/di"

type appRunner interface {
	Run()
}

var newApp = func() appRunner {
	return di.NewApp()
}

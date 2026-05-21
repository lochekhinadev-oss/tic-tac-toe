package main

import "testing"

func TestMainRunsConfiguredApp(t *testing.T) {
	originalNewApp := newApp
	defer func() {
		newApp = originalNewApp
	}()

	runner := &appRunnerStub{}
	newApp = func() appRunner {
		return runner
	}

	main()

	if !runner.called {
		t.Fatal("expected main to run app")
	}
}

type appRunnerStub struct {
	called bool
}

func (r *appRunnerStub) Run() {
	r.called = true
}

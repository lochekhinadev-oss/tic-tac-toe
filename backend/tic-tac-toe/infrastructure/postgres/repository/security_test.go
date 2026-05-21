package repository

import (
	"strings"
	"testing"
)

const sqlInjectionPayload = `admin' OR '1'='1; DROP TABLE users; --`

func assertQueryDoesNotContainPayload(t *testing.T, query string) {
	t.Helper()

	if strings.Contains(query, sqlInjectionPayload) {
		t.Fatalf("query contains unparameterized user input: %q", query)
	}
}

func assertArgsContainPayload(t *testing.T, args []any) {
	t.Helper()

	for _, arg := range args {
		if value, ok := arg.(string); ok && value == sqlInjectionPayload {
			return
		}
	}
	t.Fatalf("expected SQL injection payload to be passed as argument, got %#v", args)
}

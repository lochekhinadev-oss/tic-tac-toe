package log

import "testing"

func TestFields(t *testing.T) {
	t.Setenv("SERVICE_NAME", "test-service")
	t.Setenv("VERSION", "v1.2.3")
	t.Setenv("COMMIT", "abc123")
	t.Setenv("APP_ENV", "testing")

	fields := Fields()
	want := map[string]string{
		"service": "test-service",
		"version": "v1.2.3",
		"commit":  "abc123",
		"env":     "testing",
	}

	if len(fields) != len(want)*2 {
		t.Fatalf("unexpected fields length: got %d want %d", len(fields), len(want)*2)
	}

	for i := 0; i < len(fields); i += 2 {
		key, ok := fields[i].(string)
		if !ok {
			t.Fatalf("unexpected key type at %d: %T", i, fields[i])
		}
		value, ok := fields[i+1].(string)
		if !ok {
			t.Fatalf("unexpected value type at %d: %T", i+1, fields[i+1])
		}
		if wantValue, ok := want[key]; !ok {
			t.Fatalf("unexpected key %q", key)
		} else if wantValue != value {
			t.Fatalf("unexpected value for %q: got %q want %q", key, value, wantValue)
		}
	}
}

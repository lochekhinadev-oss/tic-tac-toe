package main

import (
	"math/rand"
	"testing"

	"tic-tac-toe/app/domain"
)

func TestSeedIdentifiersAndLogin(t *testing.T) {
	if got := userLogin(0); got != "0000" {
		t.Fatalf("expected login 0000, got %q", got)
	}
	if got := userLogin(9999); got != "9999" {
		t.Fatalf("expected login 9999, got %q", got)
	}
	if got := userUUID(42); got != "00000000-0000-4000-8000-000000000042" {
		t.Fatalf("unexpected user uuid: %q", got)
	}
	if got := gameUUID(42); got != "10000000-0000-4000-8000-000000000042" {
		t.Fatalf("unexpected game uuid: %q", got)
	}
}

func TestGameRowUsesDomainConstants(t *testing.T) {
	row, err := gameRow(0, 10, rand.New(rand.NewSource(seedRandomSource)))
	if err != nil {
		t.Fatalf("unexpected game row error: %v", err)
	}

	if row[2] != string(domain.GameModePlayer) {
		t.Fatalf("expected player mode, got %#v", row[2])
	}
	if row[3] != string(domain.GameStatePlayerWins) {
		t.Fatalf("expected player wins state, got %#v", row[3])
	}
}

func TestCopySourcesStreamRows(t *testing.T) {
	users := &userCopySource{total: 2}
	for i := 0; i < 2; i++ {
		if !users.Next() {
			t.Fatalf("expected user row %d", i)
		}
		values, err := users.Values()
		if err != nil {
			t.Fatalf("unexpected user values error: %v", err)
		}
		if len(values) != 3 {
			t.Fatalf("unexpected user values: %#v", values)
		}
	}
	if users.Next() {
		t.Fatal("expected user source to stop")
	}
	if err := users.Err(); err != nil {
		t.Fatalf("unexpected user source error: %v", err)
	}

	games := newGameCopySource(2, 10)
	for i := 0; i < 2; i++ {
		if !games.Next() {
			t.Fatalf("expected game row %d", i)
		}
		values, err := games.Values()
		if err != nil {
			t.Fatalf("unexpected game values error: %v", err)
		}
		if len(values) != 9 {
			t.Fatalf("unexpected game values: %#v", values)
		}
	}
	if games.Next() {
		t.Fatal("expected game source to stop")
	}
	if err := games.Err(); err != nil {
		t.Fatalf("unexpected game source error: %v", err)
	}
}

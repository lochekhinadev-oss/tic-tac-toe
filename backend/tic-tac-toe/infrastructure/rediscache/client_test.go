package rediscache

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"testing"
	"time"

	"tic-tac-toe/app/domain"
)

func TestLeaderboardCacheRoundTrip(t *testing.T) {
	cache, state := newFakeRedisClient(t)
	defer cache.Close()

	ctx := context.Background()
	players := []domain.WonGameInfo{
		{UserUUID: "user-1", Login: "alpha", WinRatio: 2.5},
		{UserUUID: "user-2", Login: "beta", WinRatio: 1.25},
	}

	if err := cache.SetLeaderboard(ctx, 10, players, time.Minute); err != nil {
		t.Fatalf("unexpected set error: %v", err)
	}

	got, ok, err := cache.GetLeaderboard(ctx, 10)
	if err != nil {
		t.Fatalf("unexpected get error: %v", err)
	}
	if !ok {
		t.Fatal("expected cache hit")
	}
	if len(got) != len(players) || got[0].Login != "alpha" || got[1].Login != "beta" {
		t.Fatalf("unexpected leaderboard payload: %#v", got)
	}

	if err := cache.InvalidateLeaderboard(ctx); err != nil {
		t.Fatalf("unexpected invalidate error: %v", err)
	}
	if state.version != 1 {
		t.Fatalf("expected version to advance after invalidation, got %d", state.version)
	}

	miss, ok, err := cache.GetLeaderboard(ctx, 10)
	if err != nil {
		t.Fatalf("unexpected post-invalidate get error: %v", err)
	}
	if ok || miss != nil {
		t.Fatalf("expected cache miss after invalidation, got ok=%t payload=%#v", ok, miss)
	}
}

func newFakeRedisClient(t *testing.T) (*Client, *fakeRedisState) {
	t.Helper()

	state := &fakeRedisState{values: make(map[string]string)}
	client := &Client{
		addr:      "pipe",
		opTimeout: defaultOperationTimeout,
		dial: func(context.Context, string) (net.Conn, error) {
			clientConn, serverConn := net.Pipe()
			go serveFakeRedis(serverConn, state)
			return clientConn, nil
		},
	}

	return client, state
}

type fakeRedisState struct {
	values  map[string]string
	version int64
}

func serveFakeRedis(conn net.Conn, state *fakeRedisState) {
	defer conn.Close()

	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)
	for {
		args, err := readFakeCommand(reader)
		if err != nil {
			return
		}
		if err := state.handle(writer, args); err != nil {
			return
		}
	}
}

func (s *fakeRedisState) handle(writer *bufio.Writer, args []string) error {
	if len(args) == 0 {
		return nil
	}

	switch strings.ToUpper(args[0]) {
	case "GET":
		value, ok := s.values[args[1]]
		if !ok {
			_, err := writer.WriteString("$-1\r\n")
			if err != nil {
				return err
			}
			return writer.Flush()
		}
		if _, err := fmt.Fprintf(writer, "$%d\r\n%s\r\n", len(value), value); err != nil {
			return err
		}
		return writer.Flush()
	case "SET":
		s.values[args[1]] = args[2]
		_, err := writer.WriteString("+OK\r\n")
		if err != nil {
			return err
		}
		return writer.Flush()
	case "INCR":
		next := int64(1)
		if current, ok := s.values[args[1]]; ok {
			parsed, err := strconv.ParseInt(current, 10, 64)
			if err != nil {
				return err
			}
			next = parsed + 1
		}
		s.values[args[1]] = strconv.FormatInt(next, 10)
		if args[1] == leaderboardVersionKey {
			s.version = next
		}
		if _, err := fmt.Fprintf(writer, ":%d\r\n", next); err != nil {
			return err
		}
		return writer.Flush()
	case "AUTH", "SELECT", "PING":
		_, err := writer.WriteString("+OK\r\n")
		if err != nil {
			return err
		}
		return writer.Flush()
	default:
		_, err := writer.WriteString("-ERR unknown command\r\n")
		if err != nil {
			return err
		}
		return writer.Flush()
	}
}

func readFakeCommand(reader *bufio.Reader) ([]string, error) {
	prefix, err := reader.ReadByte()
	if err != nil {
		return nil, err
	}
	if prefix != '*' {
		return nil, fmt.Errorf("unexpected prefix %q", prefix)
	}

	count, err := readFakeLine(reader)
	if err != nil {
		return nil, err
	}

	total, err := strconv.Atoi(count)
	if err != nil {
		return nil, err
	}

	args := make([]string, 0, total)
	for i := 0; i < total; i++ {
		prefix, err := reader.ReadByte()
		if err != nil {
			return nil, err
		}
		if prefix != '$' {
			return nil, fmt.Errorf("unexpected bulk prefix %q", prefix)
		}

		lengthLine, err := readFakeLine(reader)
		if err != nil {
			return nil, err
		}
		length, err := strconv.Atoi(lengthLine)
		if err != nil {
			return nil, err
		}

		payload := make([]byte, length+2)
		if _, err := io.ReadFull(reader, payload); err != nil {
			return nil, err
		}
		args = append(args, string(payload[:length]))
	}

	return args, nil
}

func readFakeLine(reader *bufio.Reader) (string, error) {
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSuffix(strings.TrimSuffix(line, "\n"), "\r"), nil
}

func TestLeaderboardCacheSerializesPlayers(t *testing.T) {
	cache, _ := newFakeRedisClient(t)
	defer cache.Close()

	ctx := context.Background()
	players := []domain.WonGameInfo{{UserUUID: "user-1", Login: "alpha", WinRatio: 2.5}}
	if err := cache.SetLeaderboard(ctx, 3, players, 10*time.Second); err != nil {
		t.Fatalf("unexpected set error: %v", err)
	}

	raw, ok := mustReadStoredValue(t, cache, ctx, 3)
	if !ok {
		t.Fatal("expected stored leaderboard value")
	}

	var decoded []domain.WonGameInfo
	if err := json.Unmarshal([]byte(raw), &decoded); err != nil {
		t.Fatalf("unexpected json error: %v", err)
	}
	if len(decoded) != 1 || decoded[0].Login != "alpha" {
		t.Fatalf("unexpected serialized payload: %#v", decoded)
	}
}

func mustReadStoredValue(t *testing.T, cache *Client, ctx context.Context, limit int) (string, bool) {
	t.Helper()

	version, err := cache.leaderboardVersion(ctx)
	if err != nil {
		t.Fatalf("unexpected version error: %v", err)
	}

	raw, ok, err := cache.getString(ctx, leaderboardCacheKey(version, limit))
	if err != nil {
		t.Fatalf("unexpected get error: %v", err)
	}
	return raw, ok
}

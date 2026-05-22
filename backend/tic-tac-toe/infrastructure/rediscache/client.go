package rediscache

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"tic-tac-toe/app/domain"
)

const (
	leaderboardVersionKey   = "leaderboard:version"
	defaultOperationTimeout = 2 * time.Second
)

type LeaderboardCache interface {
	GetLeaderboard(ctx context.Context, limit int) ([]domain.WonGameInfo, bool, error)
	SetLeaderboard(ctx context.Context, limit int, players []domain.WonGameInfo, ttl time.Duration) error
	InvalidateLeaderboard(ctx context.Context) error
	Close() error
}

type Client struct {
	addr      string
	password  string
	db        int
	opTimeout time.Duration
	dial      func(context.Context, string) (net.Conn, error)

	mu     sync.Mutex
	conn   net.Conn
	reader *bufio.Reader
	writer *bufio.Writer
}

func NewClient(cfg Config) (*Client, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	parsed, err := url.Parse(cfg.URL)
	if err != nil {
		return nil, err
	}

	dbIndex, err := parseDatabaseIndex(parsed.Path)
	if err != nil {
		return nil, err
	}

	password := ""
	if parsed.User != nil {
		password, _ = parsed.User.Password()
	}

	return &Client{
		addr:      parsed.Host,
		password:  password,
		db:        dbIndex,
		opTimeout: defaultOperationTimeout,
		dial: func(ctx context.Context, addr string) (net.Conn, error) {
			dialer := net.Dialer{Timeout: defaultOperationTimeout}
			return dialer.DialContext(ctx, "tcp", addr)
		},
	}, nil
}

func (c *Client) Close() error {
	if c == nil {
		return nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	return c.closeLocked()
}

func (c *Client) GetLeaderboard(ctx context.Context, limit int) ([]domain.WonGameInfo, bool, error) {
	version, err := c.leaderboardVersion(ctx)
	if err != nil {
		return nil, false, err
	}

	raw, ok, err := c.getString(ctx, leaderboardCacheKey(version, limit))
	if err != nil || !ok {
		return nil, ok, err
	}

	var players []domain.WonGameInfo
	if err := json.Unmarshal([]byte(raw), &players); err != nil {
		return nil, false, err
	}

	return players, true, nil
}

func (c *Client) SetLeaderboard(ctx context.Context, limit int, players []domain.WonGameInfo, ttl time.Duration) error {
	version, err := c.leaderboardVersion(ctx)
	if err != nil {
		return err
	}

	payload, err := json.Marshal(players)
	if err != nil {
		return err
	}

	return c.setString(ctx, leaderboardCacheKey(version, limit), string(payload), ttl)
}

func (c *Client) InvalidateLeaderboard(ctx context.Context) error {
	_, err := c.incr(ctx, leaderboardVersionKey)
	return err
}

func (c *Client) leaderboardVersion(ctx context.Context) (int64, error) {
	raw, ok, err := c.getString(ctx, leaderboardVersionKey)
	if err != nil {
		return 0, err
	}
	if !ok || strings.TrimSpace(raw) == "" {
		return 0, nil
	}

	version, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, err
	}
	return version, nil
}

func leaderboardCacheKey(version int64, limit int) string {
	return fmt.Sprintf("leaderboard:v%d:top:%d", version, limit)
}

func parseDatabaseIndex(path string) (int, error) {
	trimmed := strings.Trim(path, "/")
	if trimmed == "" {
		return 0, nil
	}

	index, err := strconv.Atoi(trimmed)
	if err != nil || index < 0 {
		return 0, fmt.Errorf("invalid redis database index: %q", path)
	}
	return index, nil
}

func (c *Client) getString(ctx context.Context, key string) (string, bool, error) {
	reply, err := c.call(ctx, "GET", key)
	if err != nil {
		return "", false, err
	}

	switch value := reply.(type) {
	case nil:
		return "", false, nil
	case string:
		return value, true, nil
	case []byte:
		return string(value), true, nil
	default:
		return "", false, fmt.Errorf("unexpected redis reply for GET: %T", reply)
	}
}

func (c *Client) setString(ctx context.Context, key string, value string, ttl time.Duration) error {
	if ttl > 0 {
		_, err := c.call(ctx, "SET", key, value, "EX", strconv.Itoa(int(ttl.Seconds())))
		return err
	}

	_, err := c.call(ctx, "SET", key, value)
	return err
}

func (c *Client) incr(ctx context.Context, key string) (int64, error) {
	reply, err := c.call(ctx, "INCR", key)
	if err != nil {
		return 0, err
	}

	value, ok := reply.(int64)
	if !ok {
		return 0, fmt.Errorf("unexpected redis reply for INCR: %T", reply)
	}
	return value, nil
}

func (c *Client) call(ctx context.Context, args ...string) (any, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.ensureConnLocked(ctx); err != nil {
		return nil, err
	}

	if err := c.setDeadlineLocked(ctx); err != nil {
		_ = c.closeLocked()
		return nil, err
	}

	if err := writeCommand(c.writer, args...); err != nil {
		_ = c.closeLocked()
		return nil, err
	}

	reply, err := readReply(c.reader)
	if err != nil {
		_ = c.closeLocked()
		return nil, err
	}

	return reply, nil
}

func (c *Client) ensureConnLocked(ctx context.Context) error {
	if c.conn != nil {
		return nil
	}

	dial := c.dial
	if dial == nil {
		dial = func(ctx context.Context, addr string) (net.Conn, error) {
			dialer := net.Dialer{Timeout: c.opTimeout}
			return dialer.DialContext(ctx, "tcp", addr)
		}
	}

	conn, err := dial(ctx, c.addr)
	if err != nil {
		return err
	}

	c.conn = conn
	c.reader = bufio.NewReader(conn)
	c.writer = bufio.NewWriter(conn)

	if c.password != "" {
		if _, err := c.callNoLock(ctx, "AUTH", c.password); err != nil {
			_ = c.closeLocked()
			return err
		}
	}

	if c.db != 0 {
		if _, err := c.callNoLock(ctx, "SELECT", strconv.Itoa(c.db)); err != nil {
			_ = c.closeLocked()
			return err
		}
	}

	return nil
}

func (c *Client) callNoLock(ctx context.Context, args ...string) (any, error) {
	if err := c.setDeadlineLocked(ctx); err != nil {
		return nil, err
	}

	if err := writeCommand(c.writer, args...); err != nil {
		return nil, err
	}

	return readReply(c.reader)
}

func (c *Client) setDeadlineLocked(ctx context.Context) error {
	deadline, ok := ctx.Deadline()
	if !ok {
		deadline = time.Now().Add(c.opTimeout)
	}

	if err := c.conn.SetDeadline(deadline); err != nil {
		return err
	}
	return nil
}

func (c *Client) closeLocked() error {
	if c.conn == nil {
		return nil
	}

	err := c.conn.Close()
	c.conn = nil
	c.reader = nil
	c.writer = nil
	return err
}

func writeCommand(writer *bufio.Writer, args ...string) error {
	if _, err := fmt.Fprintf(writer, "*%d\r\n", len(args)); err != nil {
		return err
	}

	for _, arg := range args {
		if _, err := fmt.Fprintf(writer, "$%d\r\n%s\r\n", len(arg), arg); err != nil {
			return err
		}
	}

	return writer.Flush()
}

func readReply(reader *bufio.Reader) (any, error) {
	prefix, err := reader.ReadByte()
	if err != nil {
		return nil, err
	}

	switch prefix {
	case '+':
		line, err := readLine(reader)
		if err != nil {
			return nil, err
		}
		return line, nil
	case ':':
		line, err := readLine(reader)
		if err != nil {
			return nil, err
		}
		value, err := strconv.ParseInt(line, 10, 64)
		if err != nil {
			return nil, err
		}
		return value, nil
	case '$':
		line, err := readLine(reader)
		if err != nil {
			return nil, err
		}
		length, err := strconv.Atoi(line)
		if err != nil {
			return nil, err
		}
		if length < 0 {
			return nil, nil
		}

		payload := make([]byte, length+2)
		if _, err := io.ReadFull(reader, payload); err != nil {
			return nil, err
		}
		return string(payload[:length]), nil
	case '-':
		line, err := readLine(reader)
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf(strings.TrimSpace(line))
	default:
		return nil, fmt.Errorf("unexpected redis reply prefix: %q", prefix)
	}
}

func readLine(reader *bufio.Reader) (string, error) {
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}

	return strings.TrimSuffix(strings.TrimSuffix(line, "\n"), "\r"), nil
}

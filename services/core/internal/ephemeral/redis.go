package ephemeral

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"time"
)

type RedisConfig struct {
	Enabled   bool
	Addr      string
	KeyPolicy KeyPolicy
	Timeout   time.Duration
}

type RedisStore struct {
	cfg RedisConfig
}

const (
	maxRedisRESPBulkBytes  = 1 << 20
	maxRedisRESPArrayItems = maxEphemeralProgressReadEntries
	maxRedisRESPLineBytes  = 4096
)

func NewRedisStore(cfg RedisConfig) (*RedisStore, error) {
	if !cfg.Enabled {
		return &RedisStore{cfg: cfg}, nil
	}
	if strings.TrimSpace(cfg.Addr) == "" || cfg.Timeout <= 0 {
		return nil, ErrInvalidConfig
	}
	if err := cfg.KeyPolicy.validatePrefix(); err != nil {
		return nil, err
	}
	return &RedisStore{cfg: cfg}, nil
}

func (s *RedisStore) SetCache(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if err := s.ensureEnabled(); err != nil {
		return err
	}
	if err := s.cfg.KeyPolicy.RequireTTL(KeyKindCache, ttl); err != nil {
		return err
	}
	if err := validateFullyQualifiedKey(s.cfg.KeyPolicy, key); err != nil {
		return err
	}
	if err := validateEphemeralValueBytes(value); err != nil {
		return err
	}
	_, err := s.command(ctx, "SET", key, string(value), "PX", strconv.FormatInt(ttl.Milliseconds(), 10))
	return err
}

func (s *RedisStore) GetCache(ctx context.Context, key string) ([]byte, bool, error) {
	if err := s.ensureEnabled(); err != nil {
		return nil, false, err
	}
	if err := validateFullyQualifiedKey(s.cfg.KeyPolicy, key); err != nil {
		return nil, false, err
	}
	result, err := s.command(ctx, "GET", key)
	if err != nil {
		return nil, false, err
	}
	if result == nil {
		return nil, false, nil
	}
	value, ok := result.(string)
	if !ok {
		return nil, false, ErrUnexpectedRedis
	}
	return []byte(value), true, nil
}

func (s *RedisStore) PushQueue(ctx context.Context, key string, value []byte) error {
	if err := s.ensureEnabled(); err != nil {
		return err
	}
	if err := validateFullyQualifiedKey(s.cfg.KeyPolicy, key); err != nil {
		return err
	}
	if err := validateEphemeralValueBytes(value); err != nil {
		return err
	}
	_, err := s.command(ctx, "RPUSH", key, string(value))
	return err
}

func (s *RedisStore) PopQueue(ctx context.Context, key string) ([]byte, bool, error) {
	if err := s.ensureEnabled(); err != nil {
		return nil, false, err
	}
	if err := validateFullyQualifiedKey(s.cfg.KeyPolicy, key); err != nil {
		return nil, false, err
	}
	result, err := s.command(ctx, "LPOP", key)
	if err != nil {
		return nil, false, err
	}
	if result == nil {
		return nil, false, nil
	}
	value, ok := result.(string)
	if !ok {
		return nil, false, ErrUnexpectedRedis
	}
	return []byte(value), true, nil
}

func (s *RedisStore) AcquireLock(ctx context.Context, key string, owner string, ttl time.Duration) (bool, error) {
	if err := s.ensureEnabled(); err != nil {
		return false, err
	}
	if err := s.cfg.KeyPolicy.RequireTTL(KeyKindLock, ttl); err != nil {
		return false, err
	}
	if err := validateFullyQualifiedKey(s.cfg.KeyPolicy, key); err != nil {
		return false, err
	}
	if err := validateEphemeralValueString(owner); err != nil {
		return false, err
	}
	result, err := s.command(ctx, "SET", key, owner, "NX", "PX", strconv.FormatInt(ttl.Milliseconds(), 10))
	if err != nil {
		return false, err
	}
	return result == "OK", nil
}

func (s *RedisStore) ReleaseLock(ctx context.Context, key string, owner string) error {
	if err := s.ensureEnabled(); err != nil {
		return err
	}
	if err := validateFullyQualifiedKey(s.cfg.KeyPolicy, key); err != nil {
		return err
	}
	result, err := s.command(ctx, "GET", key)
	if err != nil {
		return err
	}
	if result == nil {
		return ErrLockNotHeld
	}
	if result != owner {
		return ErrLockHeld
	}
	_, err = s.command(ctx, "DEL", key)
	return err
}

func (s *RedisStore) AppendProgress(ctx context.Context, key string, entry ProgressEntry, ttl time.Duration) error {
	if err := s.ensureEnabled(); err != nil {
		return err
	}
	if err := s.cfg.KeyPolicy.RequireTTL(KeyKindProgress, ttl); err != nil {
		return err
	}
	if err := validateFullyQualifiedKey(s.cfg.KeyPolicy, key); err != nil {
		return err
	}
	if err := validateProgressEntryValue(entry); err != nil {
		return err
	}
	if entry.ID == "" {
		entry.ID = StableOpaqueSegment(key, entry.Message, time.Now().UTC().Format(time.RFC3339Nano))
	}
	value := entry.ID + "|" + strings.ReplaceAll(entry.Message, "\n", " ")
	if err := validateEphemeralValueString(value); err != nil {
		return err
	}
	if _, err := s.command(ctx, "RPUSH", key, value); err != nil {
		return err
	}
	_, err := s.command(ctx, "PEXPIRE", key, strconv.FormatInt(ttl.Milliseconds(), 10))
	return err
}

func (s *RedisStore) ReadProgress(ctx context.Context, key string, limit int) ([]ProgressEntry, error) {
	if err := s.ensureEnabled(); err != nil {
		return nil, err
	}
	if err := validateFullyQualifiedKey(s.cfg.KeyPolicy, key); err != nil {
		return nil, err
	}
	limit = normalizeProgressReadLimit(limit)
	start := int64(-limit)
	result, err := s.command(ctx, "LRANGE", key, strconv.FormatInt(start, 10), "-1")
	if err != nil {
		return nil, err
	}
	values, ok := result.([]any)
	if !ok {
		return nil, ErrUnexpectedRedis
	}
	entries := make([]ProgressEntry, 0, len(values))
	for _, value := range values {
		text, ok := value.(string)
		if !ok {
			return nil, ErrUnexpectedRedis
		}
		id, message, _ := strings.Cut(text, "|")
		entries = append(entries, ProgressEntry{ID: id, Message: message})
	}
	return entries, nil
}

func (s *RedisStore) Publish(ctx context.Context, channel string, value []byte) error {
	if err := s.ensureEnabled(); err != nil {
		return err
	}
	if err := validateFullyQualifiedKey(s.cfg.KeyPolicy, channel); err != nil {
		return err
	}
	if err := validateEphemeralValueBytes(value); err != nil {
		return err
	}
	_, err := s.command(ctx, "PUBLISH", channel, string(value))
	return err
}

func (s *RedisStore) Health(ctx context.Context) HealthStatus {
	if !s.cfg.Enabled {
		return HealthStatus{Enabled: false, OK: true, Message: "redis disabled"}
	}
	result, err := s.command(ctx, "PING")
	if err != nil {
		return HealthStatus{Enabled: true, OK: false, Message: err.Error()}
	}
	return HealthStatus{Enabled: true, OK: result == "PONG", Message: "redis ping complete"}
}

func (s *RedisStore) ensureEnabled() error {
	if !s.cfg.Enabled {
		return ErrDisabled
	}
	return nil
}

func (s *RedisStore) command(ctx context.Context, args ...string) (any, error) {
	timeout := s.cfg.Timeout
	if timeout <= 0 {
		timeout = time.Second
	}
	dialer := net.Dialer{Timeout: timeout}
	conn, err := dialer.DialContext(ctx, "tcp", s.cfg.Addr)
	if err != nil {
		return nil, errors.Join(ErrRedisUnavailable, err)
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(timeout))
	if err := writeRESP(conn, args...); err != nil {
		return nil, err
	}
	return readRESP(bufio.NewReader(conn))
}

func writeRESP(w io.Writer, args ...string) error {
	for _, arg := range args {
		if err := validateEphemeralValueString(arg); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintf(w, "*%d\r\n", len(args)); err != nil {
		return err
	}
	for _, arg := range args {
		if _, err := fmt.Fprintf(w, "$%d\r\n", len(arg)); err != nil {
			return err
		}
		if _, err := io.WriteString(w, arg); err != nil {
			return err
		}
		if _, err := io.WriteString(w, "\r\n"); err != nil {
			return err
		}
	}
	return nil
}

func readRESP(r *bufio.Reader) (any, error) {
	prefix, err := r.ReadByte()
	if err != nil {
		return nil, err
	}
	switch prefix {
	case '+':
		line, err := readLine(r)
		return line, err
	case '-':
		line, _ := readLine(r)
		return nil, fmt.Errorf("%w: %s", ErrUnexpectedRedis, line)
	case ':':
		line, err := readLine(r)
		if err != nil {
			return nil, err
		}
		return strconv.ParseInt(line, 10, 64)
	case '$':
		line, err := readLine(r)
		if err != nil {
			return nil, err
		}
		size, err := strconv.Atoi(line)
		if err != nil {
			return nil, err
		}
		if size < 0 {
			return nil, nil
		}
		if size > maxRedisRESPBulkBytes {
			return nil, fmt.Errorf("redis bulk response too large: %d > %d", size, maxRedisRESPBulkBytes)
		}
		buf := make([]byte, size+2)
		if _, err := io.ReadFull(r, buf); err != nil {
			return nil, err
		}
		return string(buf[:size]), nil
	case '*':
		line, err := readLine(r)
		if err != nil {
			return nil, err
		}
		count, err := strconv.Atoi(line)
		if err != nil {
			return nil, err
		}
		if count < 0 {
			return nil, fmt.Errorf("invalid redis array response size %d", count)
		}
		if count > maxRedisRESPArrayItems {
			return nil, fmt.Errorf("redis array response too large: %d > %d", count, maxRedisRESPArrayItems)
		}
		values := make([]any, 0, count)
		for i := 0; i < count; i++ {
			value, err := readRESP(r)
			if err != nil {
				return nil, err
			}
			values = append(values, value)
		}
		return values, nil
	default:
		return nil, ErrUnexpectedRedis
	}
}

func readLine(r *bufio.Reader) (string, error) {
	var buf strings.Builder
	for {
		chunk, err := r.ReadSlice('\n')
		if len(chunk) > 0 {
			if buf.Len()+len(chunk) > maxRedisRESPLineBytes {
				return "", fmt.Errorf("redis line response too large: > %d bytes", maxRedisRESPLineBytes)
			}
			_, _ = buf.Write(chunk)
		}
		if err == nil {
			break
		}
		if errors.Is(err, bufio.ErrBufferFull) {
			continue
		}
		return "", err
	}
	line := buf.String()
	return strings.TrimSuffix(strings.TrimSuffix(line, "\n"), "\r"), nil
}

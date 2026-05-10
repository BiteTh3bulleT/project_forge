package ephemeral

import "errors"

var (
	ErrDisabled          = errors.New("ephemeral store disabled")
	ErrInvalidConfig     = errors.New("invalid ephemeral store config")
	ErrInvalidRole       = errors.New("invalid ephemeral role")
	ErrForbiddenRole     = errors.New("forbidden redis role")
	ErrUnsafeKey         = errors.New("unsafe ephemeral key")
	ErrTTLRequired       = errors.New("ttl required for ephemeral operation")
	ErrLockHeld          = errors.New("ephemeral lock already held")
	ErrLockNotHeld       = errors.New("ephemeral lock not held")
	ErrQueueEmpty        = errors.New("ephemeral queue empty")
	ErrValueTooLarge     = errors.New("ephemeral value too large")
	ErrUnexpectedRedis   = errors.New("unexpected redis response")
	ErrRedisUnavailable  = errors.New("redis unavailable")
	ErrCanonicalMisuse   = errors.New("redis cannot be canonical truth")
	ErrRecoverableByLoss = errors.New("redis loss must be recoverable from durable state")
)

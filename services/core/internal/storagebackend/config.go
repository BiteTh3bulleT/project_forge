package storagebackend

import "strings"

type ConfigInput struct {
	Backend     string
	PostgresDSN string
	RedisAddr   string
	QdrantURL   string
}

type Config struct {
	Kind         BackendKind
	PostgresDSN  string
	RedisAddr    string
	QdrantURL    string
	Capabilities BackendCapabilities
}

func NewConfig(input ConfigInput) (Config, error) {
	kind, err := ParseBackendKind(input.Backend)
	if err != nil {
		return Config{}, err
	}

	postgresDSN := strings.TrimSpace(input.PostgresDSN)
	if kind == BackendPostgres && postgresDSN == "" {
		return Config{}, ErrPostgresDSNRequired
	}

	return Config{
		Kind:         kind,
		PostgresDSN:  postgresDSN,
		RedisAddr:    strings.TrimSpace(input.RedisAddr),
		QdrantURL:    strings.TrimSpace(input.QdrantURL),
		Capabilities: CapabilitiesFor(kind),
	}, nil
}

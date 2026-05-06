package storagebackend

import "strings"

type BackendKind string

const (
	BackendSQLite   BackendKind = "sqlite"
	BackendPostgres BackendKind = "postgres"
)

func ParseBackendKind(raw string) (BackendKind, error) {
	switch BackendKind(strings.ToLower(strings.TrimSpace(raw))) {
	case "", BackendSQLite:
		return BackendSQLite, nil
	case BackendPostgres:
		return BackendPostgres, nil
	default:
		return "", ErrInvalidBackend
	}
}

func (k BackendKind) String() string {
	return string(k)
}

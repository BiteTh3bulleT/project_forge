package contextcompiler

import "errors"

var (
	ErrInvalidContextBlock   = errors.New("invalid context block")
	ErrInvalidBlockType      = errors.New("invalid context block type")
	ErrInvalidCompileRequest = errors.New("invalid context compile request")
	ErrInvalidPromptLayout   = errors.New("invalid prompt layout")
	ErrContextBundleNotFound = errors.New("context bundle not found")
	ErrContextBlockNotFound  = errors.New("context block not found")
	ErrWorkspaceMismatch     = errors.New("workspace mismatch")
)

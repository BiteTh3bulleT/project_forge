package forgekshadow

import "errors"

var (
	ErrUnsafeMetadata             = errors.New("unsafe shadow metadata")
	ErrPolicyRejected             = errors.New("shadow policy rejected")
	ErrDiagnosticPayloadTooLarge  = errors.New("shadow diagnostic payload too large")
	ErrDiagnosticPersistenceInput = errors.New("invalid shadow diagnostic persistence input")
)

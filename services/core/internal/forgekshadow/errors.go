package forgekshadow

import "errors"

var (
	ErrUnsafeMetadata = errors.New("unsafe shadow metadata")
	ErrPolicyRejected = errors.New("shadow policy rejected")
)

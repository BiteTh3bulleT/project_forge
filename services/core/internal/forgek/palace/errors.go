package palace

import "errors"

var (
	ErrInvalidRoom      = errors.New("invalid memory room")
	ErrInvalidAnchor    = errors.New("invalid memory anchor")
	ErrInvalidRoute     = errors.New("invalid palace route")
	ErrInvalidCandidate = errors.New("invalid candidate object")
	ErrRoomNotFound     = errors.New("memory room not found")
	ErrAnchorNotFound   = errors.New("memory anchor not found")
	ErrRouteNotFound    = errors.New("palace route not found")
)

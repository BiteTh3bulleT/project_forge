package court

import "errors"

var (
	ErrInvalidExhibit       = errors.New("invalid exhibit")
	ErrInvalidRuling        = errors.New("invalid ruling")
	ErrInvalidContradiction = errors.New("invalid contradiction")
	ErrInvalidSupersession  = errors.New("invalid supersession")
	ErrExhibitNotFound      = errors.New("exhibit not found")
)

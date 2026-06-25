package validator

import (
	"errors"
)

var (
	ErrRequestEmpty = errors.New("empty request not allowed")
	ErrEmptyString  = errors.New("empty string")
)

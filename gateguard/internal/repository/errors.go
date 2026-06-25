package repository

import (
	"errors"

	"github.com/go-pg/pg/v10"
)

var (
	ErrMultipleRows       = pg.ErrMultiRows
	ErrTxDone             = pg.ErrTxDone
	ErrRowAlreadyExists   = errors.New("row already exists")
	ErrUserNotFound       = errors.New("user not found")
	ErrInvitationNotFound = errors.New("invitation not found")
)

package repository

import "errors"

var (
	ErrNotFound            = errors.New("data not found")
	ErrInvalidData         = errors.New("invalid data provided")
	ErrDBNotImplemented    = errors.New("database mongodb belum diimplementasikan")
)

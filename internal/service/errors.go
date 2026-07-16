package service

import "errors"

var (
	ErrEmptyPath = errors.New("filepath is empty")
	ErrNotExist  = errors.New("not exist")
	ErrNotDir    = errors.New("not directory")
	TimeFormat   = "2006-01-02T15-04-05"
)

package main

import "errors"

var (
	ErrInvalidSubCommand = errors.New("invalid sub command")
	ErrMissingArgument = errors.New("one or more missing arguments")
)
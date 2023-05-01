package main

import "errors"

var (
	ErrInvalidSubCommand     = errors.New("invalid sub command")
	ErrMissingArgument       = errors.New("one or more missing arguments")
	ErrReportAlreadyImported = errors.New("report already imported")
	ErrAlreadyInQueue        = errors.New("report is already in queue")
)

package main

import "errors"

var (
	ErrReportAlreadyImported = errors.New("report already imported")
	ErrAlreadyInQueue        = errors.New("report is already in queue")
)

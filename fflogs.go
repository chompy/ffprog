package main

import (
	"github.com/RyuaNerin/go-fflogs"
)

type FFLogsHandler struct {
	client *fflogs.Client
}

func NewFFLogsHandler(config *Config) (*FFLogsHandler, error) {
	opts := fflogs.ClientOpt{
		ApiKey: config.ApiKey,
	}

	client, err := fflogs.NewClient(&opts)
	if err != nil {
		return nil, err
	}

	return &FFLogsHandler{
		client: client,
	}, nil
}

package main

import (
	"encoding/json"
	"os"
)

const configFilePath = "config.json"
const appName = "FFProg"
const appVersion = "0.0.1"

type Config struct {
	FFLogsApiKey string `json:"fflogsApiKey"`
	DatabaseFile string `json:"databaseFile"`
}

func LoadConfig() (Config, error) {
	config := Config{}
	rawConfigData, err := os.Open(configFilePath)
	if err != nil {
		return config, err
	}
	jsonParser := json.NewDecoder(rawConfigData)
	if err = jsonParser.Decode(&config); err != nil {
		return config, err
	}
	return config, nil
}

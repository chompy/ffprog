package main

import "os"
import "encoding/json"

const configFilePath = "config.json"

type Config struct {
	ApiKey string `json:"apiKey"`
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
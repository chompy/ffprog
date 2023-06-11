package main

import (
	"encoding/json"

	"muzzammil.xyz/jsonc"
)

const configFilePath = "config.json"
const appName = "FFProg"
const appVersion = "0.0.1"

type DisplayEncounterCategory struct {
	Name    string `json:"category"`
	BossIDs []int  `json:"boss_ids"`
}

type Config struct {
	FFLogsApiKey        string                     `json:"fflogs_api_key"`
	DatabaseFile        string                     `json:"database_file"`
	DisplayedEncounters []DisplayEncounterCategory `json:"displayed_encounters"`
	HTTPPort            int                        `json:"http_port"`
}

func LoadConfig() (Config, error) {
	config := Config{}
	_, rawConfigData, err := jsonc.ReadFromFile(configFilePath)
	if err != nil {
		return config, err
	}
	if err := json.Unmarshal(rawConfigData, &config); err != nil {
		return config, err
	}
	return config, nil
}

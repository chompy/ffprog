package main

import (
	"encoding/json"
	"os"
)

const dcServerMapJson = "data/dc_servers.json"
const dcRegionMapJson = "data/dc_regions.json"

var dcServerMap = map[string][]string{}
var dcRegionMap = map[string]string{}

func fetchDCServerMap() error {
	dcServerMap = make(map[string][]string)
	rawData, err := os.ReadFile(dcServerMapJson)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(rawData, &dcServerMap); err != nil {
		return err
	}
	return nil
}

func fetchDCRegionMap() error {
	dcRegionMap = make(map[string]string)
	rawData, err := os.ReadFile(dcRegionMapJson)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(rawData, &dcRegionMap); err != nil {
		return err
	}
	return nil
}

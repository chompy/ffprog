package main

import "log"

func main() {

	// load global config
	log.Println("Load config JSON.")
	config, err := LoadConfig()
	if err != nil {
		log.Panic(err)
	}

	// fetch mappings data
	log.Println("Load data mappings.")
	if err := fetchDCRegionMap(); err != nil {
		log.Panic(err)
	}
	if err := fetchDCServerMap(); err != nil {
		log.Panic(err)
	}

	// start web server
	log.Println("Start web server.")
	if err := StartWeb(&config); err != nil {
		log.Panic(err)
	}

}

package main

import "fmt"
import "os"

func main() {
    
	// load global config
	config, err := LoadConfig()
	if err != nil {
		panic(err)
	}
	
	// parse subcommand
	subcommandName := "web"
	if len(os.Args) > 1 {
		subcommandName = os.Args[1]
	}

	// handle subcommand
	switch subcommandName {
	case "web": {
		fmt.Println("WEB NOT YET IMPLEMENTED.")
		return
	}
	case "import": {

		// parse import args
		if len(os.Args) < 3 {
			panic(ErrMissingArgument)
		}
		reportId := os.Args[2]

		ffl, err := NewFFLogsImporter(&config)
		if err != nil {
			panic(err)
		}

		if err := ffl.ImportReport(reportId); err != nil {
			panic(err)
		}

		return
	}
	case "character": {
		return
	}
	}

	panic(ErrInvalidSubCommand)





}
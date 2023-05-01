package main

import (
	"fmt"
	"os"
	"time"
)

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
	case "web":
		{
			if err := StartWeb(&config); err != nil {
				panic(err)
			}
			return
		}
	case "import":
		{

			// parse import args
			if len(os.Args) < 3 {
				panic(ErrMissingArgument)
			}
			reportID := FFLogReportURLToReportID(os.Args[2])

			ffl, err := NewFFLogsHandler(&config)
			if err != nil {
				panic(err)
			}

			db, err := NewDatabaserHandler(&config)
			if err != nil {
				panic(err)
			}

			if db.HasFFLogsReport(reportID) {
				panic(ErrReportAlreadyImported)
			}

			reportFights, err := ffl.FetchReportFights(reportID)
			if err != nil {
				panic(err)
			}

			if err := db.HandleFFLogsReportFights(reportID, reportFights); err != nil {
				panic(err)
			}
			return

		}
	case "search":
		{

			// parse import args
			if len(os.Args) < 3 {
				panic(ErrMissingArgument)
			}

			name := os.Args[2]

			db, err := NewDatabaserHandler(&config)
			if err != nil {
				panic(err)
			}

			characters, err := db.FindCharacters(name)
			if err != nil {
				panic(err)
			}

			for _, character := range characters {
				fmt.Printf("%s @ %s (%s)\n", character.Name, character.Server, character.UUID)
			}

			return

		}
	case "character":
		{
			// parse import args
			if len(os.Args) < 3 {
				panic(ErrMissingArgument)
			}

			uuid := os.Args[2]

			db, err := NewDatabaserHandler(&config)
			if err != nil {
				panic(err)
			}

			character, err := db.FetchCharacterFromUUID(uuid)
			if err != nil {
				panic(err)
			}

			fmt.Printf("%s @ %s (%s)\n\n", character.Name, character.Server, character.UUID)
			for _, progression := range character.Progression {

				statusText := "CLEARED"
				if !progression.HasKill {
					percent := progression.BestPhasePercentage / 100
					statusText = fmt.Sprintf("P%d %d%%", progression.BestPhase, percent)
				}

				fmt.Printf("%s :: %s (%s)\n", progression.EncounterInfo.ZoneName, statusText, progression.LastProgressionTime.Format(time.RFC3339))
			}

			return
		}
	}

	panic(ErrInvalidSubCommand)

}

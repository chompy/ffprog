package main

import "github.com/RyuaNerin/go-fflogs"
import "context"
import "fmt"

type FFLogsImporter struct {
	client *fflogs.Client
}

func NewFFLogsImporter(config *Config) (*FFLogsImporter, error) {
	opts := fflogs.ClientOpt{
		ApiKey: config.ApiKey,
	}

	client, err := fflogs.NewClient(&opts)
	if err != nil {
		return nil, err
	}

	return &FFLogsImporter{
		client: client,
	}, nil
}

func (ffl FFLogsImporter) ImportReport(code string) error {
	reportOpts := fflogs.ReportFightsOptions{
		Code: code,
	}
	fights, err := ffl.client.ReportFights(context.Background(), &reportOpts)
	if err != nil {
		return err
	}

	// TODO import to database here
	/*for _, f := range fights.Fights {
	
		fmt.Printf("%s (%d) - %d", f.ZoneName, f.ID, *f.FightPercentage)
		fmt.Println(f)

	}*/

	fmt.Println(fights.Friendlies)

	for _, d := range fights.Friendlies {
		fmt.Println(d.Name, d.Server, d.Type, d.Icon)
	}

	return nil
}
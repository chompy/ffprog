package main

import (
	"context"

	"github.com/RyuaNerin/go-fflogs"
	"github.com/RyuaNerin/go-fflogs/structure"
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

func (ffl FFLogsHandler) FetchReportFights(code string) (*structure.Fights, error) {
	reportOpts := fflogs.ReportFightsOptions{
		Code: code,
	}
	return ffl.client.ReportFights(context.Background(), &reportOpts)
	// TODO import to database here
	/*for _, f := range fights.Fights {

		fmt.Printf("%s (%d) - %d %d %d\n", f.ZoneName, f.ID, *f.FightPercentage, *f.LastPhaseForPercentageDisplay, *f.BossPercentage)

	}*/

	/*fmt.Println(fights.Friendlies)

	for _, d := range fights.Friendlies {
		fmt.Println(d.Name, d.Server, d.Type, d.Icon)
	}*/
}

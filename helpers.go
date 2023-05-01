package main

import (
	"crypto/sha256"
	"fmt"
	"regexp"
	"strings"

	"github.com/RyuaNerin/go-fflogs/structure"
	"github.com/martinlindhe/base36"
	gonanoid "github.com/matoous/go-nanoid"
)

var fflogReportUrlRegex = regexp.MustCompile(`https?\:\/\/www\.fflogs\.com\/reports\/([A-Za-z1-9]*)`)

var JobsMap = map[string]string{
	"whitemage":   "whm",
	"scholar":     "sch",
	"astrologian": "ast",
	"sage":        "sge",
	"darkknight":  "drk",
	"paladin":     "pld",
	"gunbreaker":  "gnb",
	"warrior":     "war",
	"redmage":     "rdm",
	"blackmage":   "blm",
	"summoner":    "smn",
	"bard":        "brd",
	"machinist":   "mch",
	"dancer":      "dnc",
	"monk":        "mnk",
	"samurai":     "sam",
	"dragoon":     "drg",
	"reaper":      "rpr",
}

func GenerateUUID() string {
	id, err := gonanoid.Nanoid()
	if err != nil {
		panic(err)
	}
	return id
}

func JobNameToJobAbbr(name string) string {
	return JobsMap[strings.ToLower(name)]
}

func FFLogsEncounterInfoHash(fflFight *structure.FightsFight) string {
	hashBytes := sha256.Sum256([]byte(fmt.Sprintf("%d", fflFight.ZoneID)))
	return base36.EncodeBytes(hashBytes[:])
}

func FFLogsCharacterHash(fightsFriendly *structure.FightsFriendly) string {
	hashBytes := sha256.Sum256([]byte(fightsFriendly.Name + fightsFriendly.Server))
	return base36.EncodeBytes(hashBytes[:])
}

func FFLogReportURLToReportID(reportURL string) string {
	results := fflogReportUrlRegex.FindAllStringSubmatch(reportURL, -1)
	if len(results) == 0 || len(results[0]) < 2 {
		return reportURL
	}
	return results[0][1]
}

func IsFFLogsEncounterValid(fflFight *structure.FightsFight) bool {
	return fflFight.Difficulty != nil && *fflFight.Difficulty != 0
}

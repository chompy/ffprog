package main

import (
	"crypto/sha256"
	"fmt"
	"strings"

	"github.com/RyuaNerin/go-fflogs/structure"
)

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

func JobNameToJobAbbr(name string) string {
	return JobsMap[strings.ToLower(name)]
}

func CharacterUID(fightsFriendly *structure.FightsFriendly) string {
	hashBytes := sha256.Sum256([]byte(fightsFriendly.Name + fightsFriendly.Server))
	return fmt.Sprintf("%x", hashBytes)
}

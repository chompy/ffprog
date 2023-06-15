package main

import (
	"crypto/sha256"
	"fmt"
	"math/rand"
	"net/http"
	"regexp"
	"strings"

	"github.com/RyuaNerin/go-fflogs/structure"
	"github.com/martinlindhe/base36"
)

var fflogReportUrlRegex = regexp.MustCompile(`https?\:\/\/www\.fflogs\.com\/reports\/([A-Za-z1-9]*)`)

var JobsMap = map[string]string{}

const uuidLength = 6

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyz1234567890")

func GenerateUID() string {
	b := make([]rune, uuidLength)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
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
		if len(reportURL) == 16 {
			return reportURL
		}
		return ""
	}
	return results[0][1]
}

func IsFFLogsEncounterValid(fflFight *structure.FightsFight) bool {
	return fflFight.HasEcho != nil && !*fflFight.HasEcho && fflFight.Difficulty != nil && *fflFight.Difficulty != 0 && fflFight.BossPercentage != nil && fflFight.FightPercentage != nil && fflFight.Kill != nil
}

func ReadUserIP(r *http.Request) string {
	IPAddress := r.Header.Get("X-Real-Ip")
	if IPAddress == "" {
		IPAddress = r.Header.Get("X-Forwarded-For")
	}
	if IPAddress == "" {
		IPAddress = r.RemoteAddr
	}
	return IPAddress
}

func EncounterDisplayListFromEncounterInfoList(list []EncounterInfo, config *Config) []displayEncounterData {
	out := make([]displayEncounterData, 0)
	for _, displayEncounterInfo := range config.DisplayedEncounters {
		displayData := displayEncounterData{
			Category:   displayEncounterInfo.Name,
			Encounters: make([]EncounterInfo, 0),
		}
		for _, bossID := range displayEncounterInfo.BossIDs {
			for _, encounter := range list {
				if encounter.BossID == int64(bossID) {
					displayData.Encounters = append(displayData.Encounters, encounter)
					break
				}
			}
		}
		if len(displayData.Encounters) > 0 {
			out = append(out, displayData)
		}
	}
	return out
}

func GetServerRegion(serverName string) string {
	for datacenter, serverList := range dcServerMap {
		for _, serverListName := range serverList {
			if strings.EqualFold(serverName, serverListName) {
				return dcRegionMap[datacenter]
			}
		}
	}
	return "na"
}

func FFLogsCharacterURL(character Character) string {
	return fmt.Sprintf(
		"https://www.fflogs.com/character/%s/%s/%s",
		GetServerRegion(character.Server),
		strings.ToLower(character.Server),
		strings.ToLower(character.Name))
}

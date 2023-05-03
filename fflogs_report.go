package main

import (
	"context"
	"time"

	"github.com/RyuaNerin/go-fflogs"
	"github.com/RyuaNerin/go-fflogs/structure"
)

// FFLogCharacterReport contains information about a character's best encounters in a report.
type FFLogCharacterReport struct {
	ReportID    string
	Character   Character
	Progression []CharacterProgression
}

func (ffl FFLogsHandler) rawFetchReportFights(reportID string) (*structure.Fights, error) {
	reportOpts := fflogs.ReportFightsOptions{
		Code: reportID,
	}
	return ffl.client.ReportFights(context.Background(), &reportOpts)
}

func isFFLogsFriendlyInEncounter(fflFightsFriendly *structure.FightsFriendly, fflFight *structure.FightsFight) bool {
	for _, fflFriendlyFight := range fflFightsFriendly.Fights {
		if fflFriendlyFight.ID == fflFight.ID {
			return true
		}
	}
	return false
}

func getFFLogsFightsEncounterHashesForFriendly(fflFights *structure.Fights, fflFightsFriendly *structure.FightsFriendly) []string {
	out := make([]string, 0)
	for _, fflFight := range fflFights.Fights {
		if !isFFLogsFriendlyInEncounter(fflFightsFriendly, &fflFight) {
			continue
		}
		compareHash := FFLogsEncounterInfoHash(&fflFight)
		hasHash := false
		for _, savedHash := range out {
			if compareHash == savedHash {
				hasHash = true
				break
			}
		}
		if !hasHash {
			out = append(out, compareHash)
		}
	}
	return out
}

func doesFFLogFriendlyHaveEncounter(fflFights *structure.Fights, fflFightsFriendly *structure.FightsFriendly, encounterHash string) bool {
	for _, fflFight := range fflFights.Fights {
		if IsFFLogsEncounterValid(&fflFight) && isFFLogsFriendlyInEncounter(fflFightsFriendly, &fflFight) && FFLogsEncounterInfoHash(&fflFight) == encounterHash {
			return true
		}
	}
	return false
}

func doesFFLogFriendlyHaveEncounterKill(fflFights *structure.Fights, fflFightsFriendly *structure.FightsFriendly, encounterHash string) bool {
	for _, fflFight := range fflFights.Fights {
		if IsFFLogsEncounterValid(&fflFight) && isFFLogsFriendlyInEncounter(fflFightsFriendly, &fflFight) && FFLogsEncounterInfoHash(&fflFight) == encounterHash && *fflFight.Kill {
			return true
		}
	}
	return false
}

func getCharacterProgressForFFLogsFightsFriendly(reportID string, fflFights *structure.Fights, fflFightsFriendly *structure.FightsFriendly) (Character, []CharacterProgression) {

	// generate character
	character := Character{
		CompareHash: FFLogsCharacterHash(fflFightsFriendly),
		Name:        fflFightsFriendly.Name,
		Server:      fflFightsFriendly.Server,
	}
	// generate character progressions
	characterProgressions := make([]CharacterProgression, 0)
	encounterHashes := getFFLogsFightsEncounterHashesForFriendly(fflFights, fflFightsFriendly)

	for _, encounterHash := range encounterHashes {

		if !doesFFLogFriendlyHaveEncounter(fflFights, fflFightsFriendly, encounterHash) {
			continue
		}

		hasFightData := false
		hasKill := doesFFLogFriendlyHaveEncounterKill(fflFights, fflFightsFriendly, encounterHash)
		bestDuration := int64(-1)
		bestFightPercent := int64(-1)
		bestPhasePercent := int64(-1)
		bestPhase := int64(-1)
		bestEndTime := time.Time{}
		bestStandardComp := false
		bestZoneID := int64(-1)
		bestZoneName := ""
		bestDifficulty := int64(-1)
		bestBossID := int64(-1)

		for _, fflFight := range fflFights.Fights {
			// check if friendly has fight
			hasFight := false
			for _, fflFriendlyFight := range fflFightsFriendly.Fights {
				if fflFriendlyFight.ID == fflFight.ID {
					hasFight = true
					break
				}
			}
			if !hasFight {
				continue
			}
			// ignore if not valid, or has player has kill but this fight wasn't a kill
			if !IsFFLogsEncounterValid(&fflFight) || FFLogsEncounterInfoHash(&fflFight) != encounterHash || hasKill && !*fflFight.Kill {
				continue
			}

			encounterDuration := fflFight.EndTime - fflFight.StartTime
			if (hasKill && (encounterDuration < bestDuration || bestDuration == -1)) || (!hasKill && encounterDuration > bestDuration) {
				bestDuration = encounterDuration
			}
			if bestFightPercent < 0 || bestFightPercent > *fflFight.FightPercentage {
				hasFightData = true
				bestFightPercent = *fflFight.FightPercentage
				bestPhasePercent = *fflFight.BossPercentage
				bestPhase = *fflFight.LastPhaseForPercentageDisplay
				bestEndTime = time.UnixMilli(fflFights.Start + fflFight.EndTime)
				bestStandardComp = *fflFight.StandardComposition
				bestZoneID = fflFights.Zone
				bestZoneName = fflFight.ZoneName
				bestDifficulty = *fflFight.Difficulty
				bestBossID = fflFight.Boss
			}
		}
		if hasFightData {
			characterProgressions = append(characterProgressions, CharacterProgression{
				ReportID:              reportID,
				GameVersion:           fflFights.GameVersion,
				Time:                  bestEndTime,
				FightPercentage:       bestFightPercent,
				Phase:                 bestPhase,
				PhasePercentage:       bestPhasePercent,
				Duration:              bestDuration,
				IsKill:                hasKill,
				IsStandardComposition: bestStandardComp,
				HasEcho:               false,
				Job:                   fflFightsFriendly.Type,
				EncounterInfo: EncounterInfo{
					CompareHash: encounterHash,
					ZoneID:      bestZoneID,
					ZoneName:    bestZoneName,
					Difficulty:  bestDifficulty,
					BossID:      bestBossID,
				},
			})
		}
	}

	return character, characterProgressions

}

func (ffl FFLogsHandler) FetchCharacterReports(reportID string) ([]FFLogCharacterReport, error) {
	// fetch report
	fflFights, err := ffl.rawFetchReportFights(reportID)
	if err != nil {
		return nil, err
	}
	// generate character reports
	out := make([]FFLogCharacterReport, 0)
	for _, fflFightFriendly := range fflFights.Friendlies {
		if fflFightFriendly.Server == "" {
			continue
		}
		character, characterProgression := getCharacterProgressForFFLogsFightsFriendly(reportID, fflFights, &fflFightFriendly)
		if len(characterProgression) == 0 {
			continue
		}
		out = append(out, FFLogCharacterReport{
			ReportID:    reportID,
			Character:   character,
			Progression: characterProgression,
		})
	}
	return out, nil
}

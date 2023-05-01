package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/RyuaNerin/go-fflogs/structure"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type EncounterInfo struct {
	gorm.Model
	CompareHash string `json:"-"`
	BossID      int64  `json:"-"`
	ZoneID      int64  `json:"zone_id"`
	ZoneName    string `json:"zone_name"`
	Difficulty  int64  `json:"-"`
}

func (e EncounterInfo) IsDisplayable() bool {
	return strings.Contains(e.ZoneName, "Savage") || strings.Contains(e.ZoneName, "Ultimate") || strings.Contains(e.ZoneName, "Extreme")
}

type CharacterProgression struct {
	gorm.Model
	CharacterID            uint          `json:"-"`
	EncounterInfoID        uint          `json:"-"`
	EncounterInfo          EncounterInfo `json:"encounter"`
	GameVersion            int64         `json:"game_version"`
	FirstKillTime          time.Time     `json:"first_kill_time"`       // first time character killed the encounter
	LastProgressionTime    time.Time     `json:"last_progression_time"` // last time progress was made, includes getting a faster clear time
	BestFightPercentage    int64         `json:"best_fight_percentage"`
	BestPhase              int64         `json:"best_phase"`
	BestPhasePercentage    int64         `json:"best_phase_percentage"`
	BestEncounterTime      int64         `json:"best_encounter_duration"`
	HasKill                bool          `json:"has_kill"`
	HasEcho                bool          `json:"has_echo"`
	HasStandardComposition bool          `json:"has_standard_composition"`
}

func (cp CharacterProgression) IsImprovement(fflFight *structure.FightsFight) bool {
	// no record previously stored
	if cp.ID == 0 || fflFight.FightPercentage == nil || fflFight.Kill == nil {
		return true
	}
	encounterTime := fflFight.EndTime - fflFight.StartTime
	// if player is still progressing then an improvement is when the fight percent value is lower than the best fight percent
	// if player has a win then an improvement is the best clear time
	return (!cp.HasKill && !*fflFight.Kill && *fflFight.FightPercentage < cp.BestFightPercentage) || (*fflFight.Kill && (!cp.HasKill || encounterTime < cp.BestEncounterTime))
}

type Character struct {
	gorm.Model
	UUID        string                 `json:"uuid"`
	CompareHash string                 `json:"-"`
	Name        string                 `json:"name"`
	Server      string                 `json:"server"`
	Progression []CharacterProgression `json:"progression"`
}

type FFLogsReportImportHistory struct {
	gorm.Model
	ReportID string
}

type DatabaseHandler struct {
	Conn *gorm.DB
}

func NewDatabaserHandler(config *Config) (*DatabaseHandler, error) {
	db, err := gorm.Open(sqlite.Open(config.DatabaseFile), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	if err := db.AutoMigrate(&EncounterInfo{}); err != nil {
		return nil, err
	}
	if err := db.AutoMigrate(&CharacterProgression{}); err != nil {
		return nil, err
	}
	if err := db.AutoMigrate(&Character{}); err != nil {
		return nil, err
	}
	if err := db.AutoMigrate(&FFLogsReportImportHistory{}); err != nil {
		return nil, err
	}
	return &DatabaseHandler{
		Conn: db,
	}, nil
}

func (d DatabaseHandler) FetchEncounterInfoFromCompareHash(hash string) (EncounterInfo, error) {
	encounterInfo := EncounterInfo{}
	tx := d.Conn.First(&encounterInfo, "compare_hash = ?", hash)
	return encounterInfo, tx.Error
}

func (d DatabaseHandler) FetchCharacterFromUUID(uuid string) (Character, error) {
	character := Character{}
	tx := d.Conn.Preload("Progression").Preload("Progression.EncounterInfo").First(&character, "uuid = ?", uuid)
	return character, tx.Error
}

func (d DatabaseHandler) FetchCharacterFromCompareHash(hash string) (Character, error) {
	character := Character{}
	tx := d.Conn.First(&character, "compare_hash = ?", hash)
	return character, tx.Error
}

func (d DatabaseHandler) HasFFLogsReport(reportID string) bool {
	var count int64
	d.Conn.Model(&FFLogsReportImportHistory{}).Where("report_id = ?", reportID).Count(&count)
	return count > 0
}

func (d DatabaseHandler) syncEncounterInfoFromFFLogsReportFights(fflReportFights *structure.Fights) error {
	for _, fflFight := range fflReportFights.Fights {
		if !IsFFLogsEncounterValid(&fflFight) {
			continue
		}
		compareHash := FFLogsEncounterInfoHash(&fflFight)
		encounterInfo, err := d.FetchEncounterInfoFromCompareHash(compareHash)
		if err != nil {
			if err != gorm.ErrRecordNotFound {
				return err
			}
			// create if not exist
			encounterInfo.BossID = fflFight.Boss
			encounterInfo.ZoneID = fflFight.ZoneID
			encounterInfo.ZoneName = fflFight.ZoneName
			if fflFight.Difficulty != nil {
				encounterInfo.Difficulty = *fflFight.Difficulty
			}
			encounterInfo.CompareHash = compareHash
			if tx := d.Conn.Create(&encounterInfo); tx.Error != nil {
				return tx.Error
			}
		}
	}
	return nil
}

func (d DatabaseHandler) syncCharacterFromFFLogsReportFights(reportFights *structure.Fights) error {
	for _, fflCharacter := range reportFights.Friendlies {
		if fflCharacter.Name == "" || fflCharacter.Server == "" || fflCharacter.Type == "" {
			continue
		}
		compareHash := FFLogsCharacterHash(&fflCharacter)
		character, err := d.FetchCharacterFromCompareHash(compareHash)
		if err != nil {
			if err != gorm.ErrRecordNotFound {
				return err
			}
			// create if not exist
			character.Name = fflCharacter.Name
			character.Server = fflCharacter.Server
			character.CompareHash = compareHash
			character.UUID = GenerateUUID()
			if tx := d.Conn.Create(&character); tx.Error != nil {
				return tx.Error
			}
		}
	}
	return nil
}

func (d DatabaseHandler) syncCharacterProgressionFromFFLogsReportFights(reportID string, reportFights *structure.Fights) error {
	for _, fflFight := range reportFights.Fights {
		if !IsFFLogsEncounterValid(&fflFight) {
			continue
		}
		encounter, err := d.FetchEncounterInfoFromCompareHash(FFLogsEncounterInfoHash(&fflFight))
		if err != nil {
			return err
		}
		for _, fflCharacter := range reportFights.Friendlies {
			character, err := d.FetchCharacterFromCompareHash(FFLogsCharacterHash(&fflCharacter))
			if err != nil {
				if err == gorm.ErrRecordNotFound {
					continue
				}
				return err
			}
			characterProgression := CharacterProgression{}
			tx := d.Conn.First(&characterProgression, "encounter_info_id = ? AND character_id = ?", encounter.ID, character.ID)
			if tx.Error != nil && tx.Error != gorm.ErrRecordNotFound {
				return tx.Error
			}
			if characterProgression.IsImprovement(&fflFight) {
				encounterTime := fflFight.EndTime - fflFight.StartTime
				endTime := time.UnixMilli(reportFights.Start + fflFight.EndTime)
				characterProgression.BestEncounterTime = encounterTime
				characterProgression.BestFightPercentage = 10000
				if fflFight.FightPercentage != nil {
					characterProgression.BestFightPercentage = *fflFight.FightPercentage
				}
				characterProgression.BestPhasePercentage = 10000
				if fflFight.BossPercentage != nil {
					characterProgression.BestPhasePercentage = *fflFight.BossPercentage
				}
				characterProgression.BestPhase = 0
				if fflFight.LastPhaseForPercentageDisplay != nil {
					characterProgression.BestPhase = *fflFight.LastPhaseForPercentageDisplay
				}
				characterProgression.GameVersion = reportFights.GameVersion
				if fflFight.Kill != nil {
					if *fflFight.Kill && (!characterProgression.HasKill || characterProgression.FirstKillTime.After(endTime)) {
						characterProgression.FirstKillTime = endTime
						characterProgression.HasKill = true
					}
				}
				if fflFight.HasEcho != nil {
					characterProgression.HasEcho = *fflFight.HasEcho
				}
				if fflFight.StandardComposition != nil {
					characterProgression.HasStandardComposition = *fflFight.StandardComposition
				}
				characterProgression.CharacterID = character.ID
				characterProgression.EncounterInfoID = encounter.ID
				characterProgression.LastProgressionTime = endTime
				if tx := d.Conn.Save(&characterProgression); tx.Error != nil {
					return tx.Error
				}
			}

		}
	}
	return nil
}

func (d DatabaseHandler) HandleFFLogsReportFights(reportID string, reportFights *structure.Fights) error {
	if err := d.syncEncounterInfoFromFFLogsReportFights(reportFights); err != nil {
		return err
	}
	if err := d.syncCharacterFromFFLogsReportFights(reportFights); err != nil {
		return err
	}
	if err := d.syncCharacterProgressionFromFFLogsReportFights(reportID, reportFights); err != nil {
		return err
	}
	importHistory := FFLogsReportImportHistory{ReportID: reportID}
	if tx := d.Conn.Save(&importHistory); tx.Error != nil {
		return tx.Error
	}
	return nil
}

func (d DatabaseHandler) FindCharacters(name string) ([]Character, error) {
	characters := make([]Character, 0)
	tx := d.Conn.Where("name LIKE ?", fmt.Sprintf("%%%s%%", name)).Find(&characters)
	return characters, tx.Error
}

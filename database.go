package main

import (
	"fmt"

	"github.com/RyuaNerin/go-fflogs/structure"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type EncounterInfo struct {
	gorm.Model
	CompareHash string
	BossID      int64
	ZoneID      int64
	ZoneName    string
	Difficulty  int64
}

type CharacterProgression struct {
	gorm.Model
	CharacterID            uint
	EncounterInfoID        uint
	EncounterInfo          EncounterInfo
	GameVersion            int64
	BestFightPercentage    int64
	BestPhase              int64
	BestPhasePercentage    int64
	BestEncounterTime      int64
	HasKill                bool
	HasEcho                bool
	HasStandardComposition bool
}

func (cp CharacterProgression) IsImprovement(fflFight *structure.FightsFight) bool {
	// no record previously stored
	if cp.ID == 0 {
		return true
	}
	encounterTime := fflFight.EndTime - fflFight.StartTime
	// if player is still progressing then an improvement is when the fight percent value is lower than the best fight percent
	// if player has a win then an improvement is the best clear time
	return (!*fflFight.Kill && *fflFight.FightPercentage < cp.BestFightPercentage) || (*fflFight.Kill && (!cp.HasKill || encounterTime < cp.BestEncounterTime))
}

type Character struct {
	gorm.Model
	UUID        string
	CompareHash string
	Name        string
	Server      string
	Job         string
	Progression []CharacterProgression
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

func (d DatabaseHandler) syncEncounterInfoFromFFLogsReportFights(fflReportFights *structure.Fights) error {
	for _, fflFight := range fflReportFights.Fights {
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
			encounterInfo.Difficulty = *fflFight.Difficulty
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
			character.Job = JobNameToJobAbbr(fflCharacter.Type)
			character.CompareHash = compareHash
			character.UUID = GenerateUUID()
			if tx := d.Conn.Create(&character); tx.Error != nil {
				return tx.Error
			}
		}
	}
	return nil
}

func (d DatabaseHandler) syncCharacterProgressionFromFFLogsReportFights(reportFights *structure.Fights) error {
	for _, fflFight := range reportFights.Fights {
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
				characterProgression.BestEncounterTime = encounterTime
				characterProgression.BestFightPercentage = *fflFight.FightPercentage
				characterProgression.BestPhasePercentage = *fflFight.BossPercentage
				characterProgression.BestPhase = *fflFight.LastPhaseForPercentageDisplay
				characterProgression.GameVersion = reportFights.GameVersion
				characterProgression.HasKill = *fflFight.Kill
				characterProgression.HasEcho = *fflFight.HasEcho
				characterProgression.HasStandardComposition = *fflFight.StandardComposition
				characterProgression.CharacterID = character.ID
				characterProgression.EncounterInfoID = encounter.ID
				if tx := d.Conn.Save(&characterProgression); tx.Error != nil {
					return tx.Error
				}
			}

		}
	}
	return nil
}

func (d DatabaseHandler) HandleFFLogsReportFights(reportFights *structure.Fights) error {
	if err := d.syncEncounterInfoFromFFLogsReportFights(reportFights); err != nil {
		return err
	}
	if err := d.syncCharacterFromFFLogsReportFights(reportFights); err != nil {
		return err
	}
	if err := d.syncCharacterProgressionFromFFLogsReportFights(reportFights); err != nil {
		return err
	}
	return nil
}

func (d DatabaseHandler) FindCharacters(name string) ([]Character, error) {
	characters := make([]Character, 0)
	tx := d.Conn.Where("name LIKE ?", fmt.Sprintf("%%%s%%", name)).Find(&characters)
	return characters, tx.Error
}

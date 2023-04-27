package main

import (
	"github.com/RyuaNerin/go-fflogs/structure"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type EncounterInfo struct {
	gorm.Model
	BossID     int64
	ZoneID     int64
	ZoneName   string
	Difficulty int64
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

type Character struct {
	gorm.Model
	UID         string
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

func (d DatabaseHandler) FetchEncounterInfoFromBossID(bossID int64) (EncounterInfo, error) {
	encounterInfo := EncounterInfo{}
	tx := d.Conn.First(&encounterInfo, "boss_id = ?", bossID)
	return encounterInfo, tx.Error
}

func (d DatabaseHandler) FetchCharacterFromUID(uid string) (Character, error) {
	character := Character{}
	tx := d.Conn.First(&character, "uid = ?", uid)
	return character, tx.Error
}

func (d DatabaseHandler) syncEncounterInfoFromFFLogsReportFights(reportFights *structure.Fights) error {
	for _, fight := range reportFights.Fights {
		encounterInfo := EncounterInfo{}
		tx := d.Conn.First(&encounterInfo, "boss_id = ?", fight.Boss)
		if tx.Error != nil {
			if tx.Error != gorm.ErrRecordNotFound {
				return tx.Error
			}
			// create if not exist
			encounterInfo.BossID = fight.Boss
			encounterInfo.ZoneID = fight.ZoneID
			encounterInfo.ZoneName = fight.ZoneName
			encounterInfo.Difficulty = *fight.Difficulty
			tx = d.Conn.Create(&encounterInfo)
			if tx.Error != nil {
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
		uid := CharacterUID(&fflCharacter)
		characterDb := Character{}
		tx := d.Conn.First(&characterDb, "uid = ?", uid)
		if tx.Error != nil {
			if tx.Error != gorm.ErrRecordNotFound {
				return tx.Error
			}
			// create if not exist
			characterDb.Name = fflCharacter.Name
			characterDb.Server = fflCharacter.Server
			characterDb.Job = JobNameToJobAbbr(fflCharacter.Type)
			characterDb.UID = uid
			tx = d.Conn.Create(&characterDb)
			if tx.Error != nil {
				return tx.Error
			}
		}
	}
	return nil
}

func (d DatabaseHandler) syncCharacterProgressionFromFFLogsReportFights(reportFights *structure.Fights) error {
	for _, fflFight := range reportFights.Fights {
		encounter, err := d.FetchEncounterInfoFromBossID(fflFight.Boss)
		if err != nil {
			return err
		}

		for _, fflCharacter := range reportFights.Friendlies {
			uid := CharacterUID(&fflCharacter)
			character, err := d.FetchCharacterFromUID(uid)
			if err != nil {
				if err == gorm.ErrRecordNotFound {
					continue
				}
				return err
			}
			characterProgression := CharacterProgression{}
			tx := d.Conn.First(&characterProgression, "encounter_info_id = ? AND character_id", encounter.ID, character.ID)
			if tx.Error != nil && tx.Error != gorm.ErrRecordNotFound {
				return tx.Error
			}
			if *fflFight.FightPercentage < characterProgression.BestFightPercentage || characterProgression.ID == 0 {
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

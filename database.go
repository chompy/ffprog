package main

import (
	"fmt"
	"strings"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type EncounterInfo struct {
	gorm.Model
	CompareHash string `json:"-" gorm:"index:idx_encounter_info_compare_hash,unique"`
	BossID      int64  `json:"-"`
	ZoneID      int64  `json:"zone_id"`
	ZoneName    string `json:"zone_name" gorm:"index:idx_encounter_info_zone_name"`
	Difficulty  int64  `json:"-"`
}

func (e EncounterInfo) IsDisplayable() bool {
	return strings.Contains(e.ZoneName, "Savage") || strings.Contains(e.ZoneName, "Ultimate") || strings.Contains(e.ZoneName, "Extreme")
}

type CharacterProgression struct {
	gorm.Model
	CharacterID           uint          `json:"-" gorm:"index:idx_character_progression_character_id;index:idx_character_progression_character_id_encounter_info_id;index:idx_character_progression_character_id_report_id_encounter_info_id"`
	ReportID              string        `json:"report_id" gorm:"index:idx_character_progression_character_id_report_id_encounter_info_id"`
	EncounterInfoID       uint          `json:"-" gorm:"index:idx_character_progression_character_id_encounter_info_id;index:idx_character_progression_character_id_report_id_encounter_info_id"`
	EncounterInfo         EncounterInfo `json:"encounter"`
	GameVersion           int64         `json:"game_version"`
	Time                  time.Time     `json:"time"` // end time from fflogs
	FightPercentage       int64         `json:"fight_percentage"`
	Phase                 int64         `json:"phase"`
	PhasePercentage       int64         `json:"phase_percentage"`
	Duration              int64         `json:"duration"`
	IsKill                bool          `json:"is_kill"`
	IsStandardComposition bool          `json:"is_standard_composition"`
	HasEcho               bool          `json:"has_echo"`
	Job                   string        `json:"job"`
}

func (prevProg CharacterProgression) IsImprovement(newProg CharacterProgression) bool {
	return (!prevProg.IsKill && newProg.IsKill) || (prevProg.IsKill && newProg.IsKill && newProg.Duration < prevProg.Duration) || (!prevProg.IsKill && !newProg.IsKill && newProg.FightPercentage < prevProg.FightPercentage)
}

type Character struct {
	gorm.Model
	UID         string `json:"uid" gorm:"index:idx_character_uid,unique"`
	CompareHash string `json:"-" gorm:"index:idx_character_compare_hash,unique"`
	Name        string `json:"name"`
	Server      string `json:"server"`
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

func (d DatabaseHandler) FetchCharacterFromUID(uid string) (Character, error) {
	character := Character{}
	tx := d.Conn.First(&character, "uid = ?", uid)
	return character, tx.Error
}

func (d DatabaseHandler) FetchBestCharacterProgressionForEncounter(characterID uint, encounterID uint) (CharacterProgression, error) {
	results := CharacterProgression{}
	tx := d.Conn.Where("character_id = ? AND encounter_info_id = ?", characterID, encounterID).Order("is_kill desc, fight_percentage asc, time desc").Preload("EncounterInfo").First(&results)
	return results, tx.Error
}

func (d DatabaseHandler) FetchBestCharacterProgressions(characterID uint) ([]CharacterProgression, error) {
	results := make([]CharacterProgression, 0)
	tx := d.Conn.Where("character_id = ?", characterID).Order("is_kill desc, fight_percentage asc, time desc").Preload("EncounterInfo").Find(&results)
	if tx.Error != nil {
		return results, tx.Error
	}
	// TODO using DISTINCT query would be better but it wasn't working
	out := make([]CharacterProgression, 0)
	for _, resultItem := range results {
		hasUniqueEncounter := false
		for _, outItem := range out {
			if resultItem.EncounterInfoID == outItem.EncounterInfoID {
				hasUniqueEncounter = true
				break
			}
		}
		if !hasUniqueEncounter {
			out = append(out, resultItem)
		}

	}
	return out, nil
}

func (d DatabaseHandler) FetchCharacterProgressionForEncounter(characterID uint, encounterID uint) ([]CharacterProgression, error) {
	characterProgression := make([]CharacterProgression, 0)
	tx := d.Conn.Where("character_id = ? AND encounter_info_id = ?", characterID, encounterID).Order("fight_percentage asc, time desc").Find(&characterProgression)
	return characterProgression, tx.Error
}

func (d DatabaseHandler) FetchCharacterFromCompareHash(hash string) (Character, error) {
	character := Character{}
	tx := d.Conn.First(&character, "compare_hash = ?", hash)
	return character, tx.Error
}

func (d DatabaseHandler) FetchCharacterProgressionFromReportID(reportID string, characterID uint, encounterInfoID uint) (CharacterProgression, error) {
	characterProgression := CharacterProgression{}
	tx := d.Conn.First(&characterProgression, "report_id = ? AND character_id = ? AND encounter_info_id = ?", reportID, characterID, encounterInfoID)
	return characterProgression, tx.Error
}

func (d DatabaseHandler) FetchEncounterList() ([]EncounterInfo, error) {
	results := make([]EncounterInfo, 0)
	tx := d.Conn.Where(`zone_name LIKE '%Savage%' OR zone_name LIKE '%Ultimate%' OR zone_name LIKE '%Extreme%'`).Order("boss_id desc, zone_name asc").Find(&results)
	return results, tx.Error
}

func (d DatabaseHandler) HasFFLogsReport(reportID string) bool {
	var count int64
	d.Conn.Model(&CharacterProgression{}).Where("report_id = ?", reportID).Count(&count)
	return count > 0
}

func (d DatabaseHandler) syncCharacterFromFFLogCharacterReport(characterReport *FFLogCharacterReport) error {
	character, err := d.FetchCharacterFromCompareHash(characterReport.Character.CompareHash)
	if err != nil && err != gorm.ErrRecordNotFound {
		return err
	}
	if character.UID == "" {
		// generate uid, ensure no collision
		character.UID = GenerateUID()
		for {
			_, err = d.FetchCharacterFromUID(character.UID)
			if err != nil {
				if err == gorm.ErrRecordNotFound {
					break
				}
				return err
			}
			character.UID = GenerateUID()
		}
	}
	character.CompareHash = characterReport.Character.CompareHash
	character.Name = characterReport.Character.Name
	character.Server = characterReport.Character.Server
	if tx := d.Conn.Save(&character); tx.Error != nil {
		return tx.Error
	}
	characterReport.Character = character
	return nil
}

func (d DatabaseHandler) syncEncounterInfoFromFFLogCharacterReport(characterReport *FFLogCharacterReport) error {
	for i, characterProgression := range characterReport.Progression {
		encounterInfo, err := d.FetchEncounterInfoFromCompareHash(characterProgression.EncounterInfo.CompareHash)
		if err != nil && err != gorm.ErrRecordNotFound {
			return err
		}
		encounterInfo.CompareHash = characterProgression.EncounterInfo.CompareHash
		encounterInfo.ZoneID = characterProgression.EncounterInfo.ZoneID
		encounterInfo.ZoneName = characterProgression.EncounterInfo.ZoneName
		encounterInfo.Difficulty = characterProgression.EncounterInfo.Difficulty
		encounterInfo.BossID = characterProgression.EncounterInfo.BossID
		if tx := d.Conn.Save(&encounterInfo); tx.Error != nil {
			return tx.Error
		}
		characterReport.Progression[i].EncounterInfo = encounterInfo
	}
	return nil
}

func (d DatabaseHandler) syncCharacterProgressionsFromFFLogCharacterReport(characterReport *FFLogCharacterReport) error {
	for i, characterProgression := range characterReport.Progression {
		// ensure actual progress was made
		bestCharacterProgressionDB, err := d.FetchBestCharacterProgressionForEncounter(characterReport.Character.ID, characterProgression.EncounterInfo.ID)
		if err != nil && err != gorm.ErrRecordNotFound {
			return err
		}
		if err != gorm.ErrRecordNotFound && !bestCharacterProgressionDB.IsImprovement(characterProgression) {
			continue
		}
		// determine if this report needs update
		characterProgressionDB, err := d.FetchCharacterProgressionFromReportID(characterReport.ReportID, characterReport.Character.ID, characterProgression.EncounterInfo.ID)
		if err != nil && err != gorm.ErrRecordNotFound {
			return err
		}
		characterProgressionDB.ReportID = characterReport.ReportID
		characterProgressionDB.CharacterID = characterReport.Character.ID
		characterProgressionDB.EncounterInfoID = characterProgression.EncounterInfo.ID
		characterProgressionDB.GameVersion = characterProgression.GameVersion
		characterProgressionDB.FightPercentage = characterProgression.FightPercentage
		characterProgressionDB.Phase = characterProgression.Phase
		characterProgressionDB.PhasePercentage = characterProgression.PhasePercentage
		characterProgressionDB.IsKill = characterProgression.IsKill
		characterProgressionDB.IsStandardComposition = characterProgression.IsStandardComposition
		characterProgressionDB.HasEcho = characterProgression.HasEcho
		characterProgressionDB.Time = characterProgression.Time
		characterProgressionDB.Duration = characterProgression.Duration
		characterProgressionDB.Job = characterProgression.Job
		if tx := d.Conn.Save(&characterProgressionDB); tx.Error != nil {
			return tx.Error
		}
		characterReport.Progression[i] = characterProgressionDB
	}
	return nil
}

func (d DatabaseHandler) HandleFFLogCharacterReport(characterReport FFLogCharacterReport) error {
	if err := d.syncCharacterFromFFLogCharacterReport(&characterReport); err != nil {
		return err
	}
	if err := d.syncEncounterInfoFromFFLogCharacterReport(&characterReport); err != nil {
		return err
	}
	if err := d.syncCharacterProgressionsFromFFLogCharacterReport(&characterReport); err != nil {
		return err
	}
	return nil
}

func (d DatabaseHandler) FindCharacters(name string) ([]Character, error) {
	characters := make([]Character, 0)
	tx := d.Conn.Where("name LIKE ?", fmt.Sprintf("%%%s%%", name)).Find(&characters)
	return characters, tx.Error
}

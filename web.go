package main

import (
	"fmt"
	"html/template"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/time/rate"
	"gorm.io/gorm"
)

// htmlTemplates map of html templates
var htmlTemplates map[string]*template.Template

// importUserTrack tracks users who import reports for rate limiting
type importUserTrack struct {
	IPAddress string
	Limiter   *rate.Limiter
}

// importUserTracking is a list of all users who have imported reports
var importUserTracking []*importUserTrack

// displayEncounterData contains data to display encounter data
type displayEncounterData struct {
	Category   string
	Encounters []EncounterInfo
}

// templateData Struct containing data to be made available to html template
type templateData struct {
	AppName              string
	VersionString        string
	Characters           []Character
	CharacterProgression []CharacterProgression
	EncounterList        []displayEncounterData
	Message              string
}

func getTemplates() (map[string]*template.Template, error) {
	// create template map
	var templates = make(map[string]*template.Template)
	// get path to all include templates
	includeFiles, err := filepath.Glob("./web/tmpl/includes/*.tmpl")
	if err != nil {
		return nil, err
	}
	// get path to all layout templates
	layoutFiles, err := filepath.Glob("./web/tmpl/layouts/*.tmpl")
	if err != nil {
		return nil, err
	}
	// template funcs
	funcMap := template.FuncMap{
		"percent": func(p int64) string {
			return fmt.Sprintf("%.2f%%", float32(p)/100.0)
		},
		"displaydate": func(t time.Time) string {
			return t.Format("2006-01-02 03:04 PM")
		},
		"timestamp": func(t time.Time) int64 {
			return t.Unix()
		},
		"duration": func(d int64) string {
			return fmt.Sprintf("%02d:%02d", (d/1000)/60, (d/1000)%60)
		},
		"fflogurl": FFLogsCharacterURL,
	}
	// make layout templates
	for _, layoutFile := range layoutFiles {
		templateFiles := append(includeFiles, layoutFile)
		templates[filepath.Base(layoutFile)], err = template.New(filepath.Base(layoutFile)).Funcs(funcMap).ParseFiles(templateFiles...)
		if err != nil {
			return nil, err
		}
	}
	return templates, nil
}

func getBaseTemplateData() templateData {
	td := templateData{
		VersionString: appVersion,
		AppName:       appName,
	}
	return td
}

func displayError(w http.ResponseWriter, message string, statusCode int) {
	w.WriteHeader(statusCode)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	td := getBaseTemplateData()
	td.Message = message
	htmlTemplates["error.tmpl"].ExecuteTemplate(w, "base.tmpl", td)
}

func displayAjaxMessage(w http.ResponseWriter, message string, statusCode int) {
	w.WriteHeader(statusCode)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	td := getBaseTemplateData()
	td.Message = message
	htmlTemplates["ajax_message.tmpl"].ExecuteTemplate(w, "blank.tmpl", td)
}

func StartWeb(config *Config) error {

	var err error

	// connect database
	db, err := NewDatabaserHandler(config)
	if err != nil {
		return err
	}

	// load html templates
	htmlTemplates, err = getTemplates()
	if err != nil {
		return err
	}

	// init import queue
	fflogsImportQueue, err := NewFFLogsImportQueue(config, db)
	if err != nil {
		return err
	}
	go fflogsImportQueue.Start()

	importUserTracking = make([]*importUserTrack, 0)
	mux := http.NewServeMux()

	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))))

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		td := getBaseTemplateData()
		htmlTemplates["home.tmpl"].ExecuteTemplate(w, "base.tmpl", td)
	})

	mux.HandleFunc("/s", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		td := getBaseTemplateData()
		name := strings.TrimSpace(r.URL.Query().Get("n"))
		if name != "" {
			td.Characters, err = db.FindCharacters(name)
			if err != nil {
				if err == gorm.ErrRecordNotFound {
					displayError(w, err.Error(), 404)
					return
				}
				displayError(w, err.Error(), 500)
				return
			}
		}
		htmlTemplates["search.tmpl"].ExecuteTemplate(w, "blank.tmpl", td)
	})

	mux.HandleFunc("/c/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		td := getBaseTemplateData()
		pathes := strings.Split(r.URL.Path, "/")
		if len(pathes) < 3 {
			displayError(w, "character id is required", 400)
			return
		}
		uid := strings.ToLower(strings.TrimSpace(pathes[2]))
		if uid == "" {
			displayError(w, "character id is required", 400)
			return
		}
		character, err := db.FetchCharacterFromUID(uid)
		if err != nil {
			displayError(w, err.Error(), 500)
			return
		}
		encounterList, err := db.FetchEncounterList()
		if err != nil {
			displayError(w, err.Error(), 500)
			return
		}
		td.EncounterList = EncounterDisplayListFromEncounterInfoList(encounterList, config)
		td.Characters = []Character{character}
		characterProgress, err := db.FetchBestCharacterProgressions(character.ID)
		if err != nil {
			displayError(w, err.Error(), 500)
			return
		}
		td.CharacterProgression = characterProgress
		htmlTemplates["character_prog_list.tmpl"].ExecuteTemplate(w, "base.tmpl", td)
	})

	mux.HandleFunc("/i/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		userIPAddress := ReadUserIP(r)
		if userIPAddress == "" {
			displayAjaxMessage(w, "Invalid client.", 400)
		}
		var importTrack *importUserTrack = nil
		for _, it := range importUserTracking {
			if it.IPAddress == userIPAddress {
				importTrack = it
				break
			}
		}
		if importTrack == nil {
			importTrack = &importUserTrack{
				IPAddress: userIPAddress,
				Limiter:   rate.NewLimiter(0.1, 1),
			}
			importUserTracking = append(importUserTracking, importTrack)
		}
		if !importTrack.Limiter.Allow() {
			displayAjaxMessage(w, "Too many import request sent, please wait a little bit.", http.StatusTooManyRequests)
			return
		}
		reportID := FFLogReportURLToReportID(r.URL.Query().Get("r"))
		if reportID == "" {
			displayAjaxMessage(w, "FFLogs report URL not provided or invalid.", 400)
			return
		}
		if db.HasFFLogsReport(reportID) {
			displayAjaxMessage(w, fmt.Sprintf("FFLogs report %s has already been processed.", reportID), 400)
			return
		}
		if err := fflogsImportQueue.Add(reportID); err != nil {
			if err == ErrAlreadyInQueue {
				displayAjaxMessage(w, fmt.Sprintf("FFLogs report %s is already being processed.", reportID), 400)
				return
			}
			displayAjaxMessage(w, fmt.Sprintf("An Error Occured: %s", err.Error()), 500)
		}
		displayAjaxMessage(w, "Your report is being processed.", 200)
	})

	return http.ListenAndServe(fmt.Sprintf(":%d", config.HTTPPort), mux)
}

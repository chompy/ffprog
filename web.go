package main

import (
	"fmt"
	"html/template"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"gorm.io/gorm"
)

// htmlTemplates map of html templates
var htmlTemplates map[string]*template.Template

// templateData Struct containing data to be made available to html template
type templateData struct {
	AppName       string
	VersionString string
	Characters    []Character
	ErrorMessage  string
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
			return fmt.Sprintf("%d%%", int(p/100))
		},
		"displaydate": func(t time.Time) string {
			return t.Format("2006-01-02 15:04 PM")
		},
		"duration": func(d int64) string {
			return fmt.Sprintf("%02d:%02d", (d/1000)/60, (d/1000)%60)
		},
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
		VersionString: "0.01",
		AppName:       "FFProg",
	}
	return td
}

func displayError(w http.ResponseWriter, message string, statusCode int) {
	w.WriteHeader(statusCode)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	td := getBaseTemplateData()
	td.ErrorMessage = message
	htmlTemplates["error.tmpl"].ExecuteTemplate(w, "base.tmpl", td)
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

	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		td := getBaseTemplateData()
		htmlTemplates["home.tmpl"].ExecuteTemplate(w, "base.tmpl", td)
	})

	http.HandleFunc("/s", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		td := getBaseTemplateData()
		name := r.URL.Query().Get("n")
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

	http.HandleFunc("/c/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		td := getBaseTemplateData()

		pathes := strings.Split(r.URL.Path, "/")
		if len(pathes) < 3 {
			displayError(w, "character id is required", 400)
			return
		}
		uuid := pathes[2]
		if uuid == "" {
			displayError(w, "character id is required", 400)
			return
		}
		character, err := db.FetchCharacterFromUUID(uuid)
		if err != nil {
			displayError(w, err.Error(), 500)
			return
		}
		td.Characters = []Character{character}
		htmlTemplates["character.tmpl"].ExecuteTemplate(w, "base.tmpl", td)
	})

	return http.ListenAndServe(":8081", nil)
}

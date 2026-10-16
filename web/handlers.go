package main

import (
	"encoding/json"
	"html/template"
	"net/http"
	"os"
)

var indexTmpl = template.Must(template.ParseFiles(BASE + "/web/templates/login.html"))
var appTmpl = template.Must(template.ParseFiles(BASE + "/web/templates/app.html"))

func handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	cfg := getServerConfig()
	if !cfg.Initialized {
		http.Redirect(w, r, "/setup", http.StatusFound)
		return
	}
	var org Organization
	if err := loadJSON(BASE+"/organization.json", &org); err != nil {
		http.Error(w, "Ошибка загрузки организации", 500)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	indexTmpl.Execute(w, org)
}

func handleApp(w http.ResponseWriter, r *http.Request) {
	_, err := r.Cookie("session_email")
	if err != nil {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	appTmpl.Execute(w, nil)
}

func handleOrganization(w http.ResponseWriter, r *http.Request) {
	data, err := os.ReadFile(BASE + "/organization.json")
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Write(data)
}

func handleProjects(w http.ResponseWriter, r *http.Request) {
	projectsDir := BASE + "/projects"
	entries, err := os.ReadDir(projectsDir)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	var projects []map[string]interface{}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		metaPath := projectsDir + "/" + e.Name() + "/meta.json"
		data, err := os.ReadFile(metaPath)
		if err != nil {
			continue
		}
		var meta map[string]interface{}
		if err := json.Unmarshal(data, &meta); err != nil {
			continue
		}
		projects = append(projects, meta)
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(projects)
}

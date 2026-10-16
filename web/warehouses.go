package main

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"time"
)

type Warehouse struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	Responsible string `json:"responsible"`
	Description string `json:"description"`
	CreatedAt   string `json:"created_at"`
}

type WarehousesFile struct {
	Warehouses []Warehouse `json:"warehouses"`
}

func loadWarehouses(projectID string) (*WarehousesFile, error) {
	path := BASE + "/projects/" + projectID + "/warehouses.json"
	var wf WarehousesFile
	if err := loadJSON(path, &wf); err != nil {
		wf.Warehouses = []Warehouse{}
	}
	if wf.Warehouses == nil {
		wf.Warehouses = []Warehouse{}
	}
	return &wf, nil
}

func saveWarehouses(projectID string, wf *WarehousesFile) error {
	path := BASE + "/projects/" + projectID + "/warehouses.json"
	return saveJSON(path, wf)
}

func handleWarehousesList(w http.ResponseWriter, r *http.Request) {
	projectID := r.URL.Query().Get("project")
	if projectID == "" {
		http.Error(w, "Не указан проект", 400)
		return
	}
	wf, err := loadWarehouses(projectID)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(wf)
}

func handleWarehouseAdd(w http.ResponseWriter, r *http.Request) {
	if !canEditItems(getRole(r)) {
		http.Error(w, "Доступ запрещён", http.StatusForbidden)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Метод не разрешён", 405)
		return
	}
	r.ParseForm()
	projectID := strings.TrimSpace(r.FormValue("project"))
	name := strings.TrimSpace(r.FormValue("name"))
	wtype := strings.TrimSpace(r.FormValue("type"))
	responsible := strings.TrimSpace(r.FormValue("responsible"))
	description := strings.TrimSpace(r.FormValue("description"))

	if projectID == "" || name == "" {
		http.Error(w, "Не хватает данных", 400)
		return
	}

	wf, _ := loadWarehouses(projectID)

	// Генерируем ID
	newID := "wh_001"
	for i := 1; i <= 999; i++ {
		candidate := "wh_" + pad3(i)
		exists := false
		for _, w := range wf.Warehouses {
			if w.ID == candidate {
				exists = true
				break
			}
		}
		if !exists {
			newID = candidate
			break
		}
	}

	wf.Warehouses = append(wf.Warehouses, Warehouse{
		ID:          newID,
		Name:        name,
		Type:        wtype,
		Responsible: responsible,
		Description: description,
		CreatedAt:   time.Now().UTC().Format(time.RFC3339),
	})

	if err := saveWarehouses(projectID, wf); err != nil {
		http.Error(w, "Ошибка сохранения", 500)
		return
	}
	w.Write([]byte("Склад добавлен: " + newID))
}

func handleWarehouseDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Метод не разрешён", 405)
		return
	}
	r.ParseForm()
	projectID := strings.TrimSpace(r.FormValue("project"))
	id := strings.TrimSpace(r.FormValue("id"))

	if projectID == "" || id == "" {
		http.Error(w, "Не хватает данных", 400)
		return
	}
	wf, _ := loadWarehouses(projectID)

	found := false
	newList := []Warehouse{}
	for _, w := range wf.Warehouses {
		if w.ID == id {
			found = true
			continue
		}
		newList = append(newList, w)
	}
	if !found {
		http.Error(w, "Склад не найден", 404)
		return
	}
	wf.Warehouses = newList
	if err := saveWarehouses(projectID, wf); err != nil {
		http.Error(w, "Ошибка сохранения", 500)
		return
	}
	w.Write([]byte("Удалён"))
}

func handleWarehouseRename(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Метод не разрешён", 405)
		return
	}
	r.ParseForm()
	projectID := strings.TrimSpace(r.FormValue("project"))
	id := strings.TrimSpace(r.FormValue("id"))
	name := strings.TrimSpace(r.FormValue("name"))

	if projectID == "" || id == "" || name == "" {
		http.Error(w, "Не хватает данных", 400)
		return
	}
	wf, _ := loadWarehouses(projectID)

	found := false
	for i := range wf.Warehouses {
		if wf.Warehouses[i].ID == id {
			wf.Warehouses[i].Name = name
			found = true
			break
		}
	}
	if !found {
		http.Error(w, "Склад не найден", 404)
		return
	}
	if err := saveWarehouses(projectID, wf); err != nil {
		http.Error(w, "Ошибка сохранения", 500)
		return
	}
	w.Write([]byte("Переименован"))
}

func pad3(n int) string {
	s := ""
	if n < 10 {
		s = "00"
	} else if n < 100 {
		s = "0"
	}
	return s + itoa(n)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	s := ""
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	return s
}

// Требуется для совместимости с os
var _ = os.Stat

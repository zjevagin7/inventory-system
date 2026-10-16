package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

type Item struct {
	ID             string `json:"id"`
	WarehouseID    string `json:"warehouse_id"`
	Name           string `json:"name"`
	Characteristic string `json:"characteristic"`
	Serial         string `json:"serial"`
	Unit           string `json:"unit"`
	QtyPlanned     int    `json:"qty_planned"`
	QtyActual      int    `json:"qty_actual"`
	Deviation      int    `json:"deviation"`
	Location       string `json:"location"`
	LocationDetail string `json:"location_detail"`
	HolderName     string `json:"holder_name"`
	Photo          string `json:"photo"`
	Favicon        string `json:"favicon"`
	CreatedAt      string `json:"created_at"`
	UpdatedAt      string `json:"updated_at"`
}

type ItemsFile struct {
	Items []Item `json:"items"`
}

func loadItems(projectID string) (*ItemsFile, error) {
	path := BASE + "/projects/" + projectID + "/items.json"
	var itf ItemsFile
	if err := loadJSON(path, &itf); err != nil {
		itf.Items = []Item{}
	}
	if itf.Items == nil {
		itf.Items = []Item{}
	}
	return &itf, nil
}

func saveItems(projectID string, itf *ItemsFile) error {
	path := BASE + "/projects/" + projectID + "/items.json"
	return saveJSON(path, itf)
}

func handleItemsList(w http.ResponseWriter, r *http.Request) {
	projectID := r.URL.Query().Get("project")
	warehouseID := r.URL.Query().Get("warehouse")
	if projectID == "" {
		http.Error(w, "Не указан проект", 400)
		return
	}
	itf, err := loadItems(projectID)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	if warehouseID != "" {
		filtered := []Item{}
		for _, it := range itf.Items {
			if it.WarehouseID == warehouseID {
				filtered = append(filtered, it)
			}
		}
		itf.Items = filtered
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(itf)
}

func handleItemAdd(w http.ResponseWriter, r *http.Request) {
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
	warehouseID := strings.TrimSpace(r.FormValue("warehouse"))
	name := strings.TrimSpace(r.FormValue("name"))
	characteristic := strings.TrimSpace(r.FormValue("characteristic"))
	serial := strings.TrimSpace(r.FormValue("serial"))
	unit := strings.TrimSpace(r.FormValue("unit"))
	location := strings.TrimSpace(r.FormValue("location"))
	locationDetail := strings.TrimSpace(r.FormValue("location_detail"))
	holderName := strings.TrimSpace(r.FormValue("holder_name"))

	if projectID == "" || warehouseID == "" || name == "" {
		http.Error(w, "Не хватает данных", 400)
		return
	}

	if location == "В ремонте" {
		locationDetail = "На базе"
	}
	if location == "Списан" {
		locationDetail = "Отдан на базу"
	}

	itf, _ := loadItems(projectID)

	newID := "item_001"
	for i := 1; i <= 9999; i++ {
		candidate := "item_" + padN(i, 3)
		exists := false
		for _, it := range itf.Items {
			if it.ID == candidate {
				exists = true
				break
			}
		}
		if !exists {
			newID = candidate
			break
		}
	}

	itf.Items = append(itf.Items, Item{
		ID:             newID,
		WarehouseID:    warehouseID,
		Name:           name,
		Characteristic: characteristic,
		Serial:         serial,
		Unit:           unit,
		QtyPlanned:     0,
		QtyActual:      0,
		Deviation:      0,
		Location:       location,
		LocationDetail: locationDetail,
		HolderName:     holderName,
		Photo:          "",
		Favicon:        "",
		CreatedAt:      time.Now().UTC().Format(time.RFC3339),
		UpdatedAt:      time.Now().UTC().Format(time.RFC3339),
	})

	if err := saveItems(projectID, itf); err != nil {
		http.Error(w, "Ошибка сохранения", 500)
		return
	}
	w.Write([]byte("Предмет добавлен: " + newID))
}

func handleItemDelete(w http.ResponseWriter, r *http.Request) {
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
	id := strings.TrimSpace(r.FormValue("id"))

	if projectID == "" || id == "" {
		http.Error(w, "Не хватает данных", 400)
		return
	}
	itf, _ := loadItems(projectID)

	found := false
	newList := []Item{}
	for _, it := range itf.Items {
		if it.ID == id {
			found = true
			continue
		}
		newList = append(newList, it)
	}
	if !found {
		http.Error(w, "Предмет не найден", 404)
		return
	}
	itf.Items = newList
	if err := saveItems(projectID, itf); err != nil {
		http.Error(w, "Ошибка сохранения", 500)
		return
	}
	w.Write([]byte("Удалён"))
}

func handleItemUpdate(w http.ResponseWriter, r *http.Request) {
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
	id := strings.TrimSpace(r.FormValue("id"))

	if projectID == "" || id == "" {
		http.Error(w, "Не хватает данных", 400)
		return
	}
	itf, _ := loadItems(projectID)

	found := false
	for i := range itf.Items {
		if itf.Items[i].ID == id {
			itf.Items[i].Name = strings.TrimSpace(r.FormValue("name"))
			itf.Items[i].Characteristic = strings.TrimSpace(r.FormValue("characteristic"))
			itf.Items[i].Serial = strings.TrimSpace(r.FormValue("serial"))
			itf.Items[i].Unit = strings.TrimSpace(r.FormValue("unit"))
			itf.Items[i].Location = strings.TrimSpace(r.FormValue("location"))
			itf.Items[i].LocationDetail = strings.TrimSpace(r.FormValue("location_detail"))
			itf.Items[i].HolderName = strings.TrimSpace(r.FormValue("holder_name"))

			if itf.Items[i].Location == "В ремонте" {
				itf.Items[i].LocationDetail = "На базе"
			}
			if itf.Items[i].Location == "Списан" {
				itf.Items[i].LocationDetail = "Отдан на базу"
			}

			if qp := r.FormValue("qty_planned"); qp != "" {
				itf.Items[i].QtyPlanned = atoiSafe(qp)
			}
			if qa := r.FormValue("qty_actual"); qa != "" {
				itf.Items[i].QtyActual = atoiSafe(qa)
			}
			itf.Items[i].Deviation = itf.Items[i].QtyActual - itf.Items[i].QtyPlanned
			itf.Items[i].UpdatedAt = time.Now().UTC().Format(time.RFC3339)
			found = true
			break
		}
	}
	if !found {
		http.Error(w, "Предмет не найден", 404)
		return
	}
	if err := saveItems(projectID, itf); err != nil {
		http.Error(w, "Ошибка сохранения", 500)
		return
	}
	w.Write([]byte("Обновлён"))
}

func padN(n, width int) string {
	s := ""
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	for len(s) < width {
		s = "0" + s
	}
	return s
}

func atoiSafe(s string) int {
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			continue
		}
		n = n*10 + int(c-'0')
	}
	return n
}

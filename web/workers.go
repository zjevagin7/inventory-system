package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

type Worker struct {
	Tab       string `json:"tab"`
	Name      string `json:"name"`
	Role      string `json:"role"`
	Position  string `json:"position"`
	CardUID   string `json:"card_uid"`
	CreatedAt string `json:"created_at"`
}

type WorkersFile struct {
	Workers []Worker `json:"workers"`
}

func loadWorkers(projectID string) (*WorkersFile, error) {
	path := BASE + "/projects/" + projectID + "/workers.json"
	var wf WorkersFile
	if err := loadJSON(path, &wf); err != nil {
		wf.Workers = []Worker{}
	}
	if wf.Workers == nil {
		wf.Workers = []Worker{}
	}
	return &wf, nil
}

func saveWorkers(projectID string, wf *WorkersFile) error {
	path := BASE + "/projects/" + projectID + "/workers.json"
	return saveJSON(path, wf)
}

func handleWorkersList(w http.ResponseWriter, r *http.Request) {
	projectID := r.URL.Query().Get("project")
	if projectID == "" {
		http.Error(w, "Не указан проект", 400)
		return
	}
	wf, err := loadWorkers(projectID)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(wf)
}

func handleWorkerAdd(w http.ResponseWriter, r *http.Request) {
	if !canEditWorkers(getRole(r)) {
		http.Error(w, "Доступ запрещён", http.StatusForbidden)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "Метод не разрешён", 405)
		return
	}
	r.ParseForm()
	projectID := strings.TrimSpace(r.FormValue("project"))
	tab := strings.TrimSpace(r.FormValue("tab"))
	name := strings.TrimSpace(r.FormValue("name"))
	role := strings.TrimSpace(r.FormValue("role"))
	position := strings.TrimSpace(r.FormValue("position"))
	cardUID := strings.TrimSpace(r.FormValue("card_uid"))

	if projectID == "" || name == "" {
		http.Error(w, "Не хватает данных", 400)
		return
	}

	wf, _ := loadWorkers(projectID)

	// Проверка уникальности табельного
	for _, wk := range wf.Workers {
		if wk.Tab == tab && tab != "" {
			http.Error(w, "Работник с таким табельным уже есть", 409)
			return
		}
	}

	// Проверка уникальности карты
	if cardUID != "" {
		for _, wk := range wf.Workers {
			if wk.CardUID == cardUID {
				http.Error(w, "Карта уже привязана к другому работнику", 409)
				return
			}
		}
	}

	wf.Workers = append(wf.Workers, Worker{
		Tab:       tab,
		Name:      name,
		Role:      role,
		Position:  position,
		CardUID:   cardUID,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	})

	if err := saveWorkers(projectID, wf); err != nil {
		http.Error(w, "Ошибка сохранения", 500)
		return
	}
	w.Write([]byte("Работник добавлен: " + name))
}

func handleWorkerDelete(w http.ResponseWriter, r *http.Request) {
	if !canEditWorkers(getRole(r)) {
		http.Error(w, "Доступ запрещён", http.StatusForbidden)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "Метод не разрешён", 405)
		return
	}
	r.ParseForm()
	projectID := strings.TrimSpace(r.FormValue("project"))
	tab := strings.TrimSpace(r.FormValue("tab"))
	name := strings.TrimSpace(r.FormValue("name"))

	if projectID == "" || (tab == "" && name == "") {
		http.Error(w, "Не хватает данных", 400)
		return
	}
	wf, _ := loadWorkers(projectID)

	found := false
	newList := []Worker{}
	for _, wk := range wf.Workers {
		if (tab != "" && wk.Tab == tab) || (tab == "" && wk.Name == name) {
			found = true
			continue
		}
		newList = append(newList, wk)
	}
	if !found {
		http.Error(w, "Работник не найден", 404)
		return
	}
	wf.Workers = newList
	if err := saveWorkers(projectID, wf); err != nil {
		http.Error(w, "Ошибка сохранения", 500)
		return
	}
	w.Write([]byte("Удалён"))
}

func handleWorkerTransfer(w http.ResponseWriter, r *http.Request) {
	if !canEditWorkers(getRole(r)) {
		http.Error(w, "Доступ запрещён", http.StatusForbidden)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "Метод не разрешён", 405)
		return
	}
	r.ParseForm()
	fromProject := strings.TrimSpace(r.FormValue("from_project"))
	toProject := strings.TrimSpace(r.FormValue("to_project"))
	tab := strings.TrimSpace(r.FormValue("tab"))
	name := strings.TrimSpace(r.FormValue("name"))

	if fromProject == "" || toProject == "" || (tab == "" && name == "") {
		http.Error(w, "Не хватает данных", 400)
		return
	}

	fromWF, _ := loadWorkers(fromProject)
	toWF, _ := loadWorkers(toProject)

	var worker *Worker
	newFromList := []Worker{}
	for _, wk := range fromWF.Workers {
		if (tab != "" && wk.Tab == tab) || (tab == "" && wk.Name == name) {
			wkCopy := wk
			worker = &wkCopy
			continue
		}
		newFromList = append(newFromList, wk)
	}
	if worker == nil {
		http.Error(w, "Работник не найден в исходном проекте", 404)
		return
	}

	for _, wk := range toWF.Workers {
		if (tab != "" && wk.Tab == tab) || (tab == "" && wk.Name == name) {
			http.Error(w, "Работник уже есть в целевом проекте", 409)
			return
		}
	}

	fromWF.Workers = newFromList
	toWF.Workers = append(toWF.Workers, *worker)

	if err := saveWorkers(fromProject, fromWF); err != nil {
		http.Error(w, "Ошибка сохранения исходного", 500)
		return
	}
	if err := saveWorkers(toProject, toWF); err != nil {
		http.Error(w, "Ошибка сохранения целевого", 500)
		return
	}
	w.Write([]byte("Работник перенесён: " + worker.Name))
}

// Привязка карты к работнику
func handleWorkerBindCard(w http.ResponseWriter, r *http.Request) {
	if !canEditWorkers(getRole(r)) {
		http.Error(w, "Доступ запрещён", http.StatusForbidden)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "Метод не разрешён", 405)
		return
	}
	r.ParseForm()
	projectID := strings.TrimSpace(r.FormValue("project"))
	tab := strings.TrimSpace(r.FormValue("tab"))
	cardUID := strings.TrimSpace(r.FormValue("card_uid"))

	if projectID == "" || tab == "" {
		http.Error(w, "Не хватает данных", 400)
		return
	}

	wf, _ := loadWorkers(projectID)

	// Проверка: карта не должна быть у другого работника
	if cardUID != "" {
		for _, wk := range wf.Workers {
			if wk.CardUID == cardUID && wk.Tab != tab {
				http.Error(w, "Карта уже привязана к другому работнику", 409)
				return
			}
		}
	}

	found := false
	for i := range wf.Workers {
		if wf.Workers[i].Tab == tab {
			wf.Workers[i].CardUID = cardUID
			found = true
			break
		}
	}
	if !found {
		http.Error(w, "Работник не найден", 404)
		return
	}
	if err := saveWorkers(projectID, wf); err != nil {
		http.Error(w, "Ошибка сохранения", 500)
		return
	}
	w.Write([]byte("Карта привязана"))
}

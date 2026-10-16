package main

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"time"
)

// ---------- СТРУКТУРЫ ----------

type TimesheetEmployee struct {
	Tab        string            `json:"tab"`
	Name       string            `json:"name"`
	Days       map[string]int    `json:"days"`
	Notes      map[string]string `json:"notes"`
	TotalHours int               `json:"total_hours"`
}

type TimesheetFile struct {
	Month     string              `json:"month"`
	ProjectID string              `json:"project_id"`
	Employees []TimesheetEmployee `json:"employees"`
	UpdatedAt string              `json:"updated_at"`
}

func timesheetPath(projectID, month string) string {
	return BASE + "/projects/" + projectID + "/timesheet/" + month + ".json"
}

func loadTimesheet(projectID, month string) (*TimesheetFile, error) {
	path := timesheetPath(projectID, month)
	var tf TimesheetFile
	if err := loadJSON(path, &tf); err != nil {
		tf = TimesheetFile{
			Month:     month,
			ProjectID: projectID,
			Employees: []TimesheetEmployee{},
		}
	}
	if tf.Employees == nil {
		tf.Employees = []TimesheetEmployee{}
	}
	for i := range tf.Employees {
		if tf.Employees[i].Days == nil {
			tf.Employees[i].Days = map[string]int{}
		}
		if tf.Employees[i].Notes == nil {
			tf.Employees[i].Notes = map[string]string{}
		}
	}
	return &tf, nil
}

func saveTimesheet(projectID string, tf *TimesheetFile) error {
	path := timesheetPath(projectID, tf.Month)
	os.MkdirAll(BASE+"/projects/"+projectID+"/timesheet", 0755)
	return saveJSON(path, tf)
}

func recalcTotals(tf *TimesheetFile) {
	for i := range tf.Employees {
		total := 0
		for _, h := range tf.Employees[i].Days {
			total += h
		}
		tf.Employees[i].TotalHours = total
	}
}

// ---------- РУЧНОЙ ТАБЕЛЬ (UI) ----------

func handleTimesheetGet(w http.ResponseWriter, r *http.Request) {
	projectID := r.URL.Query().Get("project")
	month := r.URL.Query().Get("month")
	if projectID == "" || month == "" {
		http.Error(w, "Не указан проект или месяц", 400)
		return
	}

	tf, _ := loadTimesheet(projectID, month)

	if len(tf.Employees) == 0 {
		wf, _ := loadWorkers(projectID)
		for _, wk := range wf.Workers {
			tf.Employees = append(tf.Employees, TimesheetEmployee{
				Tab:        wk.Tab,
				Name:       wk.Name,
				Days:       map[string]int{},
				Notes:      map[string]string{},
				TotalHours: 0,
			})
		}
		recalcTotals(tf)
		saveTimesheet(projectID, tf)
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(tf)
}

func handleTimesheetSet(w http.ResponseWriter, r *http.Request) {
	if !canEditTimesheet(getRole(r)) {
		http.Error(w, "Доступ запрещён", http.StatusForbidden)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "Метод не разрешён", 405)
		return
	}
	r.ParseForm()
	projectID := strings.TrimSpace(r.FormValue("project"))
	month := strings.TrimSpace(r.FormValue("month"))
	tab := strings.TrimSpace(r.FormValue("tab"))
	day := strings.TrimSpace(r.FormValue("day"))
	hoursStr := strings.TrimSpace(r.FormValue("hours"))
	note := strings.TrimSpace(r.FormValue("note"))

	if projectID == "" || month == "" || tab == "" || day == "" {
		http.Error(w, "Не хватает данных", 400)
		return
	}

	tf, _ := loadTimesheet(projectID, month)

	found := false
	for i := range tf.Employees {
		if tf.Employees[i].Tab == tab {
			if hoursStr == "" && note == "" {
				delete(tf.Employees[i].Days, day)
				delete(tf.Employees[i].Notes, day)
			} else if note != "" {
				tf.Employees[i].Notes[day] = note
				delete(tf.Employees[i].Days, day)
			} else {
				tf.Employees[i].Days[day] = atoiSafe(hoursStr)
				delete(tf.Employees[i].Notes, day)
			}
			found = true
			break
		}
	}
	if !found {
		http.Error(w, "Работник не найден в табеле", 404)
		return
	}

	recalcTotals(tf)
	tf.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	if err := saveTimesheet(projectID, tf); err != nil {
		http.Error(w, "Ошибка сохранения", 500)
		return
	}
	w.Write([]byte("Обновлено"))
}

// ---------- API ДЛЯ ESP ----------

func handleDevicePunch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Метод не разрешён", 405)
		return
	}

	var req struct {
		Token   string `json:"token"`
		CardUID string `json:"card_uid"`
	}

	if strings.Contains(r.Header.Get("Content-Type"), "application/json") {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Ошибка JSON", 400)
			return
		}
	} else {
		r.ParseForm()
		req.Token = strings.TrimSpace(r.FormValue("token"))
		req.CardUID = strings.TrimSpace(r.FormValue("card_uid"))
	}

	if req.Token == "" || req.CardUID == "" {
		http.Error(w, "Не хватает данных", 400)
		return
	}

	dt, ok := checkDeviceToken(req.Token)
	if !ok {
		http.Error(w, "Неверный токен устройства", 401)
		return
	}

	projectID := dt.Project
	if projectID == "" {
		http.Error(w, "У токена не указан проект", 400)
		return
	}

	wf, _ := loadWorkers(projectID)
	var worker *Worker
	for i := range wf.Workers {
		if wf.Workers[i].CardUID == req.CardUID {
			worker = &wf.Workers[i]
			break
		}
	}
	if worker == nil {
		http.Error(w, "Карта не привязана к работнику", 404)
		return
	}

	now := time.Now()
	month := now.Format("2006-01")
	day := now.Format("2")
	today := now.Format("2006-01-02")

	tf, _ := loadTimesheet(projectID, month)

	var emp *TimesheetEmployee
	for i := range tf.Employees {
		if tf.Employees[i].Tab == worker.Tab {
			emp = &tf.Employees[i]
			break
		}
	}
	if emp == nil {
		tf.Employees = append(tf.Employees, TimesheetEmployee{
			Tab:        worker.Tab,
			Name:       worker.Name,
			Days:       map[string]int{},
			Notes:      map[string]string{},
			TotalHours: 0,
		})
		emp = &tf.Employees[len(tf.Employees)-1]
	}

	punchKey := "punch_" + day
	if emp.Notes[punchKey] == "" {
		emp.Notes[punchKey] = "in:" + now.Format("15:04")
	} else if strings.HasPrefix(emp.Notes[punchKey], "in:") {
		inTime := strings.TrimPrefix(emp.Notes[punchKey], "in:")
		outTime := now.Format("15:04")
		hours := calcHours(inTime, outTime)
		emp.Days[day] = hours
		emp.Notes[punchKey] = "in:" + inTime + "|out:" + outTime
	}

	dtf, _ := loadDeviceTokens()
	for i := range dtf.Tokens {
		if dtf.Tokens[i].Token == req.Token {
			dtf.Tokens[i].LastUsed = now.Format(time.RFC3339)
			break
		}
	}
	saveDeviceTokens(dtf)

	recalcTotals(tf)
	tf.UpdatedAt = now.Format(time.RFC3339)
	saveTimesheet(projectID, tf)

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
		"name":   worker.Name,
		"tab":    worker.Tab,
		"time":   now.Format("15:04:05"),
		"date":   today,
		"action": emp.Notes[punchKey],
	})
}

func calcHours(inTime, outTime string) int {
	inParts := strings.Split(inTime, ":")
	outParts := strings.Split(outTime, ":")
	if len(inParts) != 2 || len(outParts) != 2 {
		return 0
	}
	inH := atoiSafe(inParts[0])
	inM := atoiSafe(inParts[1])
	outH := atoiSafe(outParts[0])
	outM := atoiSafe(outParts[1])
	totalIn := inH*60 + inM
	totalOut := outH*60 + outM
	diff := totalOut - totalIn
	if diff < 0 {
		diff = 0
	}
	return diff / 60
}

package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

var adminTmpl = template.Must(template.ParseFiles(BASE + "/web/templates/admin.html"))

// ---------- АДМИНКА ----------

func handleAdmin(w http.ResponseWriter, r *http.Request) {
	if !isAdmin(r) {
		http.Error(w, "Доступ запрещён", http.StatusForbidden)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	adminTmpl.Execute(w, nil)
}

// ---------- ПОЛЬЗОВАТЕЛИ (только админ) ----------

func handleAdminUsers(w http.ResponseWriter, r *http.Request) {
	if !isAdmin(r) {
		http.Error(w, "Доступ запрещён", http.StatusForbidden)
		return
	}
	users := loadUsers()
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(users)
}

func handleAdminUserUpdate(w http.ResponseWriter, r *http.Request) {
	if !isAdmin(r) {
		http.Error(w, "Доступ запрещён", http.StatusForbidden)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "Метод не разрешён", 405)
		return
	}
	r.ParseForm()
	oldEmail := strings.TrimSpace(r.FormValue("old_email"))
	newEmail := strings.TrimSpace(r.FormValue("email"))
	name := strings.TrimSpace(r.FormValue("name"))
	role := strings.TrimSpace(r.FormValue("role"))
	tab := strings.TrimSpace(r.FormValue("tab_number"))
	active := r.FormValue("active") == "true"

	users := loadUsers()
	user, ok := users[oldEmail]
	if !ok {
		http.Error(w, "Пользователь не найден", 404)
		return
	}
	delete(users, oldEmail)
	user.Name = name
	user.Role = role
	user.TabNumber = tab
	user.Active = active
	users[newEmail] = user
	if err := saveUsers(users); err != nil {
		http.Error(w, "Ошибка сохранения", 500)
		return
	}
	w.Write([]byte("Обновлено"))
}

func handleAdminUserDelete(w http.ResponseWriter, r *http.Request) {
	if !isAdmin(r) {
		http.Error(w, "Доступ запрещён", http.StatusForbidden)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "Метод не разрешён", 405)
		return
	}
	r.ParseForm()
	email := strings.TrimSpace(r.FormValue("email"))
	users := loadUsers()
	if _, ok := users[email]; !ok {
		http.Error(w, "Пользователь не найден", 404)
		return
	}
	delete(users, email)
	if err := saveUsers(users); err != nil {
		http.Error(w, "Ошибка сохранения", 500)
		return
	}
	w.Write([]byte("Удалён"))
}

func handleAdminUserAdd(w http.ResponseWriter, r *http.Request) {
	if !isAdmin(r) {
		http.Error(w, "Доступ запрещён", http.StatusForbidden)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "Метод не разрешён", 405)
		return
	}
	r.ParseForm()
	email := strings.TrimSpace(r.FormValue("email"))
	name := strings.TrimSpace(r.FormValue("name"))
	role := strings.TrimSpace(r.FormValue("role"))
	tab := strings.TrimSpace(r.FormValue("tab_number"))
	password := strings.TrimSpace(r.FormValue("password"))

	if email == "" || name == "" || role == "" || password == "" {
		http.Error(w, "Заполни обязательные поля", 400)
		return
	}
	if !strings.HasSuffix(email, "@yandex.ru") {
		http.Error(w, "Почта должна быть @yandex.ru", 400)
		return
	}
	users := loadUsers()
	if _, ok := users[email]; ok {
		http.Error(w, "Такой пользователь уже есть", 409)
		return
	}
	users[email] = User{
		Name:         name,
		Role:         role,
		TabNumber:    tab,
		Confirmed:    true,
		Active:       true,
		PasswordHash: hashPassword(password),
	}
	if err := saveUsers(users); err != nil {
		http.Error(w, "Ошибка сохранения", 500)
		return
	}
	w.Write([]byte("Добавлен"))
}

// ---------- ПРОЕКТЫ (админ + бухгалтер + директор) ----------

func handleAdminProjectsList(w http.ResponseWriter, r *http.Request) {
	if !canAddProjects(getRole(r)) {
		http.Error(w, "Доступ запрещён", http.StatusForbidden)
		return
	}
	projectsDir := BASE + "/projects"
	entries, _ := os.ReadDir(projectsDir)
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
		json.Unmarshal(data, &meta)
		projects = append(projects, meta)
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(projects)
}

func handleAdminProjectAdd(w http.ResponseWriter, r *http.Request) {
	if !canAddProjects(getRole(r)) {
		http.Error(w, "Доступ запрещён", http.StatusForbidden)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "Метод не разрешён", 405)
		return
	}
	r.ParseForm()
	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		http.Error(w, "Введите название", 400)
		return
	}
	projectsDir := BASE + "/projects"
	entries, _ := os.ReadDir(projectsDir)
	used := map[string]bool{}
	for _, e := range entries {
		used[e.Name()] = true
	}
	var pid string
	for i := 1; i <= 3; i++ {
		c := fmt.Sprintf("project_%d", i)
		if !used[c] {
			pid = c
			break
		}
	}
	if pid == "" {
		http.Error(w, "Максимум 3 проекта", 400)
		return
	}
	dir := projectsDir + "/" + pid
	os.MkdirAll(dir, 0755)

	meta := make(map[string]interface{})
	meta["id"] = pid
	meta["name"] = name
	meta["status"] = "active"
	meta["created_at"] = time.Now().UTC().Format(time.RFC3339)
	meta["closed_at"] = nil
	saveJSON(dir+"/meta.json", meta)

	items := make(map[string]interface{})
	items["items"] = []interface{}{}
	saveJSON(dir+"/items.json", items)

	history := make(map[string]interface{})
	history["events"] = []interface{}{}
	saveJSON(dir+"/history.json", history)

	warehouses := make(map[string]interface{})
	warehouses["warehouses"] = []interface{}{}
	saveJSON(dir+"/warehouses.json", warehouses)

	workers := make(map[string]interface{})
	workers["workers"] = []interface{}{}
	saveJSON(dir+"/workers.json", workers)

	w.Write([]byte("Проект создан: " + pid))
}

func handleAdminProjectDelete(w http.ResponseWriter, r *http.Request) {
	if !canAddProjects(getRole(r)) {
		http.Error(w, "Доступ запрещён", http.StatusForbidden)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "Метод не разрешён", 405)
		return
	}
	r.ParseForm()
	pid := strings.TrimSpace(r.FormValue("id"))
	if pid == "" {
		http.Error(w, "Не указан проект", 400)
		return
	}
	dir := BASE + "/projects/" + pid
	if _, err := os.Stat(dir); err != nil {
		http.Error(w, "Проект не найден", 404)
		return
	}
	os.RemoveAll(dir)
	w.Write([]byte("Удалён"))
}

func handleAdminProjectRename(w http.ResponseWriter, r *http.Request) {
	if !canAddProjects(getRole(r)) {
		http.Error(w, "Доступ запрещён", http.StatusForbidden)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "Метод не разрешён", 405)
		return
	}
	r.ParseForm()
	pid := strings.TrimSpace(r.FormValue("id"))
	name := strings.TrimSpace(r.FormValue("name"))
	if pid == "" || name == "" {
		http.Error(w, "Не хватает данных", 400)
		return
	}
	metaPath := BASE + "/projects/" + pid + "/meta.json"
	var meta map[string]interface{}
	if err := loadJSON(metaPath, &meta); err != nil {
		http.Error(w, "Проект не найден", 404)
		return
	}
	meta["name"] = name
	saveJSON(metaPath, meta)
	w.Write([]byte("Переименован"))
}

// ---------- БЭКАПЫ (админ + бухгалтер + директор) ----------

func handleAdminBackups(w http.ResponseWriter, r *http.Request) {
	if !canAddProjects(getRole(r)) {
		http.Error(w, "Доступ запрещён", http.StatusForbidden)
		return
	}
	if r.URL.Query().Get("download") != "" {
		name := r.URL.Query().Get("name")
		path := filepath.Join(BASE, "sync", name)
		if _, err := os.Stat(path); err != nil {
			http.Error(w, "Бэкап не найден", 404)
			return
		}
		w.Header().Set("Content-Disposition", "attachment; filename="+name)
		http.ServeFile(w, r, path)
		return
	}
	syncDir := filepath.Join(BASE, "sync")
	entries, _ := os.ReadDir(syncDir)
	var backups []map[string]string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasPrefix(name, "sync_") || !strings.HasSuffix(name, ".tar.gz") {
			continue
		}
		info, _ := e.Info()
		b := make(map[string]string)
		b["name"] = name
		b["date"] = info.ModTime().Format("2006-01-02 15:04")
		b["size"] = fmt.Sprintf("%.2f КБ", float64(info.Size())/1024)
		backups = append(backups, b)
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(backups)
}

func handleAdminBackupCreate(w http.ResponseWriter, r *http.Request) {
	if !canAddProjects(getRole(r)) {
		http.Error(w, "Доступ запрещён", http.StatusForbidden)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "Метод не разрешён", 405)
		return
	}
	out, err := exec.Command("python", BASE+"/sync.py").CombinedOutput()
	if err != nil {
		http.Error(w, "Ошибка: "+err.Error()+"\n"+string(out), 500)
		return
	}
	w.Write([]byte("Бэкап создан"))
}

// ---------- ТОКЕН (только админ) ----------

func handleAdminTokenGet(w http.ResponseWriter, r *http.Request) {
	if !isAdmin(r) {
		http.Error(w, "Доступ запрещён", http.StatusForbidden)
		return
	}
	data, err := os.ReadFile(BASE + "/.yandex_token")
	if err != nil {
		http.Error(w, "Токен не найден", 404)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write(data)
}

func handleAdminTokenSet(w http.ResponseWriter, r *http.Request) {
	if !isAdmin(r) {
		http.Error(w, "Доступ запрещён", http.StatusForbidden)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "Метод не разрешён", 405)
		return
	}
	r.ParseForm()
	token := strings.TrimSpace(r.FormValue("token"))
	if token == "" {
		http.Error(w, "Токен пустой", 400)
		return
	}
	os.WriteFile(BASE+"/.yandex_token", []byte(token), 0600)
	w.Write([]byte("Токен обновлён"))
}

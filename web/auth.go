package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ---------- ЗАГРУЗКА / СОХРАНЕНИЕ ----------

func loadUsers() map[string]User {
	users := map[string]User{}
	loadJSON(BASE+"/users.json", &users)
	return users
}

func saveUsers(users map[string]User) error {
	return saveJSON(BASE+"/users.json", users)
}

func hashPassword(p string) string {
	h := sha256.Sum256([]byte(p))
	return hex.EncodeToString(h[:])
}

func getServerConfig() *ServerConfig {
	var cfg ServerConfig
	if err := loadJSON(BASE+"/server_config.json", &cfg); err != nil {
		return &ServerConfig{Initialized: false, MaxConcurrent: 3}
	}
	return &cfg
}

// ---------- ПРОВЕРКА РОЛЕЙ ----------

func getRole(r *http.Request) string {
	cookie, err := r.Cookie("session_role")
	if err != nil {
		return ""
	}
	return cookie.Value
}

func getEmail(r *http.Request) string {
	cookie, err := r.Cookie("session_email")
	if err != nil {
		return ""
	}
	return cookie.Value
}

func isAdmin(r *http.Request) bool {
	return getRole(r) == "admin"
}

func canAddProjects(role string) bool {
	return role == "admin" || role == "accountant" || role == "director"
}

func canEditTimesheet(role string) bool {
	return role == "accountant" || role == "director" || role == "chief"
}

func canEditItems(role string) bool {
	return role == "admin" || role == "accountant" || role == "director" || role == "chief" || role == "storekeeper"
}

func canEditWorkers(role string) bool {
	return role == "admin" || role == "accountant" || role == "director" || role == "chief"
}

func canEditUsers(role string) bool {
	return role == "admin"
}

// ---------- ЛОГИН ----------

func handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Метод не разрешён", 405)
		return
	}
	r.ParseForm()
	email := strings.TrimSpace(r.FormValue("email"))
	password := strings.TrimSpace(r.FormValue("password"))

	if email == "" || password == "" {
		http.Error(w, "Введите почту и пароль", 400)
		return
	}
	users := loadUsers()
	user, ok := users[email]
	if !ok {
		http.Error(w, "Вас нет в системе", 401)
		return
	}
	if !user.Active {
		http.Error(w, "Учётная запись отключена", 403)
		return
	}
	if user.PasswordHash != hashPassword(password) {
		http.Error(w, "Неверный пароль", 401)
		return
	}

	user.LastLogin = time.Now().UTC().Format(time.RFC3339)
	users[email] = user
	saveUsers(users)

	http.SetCookie(w, &http.Cookie{
		Name:     "session_email",
		Value:    email,
		Path:     "/",
		MaxAge:   86400,
		HttpOnly: true,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     "session_role",
		Value:    user.Role,
		Path:     "/",
		MaxAge:   86400,
		HttpOnly: true,
	})

	log.Printf("✅ Вход: %s (%s)", email, user.Name)
	w.Write([]byte("Добро пожаловать, " + user.Name))
}

// ---------- SETUP ----------

func handleSetupPage(w http.ResponseWriter, r *http.Request) {
	cfg := getServerConfig()
	if cfg.Initialized {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	http.ServeFile(w, r, BASE+"/web/static/setup.html")
}

func handleSetupSave(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Метод не разрешён", 405)
		return
	}
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		http.Error(w, "Ошибка формы", 400)
		return
	}
	name := strings.TrimSpace(r.FormValue("name"))
	inn := strings.TrimSpace(r.FormValue("inn"))
	adminEmail := strings.TrimSpace(r.FormValue("admin_email"))
	yandexToken := strings.TrimSpace(r.FormValue("yandex_token"))
	if name == "" || inn == "" || adminEmail == "" || yandexToken == "" {
		http.Error(w, "Заполни обязательные поля", 400)
		return
	}
	emails := []string{adminEmail, r.FormValue("accountant_email"),
		r.FormValue("storekeeper_email"), r.FormValue("chief_email"),
		r.FormValue("director_email")}
	for _, e := range emails {
		if e != "" && !strings.HasSuffix(e, "@yandex.ru") {
			http.Error(w, "Все почты должны быть на @yandex.ru", 400)
			return
		}
	}
	if err := os.WriteFile(BASE+"/.yandex_token", []byte(yandexToken), 0600); err != nil {
		http.Error(w, "Ошибка сохранения токена", 500)
		return
	}
	org := Organization{
		Name:       name,
		INN:        inn,
		Director:   strings.TrimSpace(r.FormValue("director_name")),
		Accountant: strings.TrimSpace(r.FormValue("accountant_name")),
		Logo:       "",
		CreatedAt:  time.Now().UTC().Format(time.RFC3339),
		Registered: true,
	}
	file, header, err := r.FormFile("logo")
	if err == nil && file != nil {
		defer file.Close()
		ext := strings.ToLower(filepath.Ext(header.Filename))
		if ext == "" {
			ext = ".png"
		}
		logoPath := BASE + "/logos/logo" + ext
		out, err := os.Create(logoPath)
		if err == nil {
			io.Copy(out, file)
			out.Close()
			org.Logo = "logos/logo" + ext
		}
	}
	if err := saveJSON(BASE+"/organization.json", org); err != nil {
		http.Error(w, "Ошибка сохранения организации", 500)
		return
	}
	users := map[string]User{}
	addUser := func(email, name, role, tab string) {
		if email == "" {
			return
		}
		users[email] = User{
			Name:         name,
			Role:         role,
			TabNumber:    tab,
			Confirmed:    role == "admin",
			Active:       true,
			PasswordHash: hashPassword("12345"),
		}
	}
	addUser(r.FormValue("accountant_email"), r.FormValue("accountant_name"), "accountant", r.FormValue("accountant_tab"))
	addUser(r.FormValue("storekeeper_email"), r.FormValue("storekeeper_name"), "storekeeper", r.FormValue("storekeeper_tab"))
	addUser(r.FormValue("chief_email"), r.FormValue("chief_name"), "chief", r.FormValue("chief_tab"))
	addUser(r.FormValue("director_email"), r.FormValue("director_name"), "director", r.FormValue("director_tab"))
	addUser(adminEmail, r.FormValue("admin_name"), "admin", r.FormValue("admin_tab"))
	if err := saveUsers(users); err != nil {
		http.Error(w, "Ошибка сохранения пользователей", 500)
		return
	}
	cfg := ServerConfig{
		Version:       "1.0",
		Initialized:   true,
		Organization:  "organization.json",
		CreatedAt:     time.Now().UTC().Format(time.RFC3339),
		MaxConcurrent: 3,
		Blacklist:     map[string]interface{}{"inns": []string{}, "notes": map[string]string{}},
	}
	if mc := r.FormValue("max_concurrent"); mc != "" {
		var n int
		fmt.Sscanf(mc, "%d", &n)
		if n > 0 {
			cfg.MaxConcurrent = n
		}
	}
	if err := saveJSON(BASE+"/server_config.json", cfg); err != nil {
		http.Error(w, "Ошибка сохранения конфига", 500)
		return
	}
	w.Write([]byte("Настройка завершена! Перенаправление..."))
}
// ---------- КТО Я ----------

func handleMe(w http.ResponseWriter, r *http.Request) {
	email := getEmail(r)
	role := getRole(r)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(map[string]string{
		"email": email,
		"role":  role,
	})
}

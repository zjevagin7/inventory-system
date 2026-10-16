package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

type DeviceToken struct {
	Token     string `json:"token"`
	Name      string `json:"name"`
	Project   string `json:"project"`
	CreatedAt string `json:"created_at"`
	CreatedBy string `json:"created_by"`
	Active    bool   `json:"active"`
	LastUsed  string `json:"last_used"`
}

type DeviceTokensFile struct {
	Tokens []DeviceToken `json:"tokens"`
}

func loadDeviceTokens() (*DeviceTokensFile, error) {
	path := BASE + "/device_tokens.json"
	var dtf DeviceTokensFile
	if err := loadJSON(path, &dtf); err != nil {
		dtf.Tokens = []DeviceToken{}
	}
	if dtf.Tokens == nil {
		dtf.Tokens = []DeviceToken{}
	}
	return &dtf, nil
}

func saveDeviceTokens(dtf *DeviceTokensFile) error {
	path := BASE + "/device_tokens.json"
	return saveJSON(path, dtf)
}

func generateToken() string {
	b := make([]byte, 16)
	rand.Read(b)
	return "dev_" + hex.EncodeToString(b)
}

func checkDeviceToken(token string) (*DeviceToken, bool) {
	dtf, _ := loadDeviceTokens()
	for i := range dtf.Tokens {
		if dtf.Tokens[i].Token == token && dtf.Tokens[i].Active {
			return &dtf.Tokens[i], true
		}
	}
	return nil, false
}

// ---------- API АДМИНКИ ----------

func handleDeviceTokensList(w http.ResponseWriter, r *http.Request) {
	if !isAdmin(r) {
		http.Error(w, "Доступ запрещён", http.StatusForbidden)
		return
	}
	dtf, _ := loadDeviceTokens()
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(dtf)
}

func handleDeviceTokenCreate(w http.ResponseWriter, r *http.Request) {
	if !isAdmin(r) {
		http.Error(w, "Доступ запрещён", http.StatusForbidden)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "Метод не разрешён", 405)
		return
	}
	r.ParseForm()
	name := strings.TrimSpace(r.FormValue("name"))
	project := strings.TrimSpace(r.FormValue("project"))

	if name == "" {
		http.Error(w, "Введите название устройства", 400)
		return
	}

	dtf, _ := loadDeviceTokens()
	newToken := DeviceToken{
		Token:     generateToken(),
		Name:      name,
		Project:   project,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
		CreatedBy: getEmail(r),
		Active:    true,
		LastUsed:  "",
	}
	dtf.Tokens = append(dtf.Tokens, newToken)

	if err := saveDeviceTokens(dtf); err != nil {
		http.Error(w, "Ошибка сохранения", 500)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(newToken)
}

func handleDeviceTokenDelete(w http.ResponseWriter, r *http.Request) {
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
		http.Error(w, "Не указан токен", 400)
		return
	}

	dtf, _ := loadDeviceTokens()
	found := false
	newList := []DeviceToken{}
	for _, dt := range dtf.Tokens {
		if dt.Token == token {
			found = true
			continue
		}
		newList = append(newList, dt)
	}
	if !found {
		http.Error(w, "Токен не найден", 404)
		return
	}
	dtf.Tokens = newList
	if err := saveDeviceTokens(dtf); err != nil {
		http.Error(w, "Ошибка сохранения", 500)
		return
	}
	w.Write([]byte("Удалён"))
}

func handleDeviceTokenToggle(w http.ResponseWriter, r *http.Request) {
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
		http.Error(w, "Не указан токен", 400)
		return
	}

	dtf, _ := loadDeviceTokens()
	found := false
	for i := range dtf.Tokens {
		if dtf.Tokens[i].Token == token {
			dtf.Tokens[i].Active = !dtf.Tokens[i].Active
			found = true
			break
		}
	}
	if !found {
		http.Error(w, "Токен не найден", 404)
		return
	}
	if err := saveDeviceTokens(dtf); err != nil {
		http.Error(w, "Ошибка сохранения", 500)
		return
	}
	w.Write([]byte("Обновлён"))
}

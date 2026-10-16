package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const PYTHON_EXPORT_URL = "http://127.0.0.1:8081"

// ---------- ВЫГРУЗКА СКЛАДА (ссылка на Диск) ----------
func handleItemsExport(w http.ResponseWriter, r *http.Request) {
	projectID := r.URL.Query().Get("project")
	warehouseID := r.URL.Query().Get("warehouse")
	if projectID == "" || warehouseID == "" {
		http.Error(w, "Не указан проект или склад", 400)
		return
	}

	payload := map[string]string{
		"project":   projectID,
		"warehouse": warehouseID,
	}
	body, _ := json.Marshal(payload)

	resp, err := http.Post(PYTHON_EXPORT_URL+"/export/warehouse", "application/json", bytes.NewReader(body))
	if err != nil {
		http.Error(w, "Python-сервис недоступен: "+err.Error(), 502)
		return
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		http.Error(w, "Ошибка: "+string(respBody), resp.StatusCode)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Write(respBody)
}

// ---------- ВЫГРУЗКА ТАБЕЛЯ (ссылка на Диск) ----------
func handleTimesheetExport(w http.ResponseWriter, r *http.Request) {
	projectID := r.URL.Query().Get("project")
	month := r.URL.Query().Get("month")
	if projectID == "" || month == "" {
		http.Error(w, "Не указан проект или месяц", 400)
		return
	}

	payload := map[string]string{
		"project": projectID,
		"month":   month,
	}
	body, _ := json.Marshal(payload)

	resp, err := http.Post(PYTHON_EXPORT_URL+"/export/timesheet", "application/json", bytes.NewReader(body))
	if err != nil {
		http.Error(w, "Python-сервис недоступен: "+err.Error(), 502)
		return
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		http.Error(w, "Ошибка: "+string(respBody), resp.StatusCode)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Write(respBody)
}

// ---------- СБОРКА ZIP ПРОЕКТА (ссылка на Диск) ----------
func handleBackupCollect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Метод не разрешён", 405)
		return
	}

	role := getRole(r)
	if role != "admin" {
		http.Error(w, "Доступ запрещён", http.StatusForbidden)
		return
	}

	r.ParseForm()
	projectID := strings.TrimSpace(r.FormValue("project"))
	if projectID == "" {
		http.Error(w, "Не указан проект", 400)
		return
	}

	projectDir := filepath.Join(BASE, "projects", projectID)
	if _, err := os.Stat(projectDir); err != nil {
		http.Error(w, "Проект не найден", 404)
		return
	}

	now := time.Now()
	filename := fmt.Sprintf("inventory_%s_%s.zip", projectID, now.Format("2006-01-02_150405"))

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	err := filepath.Walk(projectDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		relPath, err := filepath.Rel(projectDir, path)
		if err != nil {
			return err
		}
		zipPath := filepath.Join(projectID, relPath)

		fw, err := zw.Create(zipPath)
		if err != nil {
			return err
		}
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()
		_, err = io.Copy(fw, f)
		return err
	})

	if err != nil {
		zw.Close()
		http.Error(w, "Ошибка сборки ZIP: "+err.Error(), 500)
		return
	}
	if err := zw.Close(); err != nil {
		http.Error(w, "Ошибка закрытия ZIP: "+err.Error(), 500)
		return
	}

	var reqBuf bytes.Buffer
	mw := multipart.NewWriter(&reqBuf)
	mw.WriteField("project", projectID)
	mw.WriteField("filename", filename)
	fw, _ := mw.CreateFormFile("file", filename)
	fw.Write(buf.Bytes())
	mw.Close()

	req, err := http.NewRequest("POST", PYTHON_EXPORT_URL+"/export/upload_to_yadisk", &reqBuf)
	if err != nil {
		http.Error(w, "Ошибка создания запроса: "+err.Error(), 500)
		return
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, "Python-сервис недоступен: "+err.Error(), 502)
		return
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		http.Error(w, "Ошибка загрузки на Яндекс.Диск: "+string(respBody), resp.StatusCode)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Write(respBody)
}

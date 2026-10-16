package main

import (
	"log"
	"net/http"
)

const (
	Port = "8080"
	BASE = "/data/data/com.termux/files/home/inventory"
)

func main() {
	// Основные страницы
	http.HandleFunc("/", handleIndex)
	http.HandleFunc("/app", handleApp)
	http.HandleFunc("/admin", handleAdmin)
	http.HandleFunc("/setup", handleSetupPage)

	// API авторизации
	http.HandleFunc("/api/login", handleLogin)
	http.HandleFunc("/api/setup", handleSetupSave)
	http.HandleFunc("/api/organization", handleOrganization)
	http.HandleFunc("/api/projects", handleProjects)
	http.HandleFunc("/api/me", handleMe)

	// API админки
	http.HandleFunc("/api/admin/users", handleAdminUsers)
	http.HandleFunc("/api/admin/users/update", handleAdminUserUpdate)
	http.HandleFunc("/api/admin/users/delete", handleAdminUserDelete)
	http.HandleFunc("/api/admin/users/add", handleAdminUserAdd)
	http.HandleFunc("/api/admin/projects/list", handleAdminProjectsList)
	http.HandleFunc("/api/admin/projects/add", handleAdminProjectAdd)
	http.HandleFunc("/api/admin/projects/delete", handleAdminProjectDelete)
	http.HandleFunc("/api/admin/projects/rename", handleAdminProjectRename)
	http.HandleFunc("/api/admin/backups", handleAdminBackups)
	http.HandleFunc("/api/admin/backup/create", handleAdminBackupCreate)
	http.HandleFunc("/api/admin/token/get", handleAdminTokenGet)
	http.HandleFunc("/api/admin/token/set", handleAdminTokenSet)
	// API складов
	http.HandleFunc("/api/warehouses", handleWarehousesList)
	http.HandleFunc("/api/warehouses/add", handleWarehouseAdd)
	http.HandleFunc("/api/warehouses/delete", handleWarehouseDelete)
	http.HandleFunc("/api/warehouses/rename", handleWarehouseRename)
	// API предметов
	http.HandleFunc("/api/items", handleItemsList)
	http.HandleFunc("/api/items/add", handleItemAdd)
	http.HandleFunc("/api/items/delete", handleItemDelete)
	http.HandleFunc("/api/items/update", handleItemUpdate)

	// API работников
	http.HandleFunc("/api/workers", handleWorkersList)
	http.HandleFunc("/api/workers/add", handleWorkerAdd)
	http.HandleFunc("/api/workers/delete", handleWorkerDelete)
	http.HandleFunc("/api/workers/transfer", handleWorkerTransfer)

	// API выгрузки
	http.HandleFunc("/api/items/export", handleItemsExport)
	http.HandleFunc("/api/timesheet/export", handleTimesheetExport)

	// API бэкапа проекта (ZIP на Яндекс.Диск)
	http.HandleFunc("/api/backup/collect", handleBackupCollect)

	// API табеля
	http.HandleFunc("/api/timesheet", handleTimesheetGet)
	http.HandleFunc("/api/timesheet/set", handleTimesheetSet)
	http.HandleFunc("/api/device/punch", handleDevicePunch)

	// Статика
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir(BASE+"/web/static"))))

	log.Printf("Сервер запущен на http://0.0.0.0:%s", Port)
	log.Fatal(http.ListenAndServe(":"+Port, nil))
}

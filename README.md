# Inventory System

**⚠️ ALPHA VERSION — НЕ ДЛЯ PRODUCTION ⚠️**

**Author:** P.o.p.o.v.a.m
**License:** MIT
**Year:** 2026

---

## О проекте

Система учёта инвентаря для строительных организаций и газового оборудования.
Работает на Android (Termux) и Debian (одноплатники).

## Функции

- Учёт проектов, складов, предметов, работников.
- Роли: admin, accountant, director, chief, storekeeper, worker.
- Табель учёта рабочего времени.
- Выгрузка в `.xls` для 1С.
- Синхронизация через Яндекс.Диск.
- Работа при белых списках.

## Компоненты

### Сервер ( Go )
- `web/main.go` — точка входа.
- `web/handlers.go` — базовые маршруты.
- `web/admin_handlers.go` — админские API.
- `web/auth.go` — авторизация и роли.
- `web/items.go` — предметы.
- `web/warehouses.go` — склады.
- `web/workers.go` — работники.
- `web/timesheet.go` — табель.
- `web/export.go` — выгрузки.
- `web/device_tokens.go` — токены устройств.

### Выгрузки ( Python )
- `export_service.py` — формирование `.xls` через Яндекс.Диск.

### Синхронизация ( Python )
- `sync.py` — выгрузка на Яндекс.Диск.
- `download.py` — скачивание с Яндекс.Диска.
- `inventory.py` — общие функции.

## Установка

### Android ( Termux )
См. `deploy/android-termux/`

### Debian
См. `deploy/debian/`

## Лицензия

MIT — см. `LICENSE`.

---

**Author:** P.o.p.o.v.a.m
**Build:** 2026-10-16
**Status:** ALPHA

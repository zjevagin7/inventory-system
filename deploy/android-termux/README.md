# Deploy: Android / Termux

ALPHA - Inventory System
Author: P.o.p.o.v.a.m

## Требования

- Termux (из F-Droid)
- Termux:Boot (из F-Droid) - для автозапуска
- Go, Python, git

## Установка

### 1. Установить зависимости

pkg install go python git -y
pip install flask xlwt yadisk pillow

### 2. Скопировать код

mkdir -p ~/inventory
cd ~/inventory
Скопировать файлы из репозитория.

### 3. Создать конфиги

- users.json - на основе users.example.json
- server_config.json - на основе server_config.example.json
- organization.json - на основе organization.example.json
- .yandex_token - вставить свой токен
- device_tokens.json - на основе device_tokens.example.json

### 4. Собрать Go-сервер

cd ~/inventory/web
go build -o inventory-web

### 5. Скопировать скрипты автозапуска

Скопировать start-*.sh в ~/.termux/boot/

### 6. Запустить

cd ~/inventory/web
./inventory-web

## Автозапуск

Скрипты из ~/.termux/boot/ запускаются через Termux:Boot.

## Файлы

- start-web.sh - Go-сервер
- start-export.sh - Python-сервис выгрузок
- start-sync.sh - синхронизация с Яндекс.Диском
- start-download.sh - скачивание с Яндекс.Диска

# Deploy: Debian / Одноплатник

ALPHA - Inventory System
Author: P.o.p.o.v.a.m

## Требования

- Debian 11+ (или Ubuntu 20.04+)
- Go 1.20+
- Python 3.9+
- systemd

## Установка

### 1. Установить зависимости

apt update
apt install golang python3 python3-pip git -y
pip3 install flask xlwt yadisk pillow

### 2. Скопировать код в /opt/inventory

mkdir -p /opt/inventory
Скопировать файлы из репозитория в /opt/inventory

### 3. Создать конфиги

- users.json - на основе users.example.json
- server_config.json - на основе server_config.example.json
- organization.json - на основе organization.example.json
- .yandex_token - вставить свой токен
- device_tokens.json - на основе device_tokens.example.json

### 4. Собрать Go-сервер

cd /opt/inventory/web
go build -o inventory-web

### 5. Установить systemd-юниты

Скопировать файлы из systemd/ в /etc/systemd/system/
systemctl daemon-reload
systemctl enable inventory-web inventory-export
systemctl start inventory-web inventory-export

## Файлы systemd

- inventory-web.service - Go-сервер
- inventory-export.service - Python-сервис выгрузок
- inventory-sync.service - синхронизация (timer)
- inventory-sync.timer - таймер

## Отличия от Android

- Пути: /opt/inventory вместо ~/inventory
- Автозапуск: systemd вместо Termux:Boot
- Пользователь: root или отдельный user

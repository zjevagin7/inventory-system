#!/usr/bin/env python3
"""
Обратная синхронизация: скачивание с Яндекс.Диска.
"""

import os
import sys
import json
import requests
from datetime import datetime, timezone

sys.path.insert(0, os.path.expanduser("~/inventory"))
from inventory import load_json, save_json

BASE = os.path.expanduser("~/inventory")
TOKEN_FILE = os.path.join(BASE, ".yandex_token")
SYNC_DIR = os.path.join(BASE, "sync")
YANDEX_API = "https://cloud-api.yandex.net/v1/disk"


def _now():
    return datetime.now(timezone.utc).isoformat()


def get_token():
    with open(TOKEN_FILE) as f:
        return f.read().strip()


def list_remote(token, path):
    """Список файлов в папке на Яндекс.Диске."""
    url = f"{YANDEX_API}/resources"
    headers = {"Authorization": f"OAuth {token}"}
    params = {"path": path, "limit": 1000}
    resp = requests.get(url, headers=headers, params=params)
    if resp.status_code != 200:
        print(f"❌ Список {path}: {resp.status_code} — {resp.text}")
        return []
    items = resp.json().get("_embedded", {}).get("items", [])
    return items


def download_file(token, remote_path, local_path):
    """Скачивает файл с Яндекс.Диска."""
    url = f"{YANDEX_API}/resources/download"
    headers = {"Authorization": f"OAuth {token}"}
    params = {"path": remote_path}

    resp = requests.get(url, headers=headers, params=params)
    if resp.status_code != 200:
        print(f"❌ URL для {remote_path}: {resp.status_code}")
        return False

    download_url = resp.json()["href"]
    with requests.get(download_url, stream=True) as r:
        if r.status_code != 200:
            print(f"❌ Скачивание {remote_path}: {r.status_code}")
            return False
        os.makedirs(os.path.dirname(local_path), exist_ok=True)
        with open(local_path, "wb") as f:
            for chunk in r.iter_content(1024 * 1024):
                f.write(chunk)

    print(f"  ✅ {remote_path} → {local_path}")
    return True


def sync_download():
    token = get_token()
    print(f"🔑 Токен: {token[:10]}...")

    # 1. Смотрим манифест на Диске
    manifest_path = "/inventory/sync/sync_manifest.json"
    local_manifest = os.path.join(SYNC_DIR, "sync_manifest.json")

    # Скачиваем манифест
    if not download_file(token, manifest_path, local_manifest):
        print("⚠️ Манифест не скачался")
        return

    # 2. Читаем манифест
    manifest = load_json(local_manifest)
    if not manifest:
        print("⚠️ Манифест пустой")
        return

    print(f"📦 Частей: {manifest.get('parts_count', 0)}")
    print(f"📦 Создан: {manifest.get('created_at')}")

    # 3. Скачиваем части
    for part in manifest.get("parts", []):
        part_name = part["name"]
        remote_path = f"/inventory/sync/{part_name}"
        local_path = os.path.join(SYNC_DIR, part_name)

        if os.path.exists(local_path):
            size = os.path.getsize(local_path)
            if size == part["size"]:
                print(f"  ⏭️ {part_name} уже есть")
                continue

        download_file(token, remote_path, local_path)

    # 4. Обновляем sync_state
    state = load_json(os.path.join(BASE, "sync_state.json")) or {}
    state["last_download"] = _now()
    state["status"] = "ok"
    save_json(os.path.join(BASE, "sync_state.json"), state)

    print(f"\n✅ Скачивание завершено")


if __name__ == "__main__":
    sync_download()

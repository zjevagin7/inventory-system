#!/usr/bin/env python3
"""
Синхронизация с Яндекс.Диском.
Собирает всё в tar.gz, режет по 2.5 ГБ, заливает, пишет манифест.
"""

import os
import sys
import json
import tarfile
import hashlib
import requests
from datetime import datetime, timezone

sys.path.insert(0, os.path.expanduser("~/inventory"))
from inventory import load_json, save_json

BASE = os.path.expanduser("~/inventory")
TOKEN_FILE = os.path.join(BASE, ".yandex_token")
SYNC_DIR = os.path.join(BASE, "sync")
PART_SIZE = 2_500_000_000  # 2.5 ГБ
YANDEX_API = "https://cloud-api.yandex.net/v1/disk"


def _now():
    return datetime.now(timezone.utc).isoformat()


def get_token():
    with open(TOKEN_FILE) as f:
        return f.read().strip()


def ensure_folder(token, path):
    """Создаёт папку на Яндекс.Диске."""
    url = f"{YANDEX_API}/resources"
    headers = {"Authorization": f"OAuth {token}"}
    params = {"path": path}
    resp = requests.put(url, headers=headers, params=params)
    if resp.status_code in (201, 409):
        return True
    print(f"⚠️ Папка {path}: {resp.status_code} — {resp.text}")
    return False


def upload_file(token, local_path, remote_path):
    """Заливает файл на Яндекс.Диск."""
    url = f"{YANDEX_API}/resources/upload"
    headers = {"Authorization": f"OAuth {token}"}
    params = {"path": remote_path, "overwrite": "true"}

    resp = requests.get(url, headers=headers, params=params)
    if resp.status_code != 200:
        print(f"❌ URL для {remote_path}: {resp.status_code}")
        return False

    upload_url = resp.json()["href"]
    with open(local_path, "rb") as f:
        up = requests.put(upload_url, files={"file": f})

    if up.status_code in (200, 201):
        print(f"  ✅ {remote_path}")
        return True

    print(f"❌ Загрузка {remote_path}: {up.status_code}")
    return False


def sha256_file(path):
    h = hashlib.sha256()
    with open(path, "rb") as f:
        for chunk in iter(lambda: f.read(1024 * 1024), b""):
            h.update(chunk)
    return h.hexdigest()


def collect_files():
    """Собирает список файлов для синхронизации."""
    files = []

    # Корневые JSON
    for name in ["config.json", "organization.json", "clients.json",
                 "people.json", "users.json", "notifications.json"]:
        p = os.path.join(BASE, name)
        if os.path.exists(p):
            files.append((p, name))

    # Проекты
    projects_dir = os.path.join(BASE, "projects")
    if os.path.isdir(projects_dir):
        for proj in sorted(os.listdir(projects_dir)):
            proj_dir = os.path.join(projects_dir, proj)
            if not os.path.isdir(proj_dir):
                continue
            for name in ["meta.json", "items.json", "history.json"]:
                p = os.path.join(proj_dir, name)
                if os.path.exists(p):
                    files.append((p, f"projects/{proj}/{name}"))

    return files


def build_archive():
    os.makedirs(SYNC_DIR, exist_ok=True)
    stamp = datetime.now(timezone.utc).strftime("%Y-%m-%d_%H%M%S")
    archive_path = os.path.join(SYNC_DIR, f"sync_{stamp}.tar.gz")

    files = collect_files()
    print(f"📦 Файлов: {len(files)}")

    # Считаем хэши и пишем manifest
    manifest_lines = []
    for local, remote in files:
        h = sha256_file(local)
        manifest_lines.append(f"{h}  {remote}")

    manifest_txt = os.path.join(SYNC_DIR, "backup_manifest.txt")
    with open(manifest_txt, "w") as f:
        f.write("\n".join(manifest_lines) + "\n")
    print(f"📋 Manifest: {manifest_txt} ({len(manifest_lines)} строк)")

    # Собираем архив (включая manifest)
    with tarfile.open(archive_path, "w:gz") as tar:
        for local, remote in files:
            tar.add(local, arcname=remote)
        tar.add(manifest_txt, arcname="backup_manifest.txt")

    size = os.path.getsize(archive_path)
    print(f"📦 Архив: {os.path.basename(archive_path)}")
    print(f"📦 Размер: {size / 1024:.2f} КБ")

    return archive_path, files

def split_archive(archive_path):
    size = os.path.getsize(archive_path)
    if size <= PART_SIZE:
        return [archive_path]

    parts = []
    with open(archive_path, "rb") as f:
        part_num = 1
        while True:
            chunk = f.read(PART_SIZE)
            if not chunk:
                break
            part_path = f"{archive_path}.part_{part_num:02d}"
            with open(part_path, "wb") as pf:
                pf.write(chunk)
            parts.append(part_path)
            print(f"  📦 Часть {part_num}: {os.path.getsize(part_path) / 1024 / 1024:.2f} МБ")
            part_num += 1
    return parts


def sync():
    token = get_token()
    print(f"🔑 Токен: {token[:10]}...")

    ensure_folder(token, "/inventory")
    ensure_folder(token, "/inventory/sync")

    archive_path, files = build_archive()
    parts = split_archive(archive_path)

    remote_parts = []
    for part in parts:
        remote_name = f"/inventory/sync/{os.path.basename(part)}"
        if upload_file(token, part, remote_name):
            remote_parts.append({
                "name": os.path.basename(part),
                "size": os.path.getsize(part),
                "sha256": sha256_file(part)
            })

    manifest = {
        "created_at": _now(),
        "parts_count": len(remote_parts),
        "parts": remote_parts,
        "files_count": len(files),
        "total_size": sum(p["size"] for p in remote_parts)
    }

    manifest_path = os.path.join(SYNC_DIR, "sync_manifest.json")
    save_json(manifest_path, manifest)
    upload_file(token, manifest_path, "/inventory/sync/sync_manifest.json")

    state = load_json(os.path.join(BASE, "sync_state.json")) or {}
    state["last_sync"] = _now()
    state["last_upload"] = _now()
    state["status"] = "ok"
    state.setdefault("yandex", {})
    state["yandex"]["files_synced"] = len(remote_parts) + 1
    save_json(os.path.join(BASE, "sync_state.json"), state)

    # Чистим старые архивы (оставляем только последний)
    all_archives = sorted([
        f for f in os.listdir(SYNC_DIR)
        if f.startswith("sync_") and f.endswith(".tar.gz")
    ])
    for old in all_archives[:-1]:
        os.remove(os.path.join(SYNC_DIR, old))
        print(f"  🗑️ Удалён старый: {old}")
    print(f"\n✅ Синхронизация завершена. Частей: {len(remote_parts)}")


if __name__ == "__main__":
    sync()

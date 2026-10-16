#!/usr/bin/env python3
"""
Библиотека для работы с данными инвентаря.
Читает и пишет JSON-файлы, ничего больше.
"""

import json
import os
import shutil
from datetime import datetime, timezone

BASE = os.path.expanduser("~/inventory")


def load_json(path):
    if not os.path.exists(path):
        return None
    with open(path, "r", encoding="utf-8") as f:
        return json.load(f)


def save_json(path, data):
    os.makedirs(os.path.dirname(path), exist_ok=True)
    with open(path, "w", encoding="utf-8") as f:
        json.dump(data, f, ensure_ascii=False, indent=2)


def _now():
    return datetime.now(timezone.utc).isoformat()


# ---------- КОНФИГ ----------
def get_config():
    return load_json(os.path.join(BASE, "config.json"))


# ---------- ОРГАНИЗАЦИЯ ----------
def get_organization():
    return load_json(os.path.join(BASE, "organization.json")) or {
        "name": "", "inn": "", "logo": None, "created_at": None
    }


def set_organization(name, inn, logo=None):
    data = {
        "name": name, "inn": inn, "logo": logo,
        "created_at": _now()
    }
    save_json(os.path.join(BASE, "organization.json"), data)
    return data


def is_organization_set():
    org = get_organization()
    return bool(org.get("name") and org.get("inn"))


# ---------- КЛИЕНТЫ ----------
def get_clients():
    return load_json(os.path.join(BASE, "clients.json")) or {}


def get_client(email):
    return get_clients().get(email)


def add_client(email, name, role, projects):
    clients = get_clients()
    clients[email] = {
        "name": name, "role": role, "projects": projects,
        "created_at": _now(), "last_sync": None, "active": True
    }
    save_json(os.path.join(BASE, "clients.json"), clients)


# ---------- ЛЮДИ ----------
def get_people():
    return load_json(os.path.join(BASE, "people.json")) or {}


# ---------- ПРОЕКТЫ ----------
def list_projects():
    projects_dir = os.path.join(BASE, "projects")
    if not os.path.isdir(projects_dir):
        return []
    return sorted([
        d for d in os.listdir(projects_dir)
        if os.path.isdir(os.path.join(projects_dir, d))
    ])


def get_project_meta(project_id):
    return load_json(os.path.join(BASE, "projects", project_id, "meta.json"))


def get_project_items(project_id):
    data = load_json(os.path.join(BASE, "projects", project_id, "items.json"))
    return data.get("items", []) if data else []


def get_project_history(project_id):
    data = load_json(os.path.join(BASE, "projects", project_id, "history.json"))
    return data.get("events", []) if data else []


def create_project(name):
    config = get_config()
    max_projects = config.get("max_projects", 3) if config else 3
    active = list_projects()
    if len(active) >= max_projects:
        return None

    used_ids = set(active)
    pid = None
    for i in range(1, max_projects + 1):
        candidate = f"project_{i}"
        if candidate not in used_ids:
            pid = candidate
            break
    if pid is None:
        return None

    base = os.path.join(BASE, "projects", pid)
    os.makedirs(base, exist_ok=True)

    org = get_organization()
    save_json(os.path.join(base, "meta.json"), {
        "id": pid,
        "name": name,
        "status": "active",
        "created_at": _now(),
        "closed_at": None,
        "organization": {
            "name": org.get("name", ""),
            "inn": org.get("inn", ""),
            "logo": org.get("logo")
        }
    })
    save_json(os.path.join(base, "items.json"), {"items": []})
    save_json(os.path.join(base, "history.json"), {"events": []})

    return pid


def close_project(project_id, who):
    src = os.path.join(BASE, "projects", project_id)
    if not os.path.isdir(src):
        return False

    meta = get_project_meta(project_id)
    if not meta:
        return False

    date = datetime.now(timezone.utc).strftime("%Y-%m-%d")
    safe_name = "".join(c for c in meta["name"] if c.isalnum() or c in " -_").strip()
    safe_name = safe_name.replace(" ", "_") or project_id
    archive_name = f"{date}_{safe_name}"

    dst = os.path.join(BASE, "archive", archive_name)
    if os.path.exists(dst):
        return False

    meta["status"] = "archived"
    meta["closed_at"] = _now()
    save_json(os.path.join(src, "meta.json"), meta)

    shutil.move(src, dst)
    return archive_name


def rename_project(project_id, new_name):
    path = os.path.join(BASE, "projects", project_id, "meta.json")
    meta = load_json(path)
    if not meta:
        return False
    meta["name"] = new_name
    save_json(path, meta)
    return True


def list_archive():
    archive_dir = os.path.join(BASE, "archive")
    if not os.path.isdir(archive_dir):
        return []
    return sorted(os.listdir(archive_dir))


# ---------- ПРЕДМЕТЫ ----------
def add_item(project_id, name, qty, serial, condition, holder, photo=None):
    path = os.path.join(BASE, "projects", project_id, "items.json")
    data = load_json(path) or {"items": []}

    new_id = f"item_{len(data['items']) + 1:03d}"
    item = {
        "id": new_id,
        "name": name,
        "qty": qty,
        "serial": serial,
        "condition": condition,
        "holder": holder,
        "photo": photo,
        "created_at": _now(),
        "updated_at": _now()
    }
    data["items"].append(item)
    save_json(path, data)

    add_event(project_id, "add", holder, new_id, f"Добавлен: {name}")
    return new_id


def find_item(project_id, item_id):
    for item in get_project_items(project_id):
        if item["id"] == item_id:
            return item
    return None


def delete_item(project_id, item_id, who):
    path = os.path.join(BASE, "projects", project_id, "items.json")
    data = load_json(path) or {"items": []}

    before = len(data["items"])
    data["items"] = [i for i in data["items"] if i["id"] != item_id]
    after = len(data["items"])

    if before == after:
        return False

    save_json(path, data)
    add_event(project_id, "delete", who, item_id, "Удалён")
    return True


def update_item(project_id, item_id, who, **fields):
    path = os.path.join(BASE, "projects", project_id, "items.json")
    data = load_json(path) or {"items": []}

    for item in data["items"]:
        if item["id"] == item_id:
            changed = []
            for key, value in fields.items():
                if key in item and item[key] != value:
                    item[key] = value
                    changed.append(key)
            if not changed:
                return False
            item["updated_at"] = _now()
            save_json(path, data)
            add_event(project_id, "edit", who, item_id,
                      f"Изменено: {', '.join(changed)}")
            return True
    return False


# ---------- ИСТОРИЯ ----------
def add_event(project_id, event, who, what, note=""):
    path = os.path.join(BASE, "projects", project_id, "history.json")
    data = load_json(path) or {"events": []}
    data["events"].append({
        "time": _now(),
        "event": event,
        "who": who,
        "what": what,
        "note": note
    })
    save_json(path, data)


# ---------- ТЕСТ ----------
if __name__ == "__main__":
    print("=== Конфиг ===")
    print(json.dumps(get_config(), ensure_ascii=False, indent=2))

    print("\n=== Организация ===")
    print(json.dumps(get_organization(), ensure_ascii=False, indent=2))

    print("\n=== Клиенты ===")
    for email, data in get_clients().items():
        print(f"  {email} → {data['name']} ({data['role']})")

    print("\n=== Проекты ===")
    for p in list_projects():
        meta = get_project_meta(p)
        print(f"  {p} → {meta['name']} [{meta['status']}]")

#!/usr/bin/env python3
"""
Микросервис выгрузки. ВСЕ файлы — через Яндекс.Диск.
Порт 8081. Принимает JSON, возвращает ссылку на Диск.
"""

import os
import json
import io
import time
import threading
import tempfile
from datetime import datetime
from flask import Flask, request, send_file, jsonify
import xlwt
import yadisk

BASE = os.path.expanduser("~/inventory")
TOKEN_FILE = os.path.join(BASE, ".yandex_token")
app = Flask(__name__)

# Хранилище активных ссылок: {remote_path: expiry_timestamp}
ACTIVE_UPLOADS = {}


def load_json(path):
    if not os.path.exists(path):
        return {}
    with open(path, "r", encoding="utf-8") as f:
        return json.load(f)


def get_client():
    with open(TOKEN_FILE) as f:
        token = f.read().strip()
    return yadisk.Client(token=token)


def style_header():
    style = xlwt.XFStyle()
    font = xlwt.Font()
    font.bold = True
    style.font = font
    return style


def delayed_delete(remote_path, delay=300):
    """Удаляет файл с Яндекс.Диска через delay секунд."""
    time.sleep(delay)
    try:
        client = get_client()
        with client:
            if client.exists(remote_path):
                client.remove(remote_path)
                print(f"🗑️ Удалён по таймеру: {remote_path}")
                ACTIVE_UPLOADS.pop(remote_path, None)
    except Exception as e:
        print(f"❌ Ошибка удаления {remote_path}: {e}")


def upload_to_disk(local_bytes, filename, folder="/backup_auto"):
    """Заливает байты на Яндекс.Диск, возвращает (public_url, remote_path, size)."""
    client = get_client()

    tmp_path = os.path.join(tempfile.gettempdir(), filename)
    with open(tmp_path, "wb") as f:
        f.write(local_bytes)

    size = len(local_bytes)

    with client:
        if not client.exists(folder):
            client.mkdir(folder)

        remote_path = f"{folder}/{filename}"
        client.upload(tmp_path, remote_path, overwrite=True)

        meta = client.get_meta(remote_path)
        if meta.size != size:
            os.remove(tmp_path)
            client.remove(remote_path)
            raise Exception(f"size mismatch: local={size}, remote={meta.size}")

        client.publish(remote_path)
        public_meta = client.get_meta(remote_path)
        public_url = public_meta.public_url if hasattr(public_meta, 'public_url') else None
        if not public_url:
            public_url = client.get_download_link(remote_path)

    os.remove(tmp_path)

    ACTIVE_UPLOADS[remote_path] = time.time() + 300
    timer = threading.Thread(target=delayed_delete, args=(remote_path, 300), daemon=True)
    timer.start()

    return public_url, remote_path, size


# ---------- СКЛАД -> .XLS -> ДИСК ----------
@app.route("/export/warehouse", methods=["POST"])
def export_warehouse():
    data = request.get_json()
    project_id = data.get("project")
    warehouse_id = data.get("warehouse")

    if not project_id or not warehouse_id:
        return jsonify({"error": "no project or warehouse"}), 400

    items_path = os.path.join(BASE, "projects", project_id, "items.json")
    items_data = load_json(items_path)
    all_items = items_data.get("items", [])
    items = [it for it in all_items if it.get("warehouse_id") == warehouse_id]

    wb = xlwt.Workbook(encoding="utf-8")
    ws = wb.add_sheet("Склад")

    headers = ["Наименование", "Характеристика", "Серийный номер", "Ед. изм.",
               "По учёту", "По факту", "Отклонение", "Местонахождение", "Состояние"]
    hs = style_header()
    for col, h in enumerate(headers):
        ws.write(0, col, h, hs)

    for row, it in enumerate(items, start=1):
        ws.write(row, 0, it.get("name", ""))
        ws.write(row, 1, it.get("characteristic", ""))
        ws.write(row, 2, it.get("serial", ""))
        ws.write(row, 3, it.get("unit", ""))
        ws.write(row, 4, it.get("qty_planned", 0))
        ws.write(row, 5, it.get("qty_actual", 0))
        ws.write(row, 6, it.get("deviation", 0))
        ws.write(row, 7, it.get("location_detail", ""))
        ws.write(row, 8, it.get("location", ""))

    for col, w in enumerate([30, 20, 20, 10, 10, 10, 10, 25, 20]):
        ws.col(col).width = 256 * w

    buf = io.BytesIO()
    wb.save(buf)
    local_bytes = buf.getvalue()

    filename = f"sklad_{warehouse_id}_{datetime.now().strftime('%Y-%m-%d_%H%M')}.xls"

    try:
        url, remote_path, size = upload_to_disk(local_bytes, filename)
    except Exception as e:
        return jsonify({"error": str(e)}), 500

    return jsonify({
        "status": "ok",
        "url": url,
        "expires_in": 300,
        "remote_path": remote_path,
        "size": size,
        "filename": filename,
    })


# ---------- ТАБЕЛЬ -> .XLS -> ДИСК ----------
@app.route("/export/timesheet", methods=["POST"])
def export_timesheet():
    data = request.get_json()
    project_id = data.get("project")
    month = data.get("month")

    if not project_id or not month:
        return jsonify({"error": "no project or month"}), 400

    ts_path = os.path.join(BASE, "projects", project_id, "timesheet", f"{month}.json")
    ts_data = load_json(ts_path)
    employees = ts_data.get("employees", [])

    wb = xlwt.Workbook(encoding="utf-8")
    ws = wb.add_sheet("Табель")

    hs = style_header()
    ws.write(0, 0, "Работник", hs)
    for d in range(1, 32):
        ws.write(0, d, d, hs)
    ws.write(0, 32, "Итого", hs)

    for row, emp in enumerate(employees, start=1):
        ws.write(row, 0, f"{emp.get('name', '')} (таб. {emp.get('tab', '')})")
        days = emp.get("days", {})
        notes = emp.get("notes", {})
        for d in range(1, 32):
            day_str = str(d)
            if day_str in days:
                ws.write(row, d, days[day_str])
            elif day_str in notes:
                ws.write(row, d, notes[day_str])
        ws.write(row, 32, emp.get("total_hours", 0))

    ws.col(0).width = 256 * 30
    for d in range(1, 33):
        ws.col(d).width = 256 * 4

    buf = io.BytesIO()
    wb.save(buf)
    local_bytes = buf.getvalue()

    filename = f"timesheet_{project_id}_{month}_{datetime.now().strftime('%H%M')}.xls"

    try:
        url, remote_path, size = upload_to_disk(local_bytes, filename)
    except Exception as e:
        return jsonify({"error": str(e)}), 500

    return jsonify({
        "status": "ok",
        "url": url,
        "expires_in": 300,
        "remote_path": remote_path,
        "size": size,
        "filename": filename,
    })


# ---------- ZIP (БЭКАП ПРОЕКТА) -> ДИСК ----------
@app.route("/export/upload_to_yadisk", methods=["POST"])
def upload_to_yadisk():
    try:
        project_id = request.form.get("project", "").strip()
        filename = request.form.get("filename", "").strip()

        if not project_id or not filename:
            return jsonify({"error": "no project or filename"}), 400

        if "file" not in request.files:
            return jsonify({"error": "no file uploaded"}), 400

        upload_file = request.files["file"]
        local_bytes = upload_file.read()

        if len(local_bytes) < 100:
            return jsonify({"error": "file too small"}), 400

        url, remote_path, size = upload_to_disk(local_bytes, filename)

        return jsonify({
            "status": "ok",
            "url": url,
            "expires_in": 300,
            "remote_path": remote_path,
            "size": size,
        })

    except Exception as e:
        import traceback
        traceback.print_exc()
        return jsonify({"error": str(e)}), 500


# ---------- HEALTHCHECK ----------
@app.route("/health", methods=["GET"])
def health():
    return jsonify({"status": "ok"})


if __name__ == "__main__":
    app.run(host="127.0.0.1", port=8081, debug=False)

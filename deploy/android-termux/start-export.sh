#!/data/data/com.termux/files/usr/bin/sh
# ALPHA - Inventory System
# Author: P.o.p.o.v.a.m
termux-wake-lock
cd ~/inventory
while true; do
  python3 export_service.py >> ~/inventory/export_service.log 2>&1
  sleep 10
done

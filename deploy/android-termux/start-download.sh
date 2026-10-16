#!/data/data/com.termux/files/usr/bin/sh
# ALPHA - Inventory System
# Author: P.o.p.o.v.a.m
termux-wake-lock
while true; do
  python3 ~/inventory/download.py >> ~/inventory/sync.log 2>&1
  sleep 3600
done

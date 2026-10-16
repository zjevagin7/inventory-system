#!/data/data/com.termux/files/usr/bin/sh
# ALPHA - Inventory System
# Author: P.o.p.o.v.a.m
termux-wake-lock
while true; do
  python3 ~/inventory/sync.py >> ~/inventory/sync.log 2>&1
  sleep 1800
done

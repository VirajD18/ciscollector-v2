#!/bin/sh
set -e

CERT_DIR=/etc/klouddbshield/certs
DB_DIR=/etc/klouddbshield/db

mkdir -p "$CERT_DIR" "$DB_DIR"
chmod 700 "$CERT_DIR"
chmod 755 "$DB_DIR"

if command -v systemctl >/dev/null 2>&1; then
  systemctl daemon-reload || true
fi

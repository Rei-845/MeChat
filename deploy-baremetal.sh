#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")" && pwd)"
RUN_USER="${SUDO_USER:-$USER}"

cd "$ROOT/backend"
CGO_ENABLED=0 go build -o bin/mechat-server ./cmd/server

cd "$ROOT/frontend"
npm install
npm run build

sudo rm -rf /var/www/mechat && sudo mkdir -p /var/www/mechat
sudo cp -r "$ROOT/frontend/dist/." /var/www/mechat/
sudo cp "$ROOT/deploy/nginx/mechat.conf" /etc/nginx/conf.d/mechat.conf
sudo rm -f /etc/nginx/sites-enabled/default

sudo tee /etc/systemd/system/mechat.service >/dev/null <<UNIT
[Unit]
Description=MeChat backend
After=network.target

[Service]
User=$RUN_USER
WorkingDirectory=$ROOT/backend
ExecStart=$ROOT/backend/bin/mechat-server -config $ROOT/backend/config/config.yaml
Restart=on-failure
RestartSec=5s

[Install]
WantedBy=multi-user.target
UNIT

sudo systemctl daemon-reload
sudo systemctl enable --now mechat
sudo nginx -t && sudo systemctl reload nginx

echo ">> 部署成功 http://localhost"
echo ">> 日志     journalctl -u mechat -f"

#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/deploy"

log() { printf '\033[32m>>\033[0m %s\n' "$*"; }
die() { printf '\033[31m!!\033[0m %s\n' "$*" >&2; exit 1; }

command -v docker >/dev/null || die "docker未安装"
docker info >/dev/null 2>&1 || die "请启动docker服务或尝试提升权限"

if docker compose version >/dev/null 2>&1; then
  dc="docker compose"
elif command -v docker-compose >/dev/null; then
  dc="docker-compose"
else
  die "找不到 docker compose"
fi

[ -f .env ] || log "未找到.env，将使用默认值"

log "构建并启动中"
$dc up -d --build
$dc ps

ip=$(hostname -I 2>/dev/null | awk '{print $1}') || true
log "部署成功 http://${ip:-你的IP}"
log "查看日志 $dc logs -f server"

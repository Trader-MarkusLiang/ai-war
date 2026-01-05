#!/bin/bash
set -e

cd "$(dirname "$0")"

case "$1" in
  start)
    echo "启动 NOFX 运维面板..."
    docker compose up -d --build
    echo "启动完成! 访问 http://localhost:8801"
    ;;
  stop)
    echo "停止 NOFX 运维面板..."
    docker compose down
    ;;
  restart)
    echo "重启 NOFX 运维面板..."
    docker compose restart
    ;;
  logs)
    docker compose logs -f
    ;;
  status)
    docker compose ps
    ;;
  *)
    echo "用法: $0 {start|stop|restart|logs|status}"
    exit 1
    ;;
esac

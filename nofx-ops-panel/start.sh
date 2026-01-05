#!/bin/bash
# NOFX 运维面板启动脚本

cd "$(dirname "$0")"

case "$1" in
  backend)
    echo "启动后端..."
    cd backend
    source venv/bin/activate 2>/dev/null || python3 -m venv venv && source venv/bin/activate
    pip install -r requirements.txt -q
    uvicorn app.main:app --host 0.0.0.0 --port 8800 --reload
    ;;
  frontend)
    echo "启动前端..."
    cd frontend
    npm install
    npm run dev
    ;;
  install)
    echo "安装依赖..."
    cd backend && python3 -m venv venv && source venv/bin/activate && pip install -r requirements.txt
    cd ../frontend && npm install
    echo "安装完成"
    ;;
  *)
    echo "用法: $0 {backend|frontend|install}"
    echo "  backend  - 启动后端API (端口8800)"
    echo "  frontend - 启动前端页面 (端口8801)"
    echo "  install  - 安装所有依赖"
    ;;
esac

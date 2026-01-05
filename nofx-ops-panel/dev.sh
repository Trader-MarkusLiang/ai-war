#!/bin/bash
# NOFX 运维面板启动脚本（开发模式）

cd "$(dirname "$0")"

case "$1" in
  start)
    echo "启动后端服务..."
    cd backend
    nohup python3 -m uvicorn app.main:app --host 0.0.0.0 --port 8800 > ../backend.log 2>&1 &
    echo $! > ../backend.pid
    cd ..

    echo "启动前端服务..."
    cd frontend
    nohup npm run dev > ../frontend.log 2>&1 &
    echo $! > ../frontend.pid
    cd ..

    echo "启动完成!"
    echo "后端: http://localhost:8800"
    echo "前端: http://localhost:8801"
    ;;

  stop)
    echo "停止服务..."
    [ -f backend.pid ] && kill $(cat backend.pid) 2>/dev/null && rm backend.pid
    [ -f frontend.pid ] && kill $(cat frontend.pid) 2>/dev/null && rm frontend.pid
    echo "服务已停止"
    ;;

  status)
    echo "服务状态:"
    if [ -f backend.pid ] && ps -p $(cat backend.pid) > /dev/null 2>&1; then
      echo "后端: 运行中 (PID: $(cat backend.pid))"
    else
      echo "后端: 未运行"
    fi

    if [ -f frontend.pid ] && ps -p $(cat frontend.pid) > /dev/null 2>&1; then
      echo "前端: 运行中 (PID: $(cat frontend.pid))"
    else
      echo "前端: 未运行"
    fi
    ;;

  logs)
    tail -f backend.log frontend.log
    ;;

  *)
    echo "用法: $0 {start|stop|status|logs}"
    exit 1
    ;;
esac

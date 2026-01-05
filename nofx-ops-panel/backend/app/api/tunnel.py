from fastapi import APIRouter, Depends
from pydantic import BaseModel
import subprocess
import os
import socket
import signal
from ..auth import verify_token

router = APIRouter(prefix="/tunnel", tags=["隧道"])

# 隧道配置
REMOTE_HOST = "root@47.236.159.60"
LOCAL_FRONTEND_PORT = 3333
LOCAL_BACKEND_PORT = 8888
REMOTE_FRONTEND_PORT = 3000
REMOTE_BACKEND_PORT = 8080

def check_port_listening(port):
    """检查端口是否在监听"""
    try:
        sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        sock.settimeout(1)
        result = sock.connect_ex(('localhost', port))
        sock.close()
        return result == 0
    except:
        return False

def get_tunnel_pids():
    """获取SSH隧道进程的真实PID"""
    pids = []
    try:
        # 查找包含端口转发的SSH进程
        result = subprocess.run(
            ["pgrep", "-f", f"ssh.*-L.*{LOCAL_FRONTEND_PORT}"],
            capture_output=True, text=True
        )
        if result.returncode == 0:
            for line in result.stdout.strip().split('\n'):
                if line.strip():
                    pids.append(int(line.strip()))
    except:
        pass
    return pids

def is_tunnel_running():
    """通过端口检测判断隧道是否运行"""
    frontend_ok = check_port_listening(LOCAL_FRONTEND_PORT)
    backend_ok = check_port_listening(LOCAL_BACKEND_PORT)
    return frontend_ok and backend_ok

@router.get("/status")
async def tunnel_status(_: dict = Depends(verify_token)):
    """获取隧道状态"""
    running = is_tunnel_running()
    return {
        "running": running,
        "pids": [],
        "ports": {
            "frontend": f"localhost:{LOCAL_FRONTEND_PORT} -> {REMOTE_FRONTEND_PORT}",
            "backend": f"localhost:{LOCAL_BACKEND_PORT} -> {REMOTE_BACKEND_PORT}"
        }
    }

@router.post("/start")
async def start_tunnel(_: dict = Depends(verify_token)):
    """启动SSH隧道"""
    if is_tunnel_running():
        return {"success": False, "message": "隧道已在运行"}

    cmd = [
        "ssh", "-f", "-N",
        "-L", f"{LOCAL_FRONTEND_PORT}:localhost:{REMOTE_FRONTEND_PORT}",
        "-L", f"{LOCAL_BACKEND_PORT}:localhost:{REMOTE_BACKEND_PORT}",
        "-o", "ServerAliveInterval=60",
        "-o", "ServerAliveCountMax=3",
        REMOTE_HOST
    ]

    try:
        subprocess.Popen(cmd, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
        return {"success": True, "message": "隧道已启动"}
    except Exception as e:
        return {"success": False, "message": str(e)}

@router.post("/stop")
async def stop_tunnel(_: dict = Depends(verify_token)):
    """停止SSH隧道"""
    if not is_tunnel_running():
        return {"success": True, "message": "隧道未运行"}

    # 使用 pkill 停止 SSH 隧道进程
    try:
        subprocess.run(
            ["pkill", "-f", f"ssh.*-L.*{LOCAL_FRONTEND_PORT}"],
            capture_output=True
        )
        return {"success": True, "message": "已停止隧道进程"}
    except Exception as e:
        return {"success": False, "message": str(e)}

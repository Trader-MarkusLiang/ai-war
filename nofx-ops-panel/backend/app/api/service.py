from fastapi import APIRouter, Depends, HTTPException
from pydantic import BaseModel
from ..auth import verify_token
from ..services import ssh_executor
from .. import config

router = APIRouter(prefix="/service", tags=["服务控制"])

class ServiceAction(BaseModel):
    service: str = "all"  # all, backend, frontend
    confirm: bool = False

@router.post("/start")
async def start_service(action: ServiceAction, _: dict = Depends(verify_token)):
    """启动服务"""
    cmd = f"cd {config.REMOTE_DIR} && docker compose up -d"
    stdout, stderr, code = await ssh_executor.execute_async(cmd)
    return {"success": code == 0, "output": stdout, "error": stderr}

@router.post("/stop")
async def stop_service(action: ServiceAction, _: dict = Depends(verify_token)):
    """停止服务"""
    if not action.confirm:
        raise HTTPException(400, "需要确认此操作")
    cmd = f"cd {config.REMOTE_DIR} && docker compose stop"
    stdout, stderr, code = await ssh_executor.execute_async(cmd)
    return {"success": code == 0, "output": stdout, "error": stderr}

@router.post("/restart")
async def restart_service(action: ServiceAction, _: dict = Depends(verify_token)):
    """重启服务"""
    if action.service and action.service != "all":
        cmd = f"docker restart {action.service}"
    else:
        cmd = f"cd {config.REMOTE_DIR} && docker compose restart"
    stdout, stderr, code = await ssh_executor.execute_async(cmd)
    return {"success": code == 0, "output": stdout, "error": stderr}

@router.post("/update")
async def update_code(action: ServiceAction, _: dict = Depends(verify_token)):
    """更新代码 (git pull)"""
    if not action.confirm:
        raise HTTPException(400, "需要确认此操作")
    cmd = f"cd {config.REMOTE_DIR} && git pull && docker compose restart"
    stdout, stderr, code = await ssh_executor.execute_async(cmd)
    return {"success": code == 0, "output": stdout + stderr}

@router.post("/clean-logs")
async def clean_logs(action: ServiceAction, _: dict = Depends(verify_token)):
    """清理30天前的日志"""
    if not action.confirm:
        raise HTTPException(400, "需要确认此操作")
    cmd = f"find {config.REMOTE_DIR}/decision_logs -name '*.json' -mtime +30 -delete 2>/dev/null; echo 'done'"
    stdout, _, code = await ssh_executor.execute_async(cmd)
    return {"success": code == 0, "output": "日志清理完成"}

@router.post("/rebuild")
async def rebuild_images(action: ServiceAction, _: dict = Depends(verify_token)):
    """重建Docker镜像"""
    if not action.confirm:
        raise HTTPException(400, "需要确认此操作")
    cmd = f"cd {config.REMOTE_DIR} && docker compose down && docker compose build --no-cache && docker compose up -d"
    stdout, stderr, code = await ssh_executor.execute_async(cmd)
    return {"success": code == 0, "output": stdout + stderr}

@router.get("/health")
async def health_check(_: dict = Depends(verify_token)):
    """健康检查"""
    checks = []
    # 检查Docker服务
    cmd = "systemctl is-active docker"
    stdout, _, code = await ssh_executor.execute_async(cmd)
    checks.append({"name": "Docker服务", "status": "ok" if code == 0 else "error", "detail": stdout.strip()})

    # 检查容器状态
    cmd = "docker ps --format '{{.Names}}:{{.Status}}' | head -5"
    stdout, _, _ = await ssh_executor.execute_async(cmd)
    for line in stdout.strip().split('\n'):
        if ':' in line:
            name, status = line.split(':', 1)
            checks.append({
                "name": f"容器:{name}",
                "status": "ok" if "Up" in status else "error",
                "detail": status
            })

    # 检查磁盘空间
    cmd = "df -h / | tail -1 | awk '{print $5}'"
    stdout, _, _ = await ssh_executor.execute_async(cmd)
    usage = stdout.strip().replace('%', '')
    try:
        pct = int(usage)
        checks.append({
            "name": "磁盘空间",
            "status": "ok" if pct < 80 else ("warning" if pct < 90 else "error"),
            "detail": f"使用率 {pct}%"
        })
    except:
        pass

    return {"checks": checks}

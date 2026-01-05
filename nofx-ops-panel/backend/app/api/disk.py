from fastapi import APIRouter, Depends, HTTPException
from pydantic import BaseModel
from ..auth import verify_token
from ..services import ssh_executor

router = APIRouter(prefix="/disk", tags=["磁盘管理"])

@router.get("/usage")
async def get_disk_usage(_: dict = Depends(verify_token)):
    """获取磁盘使用情况"""
    # 系统磁盘
    cmd = "df -h / | tail -1 | awk '{print $2,$3,$4,$5}'"
    stdout, _, _ = await ssh_executor.execute_async(cmd)
    parts = stdout.strip().split()
    system = {
        "total": parts[0] if len(parts) > 0 else "-",
        "used": parts[1] if len(parts) > 1 else "-",
        "available": parts[2] if len(parts) > 2 else "-",
        "percent": parts[3] if len(parts) > 3 else "-"
    }

    # Docker 使用情况
    cmd = "docker system df --format '{{.Type}}|{{.Size}}|{{.Reclaimable}}'"
    stdout, _, _ = await ssh_executor.execute_async(cmd)
    docker = []
    for line in stdout.strip().split('\n'):
        if '|' in line:
            parts = line.split('|')
            docker.append({
                "type": parts[0],
                "size": parts[1],
                "reclaimable": parts[2]
            })

    # NOFX 项目目录
    cmd = "du -sh /opt/nofx 2>/dev/null | awk '{print $1}'"
    stdout, _, _ = await ssh_executor.execute_async(cmd)
    project_size = stdout.strip() or "-"

    # 项目子目录详情
    cmd = "du -sh /opt/nofx/*/ 2>/dev/null | sort -rh | head -8"
    stdout, _, _ = await ssh_executor.execute_async(cmd)
    project_dirs = []
    for line in stdout.strip().split('\n'):
        if line:
            parts = line.split('\t')
            if len(parts) >= 2:
                project_dirs.append({
                    "size": parts[0],
                    "path": parts[1].replace('/opt/nofx/', '').rstrip('/')
                })

    return {
        "system": system,
        "docker": docker,
        "project": {
            "total": project_size,
            "dirs": project_dirs
        }
    }

class CleanAction(BaseModel):
    confirm: bool = False

@router.post("/clean/docker-cache")
async def clean_docker_cache(action: CleanAction, _: dict = Depends(verify_token)):
    """清理 Docker 构建缓存"""
    if not action.confirm:
        raise HTTPException(400, "需要确认此操作")
    cmd = "docker builder prune -af 2>&1 | tail -1"
    stdout, stderr, code = await ssh_executor.execute_async(cmd)
    return {"success": code == 0, "output": stdout.strip() or "清理完成"}

@router.post("/clean/docker-images")
async def clean_docker_images(action: CleanAction, _: dict = Depends(verify_token)):
    """清理无用 Docker 镜像"""
    if not action.confirm:
        raise HTTPException(400, "需要确认此操作")
    cmd = "docker image prune -af 2>&1 | tail -1"
    stdout, stderr, code = await ssh_executor.execute_async(cmd)
    return {"success": code == 0, "output": stdout.strip() or "清理完成"}

@router.post("/clean/logs")
async def clean_old_logs(action: CleanAction, _: dict = Depends(verify_token)):
    """清理旧日志文件"""
    if not action.confirm:
        raise HTTPException(400, "需要确认此操作")
    cmd = "find /opt/nofx/data -name '*.log' -mtime +7 -delete 2>/dev/null; echo '已清理7天前的日志'"
    stdout, _, code = await ssh_executor.execute_async(cmd)
    return {"success": code == 0, "output": stdout.strip()}

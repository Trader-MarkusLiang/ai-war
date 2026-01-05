from fastapi import APIRouter, Depends
from fastapi.responses import FileResponse
from ..auth import verify_token
from ..services import ssh_executor

router = APIRouter(prefix="/backup", tags=["备份"])

@router.get("/list")
async def list_backups(_: dict = Depends(verify_token)):
    """获取备份列表"""
    cmd = "ls -lh /opt/nofx_backups/*.tar.gz 2>/dev/null | awk '{print $9,$5,$6,$7}'"
    stdout, _, _ = await ssh_executor.execute_async(cmd)

    backups = []
    for line in stdout.strip().split('\n'):
        if line:
            parts = line.split()
            if len(parts) >= 4:
                backups.append({
                    "file": parts[0].split('/')[-1],
                    "size": parts[1],
                    "date": f"{parts[2]} {parts[3]}"
                })
    return {"backups": backups}

@router.post("/create")
async def create_backup(_: dict = Depends(verify_token)):
    """创建备份"""
    cmd = "/opt/nofx/backup.sh"
    stdout, stderr, code = await ssh_executor.execute_async(cmd)
    return {"success": code == 0, "output": stdout, "error": stderr}

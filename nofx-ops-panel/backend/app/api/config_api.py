from fastapi import APIRouter, Depends, HTTPException
from pydantic import BaseModel
from ..auth import verify_token
from ..services import ssh_executor
from .. import config

router = APIRouter(prefix="/config", tags=["配置"])

@router.get("")
async def get_config(_: dict = Depends(verify_token)):
    """获取配置文件"""
    cmd = f"cat {config.REMOTE_DIR}/.env 2>/dev/null"
    stdout, _, code = await ssh_executor.execute_async(cmd)
    return {"content": stdout, "exists": code == 0}

class ConfigUpdate(BaseModel):
    content: str
    confirm: bool = False

@router.put("")
async def update_config(data: ConfigUpdate, _: dict = Depends(verify_token)):
    """更新配置文件"""
    if not data.confirm:
        raise HTTPException(400, "需要确认此操作")

    # 备份原配置
    backup_cmd = f"cp {config.REMOTE_DIR}/.env {config.REMOTE_DIR}/.env.bak"
    await ssh_executor.execute_async(backup_cmd)

    # 写入新配置
    escaped = data.content.replace("'", "'\\''")
    cmd = f"echo '{escaped}' > {config.REMOTE_DIR}/.env"
    _, _, code = await ssh_executor.execute_async(cmd)

    return {"success": code == 0, "restart_required": True}

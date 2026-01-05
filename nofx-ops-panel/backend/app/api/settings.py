"""
系统设置 API
"""
import os
from fastapi import APIRouter, HTTPException, Depends
from pydantic import BaseModel
from typing import Optional
from ..auth import get_current_user

router = APIRouter(prefix="/settings", tags=["settings"])

# .env 文件路径
ENV_FILE = os.path.join(os.path.dirname(os.path.dirname(__file__)), "..", ".env")


class SettingsModel(BaseModel):
    """系统设置模型"""
    admin_password: Optional[str] = None
    remote_host: Optional[str] = None
    remote_user: Optional[str] = None
    remote_dir: Optional[str] = None
    api_port: Optional[int] = None
    jwt_expire_hours: Optional[int] = None


class PasswordChangeModel(BaseModel):
    """密码修改模型"""
    old_password: str
    new_password: str


def read_env_file() -> dict:
    """读取 .env 文件"""
    env_vars = {}
    if os.path.exists(ENV_FILE):
        with open(ENV_FILE, 'r') as f:
            for line in f:
                line = line.strip()
                if line and not line.startswith('#') and '=' in line:
                    key, value = line.split('=', 1)
                    env_vars[key.strip()] = value.strip()
    return env_vars


def write_env_file(env_vars: dict):
    """写入 .env 文件"""
    with open(ENV_FILE, 'w') as f:
        for key, value in env_vars.items():
            f.write(f"{key}={value}\n")


@router.get("")
async def get_settings(user: str = Depends(get_current_user)):
    """获取当前系统设置"""
    from ..config import (
        REMOTE_HOST, REMOTE_USER, REMOTE_DIR,
        API_PORT, JWT_EXPIRE_HOURS, ADMIN_PASSWORD
    )

    return {
        "remote_host": REMOTE_HOST,
        "remote_user": REMOTE_USER,
        "remote_dir": REMOTE_DIR,
        "api_port": API_PORT,
        "jwt_expire_hours": JWT_EXPIRE_HOURS,
        "password_set": ADMIN_PASSWORD != "admin123"
    }


@router.put("")
async def update_settings(settings: SettingsModel, user: str = Depends(get_current_user)):
    """更新系统设置"""
    env_vars = read_env_file()

    if settings.remote_host:
        env_vars["REMOTE_HOST"] = settings.remote_host
    if settings.remote_user:
        env_vars["REMOTE_USER"] = settings.remote_user
    if settings.remote_dir:
        env_vars["REMOTE_DIR"] = settings.remote_dir
    if settings.api_port:
        env_vars["API_PORT"] = str(settings.api_port)
    if settings.jwt_expire_hours:
        env_vars["JWT_EXPIRE_HOURS"] = str(settings.jwt_expire_hours)

    write_env_file(env_vars)

    return {"message": "设置已保存，重启服务后生效"}


@router.post("/password")
async def change_password(data: PasswordChangeModel, user: str = Depends(get_current_user)):
    """修改管理员密码"""
    from ..config import ADMIN_PASSWORD

    if data.old_password != ADMIN_PASSWORD:
        raise HTTPException(status_code=400, detail="原密码错误")

    if len(data.new_password) < 6:
        raise HTTPException(status_code=400, detail="新密码长度至少6位")

    env_vars = read_env_file()
    env_vars["ADMIN_PASSWORD"] = data.new_password
    write_env_file(env_vars)

    return {"message": "密码修改成功，重启服务后生效"}

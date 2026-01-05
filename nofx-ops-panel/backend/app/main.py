from fastapi import FastAPI, HTTPException
from fastapi.middleware.cors import CORSMiddleware
from pydantic import BaseModel
import logging
from .auth import verify_password, create_access_token
from .api import (
    status_router, service_router, logs_router,
    diagnose_router, backup_router, config_router, tunnel_router, disk_router,
    settings_router, image_history_router
)
from . import config

# 配置日志
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)

app = FastAPI(title="NOFX运维管理面板", version="1.0.0")

# CORS配置（优化：从配置文件读取）
app.add_middleware(
    CORSMiddleware,
    allow_origins=config.ALLOWED_ORIGINS,
    allow_credentials=True,
    allow_methods=["GET", "POST", "PUT", "DELETE"],
    allow_headers=["*"],
)

# 注册路由
app.include_router(status_router, prefix="/api")
app.include_router(service_router, prefix="/api")
app.include_router(logs_router, prefix="/api")
app.include_router(diagnose_router, prefix="/api")
app.include_router(backup_router, prefix="/api")
app.include_router(config_router, prefix="/api")
app.include_router(tunnel_router, prefix="/api")
app.include_router(disk_router, prefix="/api")
app.include_router(settings_router, prefix="/api")
app.include_router(image_history_router, prefix="/api")

class LoginRequest(BaseModel):
    password: str

@app.post("/api/auth/login")
async def login(req: LoginRequest):
    if not verify_password(req.password):
        raise HTTPException(401, "密码错误")
    token = create_access_token({"sub": "admin"})
    return {"token": token, "expires_in": 86400}

@app.get("/api/ping")
async def ping():
    return {"status": "ok"}

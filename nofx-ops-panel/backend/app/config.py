import os
import secrets
from dotenv import load_dotenv

load_dotenv()

# 服务器配置
REMOTE_HOST = os.getenv("REMOTE_HOST", "47.236.159.60")
REMOTE_USER = os.getenv("REMOTE_USER", "root")
REMOTE_DIR = os.getenv("REMOTE_DIR", "/opt/nofx")
SSH_KEY_PATH = os.getenv("SSH_KEY_PATH", os.path.expanduser("~/.ssh/id_rsa"))

# API配置
API_PORT = int(os.getenv("API_PORT", "8800"))

# JWT配置（安全警告）
_default_secret = "nofx-ops-panel-secret-key-change-in-production"
JWT_SECRET = os.getenv("JWT_SECRET", _default_secret)
if JWT_SECRET == _default_secret:
    import warnings
    warnings.warn("警告: 使用默认JWT密钥，生产环境请修改！", UserWarning)

JWT_ALGORITHM = "HS256"
JWT_EXPIRE_HOURS = int(os.getenv("JWT_EXPIRE_HOURS", "24"))

# 认证密码
ADMIN_PASSWORD = os.getenv("ADMIN_PASSWORD", "admin123")

# SSH连接池配置
SSH_POOL_MAX_CONNECTIONS = int(os.getenv("SSH_POOL_MAX_CONNECTIONS", "5"))
SSH_COMMAND_TIMEOUT = int(os.getenv("SSH_COMMAND_TIMEOUT", "60"))

# SSH隧道配置
TUNNEL_LOCAL_FRONTEND_PORT = int(os.getenv("TUNNEL_LOCAL_FRONTEND_PORT", "3333"))
TUNNEL_LOCAL_BACKEND_PORT = int(os.getenv("TUNNEL_LOCAL_BACKEND_PORT", "8888"))
TUNNEL_REMOTE_FRONTEND_PORT = int(os.getenv("TUNNEL_REMOTE_FRONTEND_PORT", "3000"))
TUNNEL_REMOTE_BACKEND_PORT = int(os.getenv("TUNNEL_REMOTE_BACKEND_PORT", "8080"))

# CORS配置
ALLOWED_ORIGINS = os.getenv("ALLOWED_ORIGINS", "http://localhost:8801,http://localhost:8802,http://localhost:8803").split(",")

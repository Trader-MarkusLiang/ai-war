import os
from dotenv import load_dotenv

load_dotenv()

# 服务器配置
REMOTE_HOST = os.getenv("REMOTE_HOST", "47.236.159.60")
REMOTE_USER = os.getenv("REMOTE_USER", "root")
REMOTE_DIR = os.getenv("REMOTE_DIR", "/opt/nofx")
SSH_KEY_PATH = os.getenv("SSH_KEY_PATH", os.path.expanduser("~/.ssh/id_rsa"))

# API配置
API_PORT = int(os.getenv("API_PORT", "8800"))
JWT_SECRET = os.getenv("JWT_SECRET", "nofx-ops-panel-secret-key-change-in-production")
JWT_ALGORITHM = "HS256"
JWT_EXPIRE_HOURS = 24

# 认证密码 (简单密码认证)
ADMIN_PASSWORD = os.getenv("ADMIN_PASSWORD", "admin123")

# SSH隧道配置
TUNNEL_LOCAL_FRONTEND_PORT = 3333
TUNNEL_LOCAL_BACKEND_PORT = 8888
TUNNEL_REMOTE_FRONTEND_PORT = 3000
TUNNEL_REMOTE_BACKEND_PORT = 8080

from datetime import datetime, timedelta
from typing import Optional
from jose import JWTError, jwt
from fastapi import HTTPException, Depends, status
from fastapi.security import HTTPBearer, HTTPAuthorizationCredentials
from . import config

security = HTTPBearer()

def verify_password(password: str) -> bool:
    """验证密码"""
    return password == config.ADMIN_PASSWORD

def create_access_token(data: dict, expires_delta: Optional[timedelta] = None) -> str:
    """创建JWT令牌"""
    to_encode = data.copy()
    expire = datetime.utcnow() + (expires_delta or timedelta(hours=config.JWT_EXPIRE_HOURS))
    to_encode.update({"exp": expire})
    return jwt.encode(to_encode, config.JWT_SECRET, algorithm=config.JWT_ALGORITHM)

def verify_token(credentials: HTTPAuthorizationCredentials = Depends(security)) -> dict:
    """验证JWT令牌"""
    try:
        payload = jwt.decode(
            credentials.credentials,
            config.JWT_SECRET,
            algorithms=[config.JWT_ALGORITHM]
        )
        return payload
    except JWTError:
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="无效的认证令牌"
        )

# 别名，用于依赖注入
get_current_user = verify_token

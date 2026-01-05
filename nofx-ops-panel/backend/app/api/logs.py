from fastapi import APIRouter, Depends, WebSocket
from ..auth import verify_token
from ..services import ssh_executor
import asyncio

router = APIRouter(prefix="/logs", tags=["日志"])

@router.get("/{service}")
async def get_logs(service: str, lines: int = 100, _: dict = Depends(verify_token)):
    """获取日志"""
    container = "nofx-trading" if service == "backend" else "nofx-frontend"
    cmd = f"docker logs --tail {lines} {container} 2>&1"
    stdout, _, _ = await ssh_executor.execute_async(cmd)
    return {"logs": stdout.split('\n')}

@router.websocket("/stream/{service}")
async def stream_logs(websocket: WebSocket, service: str):
    """WebSocket实时日志流"""
    await websocket.accept()
    container = "nofx-trading" if service == "backend" else "nofx-frontend"

    try:
        while True:
            cmd = f"docker logs --tail 5 {container} 2>&1"
            stdout, _, _ = await ssh_executor.execute_async(cmd)
            for line in stdout.strip().split('\n'):
                if line:
                    await websocket.send_json({"line": line})
            await asyncio.sleep(2)
    except Exception:
        pass

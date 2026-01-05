from fastapi import APIRouter, Depends
from ..auth import verify_token
from ..services import ssh_executor

router = APIRouter(prefix="/diagnose", tags=["诊断"])

@router.get("")
async def diagnose(_: dict = Depends(verify_token)):
    """系统诊断"""
    checks = []

    # 1. Docker服务状态
    stdout, _, code = await ssh_executor.execute_async(
        "systemctl is-active docker"
    )
    checks.append({
        "name": "Docker服务",
        "status": "ok" if code == 0 else "error",
        "detail": stdout.strip()
    })

    # 2. 容器状态
    stdout, _, _ = await ssh_executor.execute_async(
        "docker ps --format '{{.Names}}: {{.Status}}'"
    )
    checks.append({
        "name": "容器状态",
        "status": "ok" if stdout.strip() else "warning",
        "detail": stdout.strip() or "无运行中的容器"
    })

    # 3. 端口监听
    stdout, _, _ = await ssh_executor.execute_async(
        "netstat -tlnp 2>/dev/null | grep -E ':3000|:8080' | awk '{print $4}'"
    )
    checks.append({
        "name": "端口监听",
        "status": "ok" if "3000" in stdout and "8080" in stdout else "warning",
        "detail": stdout.strip().replace('\n', ', ') or "未检测到端口"
    })

    # 4. 磁盘空间
    stdout, _, _ = await ssh_executor.execute_async(
        "df -h / | tail -1 | awk '{print $5}'"
    )
    percent = int(stdout.strip().replace('%', '') or 0)
    checks.append({
        "name": "磁盘空间",
        "status": "ok" if percent < 80 else ("warning" if percent < 90 else "error"),
        "detail": f"已使用 {stdout.strip()}"
    })

    # 5. 内存使用
    stdout, _, _ = await ssh_executor.execute_async(
        "free | grep Mem | awk '{printf \"%.0f\", $3/$2*100}'"
    )
    mem_percent = int(stdout.strip() or 0)
    checks.append({
        "name": "内存使用",
        "status": "ok" if mem_percent < 80 else ("warning" if mem_percent < 90 else "error"),
        "detail": f"已使用 {mem_percent}%"
    })

    return {"checks": checks}

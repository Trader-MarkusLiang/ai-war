from fastapi import APIRouter, Depends
from ..auth import verify_token
from ..services import ssh_executor

router = APIRouter(prefix="/status", tags=["状态"])

@router.get("")
async def get_status(_: dict = Depends(verify_token)):
    """获取服务器状态"""
    combined_cmd = 'echo "===CONTAINERS===" && docker ps --format "{{.Names}}|{{.Status}}|{{.Ports}}" 2>/dev/null && echo "===STATS===" && docker stats --no-stream --format "{{.Name}}|{{.MemUsage}}|{{.CPUPerc}}" 2>/dev/null && echo "===SYSTEM===" && free -h | grep Mem | awk \'{print $2,$3}\' && df -h / | tail -1 | awk \'{print $2,$3,$5}\' && echo "===SERVER===" && hostname && cat /etc/os-release | grep PRETTY_NAME | cut -d\'"\' -f2 && uptime -p'
    stdout, stderr, code = await ssh_executor.execute_async(combined_cmd)
    print(f"SSH stdout: {stdout[:500] if stdout else 'empty'}")
    print(f"SSH stderr: {stderr[:200] if stderr else 'empty'}")
    print(f"SSH code: {code}")

    # 解析结果 - 使用正则匹配
    import re
    containers = []
    stats = {}
    system = {}
    server = {"ip": "47.236.159.60", "hostname": "", "os": "", "uptime": ""}

    # 提取各部分数据
    c_match = re.search(r'===CONTAINERS===\n(.*?)(?====|$)', stdout, re.DOTALL)
    if c_match:
        for line in c_match.group(1).strip().split('\n'):
            if line.strip():
                parts = line.split('|')
                containers.append({
                    "name": parts[0] if parts else "",
                    "status": parts[1] if len(parts) > 1 else "",
                    "ports": parts[2] if len(parts) > 2 else ""
                })

    s_match = re.search(r'===STATS===\n(.*?)(?====|$)', stdout, re.DOTALL)
    if s_match:
        for line in s_match.group(1).strip().split('\n'):
            if line.strip():
                parts = line.split('|')
                if parts:
                    stats[parts[0]] = {
                        "memory": parts[1] if len(parts) > 1 else "",
                        "cpu": parts[2] if len(parts) > 2 else ""
                    }

    sys_match = re.search(r'===SYSTEM===\n(.*?)(?====|$)', stdout, re.DOTALL)
    if sys_match:
        data = sys_match.group(1).strip().split('\n')
        if len(data) >= 2:
            mem = data[0].split()
            disk = data[1].split()
            system = {
                "memory_total": mem[0] if mem else "",
                "memory_used": mem[1] if len(mem) > 1 else "",
                "disk_total": disk[0] if disk else "",
                "disk_used": disk[1] if len(disk) > 1 else "",
                "disk_percent": disk[2] if len(disk) > 2 else ""
            }

    srv_match = re.search(r'===SERVER===\n(.*?)$', stdout, re.DOTALL)
    if srv_match:
        data = srv_match.group(1).strip().split('\n')
        server["hostname"] = data[0].strip() if data else ""
        server["os"] = data[1].strip() if len(data) > 1 else ""
        server["uptime"] = data[2].strip() if len(data) > 2 else ""

    # 合并容器信息
    for c in containers:
        if c["name"] in stats:
            c.update(stats[c["name"]])

    return {"containers": containers, "system": system, "server": server}

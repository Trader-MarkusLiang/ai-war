#!/bin/bash

# ═══════════════════════════════════════════════════════════════
# NOFX SSH隧道 - 简化版（推荐使用）
# 前台运行，稳定可靠
# ═══════════════════════════════════════════════════════════════

SERVER="root@47.236.159.60"
LOCAL_FRONTEND_PORT=3333
LOCAL_BACKEND_PORT=8888
REMOTE_FRONTEND_PORT=3000
REMOTE_BACKEND_PORT=8080

# 颜色定义
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}╔═══════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║                                                           ║${NC}"
echo -e "${BLUE}║        NOFX SSH隧道 - 简化版                              ║${NC}"
echo -e "${BLUE}║                                                           ║${NC}"
echo -e "${BLUE}╚═══════════════════════════════════════════════════════════╝${NC}"
echo ""

echo -e "${GREEN}[INFO]${NC} 正在建立SSH隧道..."
echo -e "${GREEN}[INFO]${NC} 服务器: ${SERVER}"
echo -e "${GREEN}[INFO]${NC} 端口映射:"
echo "  本地 ${LOCAL_FRONTEND_PORT} → 服务器 ${REMOTE_FRONTEND_PORT} (前端)"
echo "  本地 ${LOCAL_BACKEND_PORT} → 服务器 ${REMOTE_BACKEND_PORT} (后端)"
echo ""
echo -e "${YELLOW}[提示]${NC} 保持此窗口打开，按 Ctrl+C 停止隧道"
echo ""

# 检查本地端口是否被占用
check_and_kill_port() {
    local port=$1
    if lsof -i :${port} > /dev/null 2>&1; then
        echo -e "${YELLOW}[WARNING]${NC} 端口 ${port} 已被占用"
        echo "正在尝试关闭占用进程..."
        pkill -f "ssh.*${port}" || true
        sleep 2
        # 再次检查
        if lsof -i :${port} > /dev/null 2>&1; then
            echo -e "${RED}[ERROR]${NC} 端口 ${port} 仍被占用，请手动关闭占用进程"
            echo "   使用命令: lsof -i :${port} 查看占用进程"
            return 1
        fi
    fi
    return 0
}

if ! check_and_kill_port ${LOCAL_FRONTEND_PORT}; then
    exit 1
fi

if ! check_and_kill_port ${LOCAL_BACKEND_PORT}; then
    exit 1
fi

echo -e "${GREEN}[INFO]${NC} 正在连接..."
echo ""

# 先检查远程服务是否运行
echo -e "${BLUE}[检查]${NC} 正在检查远程服务状态..."
REMOTE_CHECK=$(ssh -o ConnectTimeout=5 -o BatchMode=yes ${SERVER} "netstat -tlnp 2>/dev/null | grep -E ':(3000|8080)' || ss -tlnp 2>/dev/null | grep -E ':(3000|8080)' || echo 'check_failed'" 2>/dev/null)

if echo "$REMOTE_CHECK" | grep -q "check_failed"; then
    echo -e "${YELLOW}[WARNING]${NC} 无法检查远程服务状态，继续尝试连接..."
    echo -e "${YELLOW}[提示]${NC} 如果连接失败，请运行 ./check_remote_services.sh 进行详细诊断"
elif echo "$REMOTE_CHECK" | grep -q ":3000"; then
    echo -e "${GREEN}[✓]${NC} 远程前端服务(3000)正在运行"
else
    echo -e "${YELLOW}[⚠]${NC} 远程前端服务(3000)可能未运行"
    echo -e "${YELLOW}[提示]${NC} 如果隧道连接失败，请确保前端服务已启动"
fi

if echo "$REMOTE_CHECK" | grep -q ":8080"; then
    echo -e "${GREEN}[✓]${NC} 远程后端服务(8080)正在运行"
else
    echo -e "${YELLOW}[⚠]${NC} 远程后端服务(8080)可能未运行"
    echo -e "${YELLOW}[提示]${NC} 如果隧道连接失败，请确保后端服务已启动"
fi

echo ""

# 建立SSH隧道（前台运行）
# 注意：移除 ExitOnForwardFailure=yes，允许部分端口转发失败
ssh -o ServerAliveInterval=60 \
    -o ServerAliveCountMax=3 \
    -o ExitOnForwardFailure=no \
    -o StrictHostKeyChecking=no \
    -L ${LOCAL_FRONTEND_PORT}:localhost:${REMOTE_FRONTEND_PORT} \
    -L ${LOCAL_BACKEND_PORT}:localhost:${REMOTE_BACKEND_PORT} \
    ${SERVER} \
    "echo ''; \
     echo '╔═══════════════════════════════════════════════════════════╗'; \
     echo '║                                                           ║'; \
     echo '║        ✓ SSH隧道已成功建立！                              ║'; \
     echo '║                                                           ║'; \
     echo '╚═══════════════════════════════════════════════════════════╝'; \
     echo ''; \
     echo '📱 访问地址:'; \
     echo '   前端: http://localhost:${LOCAL_FRONTEND_PORT}'; \
     echo '   后端: http://localhost:${LOCAL_BACKEND_PORT}'; \
     echo ''; \
     echo '💡 提示:'; \
     echo '   - 保持此窗口打开'; \
     echo '   - 在浏览器中访问上述地址'; \
     echo '   - 按 Ctrl+C 停止隧道'; \
     echo ''; \
     echo '⏳ 隧道运行中...'; \
     echo ''; \
     tail -f /dev/null"

echo ""
echo -e "${YELLOW}[INFO]${NC} SSH隧道已关闭"

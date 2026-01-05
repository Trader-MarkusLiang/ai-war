#!/bin/bash

# ═══════════════════════════════════════════════════════════════
# 远程服务诊断脚本
# 检查远程服务器上的服务状态
# ═══════════════════════════════════════════════════════════════

SERVER="root@47.236.159.60"
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
echo -e "${BLUE}║        远程服务诊断工具                                  ║${NC}"
echo -e "${BLUE}║                                                           ║${NC}"
echo -e "${BLUE}╚═══════════════════════════════════════════════════════════╝${NC}"
echo ""
echo -e "${GREEN}[INFO]${NC} 正在检查远程服务器: ${SERVER}"
echo ""

# 检查SSH连接
echo -e "${BLUE}[1/5]${NC} 检查SSH连接..."
if ssh -o ConnectTimeout=5 -o BatchMode=yes ${SERVER} "echo 'SSH连接成功'" 2>/dev/null; then
    echo -e "${GREEN}[✓]${NC} SSH连接正常"
else
    echo -e "${RED}[✗]${NC} SSH连接失败，请检查："
    echo "   - SSH密钥配置"
    echo "   - 服务器IP地址"
    echo "   - 网络连接"
    exit 1
fi
echo ""

# 检查端口监听情况
echo -e "${BLUE}[2/5]${NC} 检查端口监听情况..."
echo "正在检查端口 ${REMOTE_FRONTEND_PORT} (前端)..."
FRONTEND_CHECK=$(ssh ${SERVER} "netstat -tlnp 2>/dev/null | grep :${REMOTE_FRONTEND_PORT} || ss -tlnp 2>/dev/null | grep :${REMOTE_FRONTEND_PORT} || echo 'not_found'")

if echo "$FRONTEND_CHECK" | grep -q "not_found"; then
    echo -e "${RED}[✗]${NC} 端口 ${REMOTE_FRONTEND_PORT} 未监听"
else
    echo -e "${GREEN}[✓]${NC} 端口 ${REMOTE_FRONTEND_PORT} 正在监听"
    echo "   详情: $FRONTEND_CHECK"
fi

echo ""
echo "正在检查端口 ${REMOTE_BACKEND_PORT} (后端)..."
BACKEND_CHECK=$(ssh ${SERVER} "netstat -tlnp 2>/dev/null | grep :${REMOTE_BACKEND_PORT} || ss -tlnp 2>/dev/null | grep :${REMOTE_BACKEND_PORT} || echo 'not_found'")

if echo "$BACKEND_CHECK" | grep -q "not_found"; then
    echo -e "${RED}[✗]${NC} 端口 ${REMOTE_BACKEND_PORT} 未监听"
else
    echo -e "${GREEN}[✓]${NC} 端口 ${REMOTE_BACKEND_PORT} 正在监听"
    echo "   详情: $BACKEND_CHECK"
fi
echo ""

# 检查服务进程
echo -e "${BLUE}[3/5]${NC} 检查服务进程..."
echo "正在检查前端服务进程..."
FRONTEND_PROCESS=$(ssh ${SERVER} "ps aux | grep -E '(node|npm|pm2|frontend)' | grep -v grep | head -3")
if [ -z "$FRONTEND_PROCESS" ]; then
    echo -e "${YELLOW}[⚠]${NC} 未找到明显的前端服务进程"
else
    echo -e "${GREEN}[✓]${NC} 找到相关进程:"
    echo "$FRONTEND_PROCESS" | while read line; do
        echo "   $line"
    done
fi

echo ""
echo "正在检查后端服务进程..."
BACKEND_PROCESS=$(ssh ${SERVER} "ps aux | grep -E '(go|nofx|backend|server)' | grep -v grep | head -3")
if [ -z "$BACKEND_PROCESS" ]; then
    echo -e "${YELLOW}[⚠]${NC} 未找到明显的后端服务进程"
else
    echo -e "${GREEN}[✓]${NC} 找到相关进程:"
    echo "$BACKEND_PROCESS" | while read line; do
        echo "   $line"
    done
fi
echo ""

# 检查防火墙
echo -e "${BLUE}[4/5]${NC} 检查防火墙状态..."
FIREWALL_STATUS=$(ssh ${SERVER} "systemctl status firewalld 2>/dev/null | grep Active || ufw status 2>/dev/null | head -1 || echo 'unknown'")
echo "防火墙状态: $FIREWALL_STATUS"
echo ""

# 测试本地连接
echo -e "${BLUE}[5/5]${NC} 测试远程服务器本地连接..."
echo "测试 localhost:${REMOTE_FRONTEND_PORT}..."
FRONTEND_LOCAL=$(ssh ${SERVER} "curl -s -o /dev/null -w '%{http_code}' http://localhost:${REMOTE_FRONTEND_PORT} 2>/dev/null || echo 'failed'")
if [ "$FRONTEND_LOCAL" = "failed" ] || [ -z "$FRONTEND_LOCAL" ]; then
    echo -e "${RED}[✗]${NC} 无法连接到 localhost:${REMOTE_FRONTEND_PORT}"
else
    echo -e "${GREEN}[✓]${NC} localhost:${REMOTE_FRONTEND_PORT} 响应: HTTP $FRONTEND_LOCAL"
fi

echo "测试 localhost:${REMOTE_BACKEND_PORT}..."
BACKEND_LOCAL=$(ssh ${SERVER} "curl -s -o /dev/null -w '%{http_code}' http://localhost:${REMOTE_BACKEND_PORT} 2>/dev/null || echo 'failed'")
if [ "$BACKEND_LOCAL" = "failed" ] || [ -z "$BACKEND_LOCAL" ]; then
    echo -e "${RED}[✗]${NC} 无法连接到 localhost:${REMOTE_BACKEND_PORT}"
else
    echo -e "${GREEN}[✓]${NC} localhost:${REMOTE_BACKEND_PORT} 响应: HTTP $BACKEND_LOCAL"
fi
echo ""

# 总结
echo -e "${BLUE}╔═══════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║                    诊断总结                              ║${NC}"
echo -e "${BLUE}╚═══════════════════════════════════════════════════════════╝${NC}"
echo ""

if echo "$FRONTEND_CHECK" | grep -q "not_found" && echo "$BACKEND_CHECK" | grep -q "not_found"; then
    echo -e "${RED}[问题]${NC} 两个服务端口都未监听"
    echo ""
    echo -e "${YELLOW}[建议]${NC} 请检查："
    echo "   1. 服务是否已启动"
    echo "   2. 服务配置文件中的端口设置"
    echo "   3. 查看服务日志: ssh ${SERVER} 'journalctl -u your-service -n 50'"
elif echo "$FRONTEND_CHECK" | grep -q "not_found"; then
    echo -e "${YELLOW}[问题]${NC} 前端服务(3000)未监听"
elif echo "$BACKEND_CHECK" | grep -q "not_found"; then
    echo -e "${YELLOW}[问题]${NC} 后端服务(8080)未监听"
else
    echo -e "${GREEN}[✓]${NC} 所有服务端口都在监听"
    echo ""
    echo -e "${YELLOW}[提示]${NC} 如果SSH隧道仍然失败，可能是："
    echo "   1. 服务监听在 0.0.0.0 而不是 localhost（这是正常的）"
    echo "   2. SSH配置问题，尝试使用 -N 参数"
fi

echo ""


#!/bin/bash

# ═══════════════════════════════════════════════════════════════
# NOFX 阿里云一键部署脚本
# 适用于：2核1G服务器（Ubuntu 20.04+）
# ═══════════════════════════════════════════════════════════════

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# 工具函数
print_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
print_success() { echo -e "${GREEN}[SUCCESS]${NC} $1"; }
print_warning() { echo -e "${YELLOW}[WARNING]${NC} $1"; }
print_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# 显示欢迎信息
clear
cat << 'EOF'
╔═══════════════════════════════════════════════════════════╗
║                                                           ║
║        NOFX AI Trading System - 阿里云部署脚本            ║
║                                                           ║
║        适用于：2核1G服务器                                 ║
║        系统：Ubuntu 20.04+                                ║
║                                                           ║
╚═══════════════════════════════════════════════════════════╝
EOF
echo ""

# 检查是否为root用户
if [ "$EUID" -ne 0 ]; then
    print_error "请使用root用户运行此脚本"
    exit 1
fi

print_info "开始部署流程..."
echo ""

# ═══════════════════════════════════════════════════════════════
# 步骤1：系统检查和优化
# ═══════════════════════════════════════════════════════════════
print_info "步骤1/10: 系统检查和优化"

# 检查内存
TOTAL_MEM=$(free -m | awk '/^Mem:/{print $2}')
print_info "检测到内存: ${TOTAL_MEM}MB"

if [ "$TOTAL_MEM" -lt 1500 ]; then
    print_warning "内存小于1.5G，将创建2G Swap交换空间以提升稳定性"

    if [ ! -f /swapfile ]; then
        print_info "创建Swap文件..."
        fallocate -l 2G /swapfile
        chmod 600 /swapfile
        mkswap /swapfile
        swapon /swapfile

        # 永久启用
        if ! grep -q '/swapfile' /etc/fstab; then
            echo '/swapfile none swap sw 0 0' >> /etc/fstab
        fi

        # 优化swap使用策略
        sysctl vm.swappiness=10
        if ! grep -q 'vm.swappiness' /etc/sysctl.conf; then
            echo 'vm.swappiness=10' >> /etc/sysctl.conf
        fi

        print_success "Swap创建成功"
    else
        print_info "Swap已存在，跳过创建"
    fi
fi

# ═══════════════════════════════════════════════════════════════
# 步骤2：更新系统
# ═══════════════════════════════════════════════════════════════
print_info "步骤2/10: 更新系统软件包"
apt-get update -qq
print_success "系统更新完成"

# ═══════════════════════════════════════════════════════════════
# 步骤3：安装基础工具
# ═══════════════════════════════════════════════════════════════
print_info "步骤3/10: 安装基础工具"
apt-get install -y -qq curl wget git nano jq htop net-tools > /dev/null 2>&1
print_success "基础工具安装完成"

# ═══════════════════════════════════════════════════════════════
# 步骤4：安装Docker
# ═══════════════════════════════════════════════════════════════
print_info "步骤4/10: 安装Docker"

if command -v docker &> /dev/null; then
    print_info "Docker已安装，版本: $(docker --version)"
else
    print_info "正在安装Docker（这可能需要几分钟）..."

    # 清理可能的残留
    apt-get remove -y docker docker-engine docker.io containerd runc 2>/dev/null || true

    # 安装依赖
    print_info "安装依赖包..."
    apt-get install -y apt-transport-https ca-certificates curl software-properties-common

    # 添加Docker GPG密钥
    print_info "添加Docker GPG密钥..."
    curl -fsSL https://download.docker.com/linux/ubuntu/gpg | apt-key add -

    # 添加Docker仓库
    print_info "添加Docker仓库..."
    add-apt-repository "deb [arch=amd64] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable"

    # 更新并安装Docker（不包含docker-model-plugin）
    print_info "安装Docker核心组件..."
    apt-get update
    apt-get install -y docker-ce docker-ce-cli containerd.io docker-compose-plugin docker-buildx-plugin

    # 启动Docker服务
    print_info "启动Docker服务..."
    systemctl start docker
    systemctl enable docker

    # 等待Docker服务启动
    sleep 3

    # 验证安装
    if command -v docker &> /dev/null && systemctl is-active --quiet docker; then
        print_success "Docker安装完成: $(docker --version)"
    else
        print_error "Docker安装失败，请检查日志"
        systemctl status docker --no-pager
        exit 1
    fi
fi

# 验证Docker Compose
if docker compose version &> /dev/null; then
    print_success "Docker Compose已就绪: $(docker compose version --short)"
elif command -v docker-compose &> /dev/null; then
    print_success "docker-compose已就绪: $(docker-compose --version)"
else
    print_error "Docker Compose未安装，尝试安装..."
    apt-get install -y docker-compose-plugin

    if ! docker compose version &> /dev/null; then
        print_error "Docker Compose安装失败"
        exit 1
    fi
fi

# ═══════════════════════════════════════════════════════════════
# 步骤5：克隆项目
# ═══════════════════════════════════════════════════════════════
print_info "步骤5/10: 克隆NOFX项目"

PROJECT_DIR="/opt/nofx"

if [ -d "$PROJECT_DIR" ]; then
    print_warning "项目目录已存在，是否删除并重新克隆？(y/n)"
    read -r response
    if [[ "$response" =~ ^[Yy]$ ]]; then
        rm -rf "$PROJECT_DIR"
        print_info "已删除旧项目"
    else
        print_info "使用现有项目目录"
        cd "$PROJECT_DIR"
    fi
fi

if [ ! -d "$PROJECT_DIR" ]; then
    print_info "正在克隆项目..."
    git clone https://github.com/NoFxAiOS/nofx.git "$PROJECT_DIR" > /dev/null 2>&1
    cd "$PROJECT_DIR"
    print_success "项目克隆完成"
else
    cd "$PROJECT_DIR"
fi

# ═══════════════════════════════════════════════════════════════
# 步骤6：配置环境文件
# ═══════════════════════════════════════════════════════════════
print_info "步骤6/10: 配置环境文件"

# 创建.env文件
if [ ! -f .env ]; then
    cat > .env << 'ENVEOF'
NOFX_BACKEND_PORT=8080
NOFX_FRONTEND_PORT=3000
NOFX_TIMEZONE=Asia/Shanghai
ENVEOF
    print_success ".env文件创建完成"
else
    print_info ".env文件已存在"
fi

# 配置config.json
if [ ! -f config.json ]; then
    if [ -f config.json.example ]; then
        cp config.json.example config.json
        print_warning "已创建config.json，请稍后编辑填入API密钥"
    else
        print_error "未找到config.json.example"
    fi
else
    print_info "config.json已存在"
fi

# ═══════════════════════════════════════════════════════════════
# 步骤7：内存优化配置
# ═══════════════════════════════════════════════════════════════
print_info "步骤7/10: 创建内存优化配置"

cat > docker-compose.override.yml << 'OVERRIDEEOF'
version: '3.8'

services:
  nofx:
    deploy:
      resources:
        limits:
          memory: 512M
        reservations:
          memory: 256M
    environment:
      - GOGC=50  # 更激进的GC，减少内存占用

  nofx-frontend:
    deploy:
      resources:
        limits:
          memory: 256M
        reservations:
          memory: 128M
OVERRIDEEOF

print_success "内存优化配置创建完成"

# ═══════════════════════════════════════════════════════════════
# 步骤8：配置防火墙
# ═══════════════════════════════════════════════════════════════
print_info "步骤8/10: 配置防火墙"

if command -v ufw &> /dev/null; then
    print_info "配置UFW防火墙..."
    ufw allow 22/tcp > /dev/null 2>&1
    ufw allow 3000/tcp > /dev/null 2>&1
    ufw allow 8080/tcp > /dev/null 2>&1
    print_success "UFW防火墙配置完成"
else
    print_info "UFW未安装，使用iptables..."
    iptables -A INPUT -p tcp --dport 22 -j ACCEPT
    iptables -A INPUT -p tcp --dport 3000 -j ACCEPT
    iptables -A INPUT -p tcp --dport 8080 -j ACCEPT
    print_success "iptables防火墙配置完成"
fi

# ═══════════════════════════════════════════════════════════════
# 步骤9：创建systemd服务
# ═══════════════════════════════════════════════════════════════
print_info "步骤9/10: 创建systemd自动启动服务"

cat > /etc/systemd/system/nofx.service << 'SERVICEEOF'
[Unit]
Description=NOFX AI Trading System
After=docker.service
Requires=docker.service

[Service]
Type=oneshot
RemainAfterExit=yes
WorkingDirectory=/opt/nofx
ExecStart=/usr/bin/docker compose up -d
ExecStop=/usr/bin/docker compose down
TimeoutStartSec=0

[Install]
WantedBy=multi-user.target
SERVICEEOF

systemctl daemon-reload
systemctl enable nofx.service > /dev/null 2>&1
print_success "systemd服务创建完成"

# ═══════════════════════════════════════════════════════════════
# 步骤10：创建管理脚本
# ═══════════════════════════════════════════════════════════════
print_info "步骤10/10: 创建管理脚本"

# 创建监控脚本
cat > /opt/nofx/monitor.sh << 'MONITOREOF'
#!/bin/bash

echo "=== NOFX系统监控 ==="
echo ""
echo "1. 系统资源："
free -h
echo ""
df -h / | grep -v Filesystem
echo ""

echo "2. Docker容器状态："
docker ps --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"
echo ""

echo "3. 内存使用："
docker stats --no-stream --format "table {{.Name}}\t{{.MemUsage}}\t{{.CPUPerc}}"
echo ""

echo "4. 服务健康检查："
curl -s http://localhost:8080/health | jq '.' 2>/dev/null || echo "后端未响应"
echo ""

echo "5. 最近日志（最后10行）："
docker logs --tail 10 nofx-trading 2>&1 | tail -10
MONITOREOF

chmod +x /opt/nofx/monitor.sh

# 创建备份脚本
cat > /opt/nofx/backup.sh << 'BACKUPEOF'
#!/bin/bash
BACKUP_DIR="/opt/nofx_backups"
DATE=$(date +%Y%m%d_%H%M%S)

mkdir -p $BACKUP_DIR
tar -czf $BACKUP_DIR/nofx_backup_$DATE.tar.gz \
    /opt/nofx/config.json \
    /opt/nofx/decision_logs \
    /opt/nofx/data 2>/dev/null

# 保留最近7天的备份
find $BACKUP_DIR -name "nofx_backup_*.tar.gz" -mtime +7 -delete 2>/dev/null

echo "备份完成: $BACKUP_DIR/nofx_backup_$DATE.tar.gz"
BACKUPEOF

chmod +x /opt/nofx/backup.sh

# 添加到crontab（每天凌晨2点备份）
(crontab -l 2>/dev/null | grep -v "nofx/backup.sh"; echo "0 2 * * * /opt/nofx/backup.sh") | crontab -

print_success "管理脚本创建完成"

# ═══════════════════════════════════════════════════════════════
# 部署完成
# ═══════════════════════════════════════════════════════════════
echo ""
print_success "═══════════════════════════════════════════════════════════════"
print_success "                    部署准备完成！                              "
print_success "═══════════════════════════════════════════════════════════════"
echo ""

print_warning "⚠️  重要：在启动服务前，请先配置API密钥"
echo ""
print_info "1. 编辑配置文件："
echo "   nano /opt/nofx/config.json"
echo ""
print_info "2. 填入以下信息："
echo "   - 交易所API密钥（Binance/Hyperliquid/Aster）"
echo "   - AI API密钥（DeepSeek/Qwen/OpenAI）"
echo "   - 初始余额等参数"
echo ""
print_info "3. 关闭代理配置（阿里云服务器通常不需要代理）："
echo '   "proxy": { "enabled": false }'
echo ""
print_info "4. 启动服务："
echo "   cd /opt/nofx"
echo "   ./scripts/start.sh start --build"
echo ""
print_info "5. 查看服务状态："
echo "   ./scripts/start.sh status"
echo "   /opt/nofx/monitor.sh"
echo ""
print_info "6. 访问Web界面："
echo "   http://$(curl -s ifconfig.me):3000"
echo ""
print_info "常用命令："
echo "   启动: cd /opt/nofx && ./scripts/start.sh start"
echo "   停止: ./scripts/start.sh stop"
echo "   日志: ./scripts/start.sh logs"
echo "   监控: /opt/nofx/monitor.sh"
echo "   备份: /opt/nofx/backup.sh"
echo ""
print_success "═══════════════════════════════════════════════════════════════"

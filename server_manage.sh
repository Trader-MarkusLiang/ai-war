#!/bin/bash

# ═══════════════════════════════════════════════════════════════
# NOFX 服务器管理脚本
# 用于远程管理阿里云服务器上的NOFX服务
# ═══════════════════════════════════════════════════════════════

SERVER="root@47.236.159.60"
PROJECT_DIR="/opt/nofx"

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

print_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
print_success() { echo -e "${GREEN}[SUCCESS]${NC} $1"; }
print_warning() { echo -e "${YELLOW}[WARNING]${NC} $1"; }
print_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# 执行远程命令
remote_exec() {
    ssh ${SERVER} "$@"
}

# 查看服务状态
show_status() {
    print_info "查询服务器状态..."
    echo ""

    remote_exec << 'EOF'
echo "=== NOFX服务状态 ==="
echo ""
echo "1. Docker容器状态："
docker ps --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"
echo ""
echo "2. 系统资源："
free -h
echo ""
df -h / | grep -v Filesystem
echo ""
echo "3. 容器资源使用："
docker stats --no-stream --format "table {{.Name}}\t{{.MemUsage}}\t{{.CPUPerc}}"
echo ""
echo "4. 服务健康检查："
curl -s http://localhost:8080/api/health | jq '.' 2>/dev/null || echo "后端未响应"
EOF
}

# 查看日志
show_logs() {
    local service="${1:-all}"
    print_info "查看日志: ${service}"
    echo ""

    case "$service" in
        backend|nofx)
            remote_exec "docker logs --tail 50 -f nofx-trading"
            ;;
        frontend)
            remote_exec "docker logs --tail 50 -f nofx-frontend"
            ;;
        all|*)
            remote_exec "cd ${PROJECT_DIR} && ./start.sh logs"
            ;;
    esac
}

# 启动服务
start_service() {
    print_info "启动服务..."
    remote_exec "cd ${PROJECT_DIR} && ./start.sh start"
    print_success "服务启动命令已执行"
}

# 停止服务
stop_service() {
    print_info "停止服务..."
    remote_exec "cd ${PROJECT_DIR} && ./start.sh stop"
    print_success "服务已停止"
}

# 重启服务
restart_service() {
    print_info "重启服务..."
    remote_exec "cd ${PROJECT_DIR} && ./start.sh restart"
    print_success "服务重启命令已执行"
}

# 重新构建
rebuild_service() {
    print_warning "这将重新构建Docker镜像，可能需要5-10分钟"
    read -p "确认继续？(y/n): " confirm

    if [[ "$confirm" =~ ^[Yy]$ ]]; then
        print_info "重新构建服务..."
        remote_exec "cd ${PROJECT_DIR} && ./start.sh stop && ./start.sh start --build"
        print_success "重新构建完成"
    else
        print_info "已取消"
    fi
}

# 备份数据
backup_data() {
    print_info "执行数据备份..."
    remote_exec "${PROJECT_DIR}/backup.sh"

    print_info "备份文件列表："
    remote_exec "ls -lh /opt/nofx_backups/ | tail -5"

    print_info "是否下载最新备份到本地？(y/n)"
    read -p "> " download

    if [[ "$download" =~ ^[Yy]$ ]]; then
        local backup_file=$(remote_exec "ls -t /opt/nofx_backups/*.tar.gz | head -1")
        local local_dir="./backups"
        mkdir -p "$local_dir"

        print_info "下载备份文件..."
        scp ${SERVER}:${backup_file} ${local_dir}/
        print_success "备份已下载到: ${local_dir}/"
    fi
}

# 更新代码
update_code() {
    print_warning "这将从GitHub拉取最新代码并重启服务"
    read -p "确认继续？(y/n): " confirm

    if [[ "$confirm" =~ ^[Yy]$ ]]; then
        print_info "更新代码..."
        remote_exec "cd ${PROJECT_DIR} && git pull && ./start.sh restart"
        print_success "代码更新完成"
    else
        print_info "已取消"
    fi
}

# 系统监控
monitor_system() {
    print_info "系统监控（按Ctrl+C退出）"
    echo ""

    remote_exec "${PROJECT_DIR}/monitor.sh"
}

# 编辑配置
edit_config() {
    print_info "下载配置文件..."
    scp ${SERVER}:${PROJECT_DIR}/config.json ./config.json.tmp

    print_info "请编辑配置文件: config.json.tmp"
    ${EDITOR:-nano} ./config.json.tmp

    print_info "是否上传修改后的配置？(y/n)"
    read -p "> " upload

    if [[ "$upload" =~ ^[Yy]$ ]]; then
        print_info "上传配置文件..."
        scp ./config.json.tmp ${SERVER}:${PROJECT_DIR}/config.json
        rm ./config.json.tmp

        print_info "是否重启服务以应用配置？(y/n)"
        read -p "> " restart

        if [[ "$restart" =~ ^[Yy]$ ]]; then
            restart_service
        fi
    else
        rm ./config.json.tmp
        print_info "已取消"
    fi
}

# 查看配置
view_config() {
    print_info "当前配置："
    echo ""
    remote_exec "cat ${PROJECT_DIR}/config.json | jq '.'"
}

# SSH登录
ssh_login() {
    print_info "SSH登录到服务器..."
    ssh ${SERVER}
}

# 清理日志
clean_logs() {
    print_warning "这将清理30天前的日志文件"
    read -p "确认继续？(y/n): " confirm

    if [[ "$confirm" =~ ^[Yy]$ ]]; then
        print_info "清理日志..."
        remote_exec "find ${PROJECT_DIR}/decision_logs -name '*.json' -mtime +30 -delete"
        print_success "日志清理完成"
    else
        print_info "已取消"
    fi
}

# 诊断问题
diagnose() {
    print_info "运行诊断脚本..."
    echo ""

    remote_exec << 'EOF'
echo "=== NOFX服务诊断 ==="
echo ""

echo "1. Docker服务状态："
systemctl status docker --no-pager | grep -E "Active|Main PID"
echo ""

echo "2. Docker容器状态："
docker ps -a --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"
echo ""

echo "3. 端口监听："
netstat -tlnp | grep -E '3000|8080'
echo ""

echo "4. 磁盘空间："
df -h / | grep -v Filesystem
echo ""

echo "5. 内存使用："
free -h
echo ""

echo "6. Swap状态："
swapon --show
echo ""

echo "7. 最近错误日志："
docker logs --tail 20 nofx-trading 2>&1 | grep -i error || echo "无错误"
echo ""

echo "8. 配置文件检查："
if [ -f /opt/nofx/config.json ]; then
    echo "✓ config.json 存在"
    jq -e . /opt/nofx/config.json > /dev/null 2>&1 && echo "✓ JSON格式正确" || echo "✗ JSON格式错误"
else
    echo "✗ config.json 不存在"
fi
echo ""

echo "=== 诊断完成 ==="
EOF
}

# 显示帮助
show_help() {
    cat << EOF
NOFX 服务器管理脚本

用法: ./server_manage.sh [命令] [参数]

命令:
  status          查看服务状态
  logs [service]  查看日志 (backend/frontend/all)
  start           启动服务
  stop            停止服务
  restart         重启服务
  rebuild         重新构建服务
  backup          备份数据
  update          更新代码
  monitor         系统监控
  config          编辑配置文件
  view-config     查看当前配置
  ssh             SSH登录到服务器
  clean-logs      清理旧日志
  diagnose        诊断问题
  help            显示此帮助信息

示例:
  ./server_manage.sh status           # 查看状态
  ./server_manage.sh logs backend     # 查看后端日志
  ./server_manage.sh restart          # 重启服务
  ./server_manage.sh backup           # 备份数据

服务器信息:
  地址: ${SERVER}
  项目路径: ${PROJECT_DIR}

EOF
}

# 主函数
main() {
    case "${1:-help}" in
        status)
            show_status
            ;;
        logs)
            show_logs "$2"
            ;;
        start)
            start_service
            ;;
        stop)
            stop_service
            ;;
        restart)
            restart_service
            ;;
        rebuild)
            rebuild_service
            ;;
        backup)
            backup_data
            ;;
        update)
            update_code
            ;;
        monitor)
            monitor_system
            ;;
        config|edit-config)
            edit_config
            ;;
        view-config)
            view_config
            ;;
        ssh|login)
            ssh_login
            ;;
        clean-logs)
            clean_logs
            ;;
        diagnose|debug)
            diagnose
            ;;
        help|--help|-h)
            show_help
            ;;
        *)
            print_error "未知命令: $1"
            show_help
            exit 1
            ;;
    esac
}

main "$@"

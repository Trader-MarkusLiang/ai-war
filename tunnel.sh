#!/bin/bash

# ═══════════════════════════════════════════════════════════════
# NOFX SSH隧道管理脚本
# 用于在本地访问阿里云服务器上的NOFX服务
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
NC='\033[0m'

print_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
print_warning() { echo -e "${YELLOW}[WARNING]${NC} $1"; }
print_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# 检查隧道是否运行
check_tunnel() {
    if pgrep -f "ssh.*${LOCAL_FRONTEND_PORT}:localhost:${REMOTE_FRONTEND_PORT}" > /dev/null; then
        return 0
    else
        return 1
    fi
}

# 启动隧道
start_tunnel() {
    if check_tunnel; then
        print_warning "SSH隧道已经在运行"
        show_status
        return
    fi

    print_info "启动SSH隧道..."
    ssh -f -N \
        -L ${LOCAL_FRONTEND_PORT}:localhost:${REMOTE_FRONTEND_PORT} \
        -L ${LOCAL_BACKEND_PORT}:localhost:${REMOTE_BACKEND_PORT} \
        ${SERVER}

    if [ $? -eq 0 ]; then
        sleep 1
        if check_tunnel; then
            print_info "SSH隧道启动成功！"
            echo ""
            print_info "访问地址："
            echo "  前端: http://localhost:${LOCAL_FRONTEND_PORT}"
            echo "  后端: http://localhost:${LOCAL_BACKEND_PORT}"
        else
            print_error "SSH隧道启动失败"
        fi
    else
        print_error "SSH连接失败"
    fi
}

# 停止隧道
stop_tunnel() {
    if ! check_tunnel; then
        print_warning "SSH隧道未运行"
        return
    fi

    print_info "停止SSH隧道..."
    pkill -f "ssh.*${LOCAL_FRONTEND_PORT}:localhost:${REMOTE_FRONTEND_PORT}"

    sleep 1
    if ! check_tunnel; then
        print_info "SSH隧道已停止"
    else
        print_error "停止失败，请手动执行: pkill -f 'ssh.*${LOCAL_FRONTEND_PORT}'"
    fi
}

# 重启隧道
restart_tunnel() {
    print_info "重启SSH隧道..."
    stop_tunnel
    sleep 1
    start_tunnel
}

# 显示状态
show_status() {
    echo "=== SSH隧道状态 ==="
    echo ""

    if check_tunnel; then
        print_info "状态: 运行中 ✓"
        echo ""
        echo "进程信息:"
        ps aux | grep "ssh.*${LOCAL_FRONTEND_PORT}" | grep -v grep
        echo ""
        echo "端口映射:"
        echo "  本地 ${LOCAL_FRONTEND_PORT} -> 服务器 ${REMOTE_FRONTEND_PORT} (前端)"
        echo "  本地 ${LOCAL_BACKEND_PORT} -> 服务器 ${REMOTE_BACKEND_PORT} (后端)"
        echo ""
        echo "访问地址:"
        echo "  前端: http://localhost:${LOCAL_FRONTEND_PORT}"
        echo "  后端: http://localhost:${LOCAL_BACKEND_PORT}"
    else
        print_warning "状态: 未运行 ✗"
    fi
}

# 前台运行（用于调试）
run_foreground() {
    print_info "前台运行SSH隧道（按Ctrl+C停止）..."
    ssh -L ${LOCAL_FRONTEND_PORT}:localhost:${REMOTE_FRONTEND_PORT} \
        -L ${LOCAL_BACKEND_PORT}:localhost:${REMOTE_BACKEND_PORT} \
        ${SERVER}
}

# 显示帮助
show_help() {
    cat << EOF
NOFX SSH隧道管理脚本

用法: ./tunnel.sh [命令]

命令:
  start       启动SSH隧道（后台运行）
  stop        停止SSH隧道
  restart     重启SSH隧道
  status      查看隧道状态
  fg          前台运行（用于调试）
  help        显示此帮助信息

示例:
  ./tunnel.sh start    # 启动隧道
  ./tunnel.sh status   # 查看状态
  ./tunnel.sh stop     # 停止隧道

配置:
  服务器: ${SERVER}
  本地前端端口: ${LOCAL_FRONTEND_PORT}
  本地后端端口: ${LOCAL_BACKEND_PORT}

EOF
}

# 主函数
main() {
    case "${1:-status}" in
        start)
            start_tunnel
            ;;
        stop)
            stop_tunnel
            ;;
        restart)
            restart_tunnel
            ;;
        status)
            show_status
            ;;
        fg|foreground)
            run_foreground
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

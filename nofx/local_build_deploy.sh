#!/bin/bash

# ═══════════════════════════════════════════════════════════════
# NOFX 本地编译部署脚本
# 在本地编译 Docker 镜像，然后上传到远程服务器
# 解决远程服务器内存不足无法编译的问题
# ═══════════════════════════════════════════════════════════════

set -e

# ------------------------------------------------------------------------
# 配置
# ------------------------------------------------------------------------
REMOTE_HOST="root@47.236.159.60"
REMOTE_DIR="/opt/nofx"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# 镜像名称
BACKEND_IMAGE="nofx-trading"
FRONTEND_IMAGE="nofx-frontend"

# Docker 命令（自动检测是否需要 sudo）
DOCKER_CMD="docker"
if ! docker info &>/dev/null; then
    DOCKER_CMD="sudo docker"
fi

# ------------------------------------------------------------------------
# 颜色定义
# ------------------------------------------------------------------------
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

print_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
print_success() { echo -e "${GREEN}[SUCCESS]${NC} $1"; }
print_warning() { echo -e "${YELLOW}[WARNING]${NC} $1"; }
print_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# ------------------------------------------------------------------------
# 检查依赖
# ------------------------------------------------------------------------
check_dependencies() {
    print_info "检查依赖..."

    if ! command -v docker &> /dev/null; then
        print_error "Docker 未安装"
        exit 1
    fi

    if ! command -v ssh &> /dev/null; then
        print_error "SSH 未安装"
        exit 1
    fi

    print_success "依赖检查通过"
}

# ------------------------------------------------------------------------
# 测试远程连接
# ------------------------------------------------------------------------
test_connection() {
    print_info "测试远程服务器连接..."

    if ! ssh -o ConnectTimeout=10 $REMOTE_HOST "echo 'connected'" &> /dev/null; then
        print_error "无法连接到远程服务器 $REMOTE_HOST"
        exit 1
    fi

    print_success "远程服务器连接正常"
}

# ------------------------------------------------------------------------
# 本地编译后端镜像
# ------------------------------------------------------------------------
build_backend() {
    print_info "开始编译后端镜像..."
    print_info "这可能需要几分钟时间..."

    cd "$SCRIPT_DIR"

    $DOCKER_CMD build \
        -t $BACKEND_IMAGE \
        -f ./docker/Dockerfile.backend \
        . \
        2>&1 | while read line; do
            echo "  $line"
        done

    if [ ${PIPESTATUS[0]} -eq 0 ]; then
        print_success "后端镜像编译完成"
    else
        print_error "后端镜像编译失败"
        exit 1
    fi
}

# ------------------------------------------------------------------------
# 本地编译前端镜像
# ------------------------------------------------------------------------
build_frontend() {
    print_info "开始编译前端镜像..."

    cd "$SCRIPT_DIR"

    $DOCKER_CMD build \
        -t $FRONTEND_IMAGE \
        -f ./docker/Dockerfile.frontend \
        . \
        2>&1 | while read line; do
            echo "  $line"
        done

    if [ ${PIPESTATUS[0]} -eq 0 ]; then
        print_success "前端镜像编译完成"
    else
        print_error "前端镜像编译失败"
        exit 1
    fi
}

# ------------------------------------------------------------------------
# 上传镜像到远程服务器
# ------------------------------------------------------------------------
upload_image() {
    local image_name=$1
    print_info "上传镜像 $image_name 到远程服务器..."
    print_warning "这可能需要较长时间，取决于网络速度..."

    # 获取镜像大小
    local size=$($DOCKER_CMD image inspect $image_name --format='{{.Size}}' 2>/dev/null)
    local size_mb=$((size / 1024 / 1024))
    print_info "镜像大小: ${size_mb}MB"

    # 使用 pv 显示进度（如果可用），否则直接传输
    if command -v pv &> /dev/null; then
        $DOCKER_CMD save $image_name | pv -s $size | ssh $REMOTE_HOST "docker load"
    else
        $DOCKER_CMD save $image_name | ssh $REMOTE_HOST "docker load"
    fi

    if [ $? -eq 0 ]; then
        print_success "镜像 $image_name 上传完成"
    else
        print_error "镜像 $image_name 上传失败"
        exit 1
    fi
}

# ------------------------------------------------------------------------
# 同步配置文件
# ------------------------------------------------------------------------
sync_config() {
    print_info "同步配置文件到远程服务器..."

    # 同步必要的配置文件（不包括 .env）
    rsync -avz --progress \
        --exclude='node_modules' \
        --exclude='.git' \
        --exclude='data' \
        --exclude='nofx' \
        --exclude='.env' \
        "$SCRIPT_DIR/docker-compose.yml" \
        "$SCRIPT_DIR/nginx/" \
        $REMOTE_HOST:$REMOTE_DIR/

    print_success "配置文件同步完成"
}

# ------------------------------------------------------------------------
# 远程启动服务
# ------------------------------------------------------------------------
start_remote() {
    print_info "在远程服务器启动服务..."

    ssh $REMOTE_HOST "cd $REMOTE_DIR && docker compose up -d"

    if [ $? -eq 0 ]; then
        print_success "服务启动成功"
    else
        print_error "服务启动失败"
        exit 1
    fi
}

# ------------------------------------------------------------------------
# 远程停止服务
# ------------------------------------------------------------------------
stop_remote() {
    print_info "停止远程服务..."

    ssh $REMOTE_HOST "cd $REMOTE_DIR && docker compose stop" || true

    print_success "服务已停止"
}

# ------------------------------------------------------------------------
# 查看远程状态
# ------------------------------------------------------------------------
status_remote() {
    print_info "远程服务状态:"
    ssh $REMOTE_HOST "cd $REMOTE_DIR && docker compose ps"
}

# ------------------------------------------------------------------------
# 查看远程日志
# ------------------------------------------------------------------------
logs_remote() {
    local service=$1
    if [ -z "$service" ]; then
        ssh $REMOTE_HOST "cd $REMOTE_DIR && docker compose logs -f --tail=100"
    else
        ssh $REMOTE_HOST "cd $REMOTE_DIR && docker compose logs -f --tail=100 $service"
    fi
}

# ------------------------------------------------------------------------
# 完整部署流程
# ------------------------------------------------------------------------
deploy_all() {
    print_info "=========================================="
    print_info "开始完整部署流程"
    print_info "=========================================="

    check_dependencies
    test_connection

    # 停止远程服务
    stop_remote

    # 编译镜像
    build_backend
    build_frontend

    # 上传镜像
    upload_image $BACKEND_IMAGE
    upload_image $FRONTEND_IMAGE

    # 同步配置
    sync_config

    # 启动服务
    start_remote

    # 显示状态
    sleep 3
    status_remote

    print_info "=========================================="
    print_success "部署完成!"
    print_info "=========================================="
}

# ------------------------------------------------------------------------
# 仅部署后端
# ------------------------------------------------------------------------
deploy_backend() {
    print_info "仅部署后端..."

    check_dependencies
    test_connection
    stop_remote
    build_backend
    upload_image $BACKEND_IMAGE
    start_remote

    print_success "后端部署完成"
}

# ------------------------------------------------------------------------
# 仅部署前端
# ------------------------------------------------------------------------
deploy_frontend() {
    print_info "仅部署前端..."

    check_dependencies
    test_connection
    stop_remote
    build_frontend
    upload_image $FRONTEND_IMAGE
    start_remote

    print_success "前端部署完成"
}

# ------------------------------------------------------------------------
# 帮助信息
# ------------------------------------------------------------------------
show_help() {
    echo "NOFX 本地编译部署脚本"
    echo ""
    echo "用法: ./local_build_deploy.sh [command]"
    echo ""
    echo "命令:"
    echo "  deploy          完整部署（编译+上传+启动）"
    echo "  backend         仅部署后端"
    echo "  frontend        仅部署前端"
    echo "  build           仅本地编译（不上传）"
    echo "  upload          仅上传镜像（不编译）"
    echo "  start           启动远程服务"
    echo "  stop            停止远程服务"
    echo "  restart         重启远程服务"
    echo "  status          查看远程服务状态"
    echo "  logs [service]  查看远程日志"
    echo "  help            显示帮助"
    echo ""
    echo "示例:"
    echo "  ./local_build_deploy.sh deploy    # 完整部署"
    echo "  ./local_build_deploy.sh backend   # 仅部署后端"
    echo "  ./local_build_deploy.sh logs nofx # 查看后端日志"
}

# ------------------------------------------------------------------------
# 主函数
# ------------------------------------------------------------------------
main() {
    case "${1:-help}" in
        deploy)
            deploy_all
            ;;
        backend)
            deploy_backend
            ;;
        frontend)
            deploy_frontend
            ;;
        build)
            check_dependencies
            build_backend
            build_frontend
            print_success "本地编译完成"
            ;;
        upload)
            check_dependencies
            test_connection
            upload_image $BACKEND_IMAGE
            upload_image $FRONTEND_IMAGE
            print_success "镜像上传完成"
            ;;
        start)
            test_connection
            start_remote
            ;;
        stop)
            test_connection
            stop_remote
            ;;
        restart)
            test_connection
            stop_remote
            start_remote
            ;;
        status)
            test_connection
            status_remote
            ;;
        logs)
            test_connection
            logs_remote "$2"
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

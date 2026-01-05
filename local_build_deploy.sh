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
NOFX_DIR="$SCRIPT_DIR/nofx"

# 镜像名称
BACKEND_IMAGE="nofx-trading"
FRONTEND_IMAGE="nofx-frontend"

# 代理配置（SOCKS5 代理，如 clash）
# 设置为空则不使用代理
PROXY_HOST="127.0.0.1"
PROXY_PORT="7890"
USE_PROXY="false"  # 设置为 "true" 启用代理

# 是否使用 gzip 压缩（推荐，可加速 3-5 倍）
USE_GZIP="true"

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
# 本地编译后端镜像 (Docker)
# ------------------------------------------------------------------------
build_backend() {
    print_info "开始编译后端镜像..."
    print_info "这可能需要几分钟时间..."

    cd "$NOFX_DIR"

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
# 快速编译后端二进制 (Go 直接编译，不用 Docker)
# ------------------------------------------------------------------------
build_backend_binary() {
    print_info "开始编译后端二进制文件..."

    if ! command -v go &> /dev/null; then
        print_error "Go 未安装，请先安装 Go"
        exit 1
    fi

    cd "$NOFX_DIR"

    print_info "Go 版本: $(go version)"
    print_info "编译目标: linux/amd64"

    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o nofx_new .

    if [ $? -eq 0 ]; then
        local size=$(ls -lh nofx_new | awk '{print $5}')
        print_success "后端二进制编译完成 (大小: $size)"
    else
        print_error "后端二进制编译失败"
        exit 1
    fi
}

# ------------------------------------------------------------------------
# 上传后端二进制到服务器
# ------------------------------------------------------------------------
upload_backend_binary() {
    print_info "上传后端二进制到服务器..."

    local binary_path="$NOFX_DIR/nofx_new"
    if [ ! -f "$binary_path" ]; then
        print_error "二进制文件不存在: $binary_path"
        exit 1
    fi

    local size=$(ls -lh "$binary_path" | awk '{print $5}')
    print_info "二进制大小: $size"

    # 构建 SSH 命令
    local ssh_opts=""
    if [ "$USE_PROXY" = "true" ] && [ -n "$PROXY_HOST" ] && [ -n "$PROXY_PORT" ]; then
        ssh_opts="-o ProxyCommand='nc -X 5 -x ${PROXY_HOST}:${PROXY_PORT} %h %p'"
        print_info "使用代理: ${PROXY_HOST}:${PROXY_PORT}"
    fi

    print_warning "传输中，请耐心等待..."

    # 使用 gzip 压缩传输
    if [ "$USE_GZIP" = "true" ]; then
        gzip -c "$binary_path" | eval "ssh $ssh_opts $REMOTE_HOST 'gunzip > $REMOTE_DIR/nofx_new && chmod +x $REMOTE_DIR/nofx_new'"
    else
        eval "scp $ssh_opts '$binary_path' '$REMOTE_HOST:$REMOTE_DIR/nofx_new'"
        ssh $REMOTE_HOST "chmod +x $REMOTE_DIR/nofx_new"
    fi

    if [ $? -eq 0 ]; then
        print_success "二进制上传完成"
    else
        print_error "二进制上传失败"
        exit 1
    fi
}

# ------------------------------------------------------------------------
# 替换服务器上的后端二进制并重启
# ------------------------------------------------------------------------
replace_backend_binary() {
    print_info "替换服务器上的后端二进制..."

    ssh $REMOTE_HOST "cd $REMOTE_DIR && \
        mv nofx nofx.old 2>/dev/null || true && \
        mv nofx_new nofx && \
        ls -la nofx"

    if [ $? -eq 0 ]; then
        print_success "二进制替换完成"
    else
        print_error "二进制替换失败"
        exit 1
    fi
}

# ------------------------------------------------------------------------
# 快速部署后端 (Go 编译 + 上传二进制，不用 Docker)
# ------------------------------------------------------------------------
quick_backend() {
    print_info "=========================================="
    print_info "快速部署后端 (Go 直接编译)"
    print_info "=========================================="

    test_connection
    build_backend_binary
    upload_backend_binary
    replace_backend_binary

    # 重启后端服务
    print_info "重启后端服务..."
    ssh $REMOTE_HOST "cd $REMOTE_DIR && docker compose restart nofx"

    sleep 3
    status_remote

    # 清理本地临时文件
    rm -f "$NOFX_DIR/nofx_new"

    print_info "=========================================="
    print_success "快速部署完成!"
    print_info "=========================================="
}

# ------------------------------------------------------------------------
# 同步后端代码到服务器
# ------------------------------------------------------------------------
sync_backend_code() {
    print_info "同步后端代码到服务器..."

    rsync -avz --progress \
        --exclude='node_modules' \
        --exclude='.git' \
        --exclude='data' \
        --exclude='web' \
        --exclude='nofx' \
        --exclude='nofx_new' \
        --exclude='*.old' \
        "$NOFX_DIR/" \
        $REMOTE_HOST:$REMOTE_DIR/

    print_success "后端代码同步完成"
}

# ------------------------------------------------------------------------
# 本地编译前端镜像
# ------------------------------------------------------------------------
build_frontend() {
    print_info "开始编译前端镜像..."

    cd "$NOFX_DIR"

    # 检查是否需要清除缓存
    local use_no_cache=""
    local dist_mtime=""
    local image_mtime=""

    # 获取 dist 目录的最新修改时间
    if [ -d "web/dist" ]; then
        dist_mtime=$(find web/dist -type f -printf '%T@\n' 2>/dev/null | sort -n | tail -1)
    fi

    # 获取现有镜像的创建时间
    if $DOCKER_CMD image inspect $FRONTEND_IMAGE &>/dev/null; then
        image_mtime=$($DOCKER_CMD image inspect $FRONTEND_IMAGE --format='{{.Created}}' 2>/dev/null)
        image_timestamp=$(date -d "$image_mtime" +%s 2>/dev/null || echo "0")

        # 如果 dist 比镜像新，使用 --no-cache
        if [ -n "$dist_mtime" ] && [ "$dist_mtime" != "" ]; then
            dist_timestamp=$(echo $dist_mtime | cut -d. -f1)
            if [ "$dist_timestamp" -gt "$image_timestamp" ]; then
                print_warning "检测到前端代码已更新，将清除 Docker 缓存重新构建"
                use_no_cache="--no-cache"
                # 删除本地旧镜像，确保完全重新构建
                print_info "删除本地旧镜像..."
                $DOCKER_CMD rmi $FRONTEND_IMAGE 2>/dev/null || true
            fi
        fi
    else
        # 镜像不存在，强制使用 --no-cache
        print_warning "镜像不存在，将使用 --no-cache 重新构建"
        use_no_cache="--no-cache"
    fi

    # 显示构建参数
    if [ -n "$use_no_cache" ]; then
        print_info "构建参数: $use_no_cache"
    fi

    $DOCKER_CMD build \
        $use_no_cache \
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

    # 获取镜像大小
    local size=$($DOCKER_CMD image inspect $image_name --format='{{.Size}}' 2>/dev/null)
    local size_mb=$((size / 1024 / 1024))
    print_info "镜像大小: ${size_mb}MB"

    # 构建 SSH 选项
    local ssh_cmd="ssh"
    if [ "$USE_PROXY" = "true" ] && [ -n "$PROXY_HOST" ] && [ -n "$PROXY_PORT" ]; then
        # 检测可用的代理工具
        if command -v ncat &> /dev/null; then
            ssh_cmd="ssh -o ProxyCommand='ncat --proxy-type socks5 --proxy ${PROXY_HOST}:${PROXY_PORT} %h %p'"
        elif command -v connect-proxy &> /dev/null; then
            ssh_cmd="ssh -o ProxyCommand='connect-proxy -S ${PROXY_HOST}:${PROXY_PORT} %h %p'"
        else
            ssh_cmd="ssh -o ProxyCommand='nc -X 5 -x ${PROXY_HOST}:${PROXY_PORT} %h %p'"
        fi
        print_info "使用代理: ${PROXY_HOST}:${PROXY_PORT}"
    fi

    # 显示传输模式
    if [ "$USE_GZIP" = "true" ]; then
        print_info "使用 gzip 压缩传输（预计压缩后约 ${size_mb}MB -> $((size_mb / 3))MB）"
    fi
    print_warning "传输中，请耐心等待..."

    # 构建传输命令
    local result=0
    if [ "$USE_GZIP" = "true" ]; then
        # 使用 gzip 压缩
        if command -v pv &> /dev/null; then
            $DOCKER_CMD save $image_name | gzip | pv | eval "$ssh_cmd $REMOTE_HOST 'gunzip | docker load'"
            result=$?
        else
            $DOCKER_CMD save $image_name | gzip | eval "$ssh_cmd $REMOTE_HOST 'gunzip | docker load'"
            result=$?
        fi
    else
        # 不压缩
        if command -v pv &> /dev/null; then
            $DOCKER_CMD save $image_name | pv -s $size | eval "$ssh_cmd $REMOTE_HOST 'docker load'"
            result=$?
        else
            $DOCKER_CMD save $image_name | eval "$ssh_cmd $REMOTE_HOST 'docker load'"
            result=$?
        fi
    fi

    if [ $result -eq 0 ]; then
        print_success "镜像 $image_name 上传完成"
    else
        print_error "镜像 $image_name 上传失败"
        exit 1
    fi
}

# ------------------------------------------------------------------------
# 清理远程服务器上的旧镜像和容器
# ------------------------------------------------------------------------
clean_remote_cache() {
    local service=$1
    print_info "清理远程服务器上的 $service 缓存..."

    # 停止并删除容器
    ssh $REMOTE_HOST "cd $REMOTE_DIR && docker compose down $service" 2>/dev/null || true

    # 删除旧镜像
    ssh $REMOTE_HOST "docker rmi nofx-$service 2>/dev/null || true"

    # 清理悬空镜像
    ssh $REMOTE_HOST "docker image prune -f" 2>/dev/null || true

    print_success "远程 $service 缓存已清理"
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
        "$NOFX_DIR/docker-compose.yml" \
        "$NOFX_DIR/nginx/" \
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

    # 1. 停止远程服务
    stop_remote

    # 2. 清理远程缓存
    clean_remote_cache "frontend"

    # 3. 清理本地旧镜像，强制重新构建
    print_info "清理本地旧镜像..."
    $DOCKER_CMD rmi $FRONTEND_IMAGE 2>/dev/null || true

    # 4. 本地构建新镜像（使用 --no-cache）
    build_frontend

    # 5. 上传镜像到服务器
    upload_image $FRONTEND_IMAGE

    # 6. 在服务器上验证镜像
    print_info "验证服务器上的镜像..."
    ssh $REMOTE_HOST "docker image inspect $FRONTEND_IMAGE --format='镜像创建时间: {{.Created}}'"

    # 7. 启动服务
    start_remote

    # 8. 等待容器启动
    print_info "等待容器启动..."
    sleep 5

    # 9. 验证容器内文件
    print_info "验证容器内文件时间戳..."
    ssh $REMOTE_HOST "docker exec nofx-frontend ls -lh /usr/share/nginx/html/assets/ | head -5"

    # 10. 显示服务状态
    status_remote

    print_success "前端部署完成"
}

# ------------------------------------------------------------------------
# 帮助信息
# ------------------------------------------------------------------------
show_help() {
    echo "NOFX 本地编译部署脚本"
    echo ""
    echo "用法: ./local_build_deploy.sh [选项] <命令>"
    echo ""
    echo "命令:"
    echo "  deploy          完整部署（Docker 编译+上传+启动）"
    echo "  backend         部署后端（Docker 镜像方式）"
    echo "  frontend        部署前端（Docker 镜像方式）"
    echo "  quick           ⚡ 快速部署后端（Go 直接编译，推荐）"
    echo "  sync            同步后端代码到服务器"
    echo "  build           仅本地编译 Docker 镜像（不上传）"
    echo "  upload          仅上传 Docker 镜像（不编译）"
    echo "  start           启动远程服务"
    echo "  stop            停止远程服务"
    echo "  restart         重启远程服务"
    echo "  status          查看远程服务状态"
    echo "  logs [service]  查看远程日志"
    echo "  help            显示帮助"
    echo ""
    echo "选项:"
    echo "  --proxy         启用代理 (默认: ${PROXY_HOST}:${PROXY_PORT})"
    echo "  --proxy=HOST:PORT  指定代理地址"
    echo "  --no-gzip       禁用 gzip 压缩"
    echo ""
    echo "示例:"
    echo "  ./local_build_deploy.sh --proxy quick         # ⚡ 快速部署后端（推荐）"
    echo "  ./local_build_deploy.sh --proxy frontend      # 使用代理部署前端"
    echo "  ./local_build_deploy.sh --proxy backend       # Docker 方式部署后端"
    echo "  ./local_build_deploy.sh sync                  # 同步代码到服务器"
}

# ------------------------------------------------------------------------
# 主函数
# ------------------------------------------------------------------------
main() {
    # 解析选项参数
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --proxy)
                USE_PROXY="true"
                shift
                ;;
            --proxy=*)
                USE_PROXY="true"
                local proxy_addr="${1#*=}"
                PROXY_HOST="${proxy_addr%:*}"
                PROXY_PORT="${proxy_addr#*:}"
                shift
                ;;
            --no-gzip)
                USE_GZIP="false"
                shift
                ;;
            *)
                break
                ;;
        esac
    done

    # 执行命令
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
        quick|q)
            quick_backend
            ;;
        sync)
            test_connection
            sync_backend_code
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

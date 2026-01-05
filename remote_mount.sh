#!/bin/bash

# ═══════════════════════════════════════════════════════════════
# NOFX 远程目录挂载脚本
# 使用SSHFS将服务器目录挂载到本地，方便编辑
# ═══════════════════════════════════════════════════════════════

SERVER="root@47.236.159.60"
REMOTE_DIR="/opt/nofx"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
LOCAL_MOUNT="$SCRIPT_DIR/remote-nofx"

# 颜色定义
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

print_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
print_warning() { echo -e "${YELLOW}[WARNING]${NC} $1"; }
print_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# 检查SSHFS是否安装
check_sshfs() {
    if ! command -v sshfs &> /dev/null; then
        print_error "SSHFS未安装"
        print_info "安装命令: sudo apt-get install sshfs"
        exit 1
    fi
}

# 检查是否已挂载
is_mounted() {
    mount | grep -q "$LOCAL_MOUNT"
}

# 挂载远程目录
mount_remote() {
    check_sshfs

    if is_mounted; then
        print_warning "远程目录已挂载"
        print_info "挂载点: $LOCAL_MOUNT"
        return
    fi

    print_info "创建挂载点..."
    mkdir -p "$LOCAL_MOUNT"

    print_info "挂载远程目录..."
    sshfs ${SERVER}:${REMOTE_DIR} "$LOCAL_MOUNT" \
        -o reconnect,ServerAliveInterval=15,ServerAliveCountMax=3

    if [ $? -eq 0 ]; then
        print_info "挂载成功！"
        echo ""
        print_info "远程目录: ${SERVER}:${REMOTE_DIR}"
        print_info "本地挂载点: $LOCAL_MOUNT"
        echo ""
        print_info "在Cursor中打开: cursor $LOCAL_MOUNT"
        echo ""
        print_info "卸载命令: $0 umount"
    else
        print_error "挂载失败"
        exit 1
    fi
}

# 卸载远程目录
umount_remote() {
    if ! is_mounted; then
        print_warning "远程目录未挂载"
        return
    fi

    print_info "卸载远程目录..."
    fusermount -u "$LOCAL_MOUNT"

    if [ $? -eq 0 ]; then
        print_info "卸载成功"
    else
        print_error "卸载失败，尝试强制卸载..."
        sudo umount -l "$LOCAL_MOUNT"
    fi
}

# 查看状态
show_status() {
    if is_mounted; then
        print_info "状态: 已挂载 ✓"
        echo ""
        echo "挂载信息:"
        mount | grep "$LOCAL_MOUNT"
        echo ""
        print_info "本地路径: $LOCAL_MOUNT"
        print_info "远程路径: ${SERVER}:${REMOTE_DIR}"
    else
        print_warning "状态: 未挂载 ✗"
    fi
}

# 在Cursor中打开
open_in_cursor() {
    if ! is_mounted; then
        print_warning "远程目录未挂载，正在挂载..."
        mount_remote
    fi

    print_info "在Cursor中打开..."
    cursor "$LOCAL_MOUNT" &
}

# 显示帮助
show_help() {
    cat << EOF
NOFX 远程目录挂载脚本

用法: $0 [命令]

命令:
  mount       挂载远程目录
  umount      卸载远程目录
  status      查看挂载状态
  open        挂载并在Cursor中打开
  help        显示此帮助信息

示例:
  $0 mount        # 挂载远程目录
  $0 open         # 挂载并在Cursor中打开
  $0 umount       # 卸载远程目录

配置:
  服务器: ${SERVER}
  远程目录: ${REMOTE_DIR}
  本地挂载点: ${LOCAL_MOUNT}

EOF
}

# 主函数
main() {
    case "${1:-mount}" in
        mount)
            mount_remote
            ;;
        umount|unmount)
            umount_remote
            ;;
        status)
            show_status
            ;;
        open)
            open_in_cursor
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

#!/bin/bash
# NOFX 运维面板一键部署脚本

set -e

echo "================================"
echo "NOFX 运维面板部署脚本"
echo "================================"

# 检查环境
check_requirements() {
    echo "检查环境依赖..."

    if ! command -v python3 &> /dev/null; then
        echo "错误: 未安装 Python 3"
        exit 1
    fi

    if ! command -v node &> /dev/null; then
        echo "错误: 未安装 Node.js"
        exit 1
    fi

    echo "✓ 环境检查通过"
}

# 安装后端依赖
install_backend() {
    echo "安装后端依赖..."
    cd backend
    python3 -m venv venv
    source venv/bin/activate
    pip install -r requirements.txt
    cd ..
    echo "✓ 后端依赖安装完成"
}

# 安装前端依赖
install_frontend() {
    echo "安装前端依赖..."
    cd frontend
    npm install
    cd ..
    echo "✓ 前端依赖安装完成"
}

# 配置环境变量
setup_env() {
    echo "配置环境变量..."
    if [ ! -f .env ]; then
        cp .env.example .env
        echo "✓ 已创建 .env 文件，请修改配置"
    else
        echo "✓ .env 文件已存在"
    fi
}

# 主函数
main() {
    check_requirements
    install_backend
    install_frontend
    setup_env

    echo ""
    echo "================================"
    echo "部署完成！"
    echo "================================"
    echo "启动命令："
    echo "  后端: ./start.sh backend"
    echo "  前端: ./start.sh frontend"
    echo ""
}

main


# NOFX 运维管理面板

一个基于 FastAPI + Vue 3 的轻量级运维管理面板，用于远程管理 NOFX 交易系统。

## ✨ 特性

- 🚀 实时监控服务器状态和 Docker 容器
- 📊 系统资源监控（CPU、内存、磁盘）
- 📝 日志查看和管理
- 🔧 服务控制（启动、停止、重启）
- 🔐 JWT 认证保护
- ⚡ SSH 连接池优化，性能提升 83%

## 快速启动

### 方式一：一键部署（推荐）
```bash
./deploy.sh
```

### 方式二：手动安装

#### 1. 安装依赖
```bash
./start.sh install
```

### 2. 启动后端 (新终端)
```bash
./start.sh backend
```

### 3. 启动前端 (新终端)
```bash
./start.sh frontend
```

### 4. 访问
- 地址: http://localhost:8801
- 默认密码: admin123

## 配置

### 环境变量

创建 `.env` 文件并配置以下选项：

```bash
# 服务器配置
REMOTE_HOST=47.236.159.60
REMOTE_USER=root
SSH_KEY_PATH=~/.ssh/id_rsa

# API 配置
API_PORT=8800
JWT_SECRET=your-secret-key-here
ADMIN_PASSWORD=admin123

# SSH 连接池配置（新增）
SSH_POOL_MAX_CONNECTIONS=5
SSH_COMMAND_TIMEOUT=60

# CORS 配置（新增）
ALLOWED_ORIGINS=http://localhost:8801,http://localhost:8802
```

## 最近更新

查看 [OPTIMIZATION.md](./OPTIMIZATION.md) 了解最新优化详情。

### v1.3.0 最新优化
- ✅ API 响应缓存（5秒TTL）
- ✅ 全局加载状态管理
- ✅ 优化错误提示
- ✅ 改进日志格式
- ✅ API 文档支持（/docs）
- ✅ 一键部署脚本

### 历史版本
- **v1.2.0**: 请求限流、健康检查、错误处理
- **v1.1.0**: SSH 连接池、CORS 安全、配置管理

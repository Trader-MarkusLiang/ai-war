# NOFX 运维管理网页端设计文档

## 一、项目概述

将 `/home/mc/nofx-aliyun-deploy/` 目录下的运维脚本整合为一个Web管理界面，方便通过浏览器进行服务器运维操作。

### 现有脚本功能汇总

| 脚本 | 主要功能 |
|------|----------|
| `server_manage.sh` | 服务状态、启停重启、日志、备份、诊断、配置管理 |
| `local_build_deploy.sh` | 本地编译、快速部署、完整部署、代码同步 |
| `check_remote_services.sh` | 5步远程诊断检查 |
| `tunnel_simple.sh` | SSH隧道管理 |

### 服务器信息
- SSH地址: `root@47.236.159.60`
- 项目路径: `/opt/nofx`
- 前端端口: 3000, 后端端口: 8080

### 运维面板配置
- **部署位置**: 本地部署（通过SSH连接远程服务器）
- **后端API端口**: 8800
- **前端页面端口**: 8801
- **认证方式**: 简单密码认证

---

## 二、技术架构

```
浏览器 (Vue 3 + Element Plus)
        │
        │ HTTP/WebSocket
        ▼
运维后端 (Python FastAPI)
        │
        │ SSH (Paramiko)
        ▼
远程服务器 (47.236.159.60)
```

### 技术栈
- **后端**: Python FastAPI + Paramiko (SSH)
- **前端**: Vue 3 + Vite + Element Plus
- **通信**: REST API + WebSocket (实时日志)
- **认证**: JWT + 简单密码

---

## 三、目录结构

```
nofx-ops-panel/
├── backend/
│   ├── app/
│   │   ├── main.py              # FastAPI入口
│   │   ├── config.py            # 配置
│   │   ├── auth.py              # JWT认证
│   │   ├── api/
│   │   │   ├── status.py        # 状态API
│   │   │   ├── service.py       # 服务控制API
│   │   │   ├── logs.py          # 日志API
│   │   │   ├── deploy.py        # 部署API
│   │   │   ├── diagnose.py      # 诊断API
│   │   │   ├── tunnel.py        # SSH隧道API
│   │   │   ├── config_api.py    # 配置管理API
│   │   │   └── backup.py        # 备份管理API
│   │   └── services/
│   │       ├── ssh_executor.py  # SSH执行器
│   │       ├── tunnel_manager.py # 隧道管理器
│   │       └── docker_service.py
│   └── requirements.txt
├── frontend/
│   ├── src/
│   │   ├── views/
│   │   │   ├── Dashboard.vue    # 仪表盘
│   │   │   ├── Services.vue     # 服务管理
│   │   │   ├── Logs.vue         # 日志查看
│   │   │   ├── Deploy.vue       # 部署管理
│   │   │   ├── Diagnose.vue     # 系统诊断
│   │   │   ├── Tunnel.vue       # SSH隧道管理
│   │   │   ├── Config.vue       # 配置管理
│   │   │   └── Backup.vue       # 备份管理
│   │   └── components/
│   │       ├── StatusCard.vue
│   │       ├── LogViewer.vue
│   │       └── CodeEditor.vue   # 配置编辑器
│   └── package.json
└── docker-compose.yml
```

---

## 四、功能模块

### 1. 仪表盘 (Dashboard)
- 服务状态卡片（后端/前端运行状态）
- 系统资源（CPU/内存/磁盘）
- 健康检查状态
- 最近操作记录

### 2. 服务管理 (Services)
- 启动/停止/重启服务
- 重建Docker镜像（带确认）
- 容器资源使用情况

### 3. 日志查看 (Logs)
- 选择服务（后端/前端/全部）
- WebSocket实时日志流
- 日志搜索/过滤
- 清理旧日志

### 4. 部署管理 (Deploy)
- 快速部署后端（Go编译，推荐）
- 完整部署（Docker镜像）
- 代码同步
- 部署进度实时显示

### 5. 系统诊断 (Diagnose)
- Docker服务状态
- 端口监听检查
- 磁盘/内存使用
- 错误日志检查
- 配置文件检查

### 6. SSH隧道管理 (Tunnel)
- 启动/停止SSH隧道
- 隧道状态显示
- 端口映射配置（本地3333→远程3000，本地8888→远程8080）

### 7. 配置管理 (Config)
- 查看服务器配置文件
- 在线编辑配置（带语法高亮）
- 配置变更确认和重启提示

### 8. 备份管理 (Backup)
- 执行数据备份
- 备份列表查看
- 备份文件下载到本地
- 清理旧备份

---

## 五、核心API设计

```
POST /api/auth/login          # 登录
GET  /api/status              # 获取状态
POST /api/service/{action}    # 启动/停止/重启
GET  /api/logs/{service}      # 获取日志
WS   /api/logs/stream/{svc}   # 实时日志流
POST /api/deploy/quick        # 快速部署
POST /api/deploy/full         # 完整部署
GET  /api/diagnose            # 系统诊断
GET  /api/tunnel/status       # 隧道状态
POST /api/tunnel/{action}     # 启动/停止隧道
GET  /api/config              # 获取配置
PUT  /api/config              # 更新配置
GET  /api/backup/list         # 备份列表
POST /api/backup/create       # 创建备份
GET  /api/backup/download/{f} # 下载备份
```

---

## 六、实现步骤

### 步骤1: 创建项目结构
- 创建 `nofx-ops-panel/` 目录
- 初始化后端 FastAPI 项目
- 初始化前端 Vue 3 项目

### 步骤2: 实现后端核心服务
- SSH执行器 (Paramiko)
- Docker服务操作
- JWT认证中间件

### 步骤3: 实现后端API
- 状态查询API
- 服务控制API
- 日志API (含WebSocket)
- 部署API
- 诊断API

### 步骤4: 实现前端页面
- 登录页
- 仪表盘
- 服务管理页
- 日志查看页
- 部署管理页
- 系统诊断页

### 步骤5: 集成测试
- 测试所有API
- 测试WebSocket日志流
- 测试部署流程

### 步骤6: 部署配置
- 编写docker-compose.yml
- 配置环境变量
- 部署到本地或服务器

---

## 七、关键文件清单

### 后端文件
- `backend/app/main.py` - FastAPI入口
- `backend/app/config.py` - 配置管理
- `backend/app/auth.py` - JWT认证
- `backend/app/services/ssh_executor.py` - SSH执行器
- `backend/app/api/status.py` - 状态API
- `backend/app/api/service.py` - 服务控制API
- `backend/app/api/logs.py` - 日志API
- `backend/app/api/deploy.py` - 部署API
- `backend/app/api/diagnose.py` - 诊断API
- `backend/requirements.txt` - Python依赖

### 前端文件
- `frontend/src/main.js` - Vue入口
- `frontend/src/App.vue` - 根组件
- `frontend/src/router/index.js` - 路由
- `frontend/src/views/Dashboard.vue` - 仪表盘
- `frontend/src/views/Services.vue` - 服务管理
- `frontend/src/views/Logs.vue` - 日志查看
- `frontend/src/views/Deploy.vue` - 部署管理
- `frontend/src/views/Diagnose.vue` - 系统诊断
- `frontend/src/components/StatusCard.vue` - 状态卡片
- `frontend/src/components/LogViewer.vue` - 日志查看器
- `frontend/package.json` - 前端依赖

### 部署文件
- `docker-compose.yml` - Docker部署配置
- `.env` - 环境变量

---

## 八、安全设计

1. **认证**: JWT令牌，24小时过期
2. **危险操作确认**: 停止服务、重建镜像需二次确认
3. **敏感信息脱敏**: API密钥等字段显示为 `***xxxx`
4. **操作审计**: 记录所有操作日志

---

## 九、部署方式

### 本地开发
```bash
# 后端
cd backend && pip install -r requirements.txt
uvicorn app.main:app --reload --port 8800

# 前端
cd frontend && npm install && npm run dev -- --port 8801
```

### Docker部署
```bash
docker-compose up -d
# 访问 http://localhost:8801
```

---

## 十、创建时间

文档创建时间: 2024-12-29

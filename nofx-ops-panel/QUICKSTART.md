# 快速开始指南

## 5分钟快速部署

### 前置要求

- Python 3.11+
- Node.js 16+
- SSH 密钥配置

### 方式一：一键部署（推荐）

```bash
# 1. 克隆项目
git clone <your-repo-url>
cd nofx-ops-panel

# 2. 运行部署脚本
./deploy.sh

# 3. 配置环境变量
cp .env.example .env
# 编辑 .env 文件，修改服务器配置

# 4. 启动服务
./start.sh backend  # 终端1
./start.sh frontend # 终端2
```

### 方式二：Docker 部署

```bash
# 1. 配置环境变量
cp .env.example .env

# 2. 启动容器
docker-compose up -d

# 3. 查看日志
docker-compose logs -f
```

### 访问应用

- 前端地址：http://localhost:8801
- 后端 API：http://localhost:8800
- API 文档：http://localhost:8800/docs
- 默认密码：admin123


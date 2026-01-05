# NOFX 阿里云部署 - 快速开始指南

## 🚀 最简单的使用方式

### 方式1：访问前端（推荐）

使用简化版SSH隧道脚本：

```bash
cd ~/nofx-aliyun-deploy
./tunnel_simple.sh
```

保持终端窗口打开，然后在浏览器访问：
```
http://localhost:3333
```

### 方式2：编辑服务器文件 ⭐ 新增

使用远程挂载功能，直接在Cursor中编辑服务器文件：

```bash
cd ~/nofx-aliyun-deploy
./remote_mount.sh open
```

这会：
- ✅ 将服务器的 `/opt/nofx` 挂载到 `./remote-nofx/`
- ✅ 自动在Cursor中打开
- ✅ 所有修改实时同步到服务器

---

## 📋 其他管理命令

### 查看服务器状态

```bash
./server_manage.sh status
```

### 查看日志

```bash
./server_manage.sh logs
```

### 重启服务

```bash
./server_manage.sh restart
```

### 备份数据

```bash
./server_manage.sh backup
```

---

## 🔧 远程挂载详细说明

### 挂载远程目录

```bash
# 挂载并在Cursor中打开
./remote_mount.sh open

# 只挂载（不打开Cursor）
./remote_mount.sh mount

# 查看挂载状态
./remote_mount.sh status

# 卸载远程目录
./remote_mount.sh umount
```

### 挂载位置

- **远程目录**：`/opt/nofx`（服务器）
- **本地挂载点**：`~/nofx-aliyun-deploy/remote-nofx/`

### 使用场景

1. **编辑配置文件**
   ```bash
   ./remote_mount.sh open
   # 在Cursor中编辑 remote-nofx/config.json
   ./server_manage.sh restart
   ```

2. **修改代码**
   ```bash
   ./remote_mount.sh open
   # 在Cursor中编辑代码
   ./server_manage.sh rebuild  # 如果修改了Go代码
   ```

3. **查看日志**
   ```bash
   ./remote_mount.sh open
   # 在Cursor中查看 remote-nofx/decision_logs/
   ```

---

## 🎯 完整工作流程

### 日常使用

```bash
# 1. 启动SSH隧道（访问前端）
cd ~/nofx-aliyun-deploy
./tunnel_simple.sh

# 2. 在另一个终端，挂载远程目录（编辑文件）
cd ~/nofx-aliyun-deploy
./remote_mount.sh open

# 3. 工作期间
./server_manage.sh status    # 检查状态
./server_manage.sh logs      # 查看日志

# 4. 完成后
./remote_mount.sh umount     # 卸载远程目录
# Ctrl+C 停止SSH隧道
```

### 修改配置并重启

```bash
# 1. 挂载远程目录
./remote_mount.sh open

# 2. 在Cursor中编辑 remote-nofx/config.json

# 3. 重启服务
./server_manage.sh restart

# 4. 查看日志确认
./server_manage.sh logs

# 5. 卸载
./remote_mount.sh umount
```

---

## ⚠️ 常见问题

### 问题1：浏览器显示"无法访问"

**解决方案：**
1. 确保SSH隧道正在运行（终端窗口保持打开）
2. 等待5-10秒让隧道完全建立
3. 刷新浏览器页面
4. 检查是否访问了正确的地址：`http://localhost:3333`

### 问题2：远程挂载失败

**解决方案：**
```bash
# 1. 检查SSH连接
ssh root@47.236.159.60 "echo 'SSH连接正常'"

# 2. 配置SSH密钥（如果还没有）
ssh-copy-id root@47.236.159.60

# 3. 检查SSHFS是否安装
which sshfs || sudo apt-get install sshfs

# 4. 重新挂载
./remote_mount.sh umount
./remote_mount.sh mount
```

### 问题3：端口被占用

**解决方案：**
```bash
# 查找占用进程
lsof -i :3333

# 杀掉占用进程
kill <进程ID>

# 或者使用脚本自动处理
./tunnel_simple.sh
```

### 问题4：SSH连接失败

**解决方案：**
```bash
# 测试SSH连接
ssh root@47.236.159.60 "echo 'SSH连接正常'"

# 如果需要密码，配置SSH密钥
ssh-keygen -t rsa -b 4096
ssh-copy-id root@47.236.159.60
```

### 问题5：无法卸载远程目录

**解决方案：**
```bash
# 强制卸载
fusermount -u ~/nofx-aliyun-deploy/remote-nofx

# 或使用sudo
sudo umount -l ~/nofx-aliyun-deploy/remote-nofx
```

---

## 📞 快速命令速查表

| 操作 | 命令 |
|------|------|
| 启动隧道 | `./tunnel_simple.sh` |
| 挂载远程目录 | `./remote_mount.sh open` |
| 卸载远程目录 | `./remote_mount.sh umount` |
| 查看状态 | `./server_manage.sh status` |
| 查看日志 | `./server_manage.sh logs` |
| 重启服务 | `./server_manage.sh restart` |
| SSH登录 | `./server_manage.sh ssh` |

---

## 🎯 推荐工作流程

1. **启动隧道**
   ```bash
   cd ~/nofx-aliyun-deploy
   ./tunnel_simple.sh
   ```

2. **访问前端**
   - 在浏览器打开：`http://localhost:3333`
   - 注册账号并登录

3. **编辑文件**（如果需要）
   ```bash
   # 在另一个终端
   cd ~/nofx-aliyun-deploy
   ./remote_mount.sh open
   ```

4. **监控运行**
   ```bash
   ./server_manage.sh status
   ./server_manage.sh logs
   ```

5. **完成工作**
   ```bash
   ./remote_mount.sh umount
   # Ctrl+C 停止隧道
   ```

---

## 💡 提示

- **SSH隧道和远程挂载可以同时使用**
- **远程挂载需要SSH密钥认证**
- **使用完毕后记得卸载远程目录**
- **所有脚本都有 `--help` 参数**

---

**最后更新**: 2024-12-20

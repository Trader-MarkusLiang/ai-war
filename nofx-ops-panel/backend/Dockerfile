FROM python:3.11-slim

WORKDIR /app

# 安装系统依赖（SSH客户端、procps用于pgrep）
RUN apt-get update && apt-get install -y --no-install-recommends \
    openssh-client \
    procps \
    && rm -rf /var/lib/apt/lists/*

# 安装依赖
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt -i https://pypi.tuna.tsinghua.edu.cn/simple

# 复制代码
COPY . .

# 暴露端口
EXPOSE 8800

# 启动命令
CMD ["uvicorn", "app.main:app", "--host", "0.0.0.0", "--port", "8800"]

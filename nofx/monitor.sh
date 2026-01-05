#!/bin/bash

echo "=== NOFX系统监控 ==="
echo ""
echo "1. 系统资源："
free -h
echo ""
df -h / | grep -v Filesystem
echo ""

echo "2. Docker容器状态："
docker ps --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"
echo ""

echo "3. 内存使用："
docker stats --no-stream --format "table {{.Name}}\t{{.MemUsage}}\t{{.CPUPerc}}"
echo ""

echo "4. 服务健康检查："
curl -s http://localhost:8080/health | jq '.' 2>/dev/null || echo "后端未响应"
echo ""

echo "5. 最近日志（最后10行）："
docker logs --tail 10 nofx-trading 2>&1 | tail -10

#!/bin/bash
BACKUP_DIR="/opt/nofx_backups"
DATE=$(date +%Y%m%d_%H%M%S)

mkdir -p $BACKUP_DIR
tar -czf $BACKUP_DIR/nofx_backup_$DATE.tar.gz \
    /opt/nofx/config.json \
    /opt/nofx/decision_logs \
    /opt/nofx/data 2>/dev/null

# 保留最近7天的备份
find $BACKUP_DIR -name "nofx_backup_*.tar.gz" -mtime +7 -delete 2>/dev/null

echo "备份完成: $BACKUP_DIR/nofx_backup_$DATE.tar.gz"

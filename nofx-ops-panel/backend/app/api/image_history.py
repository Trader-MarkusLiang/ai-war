"""
Docker镜像升级历史追踪API
"""
from fastapi import APIRouter, Depends, HTTPException
from typing import List, Dict, Any
import json
import os
from datetime import datetime
from ..auth import get_current_user
from ..services.ssh_executor import SSHExecutor

router = APIRouter()

# 历史记录存储文件
HISTORY_FILE = "/tmp/nofx_image_history.json"


def load_history() -> List[Dict[str, Any]]:
    """加载历史记录"""
    if os.path.exists(HISTORY_FILE):
        try:
            with open(HISTORY_FILE, 'r', encoding='utf-8') as f:
                return json.load(f)
        except Exception:
            return []
    return []


def save_history(history: List[Dict[str, Any]]):
    """保存历史记录"""
    try:
        with open(HISTORY_FILE, 'w', encoding='utf-8') as f:
            json.dump(history, f, ensure_ascii=False, indent=2)
    except Exception as e:
        print(f"保存历史记录失败: {e}")


def add_history_record(record: Dict[str, Any]):
    """添加历史记录"""
    history = load_history()
    record['id'] = len(history) + 1
    record['timestamp'] = datetime.now().isoformat()
    history.insert(0, record)  # 最新的记录在前面

    # 只保留最近100条记录
    if len(history) > 100:
        history = history[:100]

    save_history(history)


@router.get("/image/history")
async def get_image_history(
    limit: int = 50,
    current_user: dict = Depends(get_current_user)
):
    """获取镜像升级历史"""
    history = load_history()
    return {
        "total": len(history),
        "records": history[:limit]
    }


@router.get("/image/current")
async def get_current_images(current_user: dict = Depends(get_current_user)):
    """获取当前镜像信息"""
    executor = SSHExecutor()

    try:
        # 获取Docker镜像列表
        result = executor.execute(
            'docker images --format "{{.Repository}}:{{.Tag}}|{{.ID}}|{{.Size}}|{{.CreatedAt}}" | grep nofx'
        )

        images = []
        for line in result.get('stdout', '').strip().split('\n'):
            if line:
                parts = line.split('|')
                if len(parts) >= 4:
                    images.append({
                        'name': parts[0],
                        'id': parts[1],
                        'size': parts[2],
                        'created': parts[3]
                    })

        return {"images": images}
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))


@router.post("/image/record")
async def record_image_update(
    data: dict,
    current_user: dict = Depends(get_current_user)
):
    """记录镜像更新"""
    record = {
        'action': data.get('action', 'update'),
        'image_name': data.get('image_name'),
        'old_id': data.get('old_id'),
        'new_id': data.get('new_id'),
        'old_size': data.get('old_size'),
        'new_size': data.get('new_size'),
        'user': current_user.get('username', 'unknown'),
        'note': data.get('note', '')
    }

    add_history_record(record)

    return {"success": True, "message": "记录已保存"}


@router.delete("/image/history")
async def clear_history(current_user: dict = Depends(get_current_user)):
    """清空历史记录"""
    save_history([])
    return {"success": True, "message": "历史记录已清空"}

"""
简单的内存缓存实现
"""
import time
from typing import Any, Optional
from functools import wraps
import hashlib
import json

class SimpleCache:
    """简单的内存缓存"""

    def __init__(self):
        self.cache = {}
        self.expire_times = {}

    def get(self, key: str) -> Optional[Any]:
        """获取缓存"""
        if key in self.cache:
            if time.time() < self.expire_times.get(key, 0):
                return self.cache[key]
            else:
                # 过期，删除
                self.cache.pop(key, None)
                self.expire_times.pop(key, None)
        return None

    def set(self, key: str, value: Any, ttl: int = 60):
        """设置缓存"""
        self.cache[key] = value
        self.expire_times[key] = time.time() + ttl

    def clear(self):
        """清空缓存"""
        self.cache.clear()
        self.expire_times.clear()

# 全局缓存实例
cache = SimpleCache()

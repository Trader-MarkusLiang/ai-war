"""
性能监控中间件
"""
import time
from fastapi import Request
from starlette.middleware.base import BaseHTTPMiddleware
import logging

logger = logging.getLogger(__name__)

class PerformanceMiddleware(BaseHTTPMiddleware):
    """性能监控中间件"""

    def __init__(self, app):
        super().__init__(app)
        self.request_count = 0
        self.total_time = 0.0

    async def dispatch(self, request: Request, call_next):
        start_time = time.time()

        response = await call_next(request)

        process_time = time.time() - start_time
        self.request_count += 1
        self.total_time += process_time

        # 添加响应头
        response.headers["X-Process-Time"] = str(process_time)

        # 记录慢请求
        if process_time > 1.0:
            logger.warning(
                f"慢请求: {request.method} {request.url.path} "
                f"耗时 {process_time:.2f}s"
            )

        return response

    def get_stats(self):
        """获取统计信息"""
        avg_time = self.total_time / self.request_count if self.request_count > 0 else 0
        return {
            "total_requests": self.request_count,
            "average_time": round(avg_time, 3)
        }

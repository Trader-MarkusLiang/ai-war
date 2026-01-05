import paramiko
import asyncio
import logging
from typing import Tuple, Optional
from contextlib import contextmanager
from .. import config

logger = logging.getLogger(__name__)

class SSHConnectionPool:
    """SSH连接池"""

    def __init__(self, max_connections: int = 5):
        self.max_connections = max_connections
        self.pool = []
        self.in_use = set()

    def get_connection(self) -> paramiko.SSHClient:
        """从池中获取连接"""
        # 尝试复用现有连接
        for client in self.pool:
            if client not in self.in_use:
                try:
                    # 测试连接是否有效
                    transport = client.get_transport()
                    if transport and transport.is_active():
                        self.in_use.add(client)
                        return client
                    else:
                        self.pool.remove(client)
                except:
                    self.pool.remove(client)

        # 创建新连接
        if len(self.pool) < self.max_connections:
            client = self._create_connection()
            self.pool.append(client)
            self.in_use.add(client)
            return client

        raise Exception("连接池已满")

    def release_connection(self, client: paramiko.SSHClient):
        """释放连接回池"""
        if client in self.in_use:
            self.in_use.remove(client)

    def _create_connection(self) -> paramiko.SSHClient:
        """创建新的SSH连接"""
        client = paramiko.SSHClient()
        client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
        client.connect(
            config.REMOTE_HOST,
            username=config.REMOTE_USER,
            key_filename=config.SSH_KEY_PATH,
            timeout=10
        )
        return client

    def close_all(self):
        """关闭所有连接"""
        for client in self.pool:
            try:
                client.close()
            except:
                pass
        self.pool.clear()
        self.in_use.clear()

class SSHExecutor:
    """SSH命令执行器（优化版）"""

    def __init__(self):
        self.host = config.REMOTE_HOST
        self.user = config.REMOTE_USER
        self.key_path = config.SSH_KEY_PATH
        self.pool = SSHConnectionPool()

    @contextmanager
    def _get_client(self):
        """获取SSH客户端的上下文管理器"""
        client = None
        try:
            client = self.pool.get_connection()
            yield client
        except Exception as e:
            logger.error(f"SSH连接错误: {e}")
            raise
        finally:
            if client:
                self.pool.release_connection(client)

    def execute(self, command: str, timeout: int = 60) -> Tuple[str, str, int]:
        """执行命令，返回 (stdout, stderr, exit_code)"""
        try:
            with self._get_client() as client:
                logger.info(f"执行SSH命令: {command[:100]}...")
                stdin, stdout, stderr = client.exec_command(command, timeout=timeout)
                exit_code = stdout.channel.recv_exit_status()

                stdout_data = stdout.read().decode('utf-8', errors='ignore')
                stderr_data = stderr.read().decode('utf-8', errors='ignore')

                logger.info(f"命令执行完成，退出码: {exit_code}")
                return (stdout_data, stderr_data, exit_code)
        except Exception as e:
            logger.error(f"命令执行失败: {e}")
            return ("", str(e), -1)

    async def execute_async(self, command: str, timeout: int = 60) -> Tuple[str, str, int]:
        """异步执行命令"""
        loop = asyncio.get_event_loop()
        return await loop.run_in_executor(None, self.execute, command, timeout)

    def close(self):
        """关闭所有连接"""
        self.pool.close_all()

# 全局SSH执行器实例
ssh_executor = SSHExecutor()

import paramiko
import asyncio
from typing import Tuple, AsyncGenerator
from .. import config

class SSHExecutor:
    """SSH命令执行器"""

    def __init__(self):
        self.host = config.REMOTE_HOST
        self.user = config.REMOTE_USER
        self.key_path = config.SSH_KEY_PATH
        self.client = None

    def connect(self):
        """建立SSH连接"""
        if self.client is None:
            self.client = paramiko.SSHClient()
            self.client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
            self.client.connect(
                self.host,
                username=self.user,
                key_filename=self.key_path,
                timeout=10
            )
        return self.client

    def execute(self, command: str) -> Tuple[str, str, int]:
        """执行命令，返回 (stdout, stderr, exit_code)"""
        client = paramiko.SSHClient()
        client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
        try:
            client.connect(
                self.host,
                username=self.user,
                key_filename=self.key_path,
                timeout=10
            )
            stdin, stdout, stderr = client.exec_command(command, timeout=60)
            exit_code = stdout.channel.recv_exit_status()
            return (
                stdout.read().decode('utf-8'),
                stderr.read().decode('utf-8'),
                exit_code
            )
        finally:
            client.close()

    async def execute_async(self, command: str) -> Tuple[str, str, int]:
        """异步执行命令"""
        loop = asyncio.get_event_loop()
        return await loop.run_in_executor(None, self.execute, command)

    def close(self):
        """关闭连接"""
        if self.client:
            self.client.close()
            self.client = None

# 全局SSH执行器实例
ssh_executor = SSHExecutor()

"""
ADDE – Agent-Driven Docker Executor.

Python client for the ADDE Go CLI. Use from agent code to:
- pull_image: pull an image from the registry (call before create_runtime_env if needed)
- create_runtime_env: provision a container with workspace mount and limits
- execute_code_block: write code into the container and run it (returns structured log)
- get_container_logs: fetch the last execution's stdout/stderr/exit_code/execution_time
- cleanup_env: stop and remove the container
"""

from .client import (
    create_runtime_env,
    execute_code_block,
    get_container_logs,
    cleanup_env,
    pull_image,
)

__all__ = [
    "create_runtime_env",
    "execute_code_block",
    "get_container_logs",
    "cleanup_env",
    "pull_image",
]

"""
ADDE – Agent-Driven Docker Executor.

Python client for the ADDE Go CLI. Use from agent code to:
- pull_image: pull an image from the registry (call before create_runtime_env if needed)
- create_runtime_env: provision a container with workspace mount and limits
- execute_code_block: write code into the container and run it (returns structured log)
- get_container_logs: fetch the last execution's stdout/stderr/exit_code/execution_time
- cleanup_env: stop and remove the container
- prepare_build_context: stage files into a temp dir for Docker build (optional Dockerfile)
- build_image_from_context: run docker build from context; returns image_id for create_runtime_env
- build_image_from_path: build from an existing directory (e.g. cloned repo) that has a Dockerfile
- list_agent_images: list custom images (agent-env:...)
- prune_build_cache: clean up build cache
"""

from .client import (
    build_image_from_context,
    build_image_from_path,
    cleanup_env,
    create_runtime_env,
    execute_code_block,
    get_container_logs,
    list_agent_images,
    prepare_build_context,
    prune_build_cache,
    pull_image,
)

__all__ = [
    "build_image_from_context",
    "build_image_from_path",
    "cleanup_env",
    "create_runtime_env",
    "execute_code_block",
    "get_container_logs",
    "list_agent_images",
    "prepare_build_context",
    "prune_build_cache",
    "pull_image",
]

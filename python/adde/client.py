"""
ADDE Python client – invokes the Go adde CLI via subprocess and returns parsed JSON.
"""

import json
import os
import subprocess
from pathlib import Path
from typing import Any, Optional

# Default path to adde binary; override with ADDE_BIN or pass bin_path=...
_ADDE_BIN = os.environ.get("ADDE_BIN", "adde")


def _find_adde() -> str:
    """Resolve adde binary: env ADDE_BIN, or 'adde' in PATH, or go/adde.exe in repo."""
    if os.environ.get("ADDE_BIN"):
        return os.environ["ADDE_BIN"]
    root = Path(__file__).resolve().parents[2]
    for name in ("adde.exe", "adde"):
        cand = root / "go" / name
        if cand.is_file():
            return str(cand)
    return "adde"


def pull_image(
    image: str,
    bin_path: Optional[str] = None,
) -> dict[str, Any]:
    """
    Pulls an image from the default registry. Call before create_runtime_env
    if the image is not already present.

    Returns dict with keys: ok, or error.
    """
    params = {"image": image}
    return _call("pull_image", params, bin_path=bin_path)


def _call(
    tool: str,
    params: dict,
    bin_path: Optional[str] = None,
    timeout: int = 120,
) -> dict:
    bin_ = bin_path or _find_adde()
    payload = json.dumps(params)
    out = subprocess.run(
        [bin_, tool, payload],
        capture_output=True,
        text=True,
        timeout=timeout,
    )
    if out.returncode != 0:
        err = out.stderr.strip() or out.stdout.strip() or f"adde {tool} failed"
        raise RuntimeError(err)
    return json.loads(out.stdout)


def create_runtime_env(
    image: str,
    dependencies: Optional[list[str]] = None,
    env_vars: Optional[dict[str, str]] = None,
    network: bool = False,
    bin_path: Optional[str] = None,
) -> dict[str, Any]:
    """
    Provisions a container with workspace at /workspace, 512MB / 0.5 CPU, network=none by default.

    Returns dict with keys: container_id, workspace, or error.
    """
    params = {
        "image": image,
        "dependencies": dependencies or [],
        "env_vars": env_vars or {},
        "network": network,
    }
    return _call("create_runtime_env", params, bin_path=bin_path)


def execute_code_block(
    container_id: str,
    filename: str,
    code_content: str,
    timeout_sec: int = 30,
    bin_path: Optional[str] = None,
) -> dict[str, Any]:
    """
    Writes code into the container and runs it. Uses put_archive (no shell on code_content).

    Returns dict with keys: log (exit_code, stdout, stderr, execution_time), or error.
    """
    params = {
        "container_id": container_id,
        "filename": filename,
        "code_content": code_content,
        "timeout_sec": timeout_sec,
    }
    return _call("execute_code_block", params, bin_path=bin_path)


def get_container_logs(
    container_id: str,
    tail_lines: int = 0,
    bin_path: Optional[str] = None,
) -> dict[str, Any]:
    """
    Returns the last execution's structured log for the refiner agent.

    Keys: log (exit_code, stdout, stderr, execution_time), or error.
    tail_lines: 0 = all; otherwise last N lines of stdout/stderr.
    """
    params = {"container_id": container_id, "tail_lines": tail_lines}
    return _call("get_container_logs", params, bin_path=bin_path)


def cleanup_env(
    container_id: str,
    bin_path: Optional[str] = None,
) -> dict[str, Any]:
    """Stops and removes the container."""
    params = {"container_id": container_id}
    return _call("cleanup_env", params, bin_path=bin_path)


# ---- Image Builder & Factory ----


def prepare_build_context(
    files: dict[str, str],
    context_id: Optional[str] = None,
    bin_path: Optional[str] = None,
) -> dict[str, Any]:
    """
    Stages files (source code, configs, requirements) into a temporary directory for Docker build.
    Auto-generates .dockerignore if missing; injects a standard Dockerfile if requirements.txt
    or package.json exists but no Dockerfile is provided.

    Returns dict with context_id (absolute path to build context dir), or error.
    """
    params: dict[str, Any] = {"files": files}
    if context_id is not None:
        params["context_id"] = context_id
    return _call("prepare_build_context", params, bin_path=bin_path)


def build_image_from_context(
    context_id: str,
    tag: str,
    build_args: Optional[dict[str, str]] = None,
    bin_path: Optional[str] = None,
) -> dict[str, Any]:
    """
    Runs docker build from the context directory (path from prepare_build_context).
    Tag convention: agent-env:{task_id}-{timestamp}. Returns handshake:
    { status, image_id, tag, size_mb, build_log_summary } or error/failed_layer.
    """
    params: dict[str, Any] = {"context_id": context_id, "tag": tag}
    if build_args:
        params["build_args"] = build_args
    return _call(
        "build_image_from_context", params, bin_path=bin_path, timeout=600
    )


def build_image_from_path(
    path: str,
    tag: str,
    build_args: Optional[dict[str, str]] = None,
    bin_path: Optional[str] = None,
) -> dict[str, Any]:
    """
    Builds a Docker image from an existing directory on disk (e.g. a cloned repo).
    The directory must contain a Dockerfile. Same security checks and handshake as
    build_image_from_context. Use this when you already have a project directory
    with a proper Dockerfile; use prepare_build_context + build_image_from_context
    when you have in-memory files only.

    path: absolute or relative path to the project directory
    tag: e.g. agent-env:myapp-1 (agent-env: prefix is added if missing)

    Returns: { status, image_id, tag, size_mb, build_log_summary } or error.
    """
    params: dict[str, Any] = {"path": path, "tag": tag}
    if build_args:
        params["build_args"] = build_args
    return _call(
        "build_image_from_path", params, bin_path=bin_path, timeout=600
    )


def list_agent_images(
    filter_tag: Optional[str] = None,
    bin_path: Optional[str] = None,
) -> dict[str, Any]:
    """
    Returns a list of custom images created by the agent (tagged agent-env:...).
    filter_tag: optional prefix filter (e.g. 'agent-env' or 'agent-env:task-123').
    """
    params: dict[str, Any] = {}
    if filter_tag is not None:
        params["filter_tag"] = filter_tag
    return _call("list_agent_images", params, bin_path=bin_path)


def prune_build_cache(
    older_than_hrs: int = 0,
    bin_path: Optional[str] = None,
) -> dict[str, Any]:
    """
    Cleans up intermediate build stages and unused build cache.
    older_than_hrs: 0 = prune all unused; >0 = only prune cache older than that many hours.
    Returns space_reclaimed_mb or error.
    """
    params: dict[str, Any] = {}
    if older_than_hrs > 0:
        params["older_than_hrs"] = older_than_hrs
    return _call("prune_build_cache", params, bin_path=bin_path)

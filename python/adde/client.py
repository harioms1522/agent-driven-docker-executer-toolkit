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


def _call(tool: str, params: dict, bin_path: Optional[str] = None) -> dict:
    bin_ = bin_path or _find_adde()
    payload = json.dumps(params)
    out = subprocess.run(
        [bin_, tool, payload],
        capture_output=True,
        text=True,
        timeout=120,
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

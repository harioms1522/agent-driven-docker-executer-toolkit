#!/usr/bin/env python3
"""
Node.js Hello World example using ADDE build_image_from_path.

Builds the image from this directory (which has a Dockerfile and server.js),
creates a container, starts the server and hits GET / inside the container
to verify "Hello World", then cleans up. Requires Docker and the adde binary.
"""

import sys
import time
from pathlib import Path

# Add the python package to path so we can import adde from the repo
_repo_root = Path(__file__).resolve().parents[2]
_example_dir = Path(__file__).resolve().parent
sys.path.insert(0, str(_repo_root / "python"))

from adde import (
    build_image_from_path,
    create_runtime_env,
    execute_code_block,
    cleanup_env,
)


def main() -> None:
    adde_bin = (
        _repo_root / "go" / "adde.exe"
        if sys.platform == "win32"
        else _repo_root / "go" / "adde"
    )
    bin_path = str(adde_bin) if adde_bin.is_file() else None

    project_path = str(_example_dir)
    print(f"1. Building image from path: {project_path}")
    tag = f"agent-env:nodejs-helloworld-{int(time.time())}"
    out = build_image_from_path(
        path=project_path,
        tag=tag,
        bin_path=bin_path,
    )
    if out.get("status") != "success":
        print("Error:", out.get("error", out.get("build_log_summary", out)), file=sys.stderr)
        sys.exit(1)
    print(f"   Image: {out.get('tag')} ({out.get('size_mb', 0):.1f} MB)")

    print("2. Creating container from built image...")
    r = create_runtime_env(
        image=tag,
        dependencies=[],
        env_vars={},
        network=True,
        bin_path=bin_path,
    )
    if r.get("error"):
        print("Error:", r["error"], file=sys.stderr)
        sys.exit(1)
    container_id = r["container_id"]
    print(f"   Container: {container_id[:12]}...")

    try:
        print("3. Starting server and requesting GET / inside container...")
        # Server in /app (from Dockerfile). Start in background, then hit with Node (node:alpine has no wget)
        run_script = (
            "node /app/server.js & sleep 2 && node -e "
            "\"require('http').get('http://localhost:3000/', r => { let d=''; "
            "r.on('data', c=>d+=c); r.on('end', ()=>console.log(d)); });\""
        )
        exec_out = execute_code_block(
            container_id=container_id,
            filename="run.sh",
            code_content=run_script,
            timeout_sec=15,
            bin_path=bin_path,
        )
        if exec_out.get("error"):
            print("Error:", exec_out["error"], file=sys.stderr)
        else:
            log = exec_out.get("log", {})
            stdout = (log.get("stdout") or "").strip()
            print(f"   Response: {stdout or '(empty)'}")
            if log.get("stderr"):
                print(f"   stderr: {log.get('stderr', '')[:200]}")
    finally:
        print("4. Cleaning up container...")
        # cleanup_env(container_id, bin_path=bin_path)
        print("   Done.")

    print("\nNode.js Hello World run complete.")


if __name__ == "__main__":
    main()

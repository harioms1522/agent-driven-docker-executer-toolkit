#!/usr/bin/env python3
"""
Node.js Hello World example using ADDE build_image_from_path.

Builds the image from this directory (Dockerfile + server.js), creates a container
with port forwarding (container 3000 -> host 8080), starts the server in the
container, and keeps it running so you can hit http://127.0.0.1:8080/ from your
browser. Press Enter to stop the container and exit.
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
    cleanup_env,
)

# Container listens on 3000; we forward it to this host port
CONTAINER_PORT = "3000"
HOST_PORT = "8080"


def main() -> None:
    adde_bin = (
        _repo_root / "go" / "adde.exe"
        if sys.platform == "win32"
        else _repo_root / "go" / "adde"
    )
    bin_path = str(adde_bin) if adde_bin.is_file() else None

    project_path = str(_example_dir)
    print("1. Building image from path:", project_path)
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

    print("2. Creating container with port forwarding (container 3000 -> host 8080)...")
    r = create_runtime_env(
        image=tag,
        dependencies=[],
        env_vars={},
        network=True,
        port_bindings={CONTAINER_PORT: HOST_PORT},
        use_image_cmd=True,  # run image CMD (node server.js) so the server starts
        bin_path=bin_path,
    )
    if r.get("error"):
        print("Error:", r["error"], file=sys.stderr)
        sys.exit(1)
    container_id = r["container_id"]
    print(f"   Container: {container_id[:12]}...")

    try:
        print("3. Server started (container runs image CMD: node server.js).")
        print()
        print(f"   Server is running at  http://127.0.0.1:{HOST_PORT}/")
        print("   Open in browser or run: curl http://127.0.0.1:8080/")
        print()
        input("   Press Enter to stop the container and exit... ")
    finally:
        print("4. Stopping and removing container...")
        cleanup_env(container_id, bin_path=bin_path)
        print("   Done.")

    print("\nNode.js Hello World run complete.")


if __name__ == "__main__":
    main()

#!/usr/bin/env python3
"""
Hello World example using the ADDE Python client.

Builds a minimal Python app image, creates a container from it, runs the app,
prints logs, then cleans up. Requires Docker and the adde binary (build from go/).
"""

import sys
import time
from pathlib import Path

# Add the python package to path so we can import adde from the repo
_repo_root = Path(__file__).resolve().parents[2]
sys.path.insert(0, str(_repo_root / "python"))

from adde import (
    prepare_build_context,
    build_image_from_context,
    create_runtime_env,
    execute_code_block,
    get_container_logs,
    cleanup_env,
)


def main() -> None:
    # Optional: point at the adde binary if not on PATH
    adde_bin = _repo_root / "go" / "adde.exe" if sys.platform == "win32" else _repo_root / "go" / "adde"
    bin_path = str(adde_bin) if adde_bin.is_file() else None

    print("1. Preparing build context (hello world app, no Dockerfile -> one will be generated)...")
    ctx = prepare_build_context(
        files={
            "requirements.txt": "",  # empty; template still adds pip install step
            "main.py": 'print("Hello, World!")\n',
        },
        bin_path=bin_path,
    )
    if ctx.get("error"):
        print("Error:", ctx["error"], file=sys.stderr)
        sys.exit(1)
    context_path = ctx["context_id"]
    print(f"   Context: {context_path}")

    print("2. Building image (tag: agent-env:helloworld-<ts>)...")
    tag = f"agent-env:helloworld-{int(time.time())}"
    out = build_image_from_context(
        context_id=context_path,
        tag=tag,
        bin_path=bin_path,
    )
    if out.get("status") != "success":
        print("Error:", out.get("error", out.get("build_log_summary", out)), file=sys.stderr)
        sys.exit(1)
    print(f"   Image: {out.get('image_id', tag)} ({out.get('size_mb', 0):.1f} MB)")

    print("3. Creating container from built image...")
    r = create_runtime_env(
        image=tag,
        dependencies=[],
        env_vars={},
        network=False,
        bin_path=bin_path,
    )
    if r.get("error"):
        print("Error:", r["error"], file=sys.stderr)
        sys.exit(1)
    container_id = r["container_id"]
    print(f"   Container: {container_id[:12]}...")

    try:
        print("4. Running the app (exec /app/main.py inside container)...")
        run_script = "exec(open('/app/main.py').read())"
        exec_out = execute_code_block(
            container_id=container_id,
            filename="run.py",
            code_content=run_script,
            timeout_sec=10,
            bin_path=bin_path,
        )
        if exec_out.get("error"):
            print("Error:", exec_out["error"], file=sys.stderr)
        else:
            log = exec_out.get("log", {})
            print("   stdout:", repr(log.get("stdout", "").strip()))
            print("   stderr:", repr(log.get("stderr", "").strip()) if log.get("stderr") else None)
            print("   exit_code:", log.get("exit_code"), "| time:", log.get("execution_time"))

        print("5. Fetching container logs (last run)...")
        logs = get_container_logs(container_id, tail_lines=20, bin_path=bin_path)
        if logs.get("log"):
            print("   ", logs["log"].get("stdout", "").strip() or "(empty)")
    finally:
        print("6. Cleaning up container...")
        cleanup_env(container_id, bin_path=bin_path)
        print("   Done.")

    print("\nHello World run complete.")


if __name__ == "__main__":
    main()

# Hello World — ADDE Python client example

This example uses the **ADDE Python wrapper** to:

1. **Prepare** a build context (a minimal Python app: `main.py` + `requirements.txt`)
2. **Build** a Docker image from that context (no Dockerfile provided → ADDE injects one)
3. **Create** a container from the built image
4. **Run** the app inside the container and print stdout
5. **Clean up** the container

## Prerequisites

- **Docker** running and reachable
- **ADDE binary** built from the repo:
  ```bash
  cd go
  go build -o adde.exe ./cmd/adde   # Windows
  # or: go build -o adde ./cmd/adde   # Linux/macOS
  ```
- **Python 3.9+** with the `adde` package on the path (e.g. install from repo: `pip install -e python/`)

## Run the example

From the repo root:

```bash
# Install the adde Python package if you haven't
pip install -e python/

# Run the example (script finds go/adde.exe or go/adde automatically)
python examples/helloworld/run_helloworld.py
```

Or from this directory:

```bash
cd examples/helloworld
python run_helloworld.py
```

If the adde binary is not next to `go/adde.exe` or `go/adde`, set:

```bash
export ADDE_BIN=/path/to/adde    # Linux/macOS
set ADDE_BIN=C:\path\to\adde.exe # Windows
```

## Expected output

```
1. Preparing build context (hello world app, no Dockerfile → one will be generated)...
   Context: /tmp/adde-build-...
2. Building image (tag: agent-env:helloworld-<ts>)...
   Image: sha256:... (XXX.X MB)
3. Creating container from built image...
   Container: abc123def456...
4. Running the app (exec /app/main.py inside container)...
   stdout: 'Hello, World!'
   exit_code: 0 | time: 0.XXs
5. Fetching container logs (last run)...
    Hello, World!
6. Cleaning up container...
   Done.

Hello World run complete.
```

## What the script does

| Step | ADDE call | Purpose |
|------|-----------|---------|
| 1 | `prepare_build_context(files={...})` | Stage `main.py` and `requirements.txt` in a temp dir; ADDE adds `.dockerignore` and a Python Dockerfile |
| 2 | `build_image_from_context(context_id, tag)` | Run `docker build`; tag follows `agent-env:...` |
| 3 | `create_runtime_env(image=tag)` | Start a container from the new image (workspace at `/workspace`) |
| 4 | `execute_code_block(..., "run.py", "exec(open('/app/main.py').read())")` | Run the app (which lives in `/app` in the image) and capture stdout |
| 5 | `get_container_logs(container_id)` | Fetch last run’s log |
| 6 | `cleanup_env(container_id)` | Stop and remove the container |

You can change `main.py` content in the `files` dict to run a different one-liner or add more files to the build context.

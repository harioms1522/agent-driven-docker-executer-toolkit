# Agent-Driven Docker Executor (ADDE)

Go toolset + Python client for running agent-generated code in isolated Docker containers, with structured feedback for refiner agents.

## Spec alignment

| Requirement | Implementation |
|-------------|-----------------|
| **pull_image** | `image`; pulls from default registry so `create_runtime_env` can use it |
| **create_runtime_env** | `image`, `dependencies[]`, `env_vars{}`; workspace at `/workspace`; 512MB / 0.5 CPU; `--network none` unless `network: true`; optional `port_bindings` (e.g. `{"3000": "8080"}`); optional `use_image_cmd: true` to run the image CMD (e.g. server) instead of `sleep 86400` |
| **execute_code_block** | `container_id`, `filename`, `code_content`; file via **put_archive** (no shell on code); hard **timeout** (default 30s) |
| **get_container_logs** | `container_id`, `tail_lines`; returns `{ exit_code, stdout, stderr, execution_time }` (§3.B) |
| **cleanup_env** | `container_id`; stop + remove |
| **prepare_build_context** | `files{name: content}`, optional `context_id`; stages files, auto `.dockerignore`, injects Dockerfile if requirements.txt/package.json present |
| **build_image_from_context** | `context_id`, `tag`, optional `build_args{}`; runs `docker build`; tag convention `agent-env:{task_id}-{timestamp}`; security check on Dockerfile |
| **build_image_from_path** | `path`, `tag`, optional `build_args{}`; build from an **existing directory** (e.g. cloned repo) that contains a Dockerfile; same security and handshake |
| **list_agent_images** | optional `filter_tag`; returns custom images (agent-env:...) for reuse |
| **prune_build_cache** | optional `older_than_hrs`; cleans build cache |
| Security | Network disabled by default; memory/CPU capped; code injected via Docker API, not shell; Dockerfile forbidden patterns (e.g. docker.sock mount) |
| Observability | Logs captured via exec attach + stdcopy; persisted for `get_container_logs`; build returns `build_log_summary` and `failed_layer` on error |

## Layout

- **`go/`** – Go module: `pkg/executor` library + `cmd/adde` CLI.
- **`python/`** – Python package `adde` that calls the `adde` binary.

**LangGraph:** see [LANGGRAPH.md](LANGGRAPH.md) for using this toolkit in a LangGraph (Python) project. Repo: [github.com/harioms1522/agent-driven-docker-executer-toolkit](https://github.com/harioms1522/agent-driven-docker-executer-toolkit) (`main` branch).

## Build and run

### 1. Build the Go CLI

From the repo root:

```bash
cd go
go mod tidy
go build -o adde.exe ./cmd/adde   # Windows
# or
go build -o adde ./cmd/adde       # Linux/macOS
```

Ensure **Docker** is running and the daemon is reachable (e.g. `DOCKER_HOST` if remote). Use **pull_image** (or `docker pull`) before `create_runtime_env` if the image is not already present.

### 2. Install the Python client

```bash
cd python
pip install -e .
```

Or add the path to `adde` (or `adde.exe`) to `PATH`, or set `ADDE_BIN=/path/to/adde`.

### 3. Use from Python (e.g. in an agent)

**Option A: Pre-built image (existing flow)**

```python
from adde import pull_image, create_runtime_env, execute_code_block, get_container_logs, cleanup_env

# 0) Pull image if not already present
pull_image("python:3.11-slim")

# 1) Create env (network=False → --network none; set network=True for pip/npm)
r = create_runtime_env(
    image="python:3.11-slim",
    dependencies=["requests"],
    env_vars={"PYTHONUNBUFFERED": "1"},
)
if "error" in r:
    raise RuntimeError(r["error"])
cid = r["container_id"]

# 2) Run code
out = execute_code_block(cid, "script.py", "print(1+1)\nimport requests\nprint(requests.__version__)")
if "error" in out:
    raise RuntimeError(out["error"])
log = out["log"]
print("exit_code", log["exit_code"], "stdout", log["stdout"], "stderr", log["stderr"], "time", log["execution_time"])

# 3) Optionally fetch logs again (e.g. last run)
logs = get_container_logs(cid, tail_lines=50)

# 4) Cleanup
cleanup_env(cid)
```

**Option B: Image Factory (build custom image, then run)**

```python
from adde import (
    prepare_build_context,
    build_image_from_context,
    create_runtime_env,
    execute_code_block,
    get_container_logs,
    cleanup_env,
    list_agent_images,
)

# 1) Stage codebase (no Dockerfile → tool injects Python template)
ctx = prepare_build_context({
    "requirements.txt": "requests\n",
    "main.py": "import requests\nprint(requests.get('https://httpbin.org/get').status_code)\n",
})
context_path = ctx["context_id"]

# 2) Build image (tag: agent-env:task-123-1706457600)
import time
tag = f"agent-env:task-{int(time.time())}"
out = build_image_from_context(context_path, tag)
if out.get("status") != "success":
    raise RuntimeError(out.get("error", out))
image_id = out["image_id"]  # or use tag for create_runtime_env

# 3) Create env from built image (use tag; create_runtime_env accepts image name or ID)
r = create_runtime_env(image=tag, dependencies=[], env_vars={}, network=True)
cid = r["container_id"]

# 4) Run code, get logs, cleanup (same as Option A)
# execute_code_block(cid, ...); get_container_logs(cid); cleanup_env(cid)

# List / prune agent images
list_agent_images(filter_tag="agent-env")
# prune_build_cache(older_than_hrs=24)
```

## CLI usage

JSON can be passed as the second argument or via **stdin** (omit the second arg and pipe):

```bash
adde pull_image '{"image":"busybox"}'
adde create_runtime_env '{"image":"python:3.11-slim","dependencies":[],"env_vars":{},"network":false}'
adde execute_code_block '{"container_id":"<id>","filename":"main.py","code_content":"print(1)"}'
adde get_container_logs '{"container_id":"<id>","tail_lines":0}'
adde cleanup_env '{"container_id":"<id>"}'
adde prepare_build_context '{"files":{"requirements.txt":"requests","main.py":"print(1)"}}'
adde build_image_from_context '{"context_id":"/path/from/prepare","tag":"agent-env:task-1"}'
adde build_image_from_path '{"path":"/path/to/cloned/repo","tag":"agent-env:myapp-1"}'
adde list_agent_images '{"filter_tag":"agent-env"}'
adde prune_build_cache '{"older_than_hrs":24}'
```

**PowerShell on Windows:** passing JSON as an argument often breaks quoting. Use **stdin** instead:

```powershell
'{"image":"busybox"}' | .\adde.exe pull_image
'{"image":"busybox","dependencies":[],"env_vars":{},"network":false}' | .\adde.exe create_runtime_env
'{"container_id":"<id>","filename":"t.sh","code_content":"echo 42","timeout_sec":15}' | .\adde.exe execute_code_block
'{"container_id":"<id>","tail_lines":10}' | .\adde.exe get_container_logs
'{"container_id":"<id>"}' | .\adde.exe cleanup_env
```

## Flow (per spec §5)

1. Agent suggests code.
2. Manager calls `create_runtime_env`.
3. Manager calls `execute_code_block`.
4. Manager calls `get_container_logs` (or uses the `log` returned by `execute_code_block`) and sends it to the agent.
5. Agent sees errors (e.g. `ModuleNotFoundError`), adjusts (e.g. adds a `pip install` or sets `dependencies`/`network`), and repeats.

## Testing the CLI on Windows

Tests run against the built `adde.exe` binary.

### Go tests (Windows only)

From `go/`:

```powershell
go build -o adde.exe .\cmd\adde
go test -v .\cmd\adde -run Exe
```

Set `ADDE_EXE` to the full path of `adde.exe` to use a specific binary. Tests that need Docker (create, e2e) are skipped if the daemon is unreachable.

### PowerShell script

From `go/`:

```powershell
go build -o adde.exe .\cmd\adde
.\test_adde_cli.ps1
```

Use `-SkipE2E` to skip the Docker-backed end-to-end case. Set `$env:ADDE_EXE` to point at the exe if it is not in the current directory.

### Testing with BusyBox

BusyBox is a small image with `sh` but no Python. Use it to test the CLI without pulling `python:3.11-slim`.

**PowerShell script (E2E with BusyBox):**

```powershell
cd go
.\test_adde_cli.ps1 -BusyBox
```

**Go test (BusyBox E2E):**

```powershell
go test -v .\cmd\adde -run ExeE2EBusyBox
```

**Manual CLI (BusyBox):** run a shell script that prints `42`. On PowerShell use **stdin** (pipe) so JSON isn’t mangled:

```powershell
# 0) Pull image (if not already present)
'{"image":"busybox"}' | .\adde.exe pull_image

# 1) Create env
'{"image":"busybox","dependencies":[],"env_vars":{},"network":false}' | .\adde.exe create_runtime_env
# Use the container_id from the output in the next steps.

# 2) Run a .sh script (replace <id> with the container_id from step 1)
'{"container_id":"<id>","filename":"t.sh","code_content":"echo 42","timeout_sec":15}' | .\adde.exe execute_code_block

# 3) Get logs (optional)
'{"container_id":"<id>","tail_lines":10}' | .\adde.exe get_container_logs

# 4) Cleanup
'{"container_id":"<id>"}' | .\adde.exe cleanup_env
```

For BusyBox, use a **`.sh`** file and shell commands (e.g. `echo 42`); the executor runs it with `sh <path>`.

## Testing the Python client

From `python/`:

```bash
pip install -e ".[dev]"
python -m pytest tests/ -v
```

If `pytest` is not on your PATH, use `python -m pytest` instead of `pytest`.

- **Unit tests** mock subprocess and check that each client function passes the right tool and JSON to the binary. They need no binary or Docker.
- **Integration tests** call the real `adde` binary when it is found (in `../go/adde.exe` or `../go/adde`, or via `ADDE_BIN`). They are skipped if the binary is missing or Docker is down. Build the Go CLI first to run them.

## Security

- Containers use **network none** by default; set `network: true` when the agent explicitly needs access (e.g. `pip install`).
- **Memory** 512MB, **CPU** 0.5 by default.
- Code is written via the Docker **CopyToContainer** (put_archive) API, not shell, to avoid injection from `code_content`.

## License

MIT.

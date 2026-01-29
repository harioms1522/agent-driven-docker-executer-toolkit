# Node.js Hello World — ADDE build_image_from_path example

A minimal Node.js HTTP server that responds with **Hello World** on the root path (`/`). This example uses **build_image_from_path** (project directory with Dockerfile) and **port forwarding**: the container’s port 3000 is bound to **127.0.0.1:8080** on the host so you can open http://127.0.0.1:8080/ in your browser.

## Project layout

- **server.js** — HTTP server on port 3000; `GET /` returns `Hello World`
- **package.json** — Node app manifest (no extra dependencies)
- **Dockerfile** — Node 20 Alpine; installs deps, runs `node server.js`
- **.dockerignore** — Keeps build context small

## Prerequisites

- Docker running
- ADDE binary built: `cd go && go build -o adde.exe ./cmd/adde` (or `adde` on Linux/macOS)
- Python 3.9+ with `adde` installed: `pip install -e python/`

## Build and run with ADDE

### Option 1: Python script (recommended)

From the **repo root**:

```bash
python examples/nodejs-helloworld/run_nodejs_helloworld.py
```

The script will:

1. Build the image from this directory with **build_image_from_path**
2. Create a container with **port forwarding** (container 3000 → host **8080**)
3. Start the server in the container in the background
4. Print **http://127.0.0.1:8080/** — open it in your browser or run `curl http://127.0.0.1:8080/`
5. Wait for you to press Enter, then stop and remove the container

### Option 2: CLI

From the **repo root** (replace with your path if needed):

```bash
# Build image from this directory
adde build_image_from_path '{"path":"examples/nodejs-helloworld","tag":"agent-env:nodejs-helloworld-1"}'

# Create container with port forwarding (3000 -> 8080)
adde create_runtime_env '{"image":"agent-env:nodejs-helloworld-1","dependencies":[],"env_vars":{},"network":true,"port_bindings":{"3000":"8080"}}'

# Start the server in the container (background), then open http://127.0.0.1:8080/
# When done: adde cleanup_env '{"container_id":"<id>"}'
```

**PowerShell** (path with backslashes):

```powershell
$dir = (Resolve-Path "examples\nodejs-helloworld").Path
"{\"path\":\"$($dir -replace '\\','\\\\')\",\"tag\":\"agent-env:nodejs-helloworld-1\"}" | .\go\adde.exe build_image_from_path
```

### Option 3: Run with Docker only (no ADDE)

```bash
cd examples/nodejs-helloworld
docker build -t nodejs-helloworld .
docker run --rm -p 3000:3000 nodejs-helloworld
# In another terminal: curl http://localhost:3000/
```

## Expected output

When you open **http://127.0.0.1:8080/** in your browser (or `curl http://127.0.0.1:8080/`):

```
Hello World
```

The run script prints something like:

```
1. Building image from path: ...
   Image: agent-env:nodejs-helloworld-<ts>
2. Creating container with port forwarding (container 3000 -> host 8080)...
   Container: abc123def456...
3. Starting server in container (background)...
   Server started.

   Server is running at  http://127.0.0.1:8080/
   Open in browser or run: curl http://127.0.0.1:8080/

   Press Enter to stop the container and exit...
```

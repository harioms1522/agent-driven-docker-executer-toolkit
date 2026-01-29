# Node.js Hello World — ADDE build_image_from_path example

A minimal Node.js HTTP server that responds with **Hello World** on the root path (`/`). This example is meant to be built with **build_image_from_path**: you already have the project directory (with a Dockerfile), so you build the image directly from it.

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
2. Create a container from the new image
3. Start the server and hit `http://localhost:3000/` inside the container to verify "Hello World"
4. Print the response and clean up

### Option 2: CLI

From the **repo root** (replace with your path if needed):

```bash
# Build image from this directory
adde build_image_from_path '{"path":"examples/nodejs-helloworld","tag":"agent-env:nodejs-helloworld-1"}'

# Create container (use the tag from above)
adde create_runtime_env '{"image":"agent-env:nodejs-helloworld-1","dependencies":[],"env_vars":{},"network":true}'

# Then use the container_id with execute_code_block, get_container_logs, cleanup_env
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

When you hit the server (via the run script or curl inside the container):

```
Hello World
```

The run script prints something like:

```
1. Building image from path (examples/nodejs-helloworld)...
   Image: agent-env:nodejs-helloworld-<ts>
2. Creating container...
3. Starting server and requesting GET / inside container...
   Response: Hello World
4. Cleaning up...
Done.
```

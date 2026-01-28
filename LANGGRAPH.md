# Using ADDE in a LangGraph Project (Python)

This guide shows how to use the [Agent-Driven Docker Executor (ADDE)](https://github.com/harioms1522/agent-driven-docker-executer-toolkit) from a **LangGraph** application so an agent can run code in isolated Docker containers and react to execution logs (refiner loop).

**Repository:** [https://github.com/harioms1522/agent-driven-docker-executer-toolkit](https://github.com/harioms1522/agent-driven-docker-executer-toolkit) (`main` branch)

---

## 1. Prerequisites

- **Docker** running and reachable.
- **Python 3.9+**.
- **ADDE binary** built (see [README](README.md#1-build-the-go-cli)) or installed so the Python client can find it (e.g. `ADDE_BIN` or `adde` / `adde.exe` on `PATH`).

---

## 2. Install from GitHub

In your LangGraph project environment:

```bash
# Install the ADDE Python client from the main branch (installs the python/ package as "adde")
pip install "git+https://github.com/harioms1522/agent-driven-docker-executer-toolkit.git@main#subdirectory=python"
```

This installs only the `python/` package. You still need the **ADDE binary** on the host:

- Either **build it** from the repo and point the client at it:
  ```bash
  git clone https://github.com/harioms1522/agent-driven-docker-executer-toolkit.git
  cd agent-driven-docker-executer-toolkit/go
  go build -o adde.exe ./cmd/adde   # Windows
  # go build -o adde ./cmd/adde     # Linux/macOS
  export ADDE_BIN=/path/to/adde.exe   # or add to PATH
  ```
- Or **download** a built binary from a [release](https://github.com/harioms1522/agent-driven-docker-executer-toolkit/releases) (if you publish one) and set `ADDE_BIN`.

Install LangGraph and a LangChain stack (e.g. for chat and tools):

```bash
pip install langgraph langchain-core langchain-openai   # or langchain-anthropic, etc.
```

---

## 3. Expose ADDE as LangChain tools

Wrap the ADDE client so LangGraph can call it via tool calls:

```python
from langchain_core.tools import tool
from adde import (
    pull_image,
    create_runtime_env,
    execute_code_block,
    get_container_logs,
    cleanup_env,
)

# Optional: set this if the binary is not on PATH
# import os
# os.environ["ADDE_BIN"] = "/path/to/adde.exe"

@tool
def adde_pull_image(image: str) -> str:
    """Pull a Docker image. Call before creating a runtime env if the image is not present."""
    r = pull_image(image)
    if r.get("error"):
        return f"Error: {r['error']}"
    return "OK"

@tool
def adde_create_runtime_env(
    image: str,
    dependencies: str = "",  # comma-separated, e.g. "requests,numpy"
    network: bool = False,
) -> str:
    """Create an isolated container with workspace at /workspace. Returns container_id or error."""
    deps = [s.strip() for s in dependencies.split(",") if s.strip()]
    r = create_runtime_env(image=image, dependencies=deps, env_vars={}, network=network)
    if r.get("error"):
        return f"Error: {r['error']}"
    return r["container_id"]

@tool
def adde_execute_code_block(container_id: str, filename: str, code_content: str, timeout_sec: int = 30) -> str:
    """Run code in the container. Returns stdout, stderr, exit_code, execution_time."""
    r = execute_code_block(container_id, filename, code_content, timeout_sec=timeout_sec)
    if r.get("error"):
        return f"Error: {r['error']}"
    log = r["log"]
    return f"exit_code={log['exit_code']}\nstdout:\n{log['stdout']}\nstderr:\n{log['stderr']}\ntime={log['execution_time']}"

@tool
def adde_get_container_logs(container_id: str, tail_lines: int = 0) -> str:
    """Get the last run's stdout/stderr/exit_code/execution_time. tail_lines=0 means all."""
    r = get_container_logs(container_id, tail_lines=tail_lines)
    if r.get("error"):
        return f"Error: {r['error']}"
    if not r.get("log"):
        return "No log yet."
    log = r["log"]
    return f"exit_code={log['exit_code']}\nstdout:\n{log['stdout']}\nstderr:\n{log['stderr']}\ntime={log['execution_time']}"

@tool
def adde_cleanup_env(container_id: str) -> str:
    """Stop and remove the container."""
    r = cleanup_env(container_id)
    if r.get("error"):
        return f"Error: {r['error']}"
    return "OK"

ADDE_TOOLS = [
    adde_pull_image,
    adde_create_runtime_env,
    adde_execute_code_block,
    adde_get_container_logs,
    adde_cleanup_env,
]
```

---

## 4. Wire tools into a LangGraph agent

Use a model that supports tool calling and add the ADDE tools. Example with OpenAI (you can swap in another model):

```python
from langchain_openai import ChatOpenAI
from langgraph.prebuilt import create_react_agent

# Model with tool-calling
llm = ChatOpenAI(model="gpt-4o-mini", temperature=0)

# ReAct agent that can call ADDE tools
app = create_react_agent(llm, ADDE_TOOLS)
```

Then invoke the agent so it can create envs, run code, and read logs:

```python
from langchain_core.messages import HumanMessage

result = app.invoke({"messages": [HumanMessage(content="Pull busybox, create a container, run 'echo 42' in t.sh, then cleanup.")]})
# Inspect result["messages"] for tool calls and results
```

---

## 5. Refiner loop (create → run → log → retry)

For a **code-execution refiner** (agent suggests code → run in container → feed logs back → agent fixes and reruns), keep `container_id` in state and route back to the agent when execution fails or when you want another attempt:

```python
from typing import Annotated, TypedDict
from langgraph.graph import StateGraph, START, END
from langgraph.graph.message import add_messages
from langchain_core.messages import BaseMessage

class RefinerState(TypedDict):
    messages: Annotated[list[BaseMessage], add_messages]
    container_id: str | None

def agent_node(state: RefinerState) -> dict:
    llm_with_tools = llm.bind_tools(ADDE_TOOLS)
    resp = llm_with_tools.invoke(state["messages"])
    return {"messages": [resp]}

def should_continue(state: RefinerState) -> str:
    last = state["messages"][-1]
    if hasattr(last, "tool_calls") and last.tool_calls:
        return "tools"
    return END

from langgraph.prebuilt import ToolNode
tool_node = ToolNode(ADDE_TOOLS)

graph = StateGraph(RefinerState)
graph.add_node("agent", agent_node)
graph.add_node("tools", tool_node)
graph.add_edge(START, "agent")
graph.add_conditional_edges("agent", should_continue)
graph.add_edge("tools", "agent")
graph.set_entry_point("agent")

app = graph.compile()
```

Your agent can then call `adde_create_runtime_env` → `adde_execute_code_block` →, and on non‑zero exit or stderr, the next turn can use `adde_get_container_logs` and suggest a fix before calling `adde_execute_code_block` again. When done, it calls `adde_cleanup_env`.

---

## 6. Minimal end-to-end script

Assumes `OPENAI_API_KEY` and Docker are set; ADDE binary is on `PATH` or `ADDE_BIN` is set.

```python
from langchain_openai import ChatOpenAI
from langchain_core.messages import HumanMessage
from langgraph.prebuilt import create_react_agent

# Define ADDE_TOOLS as in section 3
# …

llm = ChatOpenAI(model="gpt-4o-mini", temperature=0)
app = create_react_agent(llm, ADDE_TOOLS)

out = app.invoke({
    "messages": [
        HumanMessage(content=(
            "Use the adde tools to: 1) pull_image busybox, 2) create_runtime_env busybox, "
            "3) execute_code_block with filename t.sh and code_content 'echo 42', "
            "4) get_container_logs to show the result, 5) cleanup_env."
        ))
    ],
})

for m in out["messages"]:
    if hasattr(m, "content") and m.content:
        print(m.content)
    if hasattr(m, "tool_calls") and m.tool_calls:
        for tc in m.tool_calls:
            print("Tool:", tc["name"], tc.get("args"))
```

---

## 7. References

- **ADDE repo:** [https://github.com/harioms1522/agent-driven-docker-executer-toolkit](https://github.com/harioms1522/agent-driven-docker-executer-toolkit) (`main` branch)
- **ADDE README:** [README.md](README.md) (build, CLI, Python API, testing)
- **LangGraph:** [LangGraph docs](https://langchain-ai.github.io/langgraph/) — graphs, tools, state
- **LangChain tools:** [ToolNode / tool-calling](https://langchain-ai.github.io/langgraph/how-tos/tool-calling/) — wiring tools into a graph

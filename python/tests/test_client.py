"""
Tests for the ADDE Python client (adde.client).

Unit tests mock subprocess; integration tests call the real adde binary when
available (set ADDE_BIN or have go/adde.exe in repo).
"""

import json
import os
from pathlib import Path
from unittest.mock import MagicMock, patch

import pytest

from adde.client import (
    _call,
    _find_adde,
    cleanup_env,
    create_runtime_env,
    execute_code_block,
    get_container_logs,
    pull_image,
)


# ---- Unit tests (mocked subprocess) ----


@pytest.fixture
def mock_subprocess_run():
    """Patch subprocess.run and capture call args."""
    with patch("adde.client.subprocess.run") as m:
        yield m


def test_find_adde_uses_env_when_set(monkeypatch):
    monkeypatch.setenv("ADDE_BIN", "/custom/adde.exe")
    assert _find_adde() == "/custom/adde.exe"


def test_find_adde_returns_string(monkeypatch):
    monkeypatch.delenv("ADDE_BIN", raising=False)
    result = _find_adde()
    assert isinstance(result, str) and len(result) > 0
    # Either "adde" (PATH), "adde.exe", or a path ending in adde/adde.exe
    assert "adde" in result.lower()


def test_call_invokes_binary_with_tool_and_json(mock_subprocess_run):
    mock_subprocess_run.return_value = MagicMock(returncode=0, stdout='{"ok":true}', stderr="")
    _call("pull_image", {"image": "busybox"}, bin_path="/fake/adde.exe")
    mock_subprocess_run.assert_called_once()
    args = mock_subprocess_run.call_args[0][0]
    assert args[0] == "/fake/adde.exe"
    assert args[1] == "pull_image"
    assert json.loads(args[2]) == {"image": "busybox"}


def test_call_returns_parsed_json(mock_subprocess_run):
    mock_subprocess_run.return_value = MagicMock(returncode=0, stdout='{"ok":true}', stderr="")
    out = _call("pull_image", {"image": "busybox"}, bin_path="/fake/adde")
    assert out == {"ok": True}


def test_call_raises_on_nonzero_exit(mock_subprocess_run):
    mock_subprocess_run.return_value = MagicMock(
        returncode=1, stdout="", stderr="adde: no such image"
    )
    with pytest.raises(RuntimeError, match="no such image|adde pull_image failed"):
        _call("pull_image", {"image": "nonexistent"}, bin_path="/fake/adde")


def test_pull_image_params(mock_subprocess_run):
    mock_subprocess_run.return_value = MagicMock(returncode=0, stdout='{"ok":true}', stderr="")
    pull_image("busybox", bin_path="/fake/adde")
    call_args = json.loads(mock_subprocess_run.call_args[0][0][2])
    assert call_args == {"image": "busybox"}


def test_create_runtime_env_params(mock_subprocess_run):
    mock_subprocess_run.return_value = MagicMock(
        returncode=0,
        stdout='{"container_id":"abc","workspace":"/tmp/x"}',
        stderr="",
    )
    create_runtime_env(
        image="python:3.11-slim",
        dependencies=["requests"],
        env_vars={"X": "1"},
        network=True,
        bin_path="/fake/adde",
    )
    call_args = json.loads(mock_subprocess_run.call_args[0][0][2])
    assert call_args["image"] == "python:3.11-slim"
    assert call_args["dependencies"] == ["requests"]
    assert call_args["env_vars"] == {"X": "1"}
    assert call_args["network"] is True


def test_execute_code_block_params(mock_subprocess_run):
    mock_subprocess_run.return_value = MagicMock(
        returncode=0,
        stdout='{"log":{"exit_code":0,"stdout":"42","stderr":"","execution_time":"0.1s"}}',
        stderr="",
    )
    execute_code_block(
        container_id="cid",
        filename="t.py",
        code_content="print(42)",
        timeout_sec=15,
        bin_path="/fake/adde",
    )
    call_args = json.loads(mock_subprocess_run.call_args[0][0][2])
    assert call_args["container_id"] == "cid"
    assert call_args["filename"] == "t.py"
    assert call_args["code_content"] == "print(42)"
    assert call_args["timeout_sec"] == 15


def test_get_container_logs_params(mock_subprocess_run):
    mock_subprocess_run.return_value = MagicMock(
        returncode=0,
        stdout='{"log":{"exit_code":0,"stdout":"","stderr":"","execution_time":"0s"}}',
        stderr="",
    )
    get_container_logs(container_id="cid", tail_lines=10, bin_path="/fake/adde")
    call_args = json.loads(mock_subprocess_run.call_args[0][0][2])
    assert call_args["container_id"] == "cid"
    assert call_args["tail_lines"] == 10


def test_cleanup_env_params(mock_subprocess_run):
    mock_subprocess_run.return_value = MagicMock(returncode=0, stdout='{"ok":true}', stderr="")
    cleanup_env(container_id="cid", bin_path="/fake/adde")
    call_args = json.loads(mock_subprocess_run.call_args[0][0][2])
    assert call_args == {"container_id": "cid"}


# ---- Integration tests (real adde binary, optional) ----


def _adde_bin():
    """Path to adde binary, or None if not found."""
    if os.environ.get("ADDE_BIN"):
        p = os.environ["ADDE_BIN"]
        return p if os.path.isfile(p) else None
    root = Path(__file__).resolve().parents[2]
    for name in ("adde.exe", "adde"):
        cand = root / "go" / name
        if cand.is_file():
            return str(cand)
    return None


@pytest.fixture
def adde_bin():
    return _adde_bin()


@pytest.mark.skipif(_adde_bin() is None, reason="adde binary not found (build go/ or set ADDE_BIN)")
def test_integration_pull_image_empty_image_returns_error(adde_bin):
    """pull_image with empty image returns success exit but result contains error (CLI does not exit 1)."""
    from adde.client import _call
    r = _call("pull_image", {"image": ""}, bin_path=adde_bin)
    assert "error" in r and r["error"]
    assert r.get("ok") is not True


@pytest.mark.skipif(_adde_bin() is None, reason="adde binary not found (build go/ or set ADDE_BIN)")
def test_integration_usage_exit_code(adde_bin):
    """Calling adde with no tool shows usage and we see it via stderr when we bypass _call."""
    import subprocess
    out = subprocess.run(
        [adde_bin],
        capture_output=True,
        text=True,
        timeout=5,
    )
    assert out.returncode == 2
    assert "usage" in (out.stderr + out.stdout).lower()


@pytest.mark.skipif(_adde_bin() is None, reason="adde binary not found (build go/ or set ADDE_BIN)")
def test_integration_pull_image_ok_when_docker_up(adde_bin):
    """pull_image returns ok when image is pulled (Docker available)."""
    try:
        r = pull_image("busybox", bin_path=adde_bin)
    except RuntimeError as e:
        pytest.skip(f"Docker or pull failed: {e}")
    assert "error" not in r or not r["error"]
    assert r.get("ok") is True


@pytest.mark.skipif(_adde_bin() is None, reason="adde binary not found (build go/ or set ADDE_BIN)")
def test_integration_e2e_busybox_when_docker_up(adde_bin):
    """Full flow: pull -> create -> execute .sh -> get_container_logs -> cleanup."""
    try:
        pull_image("busybox", bin_path=adde_bin)
    except RuntimeError:
        pytest.skip("Docker or pull failed")
    try:
        r = create_runtime_env(
            image="busybox",
            dependencies=[],
            env_vars={},
            network=False,
            bin_path=adde_bin,
        )
    except RuntimeError as e:
        pytest.skip(f"create_runtime_env failed: {e}")
    if "error" in r and r["error"]:
        pytest.skip(f"create_runtime_env error: {r['error']}")
    cid = r["container_id"]
    try:
        out = execute_code_block(
            cid, "t.sh", "echo 42", timeout_sec=15, bin_path=adde_bin
        )
        assert "error" not in out or not out["error"]
        assert out["log"]["stdout"].strip() == "42"
        logs = get_container_logs(cid, tail_lines=10, bin_path=adde_bin)
        assert "error" not in logs or not logs["error"]
        if logs.get("log"):
            assert "42" in logs["log"].get("stdout", "")
    finally:
        cleanup_env(cid, bin_path=adde_bin)

//go:build windows
// +build windows

// cli_exe_test.go runs tests against the adde.exe binary on Windows.
// Build from go/: go build -o adde.exe ./cmd/adde
// Run: go test -v ./cmd/adde -run Exe
// Or set ADDE_EXE to the full path of adde.exe to use a specific binary.

package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func findExe(t *testing.T) string {
	t.Helper()
	if p := os.Getenv("ADDE_EXE"); p != "" {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	// When "go test ./cmd/adde" runs, cwd is typically go/cmd/adde; exe is often in go/
	dir, _ := os.Getwd()
	candidates := []string{
		filepath.Join(dir, "adde.exe"),
		filepath.Join(dir, "..", "adde.exe"),
		filepath.Join(dir, "..", "..", "adde.exe"),
		"adde.exe",
	}
	for _, c := range candidates {
		if abs, err := filepath.Abs(c); err == nil {
			if _, err := os.Stat(abs); err == nil {
				return abs
			}
		}
	}
	t.Skip("adde.exe not found; build with: go build -o adde.exe ./cmd/adde from go/")
	return ""
}

func runAdde(t *testing.T, exe string, tool string, payload string) (stdout, stderr string, exitCode int) {
	t.Helper()
	var args []string
	if tool != "" {
		args = append(args, strings.TrimSpace(strings.ToLower(tool)))
		if payload != "" {
			args = append(args, payload)
		}
	}
	cmd := exec.Command(exe, args...)
	var out, errBuf strings.Builder
	cmd.Stdout = &out
	cmd.Stderr = &errBuf
	err := cmd.Run()
	code := 0
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			code = ee.ExitCode()
		} else {
			t.Fatalf("run adde: %v", err)
		}
	}
	return out.String(), errBuf.String(), code
}

func TestExeUsage(t *testing.T) {
	exe := findExe(t)
	stdout, stderr, code := runAdde(t, exe, "", "")
	if code != 2 {
		t.Errorf("expected exit 2 when no args, got %d", code)
	}
	if want := "usage"; !strings.Contains(stderr, "usage") && !strings.Contains(stdout, "usage") {
		t.Errorf("expected 'usage' in output; stderr=%q stdout=%q", stderr, stdout)
	}
}

func TestExeUnknownTool(t *testing.T) {
	exe := findExe(t)
	_, stderr, code := runAdde(t, exe, "no_such_tool", "{}")
	if code != 1 {
		t.Errorf("expected exit 1 for unknown tool, got %d", code)
	}
	if want := "unknown"; !strings.Contains(stderr, "unknown") {
		t.Errorf("expected 'unknown' in stderr; got %q", stderr)
	}
}

func TestExeCreateRuntimeEnvInvalidJSON(t *testing.T) {
	exe := findExe(t)
	_, stderr, code := runAdde(t, exe, "create_runtime_env", "{invalid}")
	if code != 1 {
		t.Errorf("expected exit 1 for invalid JSON, got %d", code)
	}
	if stderr == "" && !strings.Contains(stderr, "adde") {
		t.Logf("stderr: %s", stderr)
	}
}

func TestExeExecuteCodeBlockInvalidJSON(t *testing.T) {
	exe := findExe(t)
	_, _, code := runAdde(t, exe, "execute_code_block", "not json")
	if code != 1 {
		t.Errorf("expected exit 1 for invalid JSON, got %d", code)
	}
}

func TestExeGetContainerLogsInvalidJSON(t *testing.T) {
	exe := findExe(t)
	_, _, code := runAdde(t, exe, "get_container_logs", "[]")
	if code != 1 {
		t.Errorf("expected exit 1 (array is not valid params), got %d", code)
	}
}

func TestExeCleanupEnvInvalidJSON(t *testing.T) {
	exe := findExe(t)
	_, _, code := runAdde(t, exe, "cleanup_env", "{bad}")
	if code != 1 {
		t.Errorf("expected exit 1 for invalid JSON, got %d", code)
	}
}

func TestExePullImageInvalidJSON(t *testing.T) {
	exe := findExe(t)
	_, _, code := runAdde(t, exe, "pull_image", "[]")
	if code != 1 {
		t.Errorf("expected exit 1 for invalid JSON, got %d", code)
	}
}

func TestExeCreateRuntimeEnvValidJSON(t *testing.T) {
	exe := findExe(t)
	payload := `{"image":"python:3.11-slim","dependencies":[],"env_vars":{},"network":false}`
	stdout, stderr, code := runAdde(t, exe, "create_runtime_env", payload)
	if code != 0 {
		t.Logf("create_runtime_env failed (Docker may be down): code=%d stderr=%s stdout=%s", code, stderr, stdout)
		t.Skip("create_runtime_env needs Docker; skipping")
	}
	var res struct {
		ContainerID string `json:"container_id"`
		Workspace   string `json:"workspace"`
		Error       string `json:"error"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(stdout)), &res); err != nil {
		t.Fatalf("output not JSON: %v\nraw: %s", err, stdout)
	}
	if res.Error != "" {
		t.Fatalf("create_runtime_env returned error: %s", res.Error)
	}
	if res.ContainerID == "" {
		t.Error("expected container_id in result")
	}
	// Cleanup so we don't leak
	cleanupPayload := `{"container_id":"` + res.ContainerID + `"}`
	_, _, cleanupCode := runAdde(t, exe, "cleanup_env", cleanupPayload)
	if cleanupCode != 0 {
		t.Logf("cleanup_env failed: %d (container may already be gone)", cleanupCode)
	}
}

func TestExeE2E(t *testing.T) {
	exe := findExe(t)
	createPayload := `{"image":"python:3.11-slim","dependencies":[],"env_vars":{},"network":false}`
	stdout, stderr, code := runAdde(t, exe, "create_runtime_env", createPayload)
	if code != 0 {
		t.Logf("create_runtime_env failed: code=%d stderr=%s stdout=%s", code, stderr, stdout)
		t.Skip("e2e needs Docker; skipping")
	}
	var createRes struct {
		ContainerID string `json:"container_id"`
		Error       string `json:"error"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(stdout)), &createRes); err != nil || createRes.ContainerID == "" {
		t.Fatalf("create result invalid: %v\nraw: %s", err, stdout)
	}
	cid := createRes.ContainerID
	defer func() {
		runAdde(t, exe, "cleanup_env", `{"container_id":"`+cid+`"}`)
	}()

	execPayload := `{"container_id":"` + cid + `","filename":"t.py","code_content":"print(42)","timeout_sec":15}`
	stdout, stderr, code = runAdde(t, exe, "execute_code_block", execPayload)
	if code != 0 {
		t.Fatalf("execute_code_block failed: %d stderr=%s stdout=%s", code, stderr, stdout)
	}
	var execRes struct {
		Log   *struct{ Stdout string `json:"stdout"` } `json:"log"`
		Error string `json:"error"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(stdout)), &execRes); err != nil {
		t.Fatalf("execute result not JSON: %v\nraw: %s", err, stdout)
	}
	if execRes.Log == nil {
		t.Fatalf("expected log in result; got %s", stdout)
	}
	if !strings.Contains(execRes.Log.Stdout, "42") {
		t.Errorf("expected stdout to contain 42; got %q", execRes.Log.Stdout)
	}

	logsPayload := `{"container_id":"` + cid + `","tail_lines":10}`
	stdout, _, code = runAdde(t, exe, "get_container_logs", logsPayload)
	if code != 0 {
		t.Fatalf("get_container_logs failed: %d", code)
	}
	var logsRes struct {
		Log   *struct{ Stdout string `json:"stdout"` } `json:"log"`
		Error string `json:"error"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(stdout)), &logsRes); err != nil {
		t.Fatalf("logs result not JSON: %v", err)
	}
	if logsRes.Log != nil && !strings.Contains(logsRes.Log.Stdout, "42") {
		t.Errorf("get_container_logs should reflect last run; got stdout %q", logsRes.Log.Stdout)
	}
}

// TestExeE2EBusyBox runs the full flow using busybox and a .sh script (echo 42).
func TestExeE2EBusyBox(t *testing.T) {
	exe := findExe(t)
	createPayload := `{"image":"busybox","dependencies":[],"env_vars":{},"network":false}`
	stdout, stderr, code := runAdde(t, exe, "create_runtime_env", createPayload)
	if code != 0 {
		t.Logf("create_runtime_env failed: code=%d stderr=%s stdout=%s", code, stderr, stdout)
		t.Skip("BusyBox e2e needs Docker; skipping")
	}
	var createRes struct {
		ContainerID string `json:"container_id"`
		Error       string `json:"error"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(stdout)), &createRes); err != nil || createRes.ContainerID == "" {
		t.Fatalf("create result invalid: %v\nraw: %s", err, stdout)
	}
	cid := createRes.ContainerID
	defer func() {
		runAdde(t, exe, "cleanup_env", `{"container_id":"`+cid+`"}`)
	}()

	execPayload := `{"container_id":"` + cid + `","filename":"t.sh","code_content":"echo 42","timeout_sec":15}`
	stdout, stderr, code = runAdde(t, exe, "execute_code_block", execPayload)
	if code != 0 {
		t.Fatalf("execute_code_block failed: %d stderr=%s stdout=%s", code, stderr, stdout)
	}
	var execRes struct {
		Log   *struct{ Stdout string `json:"stdout"` } `json:"log"`
		Error string `json:"error"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(stdout)), &execRes); err != nil {
		t.Fatalf("execute result not JSON: %v\nraw: %s", err, stdout)
	}
	if execRes.Log == nil {
		t.Fatalf("expected log in result; got %s", stdout)
	}
	if !strings.Contains(execRes.Log.Stdout, "42") {
		t.Errorf("expected stdout to contain 42; got %q", execRes.Log.Stdout)
	}
}

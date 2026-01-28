package executor

import (
	"archive/tar"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

const lastRunPath = ".adde_last_run.json"

// ExecuteCodeBlock writes code into the container via put_archive and runs it with a timeout.
// Returns the structured log (stdout/stderr/exit_code/execution_time) for the refiner agent.
func ExecuteCodeBlock(ctx context.Context, cli *client.Client, p ExecuteCodeBlockParams) ExecuteCodeBlockResult {
	timeout := 30
	if p.TimeoutSec > 0 {
		timeout = p.TimeoutSec
	}

	// Safe file transfer: build tar with only the file content (no shell interpolation)
	tarBuf, err := buildTarStream(p.Filename, p.CodeContent)
	if err != nil {
		return ExecuteCodeBlockResult{Error: err.Error()}
	}

	err = cli.CopyToContainer(ctx, p.ContainerID, WorkspacePathInsideContainer, tarBuf, types.CopyToContainerOptions{})
	if err != nil {
		return ExecuteCodeBlockResult{Error: err.Error()}
	}

	// Run based on extension; path in container is /workspace/<filename>
	fp := path.Join(WorkspacePathInsideContainer, p.Filename)
	cmd := runCommandForFile(fp, p.Filename)

	stdout, stderr, exitCode, dur, execErr := runExec(ctx, cli, p.ContainerID, cmd, timeout)
	if execErr != nil {
		return ExecuteCodeBlockResult{Error: execErr.Error()}
	}

	logEntry := &LogEntry{
		ExitCode:      exitCode,
		Stdout:        stdout,
		Stderr:        stderr,
		ExecutionTime: formatDuration(dur),
	}

	// Persist last run so get_container_logs can read it
	_ = persistLastRun(ctx, cli, p.ContainerID, logEntry)

	return ExecuteCodeBlockResult{Log: logEntry}
}

func buildTarStream(filename, content string) (*bytes.Buffer, error) {
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	hdr := &tar.Header{
		Name: filename,
		Mode: 0644,
		Size: int64(len(content)),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		return nil, err
	}
	if _, err := tw.Write([]byte(content)); err != nil {
		return nil, err
	}
	if err := tw.Close(); err != nil {
		return nil, err
	}
	return &buf, nil
}

func runCommandForFile(fullPath, filename string) []string {
	base := strings.ToLower(path.Ext(filename))
	switch base {
	case ".py":
		return []string{"python", fullPath}
	case ".js", ".mjs":
		return []string{"node", fullPath}
	case ".ts":
		return []string{"npx", "--yes", "ts-node", fullPath}
	case ".sh":
		return []string{"sh", fullPath}
	default:
		return []string{"sh", "-c", fullPath}
	}
}

func formatDuration(d time.Duration) string {
	sec := d.Seconds()
	if sec < 1 {
		return fmt.Sprintf("%.2fs", sec)
	}
	return fmt.Sprintf("%.2fs", sec)
}

func persistLastRun(ctx context.Context, cli *client.Client, containerID string, log *LogEntry) error {
	raw, err := json.Marshal(log)
	if err != nil {
		return err
	}
	tarBuf, err := buildTarStream(lastRunPath, string(raw))
	if err != nil {
		return err
	}
	return cli.CopyToContainer(ctx, containerID, WorkspacePathInsideContainer, tarBuf, types.CopyToContainerOptions{})
}

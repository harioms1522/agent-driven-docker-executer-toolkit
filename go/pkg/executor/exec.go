package executor

import (
	"bytes"
	"context"
	"io"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
)

// runExec runs cmd in the container and returns stdout, stderr, exitCode, duration.
// Used by create (deps) and execute_code_block.
func runExec(ctx context.Context, cli *client.Client, containerID string, cmd []string, timeoutSec int) (stdout, stderr string, exitCode int, dur time.Duration, err error) {
	if timeoutSec <= 0 {
		timeoutSec = 30
	}
	runCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSec)*time.Second)
	defer cancel()

	cfg := types.ExecConfig{
		Cmd:          cmd,
		AttachStdout: true,
		AttachStderr: true,
		WorkingDir:   WorkspacePathInsideContainer,
	}
	start := time.Now()
	createResp, err := cli.ContainerExecCreate(runCtx, containerID, cfg)
	if err != nil {
		return "", "", -1, 0, err
	}

	// Attach before Start so we have the stream when the process runs; otherwise we can
	// read "exec command has already run" instead of real stdout/stderr.
	resp, err := cli.ContainerExecAttach(runCtx, createResp.ID, types.ExecStartCheck{})
	if err != nil {
		return "", "", -1, 0, err
	}
	defer resp.Close()

	err = cli.ContainerExecStart(runCtx, createResp.ID, types.ExecStartCheck{})
	if err != nil {
		return "", "", -1, 0, err
	}

	var outBuf, errBuf bytes.Buffer
	_, err = stdcopy.StdCopy(&outBuf, &errBuf, resp.Reader)
	if err != nil && err != io.EOF {
		return "", "", -1, 0, err
	}
	dur = time.Since(start)

	inspect, err := cli.ContainerExecInspect(runCtx, createResp.ID)
	if err != nil {
		return outBuf.String(), errBuf.String(), -1, dur, err
	}
	return outBuf.String(), errBuf.String(), inspect.ExitCode, dur, nil
}

package executor

import (
	"context"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

// CleanupEnv stops and removes the container to prevent resource leaks (per spec §2).
func CleanupEnv(ctx context.Context, cli *client.Client, p CleanupEnvParams) CleanupEnvResult {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	timeoutSec := 5
	err := cli.ContainerStop(ctx, p.ContainerID, container.StopOptions{Timeout: &timeoutSec})
	if err != nil {
		// may already be stopped
	}
	err = cli.ContainerRemove(ctx, p.ContainerID, types.ContainerRemoveOptions{Force: true})
	if err != nil {
		return CleanupEnvResult{OK: false, Error: err.Error()}
	}
	return CleanupEnvResult{OK: true}
}

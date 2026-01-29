package executor

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
)

// CreateRuntimeEnv provisions a container with workspace mount, resource limits, and optional network.
// Returns the daemon error message on failure (per spec §4.2).
func CreateRuntimeEnv(ctx context.Context, cli *client.Client, p CreateRuntimeEnvParams) CreateRuntimeEnvResult {
	workspaceDir, err := os.MkdirTemp("", "adde-workspace-")
	if err != nil {
		return CreateRuntimeEnvResult{Error: fmt.Sprintf("failed to create workspace dir: %v", err)}
	}
	absWorkspace, _ := filepath.Abs(workspaceDir)

	envSlice := make([]string, 0, len(p.EnvVars)+1)
	for k, v := range p.EnvVars {
		envSlice = append(envSlice, k+"="+v)
	}

	networkMode := container.NetworkMode("none")
	if p.Network {
		networkMode = container.NetworkMode("default")
	}

	cfg := &container.Config{
		Image: p.Image,
		Env:   envSlice,
	}
	if p.UseImageCmd {
		// Run the image's default CMD (e.g. node server.js); use image's working dir so server starts correctly
		// Port bindings and /workspace mount still apply; agent can exec into /workspace later if needed
	} else {
		// Default: keep alive with sleep so agent runs code via exec
		cfg.Cmd = []string{"sleep", "86400"}
		cfg.WorkingDir = WorkspacePathInsideContainer
	}
	hostCfg := &container.HostConfig{
		Binds:       []string{absWorkspace + ":" + WorkspacePathInsideContainer},
		NetworkMode: networkMode,
		Resources: container.Resources{
			Memory:   DefaultMemoryLimitBytes,
			NanoCPUs: DefaultNanoCPUs,
		},
		AutoRemove: false,
	}

	// Port bindings: container_port -> host_port (e.g. "3000" -> "8080"); bind to 127.0.0.1
	if len(p.PortBindings) > 0 {
		exposed := make(map[nat.Port]struct{})
		portMap := make(map[nat.Port][]nat.PortBinding)
		for cPort, hPort := range p.PortBindings {
			cPort = strings.TrimSpace(cPort)
			hPort = strings.TrimSpace(hPort)
			if cPort == "" || hPort == "" {
				continue
			}
			if _, err := strconv.Atoi(hPort); err != nil {
				continue
			}
			portKey := nat.Port(cPort)
			if !strings.Contains(cPort, "/") {
				portKey = nat.Port(cPort + "/tcp")
			}
			exposed[portKey] = struct{}{}
			portMap[portKey] = []nat.PortBinding{{HostIP: "127.0.0.1", HostPort: hPort}}
		}
		cfg.ExposedPorts = exposed
		hostCfg.PortBindings = portMap
	}

	resp, err := cli.ContainerCreate(ctx, cfg, hostCfg, nil, nil, "")
	if err != nil {
		return CreateRuntimeEnvResult{Error: err.Error()}
	}

	if err := cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		_ = cli.ContainerRemove(ctx, resp.ID, types.ContainerRemoveOptions{Force: true})
		return CreateRuntimeEnvResult{Error: err.Error()}
	}

	// Install dependencies if requested (e.g. pip install / npm install)
	if len(p.Dependencies) > 0 {
		installErr := runDependencyInstall(ctx, cli, resp.ID, p.Image, p.Dependencies)
		if installErr != nil {
			_ = cli.ContainerRemove(ctx, resp.ID, types.ContainerRemoveOptions{Force: true})
			return CreateRuntimeEnvResult{Error: installErr.Error()}
		}
	}

	return CreateRuntimeEnvResult{
		ContainerID: resp.ID,
		Workspace:   absWorkspace,
	}
}

func runDependencyInstall(ctx context.Context, cli *client.Client, containerID, image string, deps []string) error {
	var cmd []string
	switch {
	case len(deps) == 0:
		return nil
	case isPythonImage(image):
		cmd = append([]string{"pip", "install", "--no-cache-dir", "-q"}, deps...)
	case isNodeImage(image):
		cmd = append([]string{"npm", "install", "-g"}, deps...)
	default:
		cmd = []string{"sh", "-c", "command -v pip >/dev/null 2>&1 && pip install --no-cache-dir -q " + strings.Join(deps, " ") + " || true"}
	}
	_, _, _, _, err := runExec(ctx, cli, containerID, cmd, 120)
	return err
}

func isPythonImage(s string) bool {
	return strings.Contains(strings.ToLower(s), "python")
}

func isNodeImage(s string) bool {
	return strings.Contains(strings.ToLower(s), "node")
}


package executor

import (
	"context"
	"io"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

// PullImage pulls the given image from the registry (Docker Hub, ECR, or custom).
// Registry auth is taken from environment variables when set; see RegistryAuthForImage.
// Call this before create_runtime_env if the image is not already present.
func PullImage(ctx context.Context, cli *client.Client, p PullImageParams) PullImageResult {
	ref := strings.TrimSpace(p.Image)
	if ref == "" {
		return PullImageResult{Error: "image name is required"}
	}
	opts := types.ImagePullOptions{}
	if auth, err := RegistryAuthForImage(ref); err != nil {
		return PullImageResult{Error: "registry auth: " + err.Error()}
	} else if auth != "" {
		opts.RegistryAuth = auth
	}
	rc, err := cli.ImagePull(ctx, ref, opts)
	if err != nil {
		return PullImageResult{Error: err.Error()}
	}
	defer rc.Close()
	_, _ = io.Copy(io.Discard, rc)
	return PullImageResult{OK: true}
}

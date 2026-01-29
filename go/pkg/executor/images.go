package executor

import (
	"context"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

// AgentImageTagPrefix is the required prefix for agent-created images.
const AgentImageTagPrefix = "agent-env:"

// ListAgentImages returns images tagged with the agent-env convention (optionally filtered by filter_tag).
func ListAgentImages(ctx context.Context, cli *client.Client, p ListAgentImagesParams) ListAgentImagesResult {
	listOpts := types.ImageListOptions{}
	list, err := cli.ImageList(ctx, listOpts)
	if err != nil {
		return ListAgentImagesResult{Error: err.Error()}
	}

	filterPrefix := AgentImageTagPrefix
	if p.FilterTag != "" {
		filterPrefix = strings.TrimSpace(p.FilterTag)
		if !strings.HasSuffix(filterPrefix, ":") && !strings.Contains(filterPrefix, ":") {
			filterPrefix = AgentImageTagPrefix + filterPrefix
		}
	}

	var out []AgentImageEntry
	for _, im := range list {
		var matchingTags []string
		for _, tag := range im.RepoTags {
			if strings.HasPrefix(tag, AgentImageTagPrefix) && strings.HasPrefix(tag, filterPrefix) {
				matchingTags = append(matchingTags, tag)
			}
		}
		if len(matchingTags) == 0 {
			continue
		}
		sizeMB := float64(im.Size) / (1024 * 1024)
		created := ""
		if im.Created > 0 {
			created = time.Unix(im.Created, 0).Format(time.RFC3339)
		}
		out = append(out, AgentImageEntry{
			ID:      im.ID,
			Tags:    matchingTags,
			SizeMB:  sizeMB,
			Created: created,
		})
	}
	return ListAgentImagesResult{Images: out}
}

// DeleteImage removes a Docker image by tag or ID. When AgentEnvOnly is true, only tags with prefix "agent-env:" are allowed.
func DeleteImage(ctx context.Context, cli *client.Client, p DeleteImageParams) DeleteImageResult {
	img := strings.TrimSpace(p.Image)
	if img == "" {
		return DeleteImageResult{Error: "image is required"}
	}
	if p.AgentEnvOnly && !strings.HasPrefix(img, AgentImageTagPrefix) {
		return DeleteImageResult{Error: "only agent-created images can be deleted (image must start with \"agent-env:\"); use list_agent_images to see allowed tags"}
	}
	opts := types.ImageRemoveOptions{Force: p.Force, PruneChildren: false}
	deleted, err := cli.ImageRemove(ctx, img, opts)
	if err != nil {
		return DeleteImageResult{Error: err.Error()}
	}
	var refs []string
	for _, d := range deleted {
		if d.Deleted != "" {
			refs = append(refs, "Deleted: "+d.Deleted)
		}
		if d.Untagged != "" {
			refs = append(refs, "Untagged: "+d.Untagged)
		}
	}
	return DeleteImageResult{OK: true, Deleted: refs}
}

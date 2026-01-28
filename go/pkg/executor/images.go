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

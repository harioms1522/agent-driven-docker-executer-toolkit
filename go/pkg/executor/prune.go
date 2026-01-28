package executor

import (
	"context"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
)

// PruneBuildCache cleans up intermediate build stages and unused build cache.
// older_than_hrs: if > 0, only prune cache older than that many hours; 0 = prune all unused.
func PruneBuildCache(ctx context.Context, cli *client.Client, p PruneBuildCacheParams) PruneBuildCacheResult {
	opts := types.BuildCachePruneOptions{
		KeepStorage: 0,
	}
	if p.OlderThanHrs > 0 {
		opts.Filters = filters.NewArgs(
			filters.Arg("until", (time.Duration(p.OlderThanHrs)*time.Hour).String()),
		)
	}
	report, err := cli.BuildCachePrune(ctx, opts)
	if err != nil {
		return PruneBuildCacheResult{Error: err.Error()}
	}
	spaceReclaimedMB := float64(report.SpaceReclaimed) / (1024 * 1024)
	return PruneBuildCacheResult{SpaceReclaimedMB: spaceReclaimedMB}
}

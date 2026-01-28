package executor

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/docker/docker/client"
)

// GetContainerLogs returns the last execution's structured log (exit_code, stdout, stderr, execution_time).
// Reads from /workspace/.adde_last_run.json written by ExecuteCodeBlock. tail_lines trims stdout/stderr to last N lines.
func GetContainerLogs(ctx context.Context, cli *client.Client, p GetContainerLogsParams) GetContainerLogsResult {
	stdout, _, _, _, err := runExec(ctx, cli, p.ContainerID, []string{"cat", ".adde_last_run.json"}, 10)
	if err != nil {
		return GetContainerLogsResult{Error: err.Error()}
	}
	raw := strings.TrimSpace(stdout)
	if raw == "" {
		return GetContainerLogsResult{Error: "no previous execution log found (run execute_code_block first)"}
	}
	var log LogEntry
	if err := json.Unmarshal([]byte(raw), &log); err != nil {
		return GetContainerLogsResult{Error: "invalid last run data: " + err.Error()}
	}
	if p.TailLines > 0 {
		log.Stdout = tailLines(log.Stdout, p.TailLines)
		log.Stderr = tailLines(log.Stderr, p.TailLines)
	}
	return GetContainerLogsResult{Log: &log}
}

func tailLines(s string, n int) string {
	lines := strings.Split(s, "\n")
	if len(lines) <= n {
		return s
	}
	return strings.Join(lines[len(lines)-n:], "\n")
}

package executor

import "time"

const (
	// DefaultExecutionTimeout is the default hard timeout for code execution.
	DefaultExecutionTimeout = 30 * time.Second
	// DefaultMemoryLimitBytes is 512 MiB.
	DefaultMemoryLimitBytes = 512 * 1024 * 1024
	// DefaultNanoCPUs is 0.5 CPU (1 CPU = 1e9 nanocpus).
	DefaultNanoCPUs = 500000000
	// WorkspacePathInsideContainer is the path mounted as workspace in the container.
	WorkspacePathInsideContainer = "/workspace"
)

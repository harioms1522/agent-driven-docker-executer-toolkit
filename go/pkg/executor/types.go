package executor

// CreateRuntimeEnvParams defines parameters for create_runtime_env.
type CreateRuntimeEnvParams struct {
	Image        string            `json:"image"`
	Dependencies []string          `json:"dependencies"`
	EnvVars      map[string]string `json:"env_vars"`
	Network      bool              `json:"network,omitempty"` // true = allow network; default false
}

// CreateRuntimeEnvResult is the return value of create_runtime_env.
type CreateRuntimeEnvResult struct {
	ContainerID string `json:"container_id,omitempty"`
	Workspace   string `json:"workspace,omitempty"`
	Error       string `json:"error,omitempty"`
}

// ExecuteCodeBlockParams defines parameters for execute_code_block.
type ExecuteCodeBlockParams struct {
	ContainerID string `json:"container_id"`
	Filename   string `json:"filename"`
	CodeContent string `json:"code_content"`
	TimeoutSec int    `json:"timeout_sec,omitempty"` // default 30
}

// ExecuteCodeBlockResult is the return value of execute_code_block; includes log for refiner feedback loop.
type ExecuteCodeBlockResult struct {
	Log   *LogEntry `json:"log,omitempty"`
	Error string    `json:"error,omitempty"`
}

// GetContainerLogsParams defines parameters for get_container_logs.
type GetContainerLogsParams struct {
	ContainerID string `json:"container_id"`
	TailLines   int    `json:"tail_lines,omitempty"` // 0 = all
}

// LogEntry is the structured feedback for the refiner agent (per spec §3.B).
type LogEntry struct {
	ExitCode       int    `json:"exit_code"`
	Stdout         string `json:"stdout"`
	Stderr         string `json:"stderr"`
	ExecutionTime  string `json:"execution_time"`
}

// GetContainerLogsResult wraps LogEntry or error.
type GetContainerLogsResult struct {
	Log  *LogEntry `json:"log,omitempty"`
	Error string   `json:"error,omitempty"`
}

// CleanupEnvParams defines parameters for cleanup_env.
type CleanupEnvParams struct {
	ContainerID string `json:"container_id"`
}

// CleanupEnvResult is the return value of cleanup_env.
type CleanupEnvResult struct {
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
}

// PullImageParams defines parameters for pull_image.
type PullImageParams struct {
	Image string `json:"image"` // e.g. "busybox", "python:3.11-slim"
}

// PullImageResult is the return value of pull_image.
type PullImageResult struct {
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
}

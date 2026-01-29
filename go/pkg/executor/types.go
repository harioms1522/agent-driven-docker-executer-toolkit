package executor

// CreateRuntimeEnvParams defines parameters for create_runtime_env.
type CreateRuntimeEnvParams struct {
	Image        string            `json:"image"`
	Dependencies []string          `json:"dependencies"`
	EnvVars      map[string]string `json:"env_vars"`
	Network      bool              `json:"network,omitempty"`   // true = allow network; default false
	PortBindings map[string]string `json:"port_bindings,omitempty"` // container_port -> host_port, e.g. {"3000": "8080"}
	UseImageCmd  bool              `json:"use_image_cmd,omitempty"` // true = run image's default CMD (e.g. server); false = run "sleep 86400" for exec-based use
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

// ---- Image Builder & Factory ----

// PrepareBuildContextParams defines parameters for prepare_build_context.
type PrepareBuildContextParams struct {
	Files    map[string]string `json:"files"`     // path -> content
	ContextID string           `json:"context_id"` // optional; if empty, a new ID is generated
}

// PrepareBuildContextResult is the return value of prepare_build_context.
type PrepareBuildContextResult struct {
	ContextID string `json:"context_id,omitempty"` // absolute path to build context dir
	Error     string `json:"error,omitempty"`
}

// BuildImageFromContextParams defines parameters for build_image_from_context.
type BuildImageFromContextParams struct {
	ContextID string            `json:"context_id"` // path from prepare_build_context
	Tag       string            `json:"tag"`        // e.g. agent-env:task-123-1706457600
	BuildArgs map[string]string `json:"build_args,omitempty"`
}

// BuildImageFromPathParams defines parameters for build_image_from_path.
// Use when the project already exists on disk (e.g. cloned repo) with a Dockerfile.
type BuildImageFromPathParams struct {
	Path      string            `json:"path"`       // absolute or relative path to directory containing Dockerfile
	Tag       string            `json:"tag"`        // e.g. agent-env:myapp-1
	BuildArgs map[string]string `json:"build_args,omitempty"`
}

// BuildImageFromContextResult is the return value of build_image_from_context (handshake format).
type BuildImageFromContextResult struct {
	Status          string  `json:"status,omitempty"`           // "success" or "error"
	ImageID         string  `json:"image_id,omitempty"`         // sha256:...
	Tag             string  `json:"tag,omitempty"`
	SizeMB          float64 `json:"size_mb,omitempty"`
	BuildLogSummary string  `json:"build_log_summary,omitempty"`
	FailedLayer     string  `json:"failed_layer,omitempty"`     // when status is error
	Error           string  `json:"error,omitempty"`
}

// ListAgentImagesParams defines parameters for list_agent_images.
type ListAgentImagesParams struct {
	FilterTag string `json:"filter_tag,omitempty"` // optional prefix filter, e.g. "agent-env"
}

// ListAgentImagesResult is the return value of list_agent_images.
type ListAgentImagesResult struct {
	Images []AgentImageEntry `json:"images,omitempty"`
	Error  string            `json:"error,omitempty"`
}

// AgentImageEntry is a single image entry for list_agent_images.
type AgentImageEntry struct {
	ID       string   `json:"id"`
	Tags     []string `json:"tags"`
	SizeMB   float64  `json:"size_mb"`
	Created  string   `json:"created,omitempty"`
}

// PruneBuildCacheParams defines parameters for prune_build_cache.
type PruneBuildCacheParams struct {
	OlderThanHrs int `json:"older_than_hrs,omitempty"` // 0 = prune all unused
}

// PruneBuildCacheResult is the return value of prune_build_cache.
type PruneBuildCacheResult struct {
	SpaceReclaimedMB float64 `json:"space_reclaimed_mb,omitempty"`
	Error            string  `json:"error,omitempty"`
}

// DeleteImageParams defines parameters for delete_image.
type DeleteImageParams struct {
	Image        string `json:"image"`                   // tag (e.g. agent-env:task-1) or image ID
	Force        bool   `json:"force,omitempty"`         // force remove even if in use (untag/remove)
	AgentEnvOnly bool   `json:"agent_env_only,omitempty"` // when true, only allow deletion of tags starting with agent-env:
}

// DeleteImageResult is the return value of delete_image.
type DeleteImageResult struct {
	OK     bool     `json:"ok"`
	Deleted []string `json:"deleted,omitempty"` // refs removed (e.g. tag or "Deleted: sha256:...")
	Error  string   `json:"error,omitempty"`
}

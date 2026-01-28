package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"adde/pkg/executor"

	"github.com/docker/docker/client"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "usage: adde <tool> [json_payload]\n")
		fmt.Fprintf(os.Stderr, "  tool: pull_image | create_runtime_env | execute_code_block | get_container_logs | cleanup_env | prepare_build_context | build_image_from_context | list_agent_images | prune_build_cache\n")
		fmt.Fprintf(os.Stderr, "  json_payload: JSON object for the tool, or omit to read from stdin\n")
		os.Exit(2)
	}
	tool := os.Args[1]
	var payload string
	if len(os.Args) >= 3 {
		payload = os.Args[2]
	} else {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			payload += scanner.Text()
		}
		if err := scanner.Err(); err != nil {
			fmt.Fprintf(os.Stderr, "adde: read stdin: %v\n", err)
			os.Exit(1)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	// Tools that don't need Docker client
	switch tool {
	case "prepare_build_context":
		var p executor.PrepareBuildContextParams
		if err := json.Unmarshal([]byte(payload), &p); err != nil {
			outErr(err)
			return
		}
		result := executor.PrepareBuildContext(p)
		outJSON(result)
		if result.Error != "" {
			os.Exit(1)
		}
		return
	}

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		fmt.Fprintf(os.Stderr, "adde: docker client: %v\n", err)
		os.Exit(1)
	}
	defer cli.Close()

	switch tool {
	case "pull_image":
		var p executor.PullImageParams
		if err := json.Unmarshal([]byte(payload), &p); err != nil {
			outErr(err)
			return
		}
		result := executor.PullImage(ctx, cli, p)
		outJSON(result)
		if result.Error != "" {
			os.Exit(1)
		}
	case "create_runtime_env":
		var p executor.CreateRuntimeEnvParams
		if err := json.Unmarshal([]byte(payload), &p); err != nil {
			outErr(err)
			return
		}
		result := executor.CreateRuntimeEnv(ctx, cli, p)
		outJSON(result)
		if result.Error != "" {
			os.Exit(1)
		}
	case "execute_code_block":
		var p executor.ExecuteCodeBlockParams
		if err := json.Unmarshal([]byte(payload), &p); err != nil {
			outErr(err)
			return
		}
		result := executor.ExecuteCodeBlock(ctx, cli, p)
		outJSON(result)
		if result.Error != "" {
			os.Exit(1)
		}
	case "get_container_logs":
		var p executor.GetContainerLogsParams
		if err := json.Unmarshal([]byte(payload), &p); err != nil {
			outErr(err)
			return
		}
		result := executor.GetContainerLogs(ctx, cli, p)
		outJSON(result)
		if result.Error != "" {
			os.Exit(1)
		}
	case "cleanup_env":
		var p executor.CleanupEnvParams
		if err := json.Unmarshal([]byte(payload), &p); err != nil {
			outErr(err)
			return
		}
		result := executor.CleanupEnv(ctx, cli, p)
		outJSON(result)
		if result.Error != "" {
			os.Exit(1)
		}
	case "build_image_from_context":
		var p executor.BuildImageFromContextParams
		if err := json.Unmarshal([]byte(payload), &p); err != nil {
			outErr(err)
			return
		}
		result := executor.BuildImageFromContext(ctx, cli, p)
		outJSON(result)
		if result.Status == "error" || result.Error != "" {
			os.Exit(1)
		}
	case "list_agent_images":
		var p executor.ListAgentImagesParams
		if err := json.Unmarshal([]byte(payload), &p); err != nil {
			outErr(err)
			return
		}
		result := executor.ListAgentImages(ctx, cli, p)
		outJSON(result)
		if result.Error != "" {
			os.Exit(1)
		}
	case "prune_build_cache":
		var p executor.PruneBuildCacheParams
		if err := json.Unmarshal([]byte(payload), &p); err != nil {
			outErr(err)
			return
		}
		result := executor.PruneBuildCache(ctx, cli, p)
		outJSON(result)
		if result.Error != "" {
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "adde: unknown tool %q\n", tool)
		os.Exit(2)
	}
}

func outJSON(v interface{}) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		fmt.Fprintf(os.Stderr, "adde: encode: %v\n", err)
		os.Exit(1)
	}
}

func outErr(err error) {
	fmt.Fprintf(os.Stderr, "adde: %v\n", err)
	outJSON(struct{ Error string }{err.Error()})
	os.Exit(1)
}

package executor

import (
	"archive/tar"
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

// Forbidden Dockerfile patterns (security: no host docker.sock or privileged mounts).
var forbiddenDockerfilePatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)/var/run/docker\.sock`),
	regexp.MustCompile(`(?i)-v\s+[^\\s]*docker\.sock`),
	regexp.MustCompile(`(?i)--mount[^\\n]*docker\.sock`),
	regexp.MustCompile(`(?i)privileged\s*true`),
}

// BuildImageFromContext runs docker build from the context directory.
// Validates Dockerfile for forbidden commands, then builds and returns the handshake result.
func BuildImageFromContext(ctx context.Context, cli *client.Client, p BuildImageFromContextParams) BuildImageFromContextResult {
	if p.ContextID == "" {
		return BuildImageFromContextResult{Status: "error", Error: "context_id is required"}
	}
	absDir := filepath.Clean(p.ContextID)
	info, err := os.Stat(absDir)
	if err != nil || !info.IsDir() {
		return BuildImageFromContextResult{Status: "error", Error: fmt.Sprintf("context_id is not a valid directory: %v", err)}
	}

	// Security: validate Dockerfile
	dockerfilePath := filepath.Join(absDir, "Dockerfile")
	dfContent, err := os.ReadFile(dockerfilePath)
	if err != nil {
		return BuildImageFromContextResult{Status: "error", Error: fmt.Sprintf("Dockerfile not found or unreadable: %v", err)}
	}
	if err := validateDockerfile(string(dfContent)); err != nil {
		return BuildImageFromContextResult{Status: "error", Error: err.Error()}
	}

	// Build tar context from directory
	tarBuf, err := tarContextFromDir(absDir)
	if err != nil {
		return BuildImageFromContextResult{Status: "error", Error: fmt.Sprintf("failed to create build context: %v", err)}
	}

	tag := strings.TrimSpace(p.Tag)
	if tag == "" {
		tag = "agent-env:build-" + fmt.Sprintf("%d", time.Now().Unix())
	}
	// Enforce tagging convention: agent-env:...
	if !strings.HasPrefix(tag, "agent-env:") {
		tag = "agent-env:" + tag
	}

	buildOpts := types.ImageBuildOptions{
		Tags:       []string{tag},
		Dockerfile: "Dockerfile",
		Remove:     true,
	}
	if len(p.BuildArgs) > 0 {
		buildOpts.BuildArgs = make(map[string]*string)
		for k, v := range p.BuildArgs {
			s := v
			buildOpts.BuildArgs[k] = &s
		}
	}

	buildCtx, cancel := context.WithTimeout(ctx, 15*time.Minute)
	defer cancel()

	resp, err := cli.ImageBuild(buildCtx, tarBuf, buildOpts)
	if err != nil {
		return BuildImageFromContextResult{Status: "error", Error: err.Error()}
	}
	defer resp.Body.Close()

	// Consume build output and parse for errors/summary
	summary, failedLayer, buildErr := parseBuildOutput(resp.Body)
	if buildErr != nil {
		return BuildImageFromContextResult{
			Status:          "error",
			Error:           buildErr.Error(),
			BuildLogSummary: summary,
			FailedLayer:     failedLayer,
		}
	}

	// Inspect image for ID and size
	imageID, sizeMB := getImageInfo(buildCtx, cli, tag)
	return BuildImageFromContextResult{
		Status:          "success",
		ImageID:         imageID,
		Tag:             tag,
		SizeMB:          sizeMB,
		BuildLogSummary: summary,
	}
}

func validateDockerfile(content string) error {
	for _, re := range forbiddenDockerfilePatterns {
		if re.MatchString(content) {
			return fmt.Errorf("Dockerfile contains forbidden pattern (e.g. docker.sock mount or privileged): security check failed")
		}
	}
	return nil
}

func tarContextFromDir(dir string) (io.Reader, error) {
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		// Use forward slashes for tar (Docker expects that)
		rel = filepath.ToSlash(rel)
		if info.IsDir() {
			tw.WriteHeader(&tar.Header{Name: rel + "/", Mode: 0755, Typeflag: tar.TypeDir})
			return nil
		}
		hdr, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		hdr.Name = rel
		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()
		_, err = io.Copy(tw, f)
		return err
	})
	if err != nil {
		return nil, err
	}
	if err := tw.Close(); err != nil {
		return nil, err
	}
	return &buf, nil
}

func parseBuildOutput(r io.Reader) (summary string, failedLayer string, err error) {
	scanner := bufio.NewScanner(r)
	var lastStream string
	var lastError string
	lineCount := 0
	for scanner.Scan() {
		line := scanner.Text()
		lineCount++
		// Docker build stream is JSON lines: {"stream": "..."} or {"error": "..."}
		if strings.HasPrefix(line, "{") {
			if strings.Contains(line, `"error"`) {
				lastError = line
			}
			if strings.Contains(line, `"stream"`) {
				lastStream = line
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return "", "", err
	}
	if lastError != "" {
		// Extract error message for failed_layer / summary
		if idx := strings.Index(lastError, `"error"`); idx >= 0 {
			summary = strings.TrimSpace(lastError)
			failedLayer = summary
		}
		return summary, failedLayer, fmt.Errorf("build failed: %s", summary)
	}
	if lastStream != "" {
		summary = lastStream
	} else {
		summary = fmt.Sprintf("Build completed. %d lines of output.", lineCount)
	}
	return summary, "", nil
}

func getImageInfo(ctx context.Context, cli *client.Client, tag string) (imageID string, sizeMB float64) {
	inspect, _, err := cli.ImageInspectWithRaw(ctx, tag)
	if err != nil {
		return "", 0
	}
	imageID = inspect.ID
	if inspect.Size > 0 {
		sizeMB = float64(inspect.Size) / (1024 * 1024)
	}
	return imageID, sizeMB
}

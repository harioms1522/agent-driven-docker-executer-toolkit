package executor

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	defaultDockerignore = `.git
.gitignore
*.md
.env
.venv
__pycache__
node_modules
*.pyc
.DS_Store
*.log
`
)

// PrepareBuildContext stages files into a temporary directory for Docker build.
// If no Dockerfile is provided but requirements.txt or package.json exists, injects a standard template.
// Writes a .dockerignore if not already in files to prevent bloat.
func PrepareBuildContext(p PrepareBuildContextParams) PrepareBuildContextResult {
	if len(p.Files) == 0 {
		return PrepareBuildContextResult{Error: "files map is required and must not be empty"}
	}
	dir, err := os.MkdirTemp("", "adde-build-")
	if err != nil {
		return PrepareBuildContextResult{Error: fmt.Sprintf("failed to create temp dir: %v", err)}
	}
	absDir, err := filepath.Abs(dir)
	if err != nil {
		os.RemoveAll(dir)
		return PrepareBuildContextResult{Error: fmt.Sprintf("failed to resolve path: %v", err)}
	}

	hasDockerfile := false
	hasRequirementsTxt := false
	hasPackageJson := false
	for name := range p.Files {
		base := filepath.Base(name)
		if base == "Dockerfile" || strings.HasPrefix(base, "Dockerfile.") {
			hasDockerfile = true
		}
		if base == "requirements.txt" {
			hasRequirementsTxt = true
		}
		if base == "package.json" {
			hasPackageJson = true
		}
	}

	// Write each file (support nested paths)
	for path, content := range p.Files {
		path = filepath.Clean(path)
		if path == ".." || strings.HasPrefix(path, ".."+string(filepath.Separator)) {
			continue // skip path traversal
		}
		full := filepath.Join(absDir, path)
		if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
			os.RemoveAll(absDir)
			return PrepareBuildContextResult{Error: fmt.Sprintf("failed to create dir for %q: %v", path, err)}
		}
		if err := os.WriteFile(full, []byte(content), 0644); err != nil {
			os.RemoveAll(absDir)
			return PrepareBuildContextResult{Error: fmt.Sprintf("failed to write %q: %v", path, err)}
		}
	}

	// Auto-generate .dockerignore if not in files
	if _, ok := p.Files[".dockerignore"]; !ok {
		if err := os.WriteFile(filepath.Join(absDir, ".dockerignore"), []byte(defaultDockerignore), 0644); err != nil {
			os.RemoveAll(absDir)
			return PrepareBuildContextResult{Error: fmt.Sprintf("failed to write .dockerignore: %v", err)}
		}
	}

	// Inject standard Dockerfile if codebase has requirements.txt or package.json but no Dockerfile
	if !hasDockerfile && (hasRequirementsTxt || hasPackageJson) {
		dockerfile := standardTemplateDockerfile(hasRequirementsTxt, hasPackageJson)
		if err := os.WriteFile(filepath.Join(absDir, "Dockerfile"), []byte(dockerfile), 0644); err != nil {
			os.RemoveAll(absDir)
			return PrepareBuildContextResult{Error: fmt.Sprintf("failed to write generated Dockerfile: %v", err)}
		}
	}

	return PrepareBuildContextResult{ContextID: absDir}
}

func standardTemplateDockerfile(python, node bool) string {
	// Prefer Python if both; otherwise Node; otherwise minimal Alpine.
	if python {
		return "FROM python:3-alpine\nWORKDIR /app\nCOPY requirements.txt .\nRUN pip install --no-cache-dir -r requirements.txt\nCOPY . .\n"
	}
	if node {
		return "FROM node:20-alpine\nWORKDIR /app\nCOPY package.json .\nRUN npm install\nCOPY . .\n"
	}
	return "FROM alpine:latest\nWORKDIR /app\nCOPY . .\n"
}

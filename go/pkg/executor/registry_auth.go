package executor

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"os/exec"
	"strings"
)

// Env var names for registry authentication. Used when pulling images from
// Docker Hub, ECR, or a custom registry.
const (
	// Docker Hub (index.docker.io). Use for public/private Docker Hub images.
	EnvDockerHubUsername = "ADDE_DOCKERHUB_USERNAME"
	EnvDockerHubPassword = "ADDE_DOCKERHUB_PASSWORD" // or personal access token

	// ECR. Either set ADDE_ECR_TOKEN (pre-fetched token) or AWS credentials + region.
	EnvECRToken   = "ADDE_ECR_TOKEN"
	EnvECRRegistry = "ADDE_ECR_REGISTRY" // optional; e.g. 123456789.dkr.ecr.us-east-1.amazonaws.com
	EnvAWSRegion  = "AWS_REGION"

	// Generic registry. ServerAddress must match the registry host in the image ref.
	EnvRegistryURL      = "ADDE_REGISTRY_URL"      // e.g. registry.example.com
	EnvRegistryUsername  = "ADDE_REGISTRY_USERNAME"
	EnvRegistryPassword  = "ADDE_REGISTRY_PASSWORD"
)

// registryHostFromImage returns the registry host from an image reference.
// e.g. "python:3.11-slim" -> "index.docker.io", "123456789.dkr.ecr.us-east-1.amazonaws.com/myimg:tag" -> "123456789.dkr.ecr.us-east-1.amazonaws.registryHostFromImage(imageRef string) string {
	ref := strings.TrimSpace(imageRef)
	if ref == "" {
		return ""
	}
	// Tag may contain ":port" so split from the right for tag; for host we look at first segment.
	first := ref
	if idx := strings.Index(ref, "/"); idx >= 0 {
		first = ref[:idx]
	}
	// If the first segment contains a dot or a colon (port), it's a registry host.
	if strings.Contains(first, ".") || strings.Contains(first, ":") {
		if strings.Contains(first, ":") {
			return first
		}
		return first
	}
	return "index.docker.io"
}

// isECRImage returns true if the image reference points to an ECR registry.
func isECRImage(imageRef string) bool {
	return strings.Contains(imageRef, ".dkr.ecr.") || strings.Contains(imageRef, "amazonaws.com")
}

// ecrRegistryFromImage extracts ECR registry host from image ref, or returns ADDE_ECR_REGISTRY / default.
func ecrRegistryFromImage(imageRef string) string {
	if s := os.Getenv(EnvECRRegistry); s != "" {
		return strings.TrimSpace(s)
	}
	if idx := strings.Index(imageRef, "/"); idx >= 0 {
		return strings.TrimSpace(imageRef[:idx])
	}
	return ""
}

// getECRToken returns ECR login password: from ADDE_ECR_TOKEN or by running aws ecr get-login-password.
func getECRToken(region string) (string, error) {
	if t := os.Getenv(EnvECRToken); t != "" {
		return strings.TrimSpace(t), nil
	}
	if region == "" {
		region = os.Getenv(EnvAWSRegion)
	}
	if region == "" {
		return "", nil
	}
	cmd := exec.Command("aws", "ecr", "get-login-password", "--region", region)
	cmd.Env = os.Environ()
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// RegistryAuthForImage returns base64-encoded Docker AuthConfig for the given image reference,
// using ADDE_* and AWS_* environment variables. Returns empty string if no auth is configured
// or the registry does not need auth from env.
func RegistryAuthForImage(imageRef string) (string, error) {
	ref := strings.TrimSpace(imageRef)
	if ref == "" {
		return "", nil
	}
	host := registryHostFromImage(ref)

	if isECRImage(ref) {
		reg := ecrRegistryFromImage(ref)
		if reg == "" {
			return "", nil
		}
		region := os.Getenv(EnvAWSRegion)
		if region == "" && strings.Contains(reg, ".dkr.ecr.") {
			// Try to parse region from host, e.g. 123456789.dkr.ecr.us-east-1.amazonaws.com
			parts := strings.Split(reg, ".")
			for i, p := range parts {
				if p == "ecr" && i+2 < len(parts) {
					region = parts[i+1]
					break
				}
			}
		}
		token, err := getECRToken(region)
		if err != nil || token == "" {
			return "", err
		}
		auth := map[string]string{
			"username":      "AWS",
			"password":      token,
			"serveraddress": reg,
		}
		return encodeAuth(auth)
	}

	if host == "index.docker.io" || host == "docker.io" {
		user := os.Getenv(EnvDockerHubUsername)
		pass := os.Getenv(EnvDockerHubPassword)
		if user == "" || pass == "" {
			return "", nil
		}
		auth := map[string]string{
			"username":      user,
			"password":      pass,
			"serveraddress": "https://index.docker.io/v1/",
		}
		return encodeAuth(auth)
	}

	// Generic registry: only use if ADDE_REGISTRY_URL matches the image's registry host
	regURL := strings.TrimSpace(os.Getenv(EnvRegistryURL))
	regURL = strings.TrimPrefix(regURL, "https://")
	regURL = strings.TrimPrefix(regURL, "http://")
	if regURL == "" {
		return "", nil
	}
	hostNorm := strings.TrimPrefix(host, "https://")
	hostNorm = strings.TrimPrefix(hostNorm, "http://")
	if hostNorm != regURL {
		return "", nil
	}
	user := os.Getenv(EnvRegistryUsername)
	pass := os.Getenv(EnvRegistryPassword)
	if user == "" || pass == "" {
		return "", nil
	}
	auth := map[string]string{
		"username":      user,
		"password":      pass,
		"serveraddress": host,
	}
	return encodeAuth(auth)
}

func encodeAuth(auth map[string]string) (string, error) {
	jsonBytes, err := json.Marshal(auth)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(jsonBytes), nil
}

package clioutput

import (
	"fmt"
	"net/url"
	pathpkg "path"
	"strings"
)

func PathKeyFromVideoPath(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", fmt.Errorf("missing video path")
	}

	if parsed, err := url.Parse(trimmed); err == nil && parsed.Path != "" {
		trimmed = parsed.Path
	}

	if !strings.HasPrefix(trimmed, "/v/") {
		return "", fmt.Errorf("invalid video path %q", raw)
	}

	key := strings.TrimSpace(pathpkg.Base(trimmed))
	if key == "" || key == "." || key == "/" {
		return "", fmt.Errorf("invalid video path %q", raw)
	}
	if strings.ContainsRune(key, '/') {
		return "", fmt.Errorf("invalid video path %q", raw)
	}

	return key, nil
}

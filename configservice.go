package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var unsafeFilenameCharacters = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

type ConfigService struct{}

func (s *ConfigService) SaveConfiguration(configurationJSON string, artifactID string) (string, error) {
	var formatted bytes.Buffer
	if err := json.Indent(&formatted, []byte(configurationJSON), "", "  "); err != nil {
		return "", fmt.Errorf("invalid project configuration: %w", err)
	}
	formatted.WriteByte('\n')

	homeDirectory, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("find home directory: %w", err)
	}

	downloadsDirectory := filepath.Join(homeDirectory, "Downloads")
	if err := os.MkdirAll(downloadsDirectory, 0o755); err != nil {
		return "", fmt.Errorf("create downloads directory: %w", err)
	}

	safeArtifactID := strings.Trim(unsafeFilenameCharacters.ReplaceAllString(artifactID, "-"), "-._")
	if safeArtifactID == "" {
		safeArtifactID = "spring-boot-project"
	}

	filePath := filepath.Join(downloadsDirectory, safeArtifactID+"-configuration.json")
	if err := os.WriteFile(filePath, formatted.Bytes(), 0o644); err != nil {
		return "", fmt.Errorf("write configuration file: %w", err)
	}

	return filePath, nil
}

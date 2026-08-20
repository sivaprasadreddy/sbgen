package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/sivaprasadreddy/sbgen/generators/springboot"
)

var (
	javaPackagePattern = regexp.MustCompile(`^[a-zA-Z_$][a-zA-Z0-9_$]*(\.[a-zA-Z_$][a-zA-Z0-9_$]*)*$`)
)

type ConfigService struct{}

type generatorConfiguration struct {
	Project struct {
		GroupID     string `json:"groupId"`
		ArtifactID  string `json:"artifactId"`
		PackageName string `json:"packageName"`
		BuildTool   string `json:"buildTool"`
	} `json:"project"`
	Frontend struct {
		Type         string  `json:"type"`
		CSSFramework *string `json:"cssFramework"`
	} `json:"frontend"`
	Database struct {
		Engine               string `json:"engine"`
		MigrationTool        string `json:"migrationTool"`
		PersistenceFramework string `json:"persistenceFramework"`
	} `json:"database"`
	Security struct {
		Mode string `json:"mode"`
	} `json:"security"`
	Features []string `json:"features"`
}

func (s *ConfigService) GenerateProject(configurationJSON string) (filePath string, err error) {
	var configuration generatorConfiguration
	if err := json.Unmarshal([]byte(configurationJSON), &configuration); err != nil {
		return "", fmt.Errorf("invalid project configuration: %w", err)
	}
	if err := validateConfiguration(configuration); err != nil {
		return "", err
	}

	homeDirectory, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("find home directory: %w", err)
	}
	downloadsDirectory := filepath.Join(homeDirectory, "Downloads")
	if err := os.MkdirAll(downloadsDirectory, 0o755); err != nil {
		return "", fmt.Errorf("create downloads directory: %w", err)
	}

	//TODO;
	//populate projectConfig data from configuration
	// Generate project and download it as a zip file and return the path to the zip file
	projectConfig := springboot.ProjectConfig{}

	err = springboot.GenerateProject(projectConfig)
	if err != nil {
		return "", err
	}

	return "", nil
}

func validateConfiguration(configuration generatorConfiguration) error {
	if strings.TrimSpace(configuration.Project.GroupID) == "" {
		return fmt.Errorf("group ID is required")
	}
	if strings.TrimSpace(configuration.Project.ArtifactID) == "" {
		return fmt.Errorf("artifact ID is required")
	}
	if !javaPackagePattern.MatchString(configuration.Project.PackageName) {
		return fmt.Errorf("invalid Java package name %q", configuration.Project.PackageName)
	}
	if !contains([]string{"maven", "gradle-groovy", "gradle-kotlin"}, configuration.Project.BuildTool) {
		return fmt.Errorf("unsupported build tool %q", configuration.Project.BuildTool)
	}
	if !contains([]string{"none", "thymeleaf", "angular"}, configuration.Frontend.Type) {
		return fmt.Errorf("unsupported frontend type %q", configuration.Frontend.Type)
	}
	if configuration.Frontend.CSSFramework != nil && !contains([]string{"tailwind", "bootstrap"}, *configuration.Frontend.CSSFramework) {
		return fmt.Errorf("unsupported CSS framework %q", *configuration.Frontend.CSSFramework)
	}
	if configuration.Frontend.Type != "none" && configuration.Frontend.CSSFramework == nil {
		return fmt.Errorf("CSS framework is required for %s frontend", configuration.Frontend.Type)
	}
	if !contains([]string{"postgresql", "mysql", "mariadb", "h2"}, configuration.Database.Engine) {
		return fmt.Errorf("unsupported database engine %q", configuration.Database.Engine)
	}
	if !contains([]string{"none", "flyway", "liquibase"}, configuration.Database.MigrationTool) {
		return fmt.Errorf("unsupported migration tool %q", configuration.Database.MigrationTool)
	}
	if !contains([]string{"spring-data-jpa", "spring-jdbc-client"}, configuration.Database.PersistenceFramework) {
		return fmt.Errorf("unsupported persistence framework %q", configuration.Database.PersistenceFramework)
	}
	if !contains([]string{"none", "http-session", "oauth-resource-server"}, configuration.Security.Mode) {
		return fmt.Errorf("unsupported security mode %q", configuration.Security.Mode)
	}
	if configuration.Security.Mode == "http-session" && configuration.Frontend.Type != "thymeleaf" {
		return fmt.Errorf("HTTP-session security requires the Thymeleaf frontend")
	}
	if configuration.Security.Mode == "oauth-resource-server" && configuration.Frontend.Type != "angular" {
		return fmt.Errorf("OAuth resource-server security requires the Angular frontend")
	}
	for _, feature := range configuration.Features {
		if !contains([]string{"email", "docker-compose", "spring-modulith", "spring-cloud-aws"}, feature) {
			return fmt.Errorf("unsupported feature %q", feature)
		}
	}
	return nil
}

func contains(values []string, target string) bool {
	return slices.Contains(values, target)
}

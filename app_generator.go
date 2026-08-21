package main

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"

	"github.com/sivaprasadreddy/sbgen/generators/springboot"
)

var (
	javaPackagePattern = regexp.MustCompile(`^[a-zA-Z_$][a-zA-Z0-9_$]*(\.[a-zA-Z_$][a-zA-Z0-9_$]*)*$`)
	generationMutex    sync.Mutex
)

type AppGeneratorService struct{}

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
		HTMXSupport  bool    `json:"htmxSupport"`
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

func (s *AppGeneratorService) GenerateProject(configurationJSON string) (filePath string, err error) {
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

	projectConfig := toProjectConfig(configuration)

	// The Spring Boot generator writes relative to the process working directory.
	// Serialize generation so changing it cannot interfere with another request.
	generationMutex.Lock()
	defer generationMutex.Unlock()

	workingDirectory, err := os.MkdirTemp("", "sbgen-")
	if err != nil {
		return "", fmt.Errorf("create generation directory: %w", err)
	}
	defer os.RemoveAll(workingDirectory)

	originalDirectory, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("find current directory: %w", err)
	}
	if err := os.Chdir(workingDirectory); err != nil {
		return "", fmt.Errorf("enter generation directory: %w", err)
	}
	if err := springboot.GenerateProject(projectConfig); err != nil {
		_ = os.Chdir(originalDirectory)
		return "", fmt.Errorf("generate project: %w", err)
	}
	if err := os.Chdir(originalDirectory); err != nil {
		return "", fmt.Errorf("restore working directory: %w", err)
	}

	zipFile, zipPath, err := createUniqueZipFile(downloadsDirectory, projectConfig.AppName)
	if err != nil {
		return "", fmt.Errorf("create project archive: %w", err)
	}
	if err := zipDirectory(zipFile, filepath.Join(workingDirectory, projectConfig.AppName)); err != nil {
		_ = zipFile.Close()
		_ = os.Remove(zipPath)
		return "", fmt.Errorf("archive project: %w", err)
	}
	if err := zipFile.Close(); err != nil {
		_ = os.Remove(zipPath)
		return "", fmt.Errorf("close project archive: %w", err)
	}

	return zipPath, nil
}

func createUniqueZipFile(directory, name string) (*os.File, string, error) {
	for suffix := 0; ; suffix++ {
		fileName := name + ".zip"
		if suffix > 0 {
			fileName = fmt.Sprintf("%s-%d.zip", name, suffix)
		}
		filePath := filepath.Join(directory, fileName)
		file, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if err == nil {
			return file, filePath, nil
		}
		if !os.IsExist(err) {
			return nil, "", err
		}
	}
}

func toProjectConfig(configuration generatorConfiguration) springboot.ProjectConfig {
	buildTool := springboot.Maven
	if configuration.Project.BuildTool != "maven" {
		buildTool = springboot.Gradle
	}

	appType := springboot.RestApi
	if configuration.Frontend.Type == "thymeleaf" {
		appType = springboot.WebApp
	} else if configuration.Frontend.Type == "angular" {
		appType = springboot.SpringBootAngular
	}
	databaseTypes := map[string]springboot.DatabaseType{
		"postgresql": springboot.PostgreSQL,
		"mysql":      springboot.MySQL,
		"mariadb":    springboot.MariaDB,
		"h2":         springboot.H2,
	}
	migrationTools := map[string]springboot.DbMigrationTool{
		"none":      springboot.None,
		"flyway":    springboot.Flyway,
		"liquibase": springboot.Liquibase,
	}

	return springboot.ProjectConfig{
		AppType:               appType,
		AppName:               strings.TrimSpace(configuration.Project.ArtifactID),
		GroupID:               strings.TrimSpace(configuration.Project.GroupID),
		ArtifactID:            strings.TrimSpace(configuration.Project.ArtifactID),
		AppVersion:            "1.0.0",
		BasePackage:           configuration.Project.PackageName,
		BuildTool:             buildTool,
		DbType:                databaseTypes[configuration.Database.Engine],
		DbMigrationTool:       migrationTools[configuration.Database.MigrationTool],
		DockerComposeSupport:  contains(configuration.Features, "docker-compose"),
		SpringModulithSupport: contains(configuration.Features, "spring-modulith"),
		SpringCloudAWSSupport: contains(configuration.Features, "spring-cloud-aws"),
		ThymeleafSupport:      configuration.Frontend.Type == "thymeleaf",
		HTMXSupport:           configuration.Frontend.HTMXSupport,
		SecuritySupport:       configuration.Security.Mode == "http-session",
		JwtSecuritySupport:    configuration.Security.Mode == "oauth-resource-server",
	}
}

func zipDirectory(destination io.Writer, sourceDirectory string) error {
	archive := zip.NewWriter(destination)
	err := filepath.Walk(sourceDirectory, func(filePath string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() {
			return nil
		}

		relativePath, err := filepath.Rel(filepath.Dir(sourceDirectory), filePath)
		if err != nil {
			return err
		}
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		header.Name = filepath.ToSlash(relativePath)
		header.Method = zip.Deflate
		entry, err := archive.CreateHeader(header)
		if err != nil {
			return err
		}
		file, err := os.Open(filePath)
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(entry, file)
		closeErr := file.Close()
		if copyErr != nil {
			return copyErr
		}
		return closeErr
	})
	if err != nil {
		_ = archive.Close()
		return err
	}
	return archive.Close()
}

func validateConfiguration(configuration generatorConfiguration) error {
	if strings.TrimSpace(configuration.Project.GroupID) == "" {
		return fmt.Errorf("group ID is required")
	}
	if strings.TrimSpace(configuration.Project.ArtifactID) == "" {
		return fmt.Errorf("artifact ID is required")
	}
	artifactID := strings.TrimSpace(configuration.Project.ArtifactID)
	if artifactID == "." || artifactID == ".." || filepath.Base(artifactID) != artifactID || strings.ContainsAny(artifactID, `/\\`) {
		return fmt.Errorf("invalid artifact ID %q", configuration.Project.ArtifactID)
	}
	if !javaPackagePattern.MatchString(configuration.Project.PackageName) {
		return fmt.Errorf("invalid Java package name %q", configuration.Project.PackageName)
	}
	if !contains([]string{"maven", "gradle-groovy"}, configuration.Project.BuildTool) {
		return fmt.Errorf("unsupported build tool %q", configuration.Project.BuildTool)
	}
	if !contains([]string{"none", "thymeleaf", "angular"}, configuration.Frontend.Type) {
		return fmt.Errorf("unsupported frontend type %q", configuration.Frontend.Type)
	}
	if configuration.Frontend.CSSFramework != nil && *configuration.Frontend.CSSFramework != "tailwind" {
		return fmt.Errorf("unsupported CSS framework %q", *configuration.Frontend.CSSFramework)
	}
	if configuration.Frontend.Type != "none" && configuration.Frontend.CSSFramework == nil {
		return fmt.Errorf("CSS framework is required for %s frontend", configuration.Frontend.Type)
	}
	if configuration.Frontend.HTMXSupport && configuration.Frontend.Type != "thymeleaf" {
		return fmt.Errorf("HTMX support requires the Thymeleaf frontend")
	}
	if !contains([]string{"postgresql", "mysql", "mariadb", "h2"}, configuration.Database.Engine) {
		return fmt.Errorf("unsupported database engine %q", configuration.Database.Engine)
	}
	if !contains([]string{"none", "flyway", "liquibase"}, configuration.Database.MigrationTool) {
		return fmt.Errorf("unsupported migration tool %q", configuration.Database.MigrationTool)
	}
	if configuration.Database.PersistenceFramework != "spring-data-jpa" {
		return fmt.Errorf("unsupported persistence framework %q", configuration.Database.PersistenceFramework)
	}
	if !contains([]string{"none", "http-session", "oauth-resource-server"}, configuration.Security.Mode) {
		return fmt.Errorf("unsupported security mode %q", configuration.Security.Mode)
	}
	if configuration.Security.Mode == "http-session" && configuration.Frontend.Type != "thymeleaf" {
		return fmt.Errorf("HTTP-session security requires the Thymeleaf frontend")
	}
	if configuration.Security.Mode == "oauth-resource-server" && !contains([]string{"none", "angular"}, configuration.Frontend.Type) {
		return fmt.Errorf("OAuth resource-server security requires no frontend or the Angular frontend")
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

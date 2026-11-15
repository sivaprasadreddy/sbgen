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

	"github.com/sivaprasadreddy/progen/generators/springboot"
)

var (
	javaPackagePattern = regexp.MustCompile(`^[a-zA-Z_$][a-zA-Z0-9_$]*(\.[a-zA-Z_$][a-zA-Z0-9_$]*)*$`)
	generationMutex    sync.Mutex
)

type AppGeneratorService struct{}

type generatorConfiguration struct {
	Project struct {
		AppName     string `json:"appName"`
		AppVersion  string `json:"appVersion"`
		GroupID     string `json:"groupId"`
		ArtifactID  string `json:"artifactId"`
		PackageName string `json:"packageName"`
		BuildTool   string `json:"buildTool"`
	} `json:"project"`
	Frontend struct {
		Type string `json:"type"`
		HTMX bool   `json:"htmxSupport"`
	} `json:"frontend"`
	Database struct {
		Engine               string `json:"engine"`
		MigrationTool        string `json:"migrationTool"`
		PersistenceFramework string `json:"persistenceFramework"`
	} `json:"database"`
	Features []string `json:"features"`
	AIModels []string `json:"aiModels"`
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
	if configuration.Project.BuildTool == "gradle" {
		buildTool = springboot.Gradle
	}

	frontends := map[string]springboot.Frontend{
		"none":      springboot.FrontendNone,
		"thymeleaf": springboot.FrontendThymeleaf,
		"jte":       springboot.FrontendJTE,
		"angular":   springboot.FrontendAngular,
	}
	databaseTypes := map[string]springboot.DatabaseType{
		"postgresql": springboot.PostgreSQL,
		"mysql":      springboot.MySQL,
		"mariadb":    springboot.MariaDB,
	}
	migrationTools := map[string]springboot.DbMigrationTool{
		"flyway":    springboot.Flyway,
		"liquibase": springboot.Liquibase,
	}
	persistenceTypes := map[string]springboot.PersistenceType{
		"spring-data-jpa":    springboot.SpringDataJPA,
		"spring-jdbc-client": springboot.SpringJdbcClient,
		"jooq":               springboot.SpringJOOQ,
	}

	frontend := frontends[configuration.Frontend.Type]
	aiModels := []springboot.AIModelProvider{}
	for _, model := range configuration.AIModels {
		aiModels = append(aiModels, springboot.AIModelProvider(model))
	}

	return springboot.ProjectConfig{
		Frontend:              frontend,
		AppName:               strings.TrimSpace(configuration.Project.AppName),
		GroupID:               strings.TrimSpace(configuration.Project.GroupID),
		ArtifactID:            strings.TrimSpace(configuration.Project.ArtifactID),
		AppVersion:            strings.TrimSpace(configuration.Project.AppVersion),
		BasePackage:           strings.TrimSpace(configuration.Project.PackageName),
		BuildTool:             buildTool,
		DbType:                databaseTypes[configuration.Database.Engine],
		DbMigrationTool:       migrationTools[configuration.Database.MigrationTool],
		PersistenceType:       persistenceTypes[configuration.Database.PersistenceFramework],
		SpringCloudAWSSupport: contains(configuration.Features, "spring-cloud-aws"),
		EmailSupport:          contains(configuration.Features, "email"),
		RabbitMQSupport:       contains(configuration.Features, "rabbitmq"),
		RedisCachingSupport:   contains(configuration.Features, "redis-caching"),
		OpenTelemetrySupport:  contains(configuration.Features, "opentelemetry"),
		K8sSupport:            contains(configuration.Features, "kubernetes"),
		HTMXSupport:           configuration.Frontend.HTMX && isServerRendered(configuration.Frontend.Type),
		SpringAIModels:        aiModels,
	}
}

func isServerRendered(frontendType string) bool {
	return frontendType == "thymeleaf" || frontendType == "jte"
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
	appName := strings.TrimSpace(configuration.Project.AppName)
	if appName == "" {
		return fmt.Errorf("application name is required")
	}
	if appName == "." || appName == ".." || filepath.Base(appName) != appName || strings.ContainsAny(appName, `/\\`) {
		return fmt.Errorf("invalid application name %q", configuration.Project.AppName)
	}
	if strings.TrimSpace(configuration.Project.AppVersion) == "" {
		return fmt.Errorf("application version is required")
	}
	if strings.TrimSpace(configuration.Project.GroupID) == "" {
		return fmt.Errorf("group ID is required")
	}
	if strings.TrimSpace(configuration.Project.ArtifactID) == "" {
		return fmt.Errorf("artifact ID is required")
	}
	if !javaPackagePattern.MatchString(strings.TrimSpace(configuration.Project.PackageName)) {
		return fmt.Errorf("invalid Java package name %q", configuration.Project.PackageName)
	}
	if !contains([]string{"maven", "gradle"}, configuration.Project.BuildTool) {
		return fmt.Errorf("unsupported build tool %q", configuration.Project.BuildTool)
	}
	if !contains([]string{"none", "thymeleaf", "jte", "angular"}, configuration.Frontend.Type) {
		return fmt.Errorf("unsupported frontend type %q", configuration.Frontend.Type)
	}
	if configuration.Frontend.HTMX && !isServerRendered(configuration.Frontend.Type) {
		return fmt.Errorf("HTMX support requires the Thymeleaf or JTE frontend")
	}
	if !contains([]string{"postgresql", "mysql", "mariadb"}, configuration.Database.Engine) {
		return fmt.Errorf("unsupported database engine %q", configuration.Database.Engine)
	}
	if !contains([]string{"flyway", "liquibase"}, configuration.Database.MigrationTool) {
		return fmt.Errorf("unsupported migration tool %q", configuration.Database.MigrationTool)
	}
	if !contains([]string{"spring-data-jpa", "spring-jdbc-client", "jooq"}, configuration.Database.PersistenceFramework) {
		return fmt.Errorf("unsupported persistence framework %q", configuration.Database.PersistenceFramework)
	}
	for _, feature := range configuration.Features {
		if !contains([]string{"email", "spring-cloud-aws", "rabbitmq", "redis-caching", "opentelemetry", "kubernetes"}, feature) {
			return fmt.Errorf("unsupported feature %q", feature)
		}
	}
	for _, model := range configuration.AIModels {
		if !springboot.AIModelProvider(model).IsValid() {
			return fmt.Errorf("unsupported Spring AI model provider %q", model)
		}
	}
	return nil
}

func contains(values []string, target string) bool {
	return slices.Contains(values, target)
}

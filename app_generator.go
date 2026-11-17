package main

import (
	"archive/zip"
	"errors"
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

// BuildTool is the build system of the generated project.
type BuildTool string

const (
	BuildToolMaven  BuildTool = "maven"
	BuildToolGradle BuildTool = "gradle"
)

// TaskRunner runs the generated project's common tasks.
type TaskRunner string

const (
	TaskRunnerMise     TaskRunner = "mise"
	TaskRunnerTaskfile TaskRunner = "taskfile"
)

// FrontendType is how the generated project's user interface is built.
type FrontendType string

const (
	FrontendNone      FrontendType = "none"
	FrontendThymeleaf FrontendType = "thymeleaf"
	FrontendJTE       FrontendType = "jte"
	FrontendAngular   FrontendType = "angular"
	FrontendReact     FrontendType = "react"
)

// DatabaseEngine is the database the generated project connects to.
type DatabaseEngine string

const (
	DatabasePostgreSQL DatabaseEngine = "postgresql"
	DatabaseMySQL      DatabaseEngine = "mysql"
	DatabaseMariaDB    DatabaseEngine = "mariadb"
)

// MigrationTool manages the generated project's database schema.
type MigrationTool string

const (
	MigrationFlyway    MigrationTool = "flyway"
	MigrationLiquibase MigrationTool = "liquibase"
)

// PersistenceFramework is how the generated project accesses the database.
type PersistenceFramework string

const (
	PersistenceSpringDataJPA    PersistenceFramework = "spring-data-jpa"
	PersistenceSpringJdbcClient PersistenceFramework = "spring-jdbc-client"
	PersistenceJOOQ             PersistenceFramework = "jooq"
)

// Feature is an optional capability of the generated project.
type Feature string

const (
	FeatureEmail          Feature = "email"
	FeatureSpringCloudAWS Feature = "spring-cloud-aws"
	FeatureRabbitMQ       Feature = "rabbitmq"
	FeatureRedisCaching   Feature = "redis-caching"
	FeatureOpenTelemetry  Feature = "opentelemetry"
	FeatureKubernetes     Feature = "kubernetes"
)

// AIModel is a Spring AI chat model provider.
type AIModel string

const (
	AIModelOllama    AIModel = "ollama"
	AIModelOpenAI    AIModel = "openai"
	AIModelAnthropic AIModel = "anthropic"
)

// ProjectSettings holds the project coordinates and build setup.
type ProjectSettings struct {
	AppName     string     `json:"appName"`
	AppVersion  string     `json:"appVersion"`
	GroupID     string     `json:"groupId"`
	ArtifactID  string     `json:"artifactId"`
	PackageName string     `json:"packageName"`
	BuildTool   BuildTool  `json:"buildTool"`
	TaskRunner  TaskRunner `json:"taskRunner"`
}

// FrontendSettings selects the generated project's user interface.
type FrontendSettings struct {
	Type FrontendType `json:"type"`
	HTMX bool         `json:"htmxSupport"`
}

// DatabaseSettings selects the generated project's database stack.
type DatabaseSettings struct {
	Engine               DatabaseEngine       `json:"engine"`
	MigrationTool        MigrationTool        `json:"migrationTool"`
	PersistenceFramework PersistenceFramework `json:"persistenceFramework"`
}

// GeneratorConfiguration is everything the user chose in the wizard.
type GeneratorConfiguration struct {
	Project  ProjectSettings  `json:"project"`
	Frontend FrontendSettings `json:"frontend"`
	Database DatabaseSettings `json:"database"`
	Features []Feature        `json:"features"`
	AIModels []AIModel        `json:"aiModels"`
}

// AppGeneratorService generates Spring Boot projects.
type AppGeneratorService struct{}

// GenerateProject generates a Spring Boot project for configuration, saves it
// as a ZIP archive in the user's Downloads directory and returns the
// archive's path.
func (s *AppGeneratorService) GenerateProject(configuration GeneratorConfiguration) (string, error) {
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

func toProjectConfig(configuration GeneratorConfiguration) springboot.ProjectConfig {
	buildTool := springboot.Maven
	if configuration.Project.BuildTool == BuildToolGradle {
		buildTool = springboot.Gradle
	}

	frontends := map[FrontendType]springboot.Frontend{
		FrontendNone:      springboot.FrontendNone,
		FrontendThymeleaf: springboot.FrontendThymeleaf,
		FrontendJTE:       springboot.FrontendJTE,
		FrontendAngular:   springboot.FrontendAngular,
		FrontendReact:     springboot.FrontendReact,
	}
	taskRunners := map[TaskRunner]springboot.TaskRunner{
		TaskRunnerMise:     springboot.TaskRunnerMise,
		TaskRunnerTaskfile: springboot.TaskRunnerTaskfile,
	}
	databaseTypes := map[DatabaseEngine]springboot.DatabaseType{
		DatabasePostgreSQL: springboot.PostgreSQL,
		DatabaseMySQL:      springboot.MySQL,
		DatabaseMariaDB:    springboot.MariaDB,
	}
	migrationTools := map[MigrationTool]springboot.DbMigrationTool{
		MigrationFlyway:    springboot.Flyway,
		MigrationLiquibase: springboot.Liquibase,
	}
	persistenceTypes := map[PersistenceFramework]springboot.PersistenceType{
		PersistenceSpringDataJPA:    springboot.SpringDataJPA,
		PersistenceSpringJdbcClient: springboot.SpringJdbcClient,
		PersistenceJOOQ:             springboot.SpringJOOQ,
	}

	aiModels := []springboot.AIModelProvider{}
	for _, model := range configuration.AIModels {
		aiModels = append(aiModels, springboot.AIModelProvider(model))
	}

	features := configuration.Features
	return springboot.ProjectConfig{
		Frontend:              frontends[configuration.Frontend.Type],
		AppName:               strings.TrimSpace(configuration.Project.AppName),
		GroupID:               strings.TrimSpace(configuration.Project.GroupID),
		ArtifactID:            strings.TrimSpace(configuration.Project.ArtifactID),
		AppVersion:            strings.TrimSpace(configuration.Project.AppVersion),
		BasePackage:           strings.TrimSpace(configuration.Project.PackageName),
		BuildTool:             buildTool,
		TaskRunner:            taskRunners[configuration.Project.TaskRunner],
		DbType:                databaseTypes[configuration.Database.Engine],
		DbMigrationTool:       migrationTools[configuration.Database.MigrationTool],
		PersistenceType:       persistenceTypes[configuration.Database.PersistenceFramework],
		SpringCloudAWSSupport: slices.Contains(features, FeatureSpringCloudAWS),
		EmailSupport:          slices.Contains(features, FeatureEmail),
		RabbitMQSupport:       slices.Contains(features, FeatureRabbitMQ),
		RedisCachingSupport:   slices.Contains(features, FeatureRedisCaching),
		OpenTelemetrySupport:  slices.Contains(features, FeatureOpenTelemetry),
		K8sSupport:            slices.Contains(features, FeatureKubernetes),
		HTMXSupport:           configuration.Frontend.HTMX && isServerRendered(configuration.Frontend.Type),
		SpringAIModels:        aiModels,
	}
}

func isServerRendered(frontendType FrontendType) bool {
	return frontendType == FrontendThymeleaf || frontendType == FrontendJTE
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

func validateConfiguration(configuration GeneratorConfiguration) error {
	project := configuration.Project
	appName := strings.TrimSpace(project.AppName)
	if appName == "" {
		return errors.New("application name is required")
	}
	if appName == "." || appName == ".." || filepath.Base(appName) != appName || strings.ContainsAny(appName, `/\`) {
		return fmt.Errorf("invalid application name %q", project.AppName)
	}
	if strings.TrimSpace(project.AppVersion) == "" {
		return errors.New("application version is required")
	}
	if strings.TrimSpace(project.GroupID) == "" {
		return errors.New("group ID is required")
	}
	if strings.TrimSpace(project.ArtifactID) == "" {
		return errors.New("artifact ID is required")
	}
	if !javaPackagePattern.MatchString(strings.TrimSpace(project.PackageName)) {
		return fmt.Errorf("invalid Java package name %q", project.PackageName)
	}
	if !slices.Contains([]BuildTool{BuildToolMaven, BuildToolGradle}, project.BuildTool) {
		return fmt.Errorf("unsupported build tool %q", project.BuildTool)
	}
	if !slices.Contains([]TaskRunner{TaskRunnerMise, TaskRunnerTaskfile}, project.TaskRunner) {
		return fmt.Errorf("unsupported task runner %q", project.TaskRunner)
	}

	frontend := configuration.Frontend
	if !slices.Contains([]FrontendType{FrontendNone, FrontendThymeleaf, FrontendJTE, FrontendAngular, FrontendReact}, frontend.Type) {
		return fmt.Errorf("unsupported frontend type %q", frontend.Type)
	}
	if frontend.HTMX && !isServerRendered(frontend.Type) {
		return errors.New("HTMX support requires the Thymeleaf or JTE frontend")
	}

	database := configuration.Database
	if !slices.Contains([]DatabaseEngine{DatabasePostgreSQL, DatabaseMySQL, DatabaseMariaDB}, database.Engine) {
		return fmt.Errorf("unsupported database engine %q", database.Engine)
	}
	if !slices.Contains([]MigrationTool{MigrationFlyway, MigrationLiquibase}, database.MigrationTool) {
		return fmt.Errorf("unsupported migration tool %q", database.MigrationTool)
	}
	if !slices.Contains([]PersistenceFramework{PersistenceSpringDataJPA, PersistenceSpringJdbcClient, PersistenceJOOQ}, database.PersistenceFramework) {
		return fmt.Errorf("unsupported persistence framework %q", database.PersistenceFramework)
	}

	supportedFeatures := []Feature{FeatureEmail, FeatureSpringCloudAWS, FeatureRabbitMQ, FeatureRedisCaching, FeatureOpenTelemetry, FeatureKubernetes}
	for _, feature := range configuration.Features {
		if !slices.Contains(supportedFeatures, feature) {
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

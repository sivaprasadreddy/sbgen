package main

import (
	"archive/zip"
	"bytes"
	"embed"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"text/template"
)

const (
	springBootVersion     = "4.1.0"
	javaVersion           = "25"
	springModulithVersion = "2.1.0"
	springCloudAWSVersion = "4.0.0"
	spotlessGradleVersion = "8.9.0"
	spotlessMavenVersion  = "3.9.0"
	gradleGitPropsVersion = "4.0.1"
	palantirFormatVersion = "2.96.0"
	jacocoVersion         = "0.8.15"
	jacocoMinimumCoverage = "0.70"
	gitCommitIDVersion    = "9.0.2"
	springdocVersion      = "3.1.0"
	archunitVersion       = "1.4.2"
	flociVersion          = "2.14.0"
)

var (
	unsafeFilenameCharacters = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)
	javaPackagePattern       = regexp.MustCompile(`^[a-zA-Z_$][a-zA-Z0-9_$]*(\.[a-zA-Z_$][a-zA-Z0-9_$]*)*$`)
	projectTemplates         = parseProjectTemplates()
)

//go:embed all:generator-templates
var projectTemplateFiles embed.FS

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

type mavenDependency struct {
	GroupID    string
	ArtifactID string
	Scope      string
	Optional   bool
}

func (dependency mavenDependency) GradleConfiguration() string {
	switch dependency.Scope {
	case "runtime":
		return "runtimeOnly"
	case "test":
		return "testImplementation"
	default:
		return "implementation"
	}
}

func (dependency mavenDependency) Coordinate() string {
	return dependency.GroupID + ":" + dependency.ArtifactID
}

func (dependency mavenDependency) Version() string {
	switch dependency.Coordinate() {
	case "org.springdoc:springdoc-openapi-starter-webmvc-ui":
		return springdocVersion
	case "com.tngtech.archunit:archunit-junit5":
		return archunitVersion
	case "io.floci:spring-boot-testcontainers-floci":
		return flociVersion
	default:
		return ""
	}
}

type pomTemplateData struct {
	GroupID               string
	ArtifactID            string
	SpringBootVersion     string
	JavaVersion           string
	SpringModulithVersion string
	SpringCloudAWSVersion string
	HasModulith           bool
	HasCloudAWS           bool
	Dependencies          []mavenDependency
	SpotlessVersion       string
	PalantirFormatVersion string
	JacocoVersion         string
	GitCommitIDVersion    string
	JacocoMinimumCoverage string
}

type gradleTemplateData struct {
	GroupID               string
	ArtifactID            string
	SpringBootVersion     string
	JavaVersion           string
	SpringModulithVersion string
	SpringCloudAWSVersion string
	HasModulith           bool
	HasCloudAWS           bool
	Dependencies          []mavenDependency
	SpotlessVersion       string
	PalantirFormatVersion string
	JacocoVersion         string
	GitPropertiesVersion  string
	JacocoMinimumCoverage string
}

type applicationTemplateData struct {
	PackageName string
}

type projectMetadataTemplateData struct {
	GroupID             string
	ArtifactID          string
	PackageName         string
	BuildCommand        string
	RunCommand          string
	DatasourceURL       string
	DatasourceDriver    string
	ExternalDatasource  bool
	FlywayEnabled       bool
	LiquibaseEnabled    bool
	OAuthResourceServer bool
	MailEnabled         bool
	CloudAWSEnabled     bool
	DockerComposeH2     bool
	SpringdocEnabled    bool
}

type migrationTemplateData struct {
	IdentityClause string
}

type securityTemplateData struct {
	PackageName   string
	PublicPath    string
	ProtectedPath string
}

type composeTemplateData struct {
	DatabaseName string
	PostgreSQL   bool
	MySQL        bool
	MariaDB      bool
	Floci        bool
}

type testcontainersTemplateData struct {
	PackageName string
	PostgreSQL  bool
	MySQL       bool
	MariaDB     bool
	Floci       bool
}

type projectSupportTemplateData struct {
	ArtifactID    string
	JavaVersion   string
	Maven         bool
	Gradle        bool
	DockerCompose bool
}

type projectFile struct {
	Name    string
	Content string
	Mode    os.FileMode
}

type projectArchiveWriter func(*zip.Writer, string, generatorConfiguration) error

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

	return generateProjectZIP(downloadsDirectory, configuration, writeProjectArchive)
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

func generateProjectZIP(downloadsDirectory string, configuration generatorConfiguration, writeArchive projectArchiveWriter) (filePath string, err error) {
	safeArtifactID := sanitizeArtifactID(configuration.Project.ArtifactID)
	filePath, output, err := createAvailableZIP(downloadsDirectory, safeArtifactID)
	if err != nil {
		return "", err
	}
	generatedPath := filePath
	completed := false
	defer func() {
		if closeErr := output.Close(); err == nil && closeErr != nil {
			err = fmt.Errorf("close generated project: %w", closeErr)
		}
		if !completed || err != nil {
			_ = os.Remove(generatedPath)
		}
	}()

	archive := zip.NewWriter(output)
	if err = writeArchive(archive, safeArtifactID, configuration); err != nil {
		_ = archive.Close()
		return "", err
	}
	if err = archive.Close(); err != nil {
		return "", fmt.Errorf("finalize project archive: %w", err)
	}

	completed = true
	return filePath, nil
}

func writeProjectArchive(archive *zip.Writer, projectDirectory string, configuration generatorConfiguration) error {
	applicationJava, err := generateApplicationClass(configuration.Project.PackageName)
	if err != nil {
		return err
	}
	applicationTest, err := renderProjectTemplate("src/test/java/ApplicationTests.java.tmpl", applicationTemplateData{PackageName: configuration.Project.PackageName})
	if err != nil {
		return err
	}
	metadata := projectMetadataTemplateData{
		GroupID:     configuration.Project.GroupID,
		ArtifactID:  configuration.Project.ArtifactID,
		PackageName: configuration.Project.PackageName,
	}
	metadata.DatasourceURL, metadata.DatasourceDriver, metadata.ExternalDatasource = datasourceConfiguration(configuration.Database.Engine, sanitizeArtifactID(configuration.Project.ArtifactID))
	metadata.FlywayEnabled = configuration.Database.MigrationTool == "flyway"
	metadata.LiquibaseEnabled = configuration.Database.MigrationTool == "liquibase"
	metadata.OAuthResourceServer = configuration.Security.Mode == "oauth-resource-server"
	metadata.MailEnabled = contains(configuration.Features, "email")
	metadata.CloudAWSEnabled = contains(configuration.Features, "spring-cloud-aws")
	metadata.DockerComposeH2 = contains(configuration.Features, "docker-compose") && configuration.Database.Engine == "h2"
	metadata.SpringdocEnabled = configuration.Frontend.Type == "angular"
	if metadata.DockerComposeH2 && metadata.CloudAWSEnabled {
		metadata.DockerComposeH2 = false
	}
	if configuration.Project.BuildTool == "maven" {
		metadata.BuildCommand = "./mvnw test"
		metadata.RunCommand = "./mvnw spring-boot:run"
	} else {
		metadata.BuildCommand = "./gradlew test"
		metadata.RunCommand = "./gradlew bootRun"
	}
	applicationProperties, err := renderProjectTemplate("src/main/resources/application.properties.tmpl", metadata)
	if err != nil {
		return err
	}
	readme, err := renderProjectTemplate("README.md.tmpl", metadata)
	if err != nil {
		return err
	}
	gitignore, err := renderProjectTemplate(".gitignore.tmpl", nil)
	if err != nil {
		return err
	}
	testProperties, err := renderProjectTemplate("src/test/resources/application-test.properties.tmpl", nil)
	if err != nil {
		return err
	}

	packagePath := strings.ReplaceAll(configuration.Project.PackageName, ".", "/")
	files := []projectFile{
		{projectDirectory + "/src/main/java/" + packagePath + "/Application.java", applicationJava, 0o644},
		{projectDirectory + "/src/main/resources/application.properties", applicationProperties, 0o644},
		{projectDirectory + "/src/test/java/" + packagePath + "/ApplicationTests.java", applicationTest, 0o644},
		{projectDirectory + "/src/test/resources/application-test.properties", testProperties, 0o644},
		{projectDirectory + "/.gitignore", gitignore, 0o644},
		{projectDirectory + "/README.md", readme, 0o644},
	}
	migrationFiles, err := generateMigrationFiles(projectDirectory, configuration)
	if err != nil {
		return err
	}
	files = append(files, migrationFiles...)
	securityFiles, err := generateSecurityFiles(projectDirectory, packagePath, configuration)
	if err != nil {
		return err
	}
	files = append(files, securityFiles...)
	optionalFeatureFiles, err := generateOptionalFeatureFiles(projectDirectory, packagePath, configuration)
	if err != nil {
		return err
	}
	files = append(files, optionalFeatureFiles...)
	architectureTest, err := renderProjectTemplate("src/test/java/ArchUnitTests.java.tmpl", applicationTemplateData{PackageName: configuration.Project.PackageName})
	if err != nil {
		return err
	}
	files = append(files, projectFile{projectDirectory + "/src/test/java/" + packagePath + "/ArchUnitTests.java", architectureTest, 0o644})
	projectSupportFiles, err := generateProjectSupportFiles(projectDirectory, configuration)
	if err != nil {
		return err
	}
	files = append(files, projectSupportFiles...)
	buildFiles, err := generateBuildFiles(projectDirectory, configuration)
	if err != nil {
		return err
	}
	files = append(files, buildFiles...)
	for _, file := range files {
		if err := writeZIPFileMode(archive, file.Name, file.Content, file.Mode); err != nil {
			return err
		}
	}
	return nil
}

func generateProjectSupportFiles(projectDirectory string, configuration generatorConfiguration) ([]projectFile, error) {
	data := projectSupportTemplateData{
		ArtifactID:    configuration.Project.ArtifactID,
		JavaVersion:   javaVersion,
		Maven:         configuration.Project.BuildTool == "maven",
		Gradle:        configuration.Project.BuildTool != "maven",
		DockerCompose: contains(configuration.Features, "docker-compose") && (configuration.Database.Engine != "h2" || contains(configuration.Features, "spring-cloud-aws")),
	}
	templates := []struct {
		template string
		target   string
	}{
		{".github/workflows/ci.yml.tmpl", "/.github/workflows/ci.yml"},
		{"renovate.json.tmpl", "/renovate.json"},
		{"Taskfile.yml.tmpl", "/Taskfile.yml"},
		{".sdkmanrc.tmpl", "/.sdkmanrc"},
	}
	files := make([]projectFile, 0, len(templates))
	for _, file := range templates {
		content, err := renderProjectTemplate(file.template, data)
		if err != nil {
			return nil, err
		}
		files = append(files, projectFile{projectDirectory + file.target, content, 0o644})
	}
	return files, nil
}

func generateOptionalFeatureFiles(projectDirectory string, packagePath string, configuration generatorConfiguration) ([]projectFile, error) {
	var files []projectFile
	cloudAWS := contains(configuration.Features, "spring-cloud-aws")
	externalDatabase := configuration.Database.Engine != "h2"
	if contains(configuration.Features, "docker-compose") && (configuration.Database.Engine != "h2" || cloudAWS) {
		compose, err := renderProjectTemplate("compose.yaml.tmpl", composeTemplateData{
			DatabaseName: sanitizeArtifactID(configuration.Project.ArtifactID),
			PostgreSQL:   configuration.Database.Engine == "postgresql",
			MySQL:        configuration.Database.Engine == "mysql",
			MariaDB:      configuration.Database.Engine == "mariadb",
			Floci:        cloudAWS,
		})
		if err != nil {
			return nil, err
		}
		files = append(files, projectFile{projectDirectory + "/compose.yaml", compose, 0o644})
	}
	if externalDatabase || cloudAWS {
		testcontainersConfig, err := renderProjectTemplate("src/test/java/TestcontainersConfig.java.tmpl", testcontainersTemplateData{
			PackageName: configuration.Project.PackageName,
			PostgreSQL:  configuration.Database.Engine == "postgresql",
			MySQL:       configuration.Database.Engine == "mysql",
			MariaDB:     configuration.Database.Engine == "mariadb",
			Floci:       cloudAWS,
		})
		if err != nil {
			return nil, err
		}
		files = append(files, projectFile{projectDirectory + "/src/test/java/" + packagePath + "/TestcontainersConfig.java", testcontainersConfig, 0o644})
		if cloudAWS {
			flociSeed, err := renderProjectTemplate("src/test/resources/floci-init/ready.d/01-seed.sh.tmpl", nil)
			if err != nil {
				return nil, err
			}
			files = append(files, projectFile{projectDirectory + "/src/test/resources/floci-init/ready.d/01-seed.sh", flociSeed, 0o755})
		}
	}
	if contains(configuration.Features, "spring-modulith") {
		verificationTest, err := renderProjectTemplate("src/test/java/ModularityTests.java.tmpl", applicationTemplateData{PackageName: configuration.Project.PackageName})
		if err != nil {
			return nil, err
		}
		files = append(files, projectFile{projectDirectory + "/src/test/java/" + packagePath + "/ModularityTests.java", verificationTest, 0o644})
	}
	return files, nil
}

func generateSecurityFiles(projectDirectory string, packagePath string, configuration generatorConfiguration) ([]projectFile, error) {
	if configuration.Security.Mode == "none" {
		return nil, nil
	}
	data := securityTemplateData{
		PackageName:   configuration.Project.PackageName,
		PublicPath:    "/public",
		ProtectedPath: "/protected",
	}
	configurationTemplate := "src/main/java/config/FormLoginSecurityConfig.java.tmpl"
	testTemplate := "src/test/java/web/FormLoginSecurityEndpointsTests.java.tmpl"
	if configuration.Security.Mode == "oauth-resource-server" {
		data.PublicPath = "/api/public"
		data.ProtectedPath = "/api/protected"
		configurationTemplate = "src/main/java/config/OAuthResourceServerSecurityConfig.java.tmpl"
		testTemplate = "src/test/java/web/OAuthSecurityEndpointsTests.java.tmpl"
	}
	securityConfiguration, err := renderProjectTemplate(configurationTemplate, data)
	if err != nil {
		return nil, err
	}
	endpoints, err := renderProjectTemplate("src/main/java/web/SecurityEndpoints.java.tmpl", data)
	if err != nil {
		return nil, err
	}
	endpointTests, err := renderProjectTemplate(testTemplate, data)
	if err != nil {
		return nil, err
	}
	return []projectFile{
		{projectDirectory + "/src/main/java/" + packagePath + "/config/SecurityConfig.java", securityConfiguration, 0o644},
		{projectDirectory + "/src/main/java/" + packagePath + "/web/SecurityEndpoints.java", endpoints, 0o644},
		{projectDirectory + "/src/test/java/" + packagePath + "/web/SecurityEndpointsTests.java", endpointTests, 0o644},
	}, nil
}

func generateMigrationFiles(projectDirectory string, configuration generatorConfiguration) ([]projectFile, error) {
	switch configuration.Database.MigrationTool {
	case "flyway":
		identityClause := "GENERATED BY DEFAULT AS IDENTITY"
		if configuration.Database.Engine == "mysql" || configuration.Database.Engine == "mariadb" {
			identityClause = "AUTO_INCREMENT"
		}
		migration, err := renderProjectTemplate("src/main/resources/db/migration/V1__create_catalog_items.sql.tmpl", migrationTemplateData{IdentityClause: identityClause})
		if err != nil {
			return nil, err
		}
		return []projectFile{{projectDirectory + "/src/main/resources/db/migration/V1__create_catalog_items.sql", migration, 0o644}}, nil
	case "liquibase":
		changelog, err := renderProjectTemplate("src/main/resources/db/changelog/db.changelog-master.xml.tmpl", nil)
		if err != nil {
			return nil, err
		}
		return []projectFile{{projectDirectory + "/src/main/resources/db/changelog/db.changelog-master.xml", changelog, 0o644}}, nil
	default:
		return nil, nil
	}
}

func datasourceConfiguration(engine string, databaseName string) (url string, driver string, external bool) {
	switch engine {
	case "postgresql":
		return "${DATABASE_URL:jdbc:postgresql://localhost:5432/" + databaseName + "}", "org.postgresql.Driver", true
	case "mysql":
		return "${DATABASE_URL:jdbc:mysql://localhost:3306/" + databaseName + "}", "com.mysql.cj.jdbc.Driver", true
	case "mariadb":
		return "${DATABASE_URL:jdbc:mariadb://localhost:3306/" + databaseName + "}", "org.mariadb.jdbc.Driver", true
	case "h2":
		return "jdbc:h2:mem:" + databaseName + ";DB_CLOSE_DELAY=-1;DB_CLOSE_ON_EXIT=FALSE", "org.h2.Driver", false
	default:
		return "", "", false
	}
}

func generateBuildFiles(projectDirectory string, configuration generatorConfiguration) ([]projectFile, error) {
	render := func(name string, data any) (string, error) {
		return renderProjectTemplate(name, data)
	}
	if configuration.Project.BuildTool == "maven" {
		pomXML, err := generatePOM(configuration)
		if err != nil {
			return nil, err
		}
		wrapperFiles, err := copyWrapperFiles("generator-templates/maven-wrapper", projectDirectory)
		if err != nil {
			return nil, err
		}
		return append([]projectFile{{projectDirectory + "/pom.xml", pomXML, 0o644}}, wrapperFiles...), nil
	}

	gradleData := gradleTemplateData{
		GroupID:               configuration.Project.GroupID,
		ArtifactID:            configuration.Project.ArtifactID,
		SpringBootVersion:     springBootVersion,
		JavaVersion:           javaVersion,
		SpringModulithVersion: springModulithVersion,
		SpringCloudAWSVersion: springCloudAWSVersion,
		HasModulith:           contains(configuration.Features, "spring-modulith"),
		HasCloudAWS:           contains(configuration.Features, "spring-cloud-aws"),
		Dependencies:          projectDependencies(configuration),
		SpotlessVersion:       spotlessGradleVersion,
		PalantirFormatVersion: palantirFormatVersion,
		JacocoVersion:         jacocoVersion,
		GitPropertiesVersion:  gradleGitPropsVersion,
		JacocoMinimumCoverage: jacocoMinimumCoverage,
	}
	buildTemplate := "build.gradle.tmpl"
	settingsTemplate := "settings.gradle.tmpl"
	buildFile := "build.gradle"
	settingsFile := "settings.gradle"
	if configuration.Project.BuildTool == "gradle-kotlin" {
		buildTemplate = "build.gradle.kts.tmpl"
		settingsTemplate = "settings.gradle.kts.tmpl"
		buildFile = "build.gradle.kts"
		settingsFile = "settings.gradle.kts"
	}
	build, err := render(buildTemplate, gradleData)
	if err != nil {
		return nil, err
	}
	settings, err := render(settingsTemplate, gradleData)
	if err != nil {
		return nil, err
	}
	wrapperFiles, err := copyWrapperFiles("generator-templates/gradle-wrapper", projectDirectory)
	if err != nil {
		return nil, err
	}
	buildFiles := []projectFile{
		{projectDirectory + "/" + buildFile, build, 0o644},
		{projectDirectory + "/" + settingsFile, settings, 0o644},
	}
	return append(buildFiles, wrapperFiles...), nil
}

func copyWrapperFiles(sourceDirectory string, projectDirectory string) ([]projectFile, error) {
	var files []projectFile
	err := fs.WalkDir(projectTemplateFiles, sourceDirectory, func(sourcePath string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		content, err := projectTemplateFiles.ReadFile(sourcePath)
		if err != nil {
			return err
		}
		relativePath := strings.TrimPrefix(sourcePath, sourceDirectory+"/")
		mode := os.FileMode(0o644)
		if relativePath == "mvnw" || relativePath == "gradlew" {
			mode = 0o755
		}
		files = append(files, projectFile{
			Name:    projectDirectory + "/" + relativePath,
			Content: string(content),
			Mode:    mode,
		})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("copy wrapper files from %s: %w", sourceDirectory, err)
	}
	return files, nil
}

func generatePOM(configuration generatorConfiguration) (string, error) {
	dependencies := projectDependencies(configuration)

	hasModulith := contains(configuration.Features, "spring-modulith")
	hasCloudAWS := contains(configuration.Features, "spring-cloud-aws")
	return renderProjectTemplate("pom.xml.tmpl", pomTemplateData{
		GroupID:               configuration.Project.GroupID,
		ArtifactID:            configuration.Project.ArtifactID,
		SpringBootVersion:     springBootVersion,
		JavaVersion:           javaVersion,
		SpringModulithVersion: springModulithVersion,
		SpringCloudAWSVersion: springCloudAWSVersion,
		HasModulith:           hasModulith,
		HasCloudAWS:           hasCloudAWS,
		Dependencies:          dependencies,
		SpotlessVersion:       spotlessMavenVersion,
		PalantirFormatVersion: palantirFormatVersion,
		JacocoVersion:         jacocoVersion,
		GitCommitIDVersion:    gitCommitIDVersion,
		JacocoMinimumCoverage: jacocoMinimumCoverage,
	})
}

func projectDependencies(configuration generatorConfiguration) []mavenDependency {
	dependencies := []mavenDependency{
		{"org.springframework.boot", "spring-boot-starter-webmvc", "", false},
		{"org.springframework.boot", "spring-boot-starter-test", "test", false},
	}

	if configuration.Frontend.Type == "thymeleaf" {
		dependencies = append(dependencies, mavenDependency{"org.springframework.boot", "spring-boot-starter-thymeleaf", "", false})
	}
	if configuration.Frontend.Type == "angular" {
		dependencies = append(dependencies, mavenDependency{"org.springdoc", "springdoc-openapi-starter-webmvc-ui", "", false})
	}
	if configuration.Database.PersistenceFramework == "spring-data-jpa" {
		dependencies = append(dependencies, mavenDependency{"org.springframework.boot", "spring-boot-starter-data-jpa", "", false})
	} else {
		dependencies = append(dependencies, mavenDependency{"org.springframework.boot", "spring-boot-starter-jdbc", "", false})
	}

	switch configuration.Database.Engine {
	case "postgresql":
		dependencies = append(dependencies, mavenDependency{"org.postgresql", "postgresql", "runtime", false})
	case "mysql":
		dependencies = append(dependencies, mavenDependency{"com.mysql", "mysql-connector-j", "runtime", false})
	case "mariadb":
		dependencies = append(dependencies, mavenDependency{"org.mariadb.jdbc", "mariadb-java-client", "runtime", false})
	case "h2":
		dependencies = append(dependencies, mavenDependency{"com.h2database", "h2", "runtime", false})
	}

	switch configuration.Database.MigrationTool {
	case "flyway":
		dependencies = append(dependencies, mavenDependency{"org.springframework.boot", "spring-boot-starter-flyway", "", false})
		if configuration.Database.Engine == "postgresql" {
			dependencies = append(dependencies, mavenDependency{"org.flywaydb", "flyway-database-postgresql", "runtime", false})
		} else if configuration.Database.Engine == "mysql" || configuration.Database.Engine == "mariadb" {
			dependencies = append(dependencies, mavenDependency{"org.flywaydb", "flyway-mysql", "runtime", false})
		}
	case "liquibase":
		dependencies = append(dependencies, mavenDependency{"org.springframework.boot", "spring-boot-starter-liquibase", "", false})
	}

	if configuration.Security.Mode != "none" {
		dependencies = append(dependencies, mavenDependency{"org.springframework.boot", "spring-boot-starter-security", "", false})
		dependencies = append(dependencies, mavenDependency{"org.springframework.boot", "spring-boot-starter-webmvc-test", "test", false})
	}
	if configuration.Security.Mode == "oauth-resource-server" {
		dependencies = append(dependencies, mavenDependency{"org.springframework.boot", "spring-boot-starter-oauth2-resource-server", "", false})
	}

	if contains(configuration.Features, "email") {
		dependencies = append(dependencies, mavenDependency{"org.springframework.boot", "spring-boot-starter-mail", "", false})
	}
	if contains(configuration.Features, "docker-compose") {
		dependencies = append(dependencies, mavenDependency{"org.springframework.boot", "spring-boot-docker-compose", "runtime", true})
	}
	if contains(configuration.Features, "spring-modulith") {
		dependencies = append(dependencies, mavenDependency{"org.springframework.modulith", "spring-modulith-starter-core", "", false})
		dependencies = append(dependencies, mavenDependency{"org.springframework.modulith", "spring-modulith-starter-test", "test", false})
	}
	if contains(configuration.Features, "spring-cloud-aws") {
		dependencies = append(dependencies, mavenDependency{"io.awspring.cloud", "spring-cloud-aws-starter", "", false})
		dependencies = append(dependencies, mavenDependency{"io.floci", "spring-boot-testcontainers-floci", "test", false})
	}
	if configuration.Database.Engine != "h2" || contains(configuration.Features, "spring-cloud-aws") {
		dependencies = append(dependencies,
			mavenDependency{"org.springframework.boot", "spring-boot-testcontainers", "test", false},
			mavenDependency{"org.testcontainers", "testcontainers-junit-jupiter", "test", false},
		)
		switch configuration.Database.Engine {
		case "postgresql":
			dependencies = append(dependencies, mavenDependency{"org.testcontainers", "testcontainers-postgresql", "test", false})
		case "mysql":
			dependencies = append(dependencies, mavenDependency{"org.testcontainers", "testcontainers-mysql", "test", false})
		case "mariadb":
			dependencies = append(dependencies, mavenDependency{"org.testcontainers", "testcontainers-mariadb", "test", false})
		}
	}
	dependencies = append(dependencies, mavenDependency{"com.tngtech.archunit", "archunit-junit5", "test", false})

	return dependencies
}

func generateApplicationClass(packageName string) (string, error) {
	return renderProjectTemplate("src/main/java/Application.java.tmpl", applicationTemplateData{PackageName: packageName})
}

func parseProjectTemplates() *template.Template {
	templates := template.New("project-templates").Funcs(template.FuncMap{"xml": escapeXML, "quote": strconv.Quote})
	err := fs.WalkDir(projectTemplateFiles, "generator-templates", func(name string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || !strings.HasSuffix(name, ".tmpl") {
			return nil
		}
		content, err := projectTemplateFiles.ReadFile(name)
		if err != nil {
			return err
		}
		templateName := strings.TrimPrefix(name, "generator-templates/")
		_, err = templates.New(templateName).Parse(string(content))
		return err
	})
	return template.Must(templates, err)
}

func renderProjectTemplate(name string, data any) (string, error) {
	var rendered bytes.Buffer
	if err := projectTemplates.ExecuteTemplate(&rendered, name, data); err != nil {
		return "", fmt.Errorf("render %s: %w", name, err)
	}
	return rendered.String(), nil
}

func writeZIPFile(archive *zip.Writer, name string, content string) error {
	return writeZIPFileMode(archive, name, content, 0o644)
}

func writeZIPFileMode(archive *zip.Writer, name string, content string, mode os.FileMode) error {
	if err := validateZIPEntryName(name); err != nil {
		return err
	}
	header := &zip.FileHeader{Name: name, Method: zip.Deflate}
	header.SetMode(mode)
	entry, err := archive.CreateHeader(header)
	if err != nil {
		return fmt.Errorf("create %s in project archive: %w", name, err)
	}
	if _, err := io.Copy(entry, bytes.NewBufferString(content)); err != nil {
		return fmt.Errorf("write %s to project archive: %w", name, err)
	}
	return nil
}

func validateZIPEntryName(name string) error {
	cleanName := path.Clean(name)
	if name == "" || name != cleanName || cleanName == "." || path.IsAbs(cleanName) || cleanName == ".." || strings.HasPrefix(cleanName, "../") || strings.Contains(name, `\`) {
		return fmt.Errorf("unsafe project archive path %q", name)
	}
	return nil
}

func createAvailableZIP(directory string, artifactID string) (string, *os.File, error) {
	for suffix := range 1000 {
		name := artifactID + ".zip"
		if suffix > 0 {
			name = fmt.Sprintf("%s-%d.zip", artifactID, suffix+1)
		}
		filePath := filepath.Join(directory, name)
		file, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
		if err == nil {
			return filePath, file, nil
		}
		if !os.IsExist(err) {
			return "", nil, fmt.Errorf("create project archive: %w", err)
		}
	}
	return "", nil, fmt.Errorf("too many existing downloads for %s", artifactID)
}

func sanitizeArtifactID(artifactID string) string {
	safeArtifactID := strings.Trim(unsafeFilenameCharacters.ReplaceAllString(artifactID, "-"), "-._")
	if safeArtifactID == "" {
		return "spring-boot-project"
	}
	return safeArtifactID
}

func contains(values []string, target string) bool {
	return slices.Contains(values, target)
}

func escapeXML(value string) string {
	var escaped strings.Builder
	_ = xml.EscapeText(&escaped, []byte(value))
	return escaped.String()
}

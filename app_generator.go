package main

import (
	"archive/zip"
	"bytes"
	"embed"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"text/template"
)

const (
	springBootVersion     = "4.1.0"
	javaVersion           = "25"
	springModulithVersion = "2.1.0"
	springCloudAWSVersion = "4.0.0"
)

var (
	unsafeFilenameCharacters = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)
	javaPackagePattern       = regexp.MustCompile(`^[a-zA-Z_$][a-zA-Z0-9_$]*(\.[a-zA-Z_$][a-zA-Z0-9_$]*)*$`)
	projectTemplates         = template.Must(template.New("project-templates").Funcs(template.FuncMap{"xml": escapeXML}).ParseFS(projectTemplateFiles, "generator-templates/*.tmpl"))
)

//go:embed generator-templates/*.tmpl
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
		Mode          string `json:"mode"`
		SpringSession bool   `json:"springSession"`
	} `json:"security"`
	Features []string `json:"features"`
}

type mavenDependency struct {
	GroupID    string
	ArtifactID string
	Scope      string
	Optional   bool
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
}

type applicationTemplateData struct {
	PackageName string
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
	if configuration.Security.SpringSession && configuration.Security.Mode != "http-session" {
		return fmt.Errorf("Spring Session requires HTTP-session security")
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
	pomXML, err := generatePOM(configuration)
	if err != nil {
		return err
	}
	applicationJava, err := generateApplicationClass(configuration.Project.PackageName)
	if err != nil {
		return err
	}
	if err := writeZIPFile(archive, projectDirectory+"/pom.xml", pomXML); err != nil {
		return err
	}
	packagePath := strings.ReplaceAll(configuration.Project.PackageName, ".", "/")
	return writeZIPFile(archive, projectDirectory+"/src/main/java/"+packagePath+"/Application.java", applicationJava)
}

func generatePOM(configuration generatorConfiguration) (string, error) {
	dependencies := []mavenDependency{{"org.springframework.boot", "spring-boot-starter-webmvc", "", false}}

	if configuration.Frontend.Type == "thymeleaf" {
		dependencies = append(dependencies, mavenDependency{"org.springframework.boot", "spring-boot-starter-thymeleaf", "", false})
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
			dependencies = append(dependencies, mavenDependency{"org.flywaydb", "flyway-database-postgresql", "", false})
		} else if configuration.Database.Engine == "mysql" || configuration.Database.Engine == "mariadb" {
			dependencies = append(dependencies, mavenDependency{"org.flywaydb", "flyway-mysql", "", false})
		}
	case "liquibase":
		dependencies = append(dependencies, mavenDependency{"org.springframework.boot", "spring-boot-starter-liquibase", "", false})
	}

	if configuration.Security.Mode != "none" {
		dependencies = append(dependencies, mavenDependency{"org.springframework.boot", "spring-boot-starter-security", "", false})
	}
	if configuration.Security.Mode == "oauth-resource-server" {
		dependencies = append(dependencies, mavenDependency{"org.springframework.boot", "spring-boot-starter-oauth2-resource-server", "", false})
	}
	if configuration.Security.SpringSession {
		dependencies = append(dependencies, mavenDependency{"org.springframework.boot", "spring-boot-starter-session-jdbc", "", false})
	}

	hasModulith := contains(configuration.Features, "spring-modulith")
	hasCloudAWS := contains(configuration.Features, "spring-cloud-aws")
	if contains(configuration.Features, "email") {
		dependencies = append(dependencies, mavenDependency{"org.springframework.boot", "spring-boot-starter-mail", "", false})
	}
	if contains(configuration.Features, "docker-compose") {
		dependencies = append(dependencies, mavenDependency{"org.springframework.boot", "spring-boot-docker-compose", "runtime", true})
	}
	if hasModulith {
		dependencies = append(dependencies, mavenDependency{"org.springframework.modulith", "spring-modulith-starter-core", "", false})
	}
	if hasCloudAWS {
		dependencies = append(dependencies, mavenDependency{"io.awspring.cloud", "spring-cloud-aws-starter", "", false})
	}

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
	})
}

func generateApplicationClass(packageName string) (string, error) {
	return renderProjectTemplate("Application.java.tmpl", applicationTemplateData{PackageName: packageName})
}

func renderProjectTemplate(name string, data any) (string, error) {
	var rendered bytes.Buffer
	if err := projectTemplates.ExecuteTemplate(&rendered, name, data); err != nil {
		return "", fmt.Errorf("render %s: %w", name, err)
	}
	return rendered.String(), nil
}

func writeZIPFile(archive *zip.Writer, name string, content string) error {
	if err := validateZIPEntryName(name); err != nil {
		return err
	}
	header := &zip.FileHeader{Name: name, Method: zip.Deflate}
	header.SetMode(0o644)
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

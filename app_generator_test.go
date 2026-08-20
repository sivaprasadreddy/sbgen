package main

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"
)

type dependencyExpectation struct {
	present []string
	absent  []string
}

func minimalConfiguration() generatorConfiguration {
	var configuration generatorConfiguration
	configuration.Project.GroupID = "com.example"
	configuration.Project.ArtifactID = "demo"
	configuration.Project.PackageName = "com.example.demo"
	configuration.Project.BuildTool = "maven"
	configuration.Frontend.Type = "none"
	configuration.Database.Engine = "h2"
	configuration.Database.MigrationTool = "none"
	configuration.Database.PersistenceFramework = "spring-jdbc-client"
	configuration.Security.Mode = "none"
	configuration.Features = []string{}
	return configuration
}

func TestGenerateProjectCreatesMavenArchive(t *testing.T) {
	configuration := minimalConfiguration()
	configuration.Project.ArtifactID = "orders"
	configuration.Project.PackageName = "com.example.orders"
	configuration.Frontend.Type = "thymeleaf"
	cssFramework := "tailwind"
	configuration.Frontend.CSSFramework = &cssFramework
	configuration.Database.Engine = "postgresql"
	configuration.Database.MigrationTool = "flyway"
	configuration.Database.PersistenceFramework = "spring-data-jpa"
	configuration.Security.Mode = "http-session"
	configuration.Security.SpringSession = true
	configuration.Features = []string{"email", "docker-compose", "spring-modulith", "spring-cloud-aws"}

	filePath, archive := generateProjectArchive(t, configuration)
	if filepath.Ext(filePath) != ".zip" {
		t.Fatalf("GenerateProject() path = %q, want a zip file", filePath)
	}

	pom := readArchiveFile(t, archive, "orders/pom.xml")
	application := readArchiveFile(t, archive, "orders/src/main/java/com/example/orders/Application.java")

	assertBuildDependencies(t, "maven", pom, dependencyExpectation{present: []string{
		"org.springframework.boot:spring-boot-starter-webmvc",
		"org.springframework.boot:spring-boot-starter-thymeleaf",
		"org.springframework.boot:spring-boot-starter-data-jpa",
		"org.postgresql:postgresql",
		"org.springframework.boot:spring-boot-starter-flyway",
		"org.springframework.boot:spring-boot-starter-security",
		"org.springframework.boot:spring-boot-starter-session-jdbc",
		"org.springframework.boot:spring-boot-starter-mail",
		"org.springframework.boot:spring-boot-docker-compose",
		"org.springframework.modulith:spring-modulith-starter-core",
		"io.awspring.cloud:spring-cloud-aws-starter",
	}})
	if !strings.Contains(application, "package com.example.orders;") || !strings.Contains(application, "SpringApplication.run(Application.class, args)") {
		t.Errorf("Application.java does not contain the expected entrypoint:\n%s", application)
	}
}

func TestMinimalGeneratedApplication(t *testing.T) {
	configuration := minimalConfiguration()
	filePath, archive := generateProjectArchive(t, configuration)

	wantFiles := []string{
		"demo/pom.xml",
		"demo/src/main/java/com/example/demo/Application.java",
		"demo/src/main/resources/application.properties",
		"demo/src/test/java/com/example/demo/ApplicationTests.java",
		"demo/src/test/resources/application-test.properties",
		"demo/.gitignore",
		"demo/README.md",
		"demo/.mvn/wrapper/maven-wrapper.properties",
		"demo/mvnw",
		"demo/mvnw.cmd",
	}
	assertArchiveContains(t, archive, wantFiles...)
	for _, name := range listArchiveFiles(archive) {
		if !strings.HasPrefix(name, "demo/") {
			t.Errorf("archive entry %q is outside the project root", name)
		}
	}

	pom := readArchiveFile(t, archive, "demo/pom.xml")
	assertBuildDependencies(t, "maven", pom, dependencyExpectation{present: []string{
		"org.springframework.boot:spring-boot-starter-webmvc",
		"org.springframework.boot:spring-boot-starter-test",
		"org.springframework.boot:spring-boot-starter-jdbc",
		"com.h2database:h2",
	}})
	assertMavenBaseline(t, pom, configuration)

	application := readArchiveFile(t, archive, "demo/src/main/java/com/example/demo/Application.java")
	if !strings.Contains(application, "package com.example.demo;") || !strings.Contains(application, "@SpringBootApplication") {
		t.Errorf("generated Application.java is invalid:\n%s", application)
	}
	applicationTest := readArchiveFile(t, archive, "demo/src/test/java/com/example/demo/ApplicationTests.java")
	if !strings.Contains(applicationTest, "@SpringBootTest") || !strings.Contains(applicationTest, "void contextLoads()") {
		t.Errorf("generated ApplicationTests.java is not a context-load test:\n%s", applicationTest)
	}
	if properties := readArchiveFile(t, archive, "demo/src/main/resources/application.properties"); !strings.Contains(properties, "spring.application.name=demo") {
		t.Errorf("application.properties does not set the application name:\n%s", properties)
	}
	if gitignore := readArchiveFile(t, archive, "demo/.gitignore"); !strings.Contains(gitignore, "target/") {
		t.Errorf("generated .gitignore does not exclude Maven output:\n%s", gitignore)
	}
	if readme := readArchiveFile(t, archive, "demo/README.md"); !strings.Contains(readme, "./mvnw test") {
		t.Errorf("generated README does not document the build:\n%s", readme)
	}
	assertArchiveFileMode(t, archive, "demo/mvnw", 0o755)
	assertArchiveFileMode(t, archive, "demo/mvnw.cmd", 0o755)

	smokeTestGeneratedProject(t, filePath, configuration.Project.BuildTool)
}

func assertMavenBaseline(t *testing.T, pom string, configuration generatorConfiguration) {
	t.Helper()
	var project struct {
		Parent struct {
			GroupID    string `xml:"groupId"`
			ArtifactID string `xml:"artifactId"`
			Version    string `xml:"version"`
		} `xml:"parent"`
		GroupID    string `xml:"groupId"`
		ArtifactID string `xml:"artifactId"`
		Version    string `xml:"version"`
		Name       string `xml:"name"`
		Properties struct {
			JavaVersion string `xml:"java.version"`
		} `xml:"properties"`
		Dependencies []struct {
			GroupID    string `xml:"groupId"`
			ArtifactID string `xml:"artifactId"`
			Scope      string `xml:"scope"`
		} `xml:"dependencies>dependency"`
	}
	if err := xml.Unmarshal([]byte(pom), &project); err != nil {
		t.Fatalf("parse baseline pom.xml: %v", err)
	}
	if project.Parent.GroupID != "org.springframework.boot" || project.Parent.ArtifactID != "spring-boot-starter-parent" || project.Parent.Version != springBootVersion {
		t.Errorf("Spring Boot parent = %s:%s:%s, want org.springframework.boot:spring-boot-starter-parent:%s", project.Parent.GroupID, project.Parent.ArtifactID, project.Parent.Version, springBootVersion)
	}
	if project.GroupID != configuration.Project.GroupID || project.ArtifactID != configuration.Project.ArtifactID || project.Version != "0.0.1-SNAPSHOT" || project.Name != configuration.Project.ArtifactID {
		t.Errorf("project metadata = %s:%s:%s name=%q, want %s:%s:0.0.1-SNAPSHOT name=%q", project.GroupID, project.ArtifactID, project.Version, project.Name, configuration.Project.GroupID, configuration.Project.ArtifactID, configuration.Project.ArtifactID)
	}
	if project.Properties.JavaVersion != javaVersion {
		t.Errorf("Java version = %q, want %q", project.Properties.JavaVersion, javaVersion)
	}
	for _, dependency := range project.Dependencies {
		if dependency.GroupID == "org.springframework.boot" && dependency.ArtifactID == "spring-boot-starter-test" {
			if dependency.Scope != "test" {
				t.Errorf("spring-boot-starter-test scope = %q, want test", dependency.Scope)
			}
			return
		}
	}
	t.Error("spring-boot-starter-test dependency not found")
}

func assertArchiveFileMode(t *testing.T, archive *zip.ReadCloser, name string, want os.FileMode) {
	t.Helper()
	for _, file := range archive.File {
		if file.Name == name {
			if got := file.Mode().Perm(); got != want {
				t.Errorf("archive file %q mode = %o, want %o", name, got, want)
			}
			return
		}
	}
	t.Errorf("archive file %q not found", name)
}

func TestGenerateProjectRejectsInvalidPackage(t *testing.T) {
	payload := `{"project":{"groupId":"com.example","artifactId":"demo","packageName":"invalid-package"}}`
	if _, err := (&ConfigService{}).GenerateProject(payload); err == nil {
		t.Fatal("GenerateProject() error = nil, want invalid package error")
	}
}

func TestGenerateProjectRejectsMalformedJSON(t *testing.T) {
	if _, err := (&ConfigService{}).GenerateProject(`{"project":`); err == nil || !strings.Contains(err.Error(), "invalid project configuration") {
		t.Fatalf("GenerateProject() error = %v, want invalid project configuration error", err)
	}
}

func TestGenerateProjectValidatesConfiguration(t *testing.T) {
	stringPointer := func(value string) *string { return &value }
	tests := []struct {
		name        string
		mutate      func(*generatorConfiguration)
		wantMessage string
	}{
		{"missing group ID", func(c *generatorConfiguration) { c.Project.GroupID = "  " }, "group ID is required"},
		{"missing artifact ID", func(c *generatorConfiguration) { c.Project.ArtifactID = "  " }, "artifact ID is required"},
		{"unsupported build tool", func(c *generatorConfiguration) { c.Project.BuildTool = "ant" }, "unsupported build tool"},
		{"unsupported frontend", func(c *generatorConfiguration) { c.Frontend.Type = "react" }, "unsupported frontend type"},
		{"unsupported CSS framework", func(c *generatorConfiguration) { c.Frontend.CSSFramework = stringPointer("foundation") }, "unsupported CSS framework"},
		{"missing CSS framework", func(c *generatorConfiguration) { c.Frontend.Type = "thymeleaf" }, "CSS framework is required"},
		{"unsupported database", func(c *generatorConfiguration) { c.Database.Engine = "oracle" }, "unsupported database engine"},
		{"unsupported migration tool", func(c *generatorConfiguration) { c.Database.MigrationTool = "dbmate" }, "unsupported migration tool"},
		{"unsupported persistence framework", func(c *generatorConfiguration) { c.Database.PersistenceFramework = "mybatis" }, "unsupported persistence framework"},
		{"unsupported security mode", func(c *generatorConfiguration) { c.Security.Mode = "basic" }, "unsupported security mode"},
		{"unsupported feature", func(c *generatorConfiguration) { c.Features = []string{"graphql"} }, "unsupported feature"},
		{"HTTP session without Thymeleaf", func(c *generatorConfiguration) { c.Security.Mode = "http-session" }, "requires the Thymeleaf frontend"},
		{"OAuth without Angular", func(c *generatorConfiguration) { c.Security.Mode = "oauth-resource-server" }, "requires the Angular frontend"},
		{"Spring Session without HTTP session", func(c *generatorConfiguration) { c.Security.SpringSession = true }, "requires HTTP-session security"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			configuration := minimalConfiguration()
			test.mutate(&configuration)
			_, err := (&ConfigService{}).GenerateProject(configurationJSON(t, configuration))
			if err == nil || !strings.Contains(err.Error(), test.wantMessage) {
				t.Fatalf("GenerateProject() error = %v, want message containing %q", err, test.wantMessage)
			}
		})
	}
}

func TestGenerateProjectEscapesCoordinatesAndSanitizesArchiveName(t *testing.T) {
	configuration := minimalConfiguration()
	configuration.Project.GroupID = "com.example&partners"
	configuration.Project.ArtifactID = "orders & billing"

	filePath, archive := generateProjectArchive(t, configuration)
	if filepath.Base(filePath) != "orders-billing.zip" {
		t.Errorf("generated filename = %q, want orders-billing.zip", filepath.Base(filePath))
	}
	assertArchiveContains(t, archive,
		"orders-billing/pom.xml",
		"orders-billing/src/main/java/com/example/demo/Application.java",
	)

	pom := readArchiveFile(t, archive, "orders-billing/pom.xml")
	if !strings.Contains(pom, "<groupId>com.example&amp;partners</groupId>") || !strings.Contains(pom, "<artifactId>orders &amp; billing</artifactId>") {
		t.Errorf("pom.xml does not XML-escape project coordinates:\n%s", pom)
	}
	groupID, artifactID := readMavenCoordinates(t, pom)
	if groupID != configuration.Project.GroupID || artifactID != configuration.Project.ArtifactID {
		t.Errorf("Maven coordinates = %q:%q, want %q:%q", groupID, artifactID, configuration.Project.GroupID, configuration.Project.ArtifactID)
	}
}

func TestGenerateProjectUsesFallbackForEmptySanitizedArtifactID(t *testing.T) {
	configuration := minimalConfiguration()
	configuration.Project.ArtifactID = "..."

	filePath, archive := generateProjectArchive(t, configuration)
	if filepath.Base(filePath) != "spring-boot-project.zip" {
		t.Errorf("generated filename = %q, want spring-boot-project.zip", filepath.Base(filePath))
	}
	assertArchiveContains(t, archive, "spring-boot-project/pom.xml")
}

func TestGenerateProjectDoesNotOverwriteExistingDownload(t *testing.T) {
	homeDirectory := t.TempDir()
	t.Setenv("HOME", homeDirectory)
	configuration := minimalConfiguration()
	payload := configurationJSON(t, configuration)

	firstPath, err := (&ConfigService{}).GenerateProject(payload)
	if err != nil {
		t.Fatalf("first GenerateProject() error = %v", err)
	}
	secondPath, err := (&ConfigService{}).GenerateProject(payload)
	if err != nil {
		t.Fatalf("second GenerateProject() error = %v", err)
	}
	if filepath.Base(firstPath) != "demo.zip" || filepath.Base(secondPath) != "demo-2.zip" {
		t.Errorf("generated filenames = %q and %q, want demo.zip and demo-2.zip", filepath.Base(firstPath), filepath.Base(secondPath))
	}
	for _, filePath := range []string{firstPath, secondPath} {
		if info, err := os.Stat(filePath); err != nil || info.Size() == 0 {
			t.Errorf("generated archive %q is missing or empty: info=%v err=%v", filePath, info, err)
		}
	}
}

func TestGenerateProjectRemovesPartialArchiveAfterFailure(t *testing.T) {
	downloadsDirectory := t.TempDir()
	configuration := minimalConfiguration()
	wantErr := errors.New("injected archive failure")

	filePath, err := generateProjectZIP(downloadsDirectory, configuration, func(archive *zip.Writer, projectDirectory string, _ generatorConfiguration) error {
		if err := writeZIPFile(archive, projectDirectory+"/partial.txt", "partial"); err != nil {
			return err
		}
		return wantErr
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("generateProjectZIP() error = %v, want %v", err, wantErr)
	}
	if filePath != "" {
		t.Errorf("generateProjectZIP() path = %q after failure, want empty", filePath)
	}
	entries, readErr := os.ReadDir(downloadsDirectory)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if len(entries) != 0 {
		t.Errorf("downloads directory contains partial files after failure: %v", entries)
	}
}

func TestGeneratedArchiveUsesOnlySafeRelativePaths(t *testing.T) {
	configuration := minimalConfiguration()
	configuration.Project.ArtifactID = "../../unsafe project"
	_, archive := generateProjectArchive(t, configuration)
	for _, name := range listArchiveFiles(archive) {
		if err := validateZIPEntryName(name); err != nil {
			t.Errorf("generated unsafe archive entry %q: %v", name, err)
		}
	}

	unsafeNames := []string{"", ".", "../outside", "project/../../outside", "/absolute", `project\windows-path`, "project//file"}
	for _, name := range unsafeNames {
		t.Run(fmt.Sprintf("reject_%q", name), func(t *testing.T) {
			var output bytes.Buffer
			writer := zip.NewWriter(&output)
			err := writeZIPFile(writer, name, "unsafe")
			_ = writer.Close()
			if err == nil {
				t.Errorf("writeZIPFile(%q) error = nil, want unsafe path error", name)
			}
		})
	}
}

func configurationJSON(t *testing.T, configuration generatorConfiguration) string {
	t.Helper()
	payload, err := json.Marshal(configuration)
	if err != nil {
		t.Fatalf("marshal project configuration: %v", err)
	}
	return string(payload)
}

func readMavenCoordinates(t *testing.T, pom string) (string, string) {
	t.Helper()
	var project struct {
		GroupID    string `xml:"groupId"`
		ArtifactID string `xml:"artifactId"`
	}
	if err := xml.Unmarshal([]byte(pom), &project); err != nil {
		t.Fatalf("parse Maven coordinates: %v", err)
	}
	return project.GroupID, project.ArtifactID
}

func TestBuildDependencyAssertions(t *testing.T) {
	tests := []struct {
		name      string
		buildTool string
		buildFile string
	}{
		{
			name:      "Maven XML",
			buildTool: "maven",
			buildFile: `<project><dependencies><dependency><groupId>org.example</groupId><artifactId>present</artifactId></dependency></dependencies></project>`,
		},
		{
			name:      "Gradle Groovy",
			buildTool: "gradle-groovy",
			buildFile: "dependencies {\n  implementation 'org.example:present:1.0.0'\n}",
		},
		{
			name:      "Gradle Kotlin",
			buildTool: "gradle-kotlin",
			buildFile: "dependencies {\n  implementation(\"org.example:present:1.0.0\")\n}",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assertBuildDependencies(t, test.buildTool, test.buildFile, dependencyExpectation{
				present: []string{"org.example:present"},
				absent:  []string{"org.example:absent"},
			})
		})
	}
}

func generateProjectArchive(t *testing.T, configuration generatorConfiguration) (string, *zip.ReadCloser) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())

	filePath, err := (&ConfigService{}).GenerateProject(configurationJSON(t, configuration))
	if err != nil {
		t.Fatalf("GenerateProject() error = %v", err)
	}
	if _, err := os.Stat(filePath); err != nil {
		t.Fatalf("generated archive not found: %v", err)
	}

	archive, err := zip.OpenReader(filePath)
	if err != nil {
		t.Fatalf("open generated archive: %v", err)
	}
	t.Cleanup(func() {
		if err := archive.Close(); err != nil {
			t.Errorf("close generated archive: %v", err)
		}
	})
	return filePath, archive
}

func listArchiveFiles(archive *zip.ReadCloser) []string {
	files := make([]string, 0, len(archive.File))
	for _, file := range archive.File {
		files = append(files, file.Name)
	}
	return files
}

func assertArchiveContains(t *testing.T, archive *zip.ReadCloser, names ...string) {
	t.Helper()
	files := listArchiveFiles(archive)
	for _, name := range names {
		if !slices.Contains(files, name) {
			t.Errorf("archive does not contain %q; entries: %v", name, files)
		}
	}
}

func assertBuildDependencies(t *testing.T, buildTool string, buildFile string, expectation dependencyExpectation) {
	t.Helper()

	extractors := map[string]func(*testing.T, string) []string{
		"maven":         extractMavenDependencies,
		"gradle-groovy": extractGradleDependencies,
		"gradle-kotlin": extractGradleDependencies,
	}
	extract, ok := extractors[buildTool]
	if !ok {
		t.Fatalf("no dependency extractor for build tool %q", buildTool)
	}
	dependencies := extract(t, buildFile)

	for _, coordinate := range expectation.present {
		if !slices.Contains(dependencies, coordinate) {
			t.Errorf("%s build is missing dependency %q; dependencies: %v", buildTool, coordinate, dependencies)
		}
	}
	for _, coordinate := range expectation.absent {
		if slices.Contains(dependencies, coordinate) {
			t.Errorf("%s build unexpectedly contains dependency %q", buildTool, coordinate)
		}
	}
}

func extractMavenDependencies(t *testing.T, buildFile string) []string {
	t.Helper()
	var project struct {
		Dependencies []struct {
			GroupID    string `xml:"groupId"`
			ArtifactID string `xml:"artifactId"`
		} `xml:"dependencies>dependency"`
	}
	if err := xml.Unmarshal([]byte(buildFile), &project); err != nil {
		t.Fatalf("parse pom.xml: %v", err)
	}
	dependencies := make([]string, 0, len(project.Dependencies))
	for _, dependency := range project.Dependencies {
		dependencies = append(dependencies, dependency.GroupID+":"+dependency.ArtifactID)
	}
	return dependencies
}

var gradleDependencyPattern = regexp.MustCompile(`(?m)^\s*(?:implementation|runtimeOnly|compileOnly|testImplementation|testRuntimeOnly)\s*\(?\s*["']([^:"']+):([^:"']+)`)

func extractGradleDependencies(t *testing.T, buildFile string) []string {
	t.Helper()
	matches := gradleDependencyPattern.FindAllStringSubmatch(buildFile, -1)
	dependencies := make([]string, 0, len(matches))
	for _, match := range matches {
		dependencies = append(dependencies, match[1]+":"+match[2])
	}
	return dependencies
}

func smokeTestGeneratedProject(t *testing.T, archivePath string, buildTool string) {
	t.Helper()
	projectDirectory := extractGeneratedProject(t, archivePath)

	commands := map[string][]string{
		"maven":         {"./mvnw", "test"},
		"gradle-groovy": {"./gradlew", "test"},
		"gradle-kotlin": {"./gradlew", "test"},
	}
	commandLine, ok := commands[buildTool]
	if !ok {
		t.Fatalf("no smoke-test command for build tool %q", buildTool)
	}
	if runtime.GOOS == "windows" {
		commandLine[0] += ".cmd"
	}
	if buildTool == "maven" {
		mavenRepository := filepath.Join(os.TempDir(), "sbgen-maven-repository")
		if err := os.MkdirAll(mavenRepository, 0o755); err != nil {
			t.Fatalf("create isolated Maven repository: %v", err)
		}
		commandLine = append(commandLine[:1], append([]string{"-Dmaven.repo.local=" + mavenRepository}, commandLine[1:]...)...)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	command := exec.CommandContext(ctx, commandLine[0], commandLine[1:]...)
	command.Dir = projectDirectory
	output, err := command.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		t.Fatalf("generated project smoke test timed out after 5 minutes:\n%s", output)
	}
	if err != nil {
		t.Fatalf("generated project smoke test failed: %v\n%s", err, output)
	}
}

func extractGeneratedProject(t *testing.T, archivePath string) string {
	t.Helper()
	archive, err := zip.OpenReader(archivePath)
	if err != nil {
		t.Fatalf("open generated archive for extraction: %v", err)
	}
	defer archive.Close()

	destination := t.TempDir()
	var projectDirectory string
	for _, file := range archive.File {
		cleanName := filepath.Clean(file.Name)
		if cleanName == "." || filepath.IsAbs(cleanName) || cleanName == ".." || strings.HasPrefix(cleanName, ".."+string(filepath.Separator)) {
			t.Fatalf("unsafe archive entry %q", file.Name)
		}
		parts := strings.Split(filepath.ToSlash(cleanName), "/")
		if projectDirectory == "" {
			projectDirectory = filepath.Join(destination, parts[0])
		}
		target := filepath.Join(destination, cleanName)
		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(target, file.Mode().Perm()); err != nil {
				t.Fatalf("create extracted directory %q: %v", target, err)
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			t.Fatalf("create extracted parent directory: %v", err)
		}
		reader, err := file.Open()
		if err != nil {
			t.Fatalf("open archive entry %q: %v", file.Name, err)
		}
		output, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, file.Mode().Perm())
		if err != nil {
			reader.Close()
			t.Fatalf("create extracted file %q: %v", target, err)
		}
		_, copyErr := io.Copy(output, reader)
		closeOutputErr := output.Close()
		closeReaderErr := reader.Close()
		if copyErr != nil || closeOutputErr != nil || closeReaderErr != nil {
			t.Fatalf("extract archive entry %q: copy=%v close-output=%v close-input=%v", file.Name, copyErr, closeOutputErr, closeReaderErr)
		}
	}
	if projectDirectory == "" {
		t.Fatal("generated archive is empty")
	}
	return projectDirectory
}

func readArchiveFile(t *testing.T, archive *zip.ReadCloser, name string) string {
	t.Helper()
	for _, file := range archive.File {
		if file.Name != name {
			continue
		}
		reader, err := file.Open()
		if err != nil {
			t.Fatal(err)
		}
		defer reader.Close()
		content, err := io.ReadAll(reader)
		if err != nil {
			t.Fatal(err)
		}
		return string(content)
	}
	t.Fatalf("archive file %q not found", name)
	return ""
}

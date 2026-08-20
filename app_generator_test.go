package main

import (
	"archive/zip"
	"context"
	"encoding/json"
	"encoding/xml"
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

func TestGenerateProjectRejectsInvalidPackage(t *testing.T) {
	payload := `{"project":{"groupId":"com.example","artifactId":"demo","packageName":"invalid-package"}}`
	if _, err := (&ConfigService{}).GenerateProject(payload); err == nil {
		t.Fatal("GenerateProject() error = nil, want invalid package error")
	}
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

	payload, err := json.Marshal(configuration)
	if err != nil {
		t.Fatalf("marshal project configuration: %v", err)
	}
	filePath, err := (&ConfigService{}).GenerateProject(string(payload))
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

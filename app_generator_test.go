package main

import (
	"archive/zip"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateProjectCreatesMavenArchive(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	configuration := map[string]any{
		"project": map[string]any{
			"groupId": "com.example", "artifactId": "orders", "packageName": "com.example.orders", "buildTool": "maven",
		},
		"frontend": map[string]any{"type": "thymeleaf", "cssFramework": "tailwind"},
		"database": map[string]any{
			"engine": "postgresql", "migrationTool": "flyway", "persistenceFramework": "spring-data-jpa",
		},
		"security": map[string]any{"mode": "http-session", "springSession": true},
		"features": []string{"email", "docker-compose", "spring-modulith", "spring-cloud-aws"},
	}
	payload, err := json.Marshal(configuration)
	if err != nil {
		t.Fatal(err)
	}

	filePath, err := (&ConfigService{}).GenerateProject(string(payload))
	if err != nil {
		t.Fatalf("GenerateProject() error = %v", err)
	}
	if filepath.Ext(filePath) != ".zip" {
		t.Fatalf("GenerateProject() path = %q, want a zip file", filePath)
	}
	if _, err := os.Stat(filePath); err != nil {
		t.Fatalf("generated archive not found: %v", err)
	}

	archive, err := zip.OpenReader(filePath)
	if err != nil {
		t.Fatal(err)
	}
	defer archive.Close()

	pom := readArchiveFile(t, archive, "orders/pom.xml")
	application := readArchiveFile(t, archive, "orders/src/main/java/com/example/orders/Application.java")

	for _, artifactID := range []string{
		"spring-boot-starter-webmvc",
		"spring-boot-starter-thymeleaf",
		"spring-boot-starter-data-jpa",
		"postgresql",
		"spring-boot-starter-flyway",
		"spring-boot-starter-security",
		"spring-boot-starter-session-jdbc",
		"spring-boot-starter-mail",
		"spring-boot-docker-compose",
		"spring-modulith-starter-core",
		"spring-cloud-aws-starter",
	} {
		if !strings.Contains(pom, "<artifactId>"+artifactID+"</artifactId>") {
			t.Errorf("pom.xml does not contain dependency %s", artifactID)
		}
	}
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

package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestHTMXSupportIsMappedToProjectConfig(t *testing.T) {
	configuration := validGeneratorConfiguration(t)
	configuration.Frontend.HTMXSupport = true

	projectConfig := toProjectConfig(configuration)
	if !projectConfig.HTMXSupport {
		t.Fatal("toProjectConfig() did not enable HTMX support")
	}
}

func TestValidateConfigurationRejectsHTMXWithoutThymeleaf(t *testing.T) {
	configuration := validGeneratorConfiguration(t)
	configuration.Frontend.Type = "angular"
	configuration.Frontend.HTMXSupport = true

	if err := validateConfiguration(configuration); err == nil {
		t.Fatal("validateConfiguration() accepted HTMX without Thymeleaf")
	}
}

func validGeneratorConfiguration(t *testing.T) generatorConfiguration {
	t.Helper()
	configurationJSON := `{
		"project":{"groupId":"dev.sivalabs","artifactId":"demo","packageName":"dev.sivalabs.demo","buildTool":"maven"},
		"frontend":{"type":"thymeleaf","cssFramework":"tailwind","htmxSupport":false},
		"database":{"engine":"postgresql","migrationTool":"flyway","persistenceFramework":"spring-data-jpa"},
		"security":{"mode":"http-session"},
		"features":[]
	}`
	var configuration generatorConfiguration
	if err := json.Unmarshal([]byte(configurationJSON), &configuration); err != nil {
		t.Fatal(err)
	}
	return configuration
}

func TestCreateUniqueZipFileUsesNextAvailableSuffix(t *testing.T) {
	directory := t.TempDir()
	for _, name := range []string{"demo.zip", "demo-1.zip"} {
		if err := os.WriteFile(filepath.Join(directory, name), []byte("existing"), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	file, filePath, err := createUniqueZipFile(directory, "demo")
	if err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}

	want := filepath.Join(directory, "demo-2.zip")
	if filePath != want {
		t.Fatalf("createUniqueZipFile() path = %q, want %q", filePath, want)
	}
	for _, name := range []string{"demo.zip", "demo-1.zip"} {
		data, err := os.ReadFile(filepath.Join(directory, name))
		if err != nil {
			t.Fatal(err)
		}
		if string(data) != "existing" {
			t.Fatalf("existing archive %q was overwritten", name)
		}
	}
}

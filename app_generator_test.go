package main

import (
	"archive/zip"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/sivaprasadreddy/progen/generators/springboot"
)

func validConfiguration() GeneratorConfiguration {
	return GeneratorConfiguration{
		Project: ProjectSettings{
			AppName:     "demo",
			AppVersion:  "1.0.0",
			GroupID:     "dev.sivalabs",
			ArtifactID:  "demo",
			PackageName: "dev.sivalabs.demo",
			BuildTool:   BuildToolMaven,
			TaskRunner:  TaskRunnerMise,
		},
		Frontend: FrontendSettings{Type: FrontendNone},
		Database: DatabaseSettings{
			Engine:               DatabasePostgreSQL,
			MigrationTool:        MigrationFlyway,
			PersistenceFramework: PersistenceSpringDataJPA,
		},
	}
}

func TestToProjectConfigMapsJTEWithHTMXAndAIModels(t *testing.T) {
	c := validConfiguration()
	c.Project.BuildTool = BuildToolGradle
	c.Frontend.Type = FrontendJTE
	c.Frontend.HTMX = true
	c.Features = []Feature{FeatureEmail, FeatureKubernetes}
	c.AIModels = []AIModel{AIModelOpenAI, AIModelOllama}

	if err := validateConfiguration(c); err != nil {
		t.Fatalf("expected valid configuration, got %v", err)
	}
	pc := toProjectConfig(c)

	if pc.Frontend != springboot.FrontendJTE || !pc.JTESupport() || !pc.HTMXSupport {
		t.Errorf("unexpected frontend mapping: %+v", pc)
	}
	if pc.BuildTool != springboot.Gradle {
		t.Errorf("expected Gradle, got %s", pc.BuildTool)
	}
	if !pc.EmailSupport || !pc.K8sSupport || pc.RabbitMQSupport {
		t.Errorf("unexpected feature flags: %+v", pc)
	}
	if !slices.Contains(pc.SpringAIModels, springboot.AIModelOpenAI) || !slices.Contains(pc.SpringAIModels, springboot.AIModelOllama) {
		t.Errorf("unexpected AI models: %v", pc.SpringAIModels)
	}
	if pc.AppName != "demo" || pc.AppVersion != "1.0.0" {
		t.Errorf("unexpected app identity: %s %s", pc.AppName, pc.AppVersion)
	}
}

func TestToProjectConfigMapsReactWithTaskfile(t *testing.T) {
	c := validConfiguration()
	c.Project.TaskRunner = TaskRunnerTaskfile
	c.Frontend.Type = FrontendReact

	if err := validateConfiguration(c); err != nil {
		t.Fatalf("expected valid configuration, got %v", err)
	}
	pc := toProjectConfig(c)

	if pc.Frontend != springboot.FrontendReact || !pc.ReactSupport() || !pc.SPASupport() || pc.HTMXSupport {
		t.Errorf("unexpected frontend mapping: %+v", pc)
	}
	if pc.TaskRunner != springboot.TaskRunnerTaskfile || !pc.TaskfileSupport() || pc.MiseSupport() {
		t.Errorf("expected Taskfile task runner, got %q", pc.TaskRunner)
	}
}

func TestToProjectConfigMapsMiseTaskRunner(t *testing.T) {
	pc := toProjectConfig(validConfiguration())
	if pc.TaskRunner != springboot.TaskRunnerMise || !pc.MiseSupport() {
		t.Errorf("expected Mise task runner, got %q", pc.TaskRunner)
	}
}

func TestValidateConfigurationRejectsInvalidCombinations(t *testing.T) {
	cases := map[string]func(*GeneratorConfiguration){
		"missing app name":    func(c *GeneratorConfiguration) { c.Project.AppName = " " },
		"path in app name":    func(c *GeneratorConfiguration) { c.Project.AppName = "../demo" },
		"missing version":     func(c *GeneratorConfiguration) { c.Project.AppVersion = "" },
		"invalid package":     func(c *GeneratorConfiguration) { c.Project.PackageName = "1dev.sivalabs" },
		"unknown build tool":  func(c *GeneratorConfiguration) { c.Project.BuildTool = "gradle-groovy" },
		"missing task runner": func(c *GeneratorConfiguration) { c.Project.TaskRunner = "" },
		"unknown task runner": func(c *GeneratorConfiguration) { c.Project.TaskRunner = "make" },
		"htmx with react":     func(c *GeneratorConfiguration) { c.Frontend.Type = FrontendReact; c.Frontend.HTMX = true },
		"htmx without views":  func(c *GeneratorConfiguration) { c.Frontend.HTMX = true },
		"unknown ai provider": func(c *GeneratorConfiguration) { c.AIModels = []AIModel{"gemini"} },
		"unknown feature":     func(c *GeneratorConfiguration) { c.Features = []Feature{"htmx"} },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			c := validConfiguration()
			mutate(&c)
			if err := validateConfiguration(c); err == nil {
				t.Errorf("expected validation error")
			}
		})
	}
}

func TestGenerateProjectWritesZipToDownloads(t *testing.T) {
	if testing.Short() {
		t.Skip("generates a full project")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	path, err := (&AppGeneratorService{}).GenerateProject(validConfiguration())
	if err != nil {
		t.Fatalf("generate project: %v", err)
	}
	if want := filepath.Join(home, "Downloads", "demo.zip"); path != want {
		t.Errorf("expected archive at %s, got %s", want, path)
	}
	archive, err := zip.OpenReader(path)
	if err != nil {
		t.Fatalf("open archive: %v", err)
	}
	defer archive.Close()
	if !slices.ContainsFunc(archive.File, func(f *zip.File) bool { return f.Name == "demo/pom.xml" }) {
		t.Errorf("archive has no demo/pom.xml")
	}

	// A second generation must not overwrite the first archive.
	second, err := (&AppGeneratorService{}).GenerateProject(validConfiguration())
	if err != nil {
		t.Fatalf("generate second project: %v", err)
	}
	if filepath.Base(second) != "demo-1.zip" {
		t.Errorf("expected demo-1.zip, got %s", second)
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("first archive is gone: %v", err)
	}
}

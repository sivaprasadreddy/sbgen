package main

import (
	"slices"
	"testing"

	"github.com/sivaprasadreddy/progen/generators/springboot"
)

func validConfiguration() generatorConfiguration {
	var c generatorConfiguration
	c.Project.AppName = "demo"
	c.Project.AppVersion = "1.0.0"
	c.Project.GroupID = "dev.sivalabs"
	c.Project.ArtifactID = "demo"
	c.Project.PackageName = "dev.sivalabs.demo"
	c.Project.BuildTool = "maven"
	c.Frontend.Type = "none"
	c.Database.Engine = "postgresql"
	c.Database.MigrationTool = "flyway"
	c.Database.PersistenceFramework = "spring-data-jpa"
	return c
}

func TestToProjectConfigMapsJTEWithHTMXAndAIModels(t *testing.T) {
	c := validConfiguration()
	c.Project.BuildTool = "gradle"
	c.Frontend.Type = "jte"
	c.Frontend.HTMX = true
	c.Features = []string{"email", "kubernetes"}
	c.AIModels = []string{"openai", "ollama"}

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

func TestValidateConfigurationRejectsInvalidCombinations(t *testing.T) {
	cases := map[string]func(*generatorConfiguration){
		"missing app name":    func(c *generatorConfiguration) { c.Project.AppName = " " },
		"path in app name":    func(c *generatorConfiguration) { c.Project.AppName = "../demo" },
		"missing version":     func(c *generatorConfiguration) { c.Project.AppVersion = "" },
		"unknown build tool":  func(c *generatorConfiguration) { c.Project.BuildTool = "gradle-groovy" },
		"htmx without views":  func(c *generatorConfiguration) { c.Frontend.HTMX = true },
		"unknown ai provider": func(c *generatorConfiguration) { c.AIModels = []string{"gemini"} },
		"unknown feature":     func(c *generatorConfiguration) { c.Features = []string{"htmx"} },
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

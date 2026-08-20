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
	"reflect"
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
	assertArchiveFileMode(t, archive, "demo/mvnw.cmd", 0o644)

	smokeTestGeneratedProject(t, filePath, configuration.Project.BuildTool)
}

func TestGeneratedBuildTools(t *testing.T) {
	tests := []struct {
		name          string
		buildTool     string
		buildFile     string
		settingsFile  string
		wrapperFiles  []string
		excludedFiles []string
	}{
		{
			name:          "Maven",
			buildTool:     "maven",
			buildFile:     "pom.xml",
			wrapperFiles:  []string{".mvn/wrapper/maven-wrapper.properties", "mvnw", "mvnw.cmd"},
			excludedFiles: []string{"build.gradle", "build.gradle.kts", "settings.gradle", "settings.gradle.kts", "gradlew", "gradlew.bat"},
		},
		{
			name:          "Gradle Groovy",
			buildTool:     "gradle-groovy",
			buildFile:     "build.gradle",
			settingsFile:  "settings.gradle",
			wrapperFiles:  []string{"gradle/wrapper/gradle-wrapper.jar", "gradle/wrapper/gradle-wrapper.properties", "gradlew", "gradlew.bat"},
			excludedFiles: []string{"pom.xml", "build.gradle.kts", "settings.gradle.kts", "mvnw", "mvnw.cmd"},
		},
		{
			name:          "Gradle Kotlin",
			buildTool:     "gradle-kotlin",
			buildFile:     "build.gradle.kts",
			settingsFile:  "settings.gradle.kts",
			wrapperFiles:  []string{"gradle/wrapper/gradle-wrapper.jar", "gradle/wrapper/gradle-wrapper.properties", "gradlew", "gradlew.bat"},
			excludedFiles: []string{"pom.xml", "build.gradle", "settings.gradle", "mvnw", "mvnw.cmd"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			configuration := minimalConfiguration()
			configuration.Project.BuildTool = test.buildTool
			filePath, archive := generateProjectArchive(t, configuration)
			root := configuration.Project.ArtifactID + "/"

			wantFiles := []string{root + test.buildFile}
			if test.settingsFile != "" {
				wantFiles = append(wantFiles, root+test.settingsFile)
			}
			for _, wrapper := range test.wrapperFiles {
				wantFiles = append(wantFiles, root+wrapper)
			}
			assertArchiveContains(t, archive, wantFiles...)
			assertArchiveExcludes(t, archive, prefixPaths(root, test.excludedFiles)...)

			buildFile := readArchiveFile(t, archive, root+test.buildFile)
			assertBuildDependencies(t, test.buildTool, buildFile, dependencyExpectation{present: []string{
				"org.springframework.boot:spring-boot-starter-webmvc",
				"org.springframework.boot:spring-boot-starter-test",
				"org.springframework.boot:spring-boot-starter-jdbc",
				"com.h2database:h2",
			}})
			if test.buildTool == "maven" {
				for _, plugin := range []string{"spotless-maven-plugin", "jacoco-maven-plugin", "git-commit-id-maven-plugin"} {
					if strings.Count(buildFile, "<artifactId>"+plugin+"</artifactId>") != 1 {
						t.Errorf("Maven build does not configure %s exactly once:\n%s", plugin, buildFile)
					}
				}
				for _, version := range []string{spotlessMavenVersion, palantirFormatVersion, jacocoVersion, gitCommitIDVersion} {
					if !strings.Contains(buildFile, version) {
						t.Errorf("Maven build does not contain plugin version %s:\n%s", version, buildFile)
					}
				}
				assertMavenBaseline(t, buildFile, configuration)
				assertArchiveFileMode(t, archive, root+"mvnw", 0o755)
			} else {
				for _, plugin := range []string{"com.diffplug.spotless", "com.gorylenko.gradle-git-properties", "jacocoTestReport", "palantirJavaFormat"} {
					if !strings.Contains(buildFile, plugin) {
						t.Errorf("Gradle build does not configure %s:\n%s", plugin, buildFile)
					}
				}
				for _, version := range []string{spotlessGradleVersion, palantirFormatVersion, jacocoVersion, gradleGitPropsVersion} {
					if !strings.Contains(buildFile, version) {
						t.Errorf("Gradle build does not contain plugin version %s:\n%s", version, buildFile)
					}
				}
				if !strings.Contains(buildFile, springBootVersion) || !strings.Contains(buildFile, "JavaLanguageVersion.of("+javaVersion+")") {
					t.Errorf("Gradle build does not configure Spring Boot %s and Java %s:\n%s", springBootVersion, javaVersion, buildFile)
				}
				settings := readArchiveFile(t, archive, root+test.settingsFile)
				if !strings.Contains(settings, `rootProject.name = "demo"`) {
					t.Errorf("Gradle settings do not configure the project name:\n%s", settings)
				}
				assertArchiveFileMode(t, archive, root+"gradlew", 0o755)
			}
			smokeTestGeneratedProject(t, filePath, test.buildTool)
		})
	}
}

func TestGeneratedProjectSupportFiles(t *testing.T) {
	tests := []struct {
		buildTool     string
		wrapper       string
		cache         string
		sdk           string
		absentWrapper string
		absentSDK     string
	}{
		{"maven", "./mvnw", "cache: maven", "maven=3.9.16", "./gradlew", "gradle="},
		{"gradle-groovy", "./gradlew", "cache: gradle", "gradle=9.7.0", "./mvnw", "maven="},
		{"gradle-kotlin", "./gradlew", "cache: gradle", "gradle=9.7.0", "./mvnw", "maven="},
	}
	for _, test := range tests {
		t.Run(test.buildTool, func(t *testing.T) {
			configuration := minimalConfiguration()
			configuration.Project.BuildTool = test.buildTool
			_, archive := generateProjectArchive(t, configuration)
			assertArchiveContains(t, archive,
				"demo/.github/workflows/ci.yml",
				"demo/renovate.json",
				"demo/Taskfile.yml",
				"demo/.sdkmanrc",
			)

			workflow := readArchiveFile(t, archive, "demo/.github/workflows/ci.yml")
			if !strings.Contains(workflow, test.cache) || !strings.Contains(workflow, test.wrapper) || strings.Contains(workflow, test.absentWrapper) {
				t.Errorf("GitHub Actions workflow does not match %s:\n%s", test.buildTool, workflow)
			}
			taskfile := readArchiveFile(t, archive, "demo/Taskfile.yml")
			if !strings.Contains(taskfile, test.wrapper) || strings.Contains(taskfile, test.absentWrapper) || strings.Contains(taskfile, "docker compose") {
				t.Errorf("Taskfile does not match %s without Docker Compose:\n%s", test.buildTool, taskfile)
			}
			sdkmanrc := readArchiveFile(t, archive, "demo/.sdkmanrc")
			if !strings.Contains(sdkmanrc, "java="+javaVersion+"-tem") || !strings.Contains(sdkmanrc, test.sdk) || strings.Contains(sdkmanrc, test.absentSDK) {
				t.Errorf(".sdkmanrc does not match %s:\n%s", test.buildTool, sdkmanrc)
			}

			var renovate map[string]any
			if err := json.Unmarshal([]byte(readArchiveFile(t, archive, "demo/renovate.json")), &renovate); err != nil {
				t.Fatalf("generated renovate.json is invalid: %v", err)
			}
			if renovate["$schema"] != "https://docs.renovatebot.com/renovate-schema.json" {
				t.Errorf("generated renovate.json has unexpected schema: %v", renovate["$schema"])
			}
		})
	}

	t.Run("Docker Compose tasks", func(t *testing.T) {
		configuration := minimalConfiguration()
		configuration.Database.Engine = "postgresql"
		configuration.Features = []string{"docker-compose"}
		_, archive := generateProjectArchive(t, configuration)
		taskfile := readArchiveFile(t, archive, "demo/Taskfile.yml")
		for _, command := range []string{"docker compose -f compose.yaml up", "docker compose -f compose.yaml stop", "docker compose -f compose.yaml rm -f"} {
			if !strings.Contains(taskfile, command) {
				t.Errorf("Docker Compose Taskfile is missing %q:\n%s", command, taskfile)
			}
		}
	})
}

func TestGeneratedBuildsEnforceSeventyPercentLineCoverage(t *testing.T) {
	for _, buildTool := range []string{"maven", "gradle-groovy", "gradle-kotlin"} {
		t.Run(buildTool, func(t *testing.T) {
			configuration := minimalConfiguration()
			configuration.Project.BuildTool = buildTool
			filePath, archive := generateProjectArchive(t, configuration)
			buildFileName := map[string]string{"maven": "pom.xml", "gradle-groovy": "build.gradle", "gradle-kotlin": "build.gradle.kts"}[buildTool]
			buildFile := readArchiveFile(t, archive, "demo/"+buildFileName)
			for _, snippet := range []string{"COVEREDRATIO", jacocoMinimumCoverage, "jacocoTestCoverageVerification"} {
				if buildTool == "maven" && snippet == "jacocoTestCoverageVerification" {
					snippet = "<goal>check</goal>"
				}
				if !strings.Contains(buildFile, snippet) {
					t.Errorf("%s coverage configuration is missing %q:\n%s", buildTool, snippet, buildFile)
				}
			}
			if buildTool == "maven" {
				runGeneratedProjectCommand(t, filePath, buildTool, "verify")
			} else {
				runGeneratedProjectCommand(t, filePath, buildTool, "check")
			}
		})
	}
}

func TestWrapperDirectoriesAreCopiedVerbatim(t *testing.T) {
	tests := []struct {
		buildTool       string
		sourceDirectory string
		files           []string
	}{
		{
			buildTool:       "maven",
			sourceDirectory: "generator-templates/maven-wrapper",
			files:           []string{".mvn/wrapper/maven-wrapper.properties", "mvnw", "mvnw.cmd"},
		},
		{
			buildTool:       "gradle-groovy",
			sourceDirectory: "generator-templates/gradle-wrapper",
			files:           []string{"gradle/wrapper/gradle-wrapper.jar", "gradle/wrapper/gradle-wrapper.properties", "gradlew", "gradlew.bat"},
		},
	}

	for _, test := range tests {
		t.Run(test.buildTool, func(t *testing.T) {
			configuration := minimalConfiguration()
			configuration.Project.BuildTool = test.buildTool
			_, archive := generateProjectArchive(t, configuration)
			for _, relativePath := range test.files {
				want, err := projectTemplateFiles.ReadFile(test.sourceDirectory + "/" + relativePath)
				if err != nil {
					t.Fatalf("read embedded wrapper file %q: %v", relativePath, err)
				}
				got := []byte(readArchiveFile(t, archive, "demo/"+relativePath))
				if !bytes.Equal(got, want) {
					t.Errorf("generated wrapper file %q differs from its source asset", relativePath)
				}
			}
		})
	}
}

func TestBuildToolsHaveEquivalentDependencySemantics(t *testing.T) {
	configuration := minimalConfiguration()
	configuration.Frontend.Type = "thymeleaf"
	cssFramework := "tailwind"
	configuration.Frontend.CSSFramework = &cssFramework
	configuration.Database.Engine = "postgresql"
	configuration.Database.MigrationTool = "flyway"
	configuration.Database.PersistenceFramework = "spring-data-jpa"
	configuration.Security.Mode = "http-session"
	configuration.Features = []string{"email", "docker-compose", "spring-modulith", "spring-cloud-aws"}

	mavenConfiguration := configuration
	mavenConfiguration.Project.BuildTool = "maven"
	_, mavenArchive := generateProjectArchive(t, mavenConfiguration)
	wantScopes := extractDependencyScopes(t, "maven", readArchiveFile(t, mavenArchive, "demo/pom.xml"))

	for _, buildTool := range []string{"maven", "gradle-groovy", "gradle-kotlin"} {
		t.Run(buildTool, func(t *testing.T) {
			buildConfiguration := configuration
			buildConfiguration.Project.BuildTool = buildTool
			_, archive := generateProjectArchive(t, buildConfiguration)
			buildFileName := map[string]string{"maven": "pom.xml", "gradle-groovy": "build.gradle", "gradle-kotlin": "build.gradle.kts"}[buildTool]
			buildFile := readArchiveFile(t, archive, "demo/"+buildFileName)

			gotScopes := extractDependencyScopes(t, buildTool, buildFile)
			if !reflect.DeepEqual(gotScopes, wantScopes) {
				t.Errorf("dependency scopes differ for %s:\n got: %v\nwant: %v", buildTool, gotScopes, wantScopes)
			}
			if buildTool == "maven" {
				if !strings.Contains(buildFile, "<artifactId>spring-modulith-bom</artifactId>") || !strings.Contains(buildFile, "<artifactId>spring-cloud-aws-dependencies</artifactId>") || !mavenDependencyIsOptional(t, buildFile, "org.springframework.boot", "spring-boot-docker-compose") {
					t.Errorf("Maven build is missing BOM imports or optional Docker Compose metadata:\n%s", buildFile)
				}
			} else {
				for _, bom := range []string{
					"org.springframework.boot:spring-boot-dependencies:" + springBootVersion,
					"org.springframework.modulith:spring-modulith-bom:" + springModulithVersion,
					"io.awspring.cloud:spring-cloud-aws-dependencies:" + springCloudAWSVersion,
				} {
					if !strings.Contains(buildFile, bom) {
						t.Errorf("%s build is missing BOM %q", buildTool, bom)
					}
				}
				if !strings.Contains(buildFile, "spring-boot-docker-compose") || !strings.Contains(buildFile, "optional runtime dependency") {
					t.Errorf("%s build does not preserve Docker Compose optional/runtime intent", buildTool)
				}
			}
		})
	}
}

func TestDatabaseEnginesGenerateIsolatedRuntimeDriversAndConfiguration(t *testing.T) {
	tests := []struct {
		engine     string
		coordinate string
		url        string
		driver     string
		external   bool
	}{
		{
			engine:     "postgresql",
			coordinate: "org.postgresql:postgresql",
			url:        "${DATABASE_URL:jdbc:postgresql://localhost:5432/demo}",
			driver:     "org.postgresql.Driver",
			external:   true,
		},
		{
			engine:     "mysql",
			coordinate: "com.mysql:mysql-connector-j",
			url:        "${DATABASE_URL:jdbc:mysql://localhost:3306/demo}",
			driver:     "com.mysql.cj.jdbc.Driver",
			external:   true,
		},
		{
			engine:     "mariadb",
			coordinate: "org.mariadb.jdbc:mariadb-java-client",
			url:        "${DATABASE_URL:jdbc:mariadb://localhost:3306/demo}",
			driver:     "org.mariadb.jdbc.Driver",
			external:   true,
		},
		{
			engine:     "h2",
			coordinate: "com.h2database:h2",
			url:        "jdbc:h2:mem:demo;DB_CLOSE_DELAY=-1;DB_CLOSE_ON_EXIT=FALSE",
			driver:     "org.h2.Driver",
			external:   false,
		},
	}
	drivers := []string{
		"org.postgresql:postgresql",
		"com.mysql:mysql-connector-j",
		"org.mariadb.jdbc:mariadb-java-client",
		"com.h2database:h2",
	}

	for _, test := range tests {
		t.Run(test.engine, func(t *testing.T) {
			configuration := minimalConfiguration()
			configuration.Database.Engine = test.engine
			_, archive := generateProjectArchive(t, configuration)

			pom := readArchiveFile(t, archive, "demo/pom.xml")
			absentDrivers := make([]string, 0, len(drivers)-1)
			for _, driver := range drivers {
				if driver != test.coordinate {
					absentDrivers = append(absentDrivers, driver)
				}
			}
			assertBuildDependencies(t, "maven", pom, dependencyExpectation{
				present: []string{test.coordinate},
				absent:  absentDrivers,
			})
			if scope := extractDependencyScopes(t, "maven", pom)[test.coordinate]; scope != "runtimeOnly" {
				t.Errorf("database driver scope = %q, want runtimeOnly", scope)
			}

			properties := readArchiveFile(t, archive, "demo/src/main/resources/application.properties")
			if !strings.Contains(properties, "spring.datasource.url="+test.url) {
				t.Errorf("%s datasource URL is missing:\n%s", test.engine, properties)
			}
			if !strings.Contains(properties, "spring.datasource.driver-class-name="+test.driver) {
				t.Errorf("%s datasource driver is missing:\n%s", test.engine, properties)
			}
			assertDatasourceCredentialsAreSafe(t, properties, test.external)
		})
	}
}

func TestPersistenceFrameworksGenerateExclusiveDependencies(t *testing.T) {
	tests := []struct {
		name              string
		framework         string
		presentDependency string
		absentDependency  string
	}{
		{
			name:              "Spring Data JPA",
			framework:         "spring-data-jpa",
			presentDependency: "org.springframework.boot:spring-boot-starter-data-jpa",
			absentDependency:  "org.springframework.boot:spring-boot-starter-jdbc",
		},
		{
			name:              "Spring JdbcClient",
			framework:         "spring-jdbc-client",
			presentDependency: "org.springframework.boot:spring-boot-starter-jdbc",
			absentDependency:  "org.springframework.boot:spring-boot-starter-data-jpa",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			configuration := minimalConfiguration()
			configuration.Database.PersistenceFramework = test.framework
			_, archive := generateProjectArchive(t, configuration)

			pom := readArchiveFile(t, archive, "demo/pom.xml")
			assertBuildDependencies(t, "maven", pom, dependencyExpectation{
				present: []string{test.presentDependency},
				absent:  []string{test.absentDependency},
			})
		})
	}
}

func TestDatabaseMigrationMatrix(t *testing.T) {
	engines := []string{"postgresql", "mysql", "mariadb", "h2"}
	migrationTools := []string{"none", "flyway", "liquibase"}

	for _, engine := range engines {
		for _, migrationTool := range migrationTools {
			t.Run(engine+"_"+migrationTool, func(t *testing.T) {
				configuration := minimalConfiguration()
				configuration.Database.Engine = engine
				configuration.Database.MigrationTool = migrationTool
				filePath, archive := generateProjectArchive(t, configuration)
				pom := readArchiveFile(t, archive, "demo/pom.xml")

				switch migrationTool {
				case "none":
					assertBuildDependencies(t, "maven", pom, dependencyExpectation{absent: []string{
						"org.springframework.boot:spring-boot-starter-flyway",
						"org.springframework.boot:spring-boot-starter-liquibase",
						"org.flywaydb:flyway-database-postgresql",
						"org.flywaydb:flyway-mysql",
					}})
					assertNoArchivePrefix(t, archive, "demo/src/main/resources/db/migration/")
					assertNoArchivePrefix(t, archive, "demo/src/main/resources/db/changelog/")
				case "flyway":
					present := []string{"org.springframework.boot:spring-boot-starter-flyway"}
					absent := []string{"org.springframework.boot:spring-boot-starter-liquibase"}
					switch engine {
					case "postgresql":
						present = append(present, "org.flywaydb:flyway-database-postgresql")
						absent = append(absent, "org.flywaydb:flyway-mysql")
					case "mysql", "mariadb":
						present = append(present, "org.flywaydb:flyway-mysql")
						absent = append(absent, "org.flywaydb:flyway-database-postgresql")
					case "h2":
						absent = append(absent, "org.flywaydb:flyway-database-postgresql", "org.flywaydb:flyway-mysql")
					}
					assertBuildDependencies(t, "maven", pom, dependencyExpectation{present: present, absent: absent})
					migration := readArchiveFile(t, archive, "demo/src/main/resources/db/migration/V1__create_catalog_items.sql")
					if !strings.Contains(migration, "CREATE TABLE catalog_items") || !strings.Contains(migration, "PRIMARY KEY") {
						t.Errorf("Flyway migration is incomplete:\n%s", migration)
					}
					if (engine == "mysql" || engine == "mariadb") && !strings.Contains(migration, "AUTO_INCREMENT") {
						t.Errorf("%s Flyway migration does not use AUTO_INCREMENT:\n%s", engine, migration)
					}
					if (engine == "postgresql" || engine == "h2") && !strings.Contains(migration, "GENERATED BY DEFAULT AS IDENTITY") {
						t.Errorf("%s Flyway migration does not use identity syntax:\n%s", engine, migration)
					}
					properties := readArchiveFile(t, archive, "demo/src/main/resources/application.properties")
					if !strings.Contains(properties, "spring.flyway.locations=classpath:db/migration") {
						t.Errorf("Flyway location is not configured:\n%s", properties)
					}
				case "liquibase":
					assertBuildDependencies(t, "maven", pom, dependencyExpectation{
						present: []string{"org.springframework.boot:spring-boot-starter-liquibase"},
						absent: []string{
							"org.springframework.boot:spring-boot-starter-flyway",
							"org.flywaydb:flyway-database-postgresql",
							"org.flywaydb:flyway-mysql",
						},
					})
					changelog := readArchiveFile(t, archive, "demo/src/main/resources/db/changelog/db.changelog-master.xml")
					var document struct {
						ChangeSets []struct {
							ID string `xml:"id,attr"`
						} `xml:"changeSet"`
					}
					if err := xml.Unmarshal([]byte(changelog), &document); err != nil || len(document.ChangeSets) != 1 || document.ChangeSets[0].ID != "1" {
						t.Errorf("Liquibase root changelog is invalid: error=%v document=%+v\n%s", err, document, changelog)
					}
					properties := readArchiveFile(t, archive, "demo/src/main/resources/application.properties")
					if !strings.Contains(properties, "spring.liquibase.change-log=classpath:db/changelog/db.changelog-master.xml") {
						t.Errorf("Liquibase changelog is not configured:\n%s", properties)
					}
				}

				if engine == "h2" {
					smokeTestGeneratedProject(t, filePath, configuration.Project.BuildTool)
				} else {
					smokeBuildGeneratedProject(t, filePath, configuration.Project.BuildTool)
				}
			})
		}
	}
}

func TestSecurityModesGenerateDependenciesConfigurationAndProtectedRoutes(t *testing.T) {
	tests := []struct {
		name            string
		frontend        string
		mode            string
		present         []string
		absent          []string
		configSnippets  []string
		publicPath      string
		protectedPath   string
		propertySnippet string
	}{
		{
			name:     "none",
			frontend: "none",
			mode:     "none",
			absent: []string{
				"org.springframework.boot:spring-boot-starter-security",
				"org.springframework.boot:spring-boot-starter-oauth2-resource-server",
				"org.springframework.boot:spring-boot-starter-webmvc-test",
			},
		},
		{
			name:     "Thymeleaf HTTP session",
			frontend: "thymeleaf",
			mode:     "http-session",
			present: []string{
				"org.springframework.boot:spring-boot-starter-security",
				"org.springframework.boot:spring-boot-starter-webmvc-test",
			},
			absent: []string{"org.springframework.boot:spring-boot-starter-oauth2-resource-server"},
			configSnippets: []string{
				`.requestMatchers("/public").permitAll()`,
				".anyRequest().authenticated()",
				".formLogin(Customizer.withDefaults())",
			},
			publicPath:    "/public",
			protectedPath: "/protected",
		},
		{
			name:     "Angular OAuth resource server",
			frontend: "angular",
			mode:     "oauth-resource-server",
			present: []string{
				"org.springframework.boot:spring-boot-starter-security",
				"org.springframework.boot:spring-boot-starter-oauth2-resource-server",
				"org.springframework.boot:spring-boot-starter-webmvc-test",
			},
			configSnippets: []string{
				"SessionCreationPolicy.STATELESS",
				".csrf(AbstractHttpConfigurer::disable)",
				`.requestMatchers("/api/public").permitAll()`,
				".oauth2ResourceServer(oauth2 -> oauth2.jwt(Customizer.withDefaults()))",
			},
			publicPath:      "/api/public",
			protectedPath:   "/api/protected",
			propertySnippet: "spring.security.oauth2.resourceserver.jwt.jwk-set-uri=${OAUTH2_JWK_SET_URI:http://localhost:9000/oauth2/jwks}",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			configuration := minimalConfiguration()
			configuration.Frontend.Type = test.frontend
			if test.frontend != "none" {
				cssFramework := "tailwind"
				configuration.Frontend.CSSFramework = &cssFramework
			}
			configuration.Security.Mode = test.mode
			filePath, archive := generateProjectArchive(t, configuration)

			pom := readArchiveFile(t, archive, "demo/pom.xml")
			assertBuildDependencies(t, "maven", pom, dependencyExpectation{present: test.present, absent: test.absent})
			securityConfigPath := "demo/src/main/java/com/example/demo/config/SecurityConfig.java"
			endpointsPath := "demo/src/main/java/com/example/demo/web/SecurityEndpoints.java"
			endpointTestsPath := "demo/src/test/java/com/example/demo/web/SecurityEndpointsTests.java"
			properties := readArchiveFile(t, archive, "demo/src/main/resources/application.properties")

			if test.mode == "none" {
				assertArchiveExcludes(t, archive, securityConfigPath, endpointsPath, endpointTestsPath)
				if strings.Contains(properties, "spring.security.oauth2") {
					t.Errorf("unsecured project contains OAuth configuration:\n%s", properties)
				}
				return
			}

			assertArchiveContains(t, archive, securityConfigPath, endpointsPath, endpointTestsPath)
			securityConfig := readArchiveFile(t, archive, securityConfigPath)
			for _, snippet := range test.configSnippets {
				if !strings.Contains(securityConfig, snippet) {
					t.Errorf("%s security configuration is missing %q:\n%s", test.mode, snippet, securityConfig)
				}
			}
			endpoints := readArchiveFile(t, archive, endpointsPath)
			if !strings.Contains(endpoints, `@GetMapping("`+test.publicPath+`")`) || !strings.Contains(endpoints, `@GetMapping("`+test.protectedPath+`")`) {
				t.Errorf("%s endpoints do not expose the intended routes:\n%s", test.mode, endpoints)
			}
			if test.propertySnippet == "" {
				if strings.Contains(properties, "spring.security.oauth2") {
					t.Errorf("form-login project contains OAuth properties:\n%s", properties)
				}
			} else if !strings.Contains(properties, test.propertySnippet) {
				t.Errorf("OAuth resource-server property is missing:\n%s", properties)
			}

			smokeTestGeneratedProject(t, filePath, configuration.Project.BuildTool)
		})
	}
}

func TestOptionalFeaturesGenerateInIsolationAndTogether(t *testing.T) {
	t.Run("email", func(t *testing.T) {
		configuration := minimalConfiguration()
		configuration.Features = []string{"email"}
		filePath, archive := generateProjectArchive(t, configuration)
		pom := readArchiveFile(t, archive, "demo/pom.xml")
		assertBuildDependencies(t, "maven", pom, dependencyExpectation{
			present: []string{"org.springframework.boot:spring-boot-starter-mail"},
			absent:  []string{"org.springframework.boot:spring-boot-docker-compose", "org.springframework.modulith:spring-modulith-starter-core", "io.awspring.cloud:spring-cloud-aws-starter"},
		})
		properties := readArchiveFile(t, archive, "demo/src/main/resources/application.properties")
		for _, property := range []string{"spring.mail.host=${MAIL_HOST:localhost}", "spring.mail.port=${MAIL_PORT:1025}", "spring.mail.username=${MAIL_USERNAME:}", "spring.mail.password=${MAIL_PASSWORD:}"} {
			if !strings.Contains(properties, property) {
				t.Errorf("mail configuration is missing %q:\n%s", property, properties)
			}
		}
		assertArchiveExcludes(t, archive, "demo/compose.yaml", "demo/src/test/java/com/example/demo/ModularityTests.java")
		smokeTestGeneratedProject(t, filePath, configuration.Project.BuildTool)
	})

	composeTests := []struct {
		engine  string
		service string
		image   string
		port    string
		health  string
		environ []string
	}{
		{"postgresql", "postgres:", "postgres:17-alpine", `"5432:5432"`, "pg_isready -U demo -d demo", []string{"POSTGRES_DB: demo", "POSTGRES_USER: demo", "POSTGRES_PASSWORD: demo"}},
		{"mysql", "mysql:", "mysql:8.4", `"3306:3306"`, "mysqladmin ping", []string{"MYSQL_DATABASE: demo", "MYSQL_USER: demo", "MYSQL_PASSWORD: demo", "MYSQL_ROOT_PASSWORD: root"}},
		{"mariadb", "mariadb:", "mariadb:11.8", `"3306:3306"`, "healthcheck.sh", []string{"MARIADB_DATABASE: demo", "MARIADB_USER: demo", "MARIADB_PASSWORD: demo", "MARIADB_ROOT_PASSWORD: root"}},
	}
	for _, test := range composeTests {
		t.Run("docker-compose "+test.engine, func(t *testing.T) {
			configuration := minimalConfiguration()
			configuration.Database.Engine = test.engine
			configuration.Features = []string{"docker-compose"}
			filePath, archive := generateProjectArchive(t, configuration)
			pom := readArchiveFile(t, archive, "demo/pom.xml")
			assertBuildDependencies(t, "maven", pom, dependencyExpectation{present: []string{"org.springframework.boot:spring-boot-docker-compose"}})
			if !mavenDependencyIsOptional(t, pom, "org.springframework.boot", "spring-boot-docker-compose") || !strings.Contains(pom, "<scope>runtime</scope>") {
				t.Errorf("Docker Compose dependency is not optional runtime:\n%s", pom)
			}
			compose := readArchiveFile(t, archive, "demo/compose.yaml")
			want := append([]string{"services:", test.service, "image: " + test.image, test.port, test.health, "interval: 10s", "timeout: 5s", "retries: 5"}, test.environ...)
			for _, snippet := range want {
				if !strings.Contains(compose, snippet) {
					t.Errorf("%s compose file is missing %q:\n%s", test.engine, snippet, compose)
				}
			}
			smokeBuildGeneratedProject(t, filePath, configuration.Project.BuildTool)
		})
	}

	t.Run("docker-compose h2", func(t *testing.T) {
		configuration := minimalConfiguration()
		configuration.Features = []string{"docker-compose"}
		filePath, archive := generateProjectArchive(t, configuration)
		assertArchiveExcludes(t, archive, "demo/compose.yaml")
		properties := readArchiveFile(t, archive, "demo/src/main/resources/application.properties")
		if !strings.Contains(properties, "spring.docker.compose.enabled=false") {
			t.Errorf("H2 Docker Compose selection is not safely disabled:\n%s", properties)
		}
		smokeTestGeneratedProject(t, filePath, configuration.Project.BuildTool)
	})

	t.Run("spring-modulith", func(t *testing.T) {
		configuration := minimalConfiguration()
		configuration.Features = []string{"spring-modulith", "spring-modulith"}
		filePath, archive := generateProjectArchive(t, configuration)
		pom := readArchiveFile(t, archive, "demo/pom.xml")
		for _, artifact := range []string{"spring-modulith-starter-core", "spring-modulith-starter-test", "spring-modulith-bom"} {
			if count := strings.Count(pom, "<artifactId>"+artifact+"</artifactId>"); count != 1 {
				t.Errorf("%s occurs %d times, want once:\n%s", artifact, count, pom)
			}
		}
		if strings.Count(pom, "<spring-modulith.version>"+springModulithVersion+"</spring-modulith.version>") != 1 {
			t.Errorf("Spring Modulith version property is missing or duplicated:\n%s", pom)
		}
		verification := readArchiveFile(t, archive, "demo/src/test/java/com/example/demo/ModularityTests.java")
		if !strings.Contains(verification, "ApplicationModules.of(Application.class).verify()") {
			t.Errorf("module verification test is invalid:\n%s", verification)
		}
		smokeTestGeneratedProject(t, filePath, configuration.Project.BuildTool)
	})

	t.Run("spring-cloud-aws", func(t *testing.T) {
		configuration := minimalConfiguration()
		configuration.Features = []string{"spring-cloud-aws", "spring-cloud-aws"}
		filePath, archive := generateProjectArchive(t, configuration)
		pom := readArchiveFile(t, archive, "demo/pom.xml")
		for _, artifact := range []string{"spring-cloud-aws-starter", "spring-cloud-aws-dependencies"} {
			if count := strings.Count(pom, "<artifactId>"+artifact+"</artifactId>"); count != 1 {
				t.Errorf("%s occurs %d times, want once:\n%s", artifact, count, pom)
			}
		}
		if strings.Count(pom, "<spring-cloud-aws.version>"+springCloudAWSVersion+"</spring-cloud-aws.version>") != 1 {
			t.Errorf("Spring Cloud AWS version property is missing or duplicated:\n%s", pom)
		}
		properties := readArchiveFile(t, archive, "demo/src/main/resources/application.properties")
		if !strings.Contains(properties, "spring.cloud.aws.region.static=${AWS_REGION:us-east-1}") || strings.Contains(properties, "access-key") || strings.Contains(properties, "secret-key") {
			t.Errorf("Spring Cloud AWS local configuration is unsafe or incomplete:\n%s", properties)
		}
		smokeTestGeneratedProject(t, filePath, configuration.Project.BuildTool)
	})

	t.Run("none", func(t *testing.T) {
		configuration := minimalConfiguration()
		_, archive := generateProjectArchive(t, configuration)
		pom := readArchiveFile(t, archive, "demo/pom.xml")
		for _, snippet := range []string{"spring-boot-starter-mail", "spring-boot-docker-compose", "spring-modulith", "spring-cloud-aws"} {
			if strings.Contains(pom, snippet) {
				t.Errorf("minimal build unexpectedly contains %q:\n%s", snippet, pom)
			}
		}
		properties := readArchiveFile(t, archive, "demo/src/main/resources/application.properties")
		for _, prefix := range []string{"spring.mail.", "spring.cloud.aws.", "spring.docker.compose."} {
			if strings.Contains(properties, prefix) {
				t.Errorf("minimal properties unexpectedly contain %q:\n%s", prefix, properties)
			}
		}
		assertArchiveExcludes(t, archive, "demo/compose.yaml", "demo/src/test/java/com/example/demo/ModularityTests.java")
	})

	t.Run("all", func(t *testing.T) {
		configuration := minimalConfiguration()
		configuration.Features = []string{"email", "docker-compose", "spring-modulith", "spring-cloud-aws"}
		filePath, archive := generateProjectArchive(t, configuration)
		pom := readArchiveFile(t, archive, "demo/pom.xml")
		for _, artifact := range []string{"spring-boot-starter-mail", "spring-boot-docker-compose", "spring-modulith-starter-core", "spring-modulith-starter-test", "spring-modulith-bom", "spring-cloud-aws-starter", "spring-cloud-aws-dependencies"} {
			if count := strings.Count(pom, "<artifactId>"+artifact+"</artifactId>"); count != 1 {
				t.Errorf("full-feature build contains %d copies of %s, want one", count, artifact)
			}
		}
		smokeTestGeneratedProject(t, filePath, configuration.Project.BuildTool)
	})
}

func TestComposeTemplateSupportsMultipleSelectedServices(t *testing.T) {
	compose, err := renderProjectTemplate("compose.yaml.tmpl", composeTemplateData{
		DatabaseName: "demo",
		PostgreSQL:   true,
		MySQL:        true,
		MariaDB:      true,
	})
	if err != nil {
		t.Fatalf("render compose template: %v", err)
	}
	for _, service := range []string{"  postgres:\n", "  mysql:\n", "  mariadb:\n"} {
		if count := strings.Count(compose, service); count != 1 {
			t.Errorf("compose service %q occurs %d times, want once:\n%s", strings.TrimSpace(service), count, compose)
		}
	}
}

func TestSpringdocFlociAndArchUnitGeneration(t *testing.T) {
	t.Run("ArchUnit baseline", func(t *testing.T) {
		configuration := minimalConfiguration()
		filePath, archive := generateProjectArchive(t, configuration)
		pom := readArchiveFile(t, archive, "demo/pom.xml")
		assertBuildDependencies(t, "maven", pom, dependencyExpectation{present: []string{"com.tngtech.archunit:archunit-junit5"}})
		if !strings.Contains(pom, "<archunit.version>"+archunitVersion+"</archunit.version>") || !strings.Contains(pom, "<version>${archunit.version}</version>") {
			t.Errorf("ArchUnit version is missing:\n%s", pom)
		}
		architectureTest := readArchiveFile(t, archive, "demo/src/test/java/com/example/demo/ArchUnitTests.java")
		if !strings.Contains(architectureTest, "new ClassFileImporter().importPackagesOf(Application.class)") || !strings.Contains(architectureTest, `.resideInAPackage("com.example.demo..")`) {
			t.Errorf("generated ArchUnit test is invalid:\n%s", architectureTest)
		}
		smokeTestGeneratedProject(t, filePath, configuration.Project.BuildTool)
	})

	t.Run("Springdoc Angular API", func(t *testing.T) {
		configuration := minimalConfiguration()
		configuration.Frontend.Type = "angular"
		cssFramework := "tailwind"
		configuration.Frontend.CSSFramework = &cssFramework
		filePath, archive := generateProjectArchive(t, configuration)
		pom := readArchiveFile(t, archive, "demo/pom.xml")
		assertBuildDependencies(t, "maven", pom, dependencyExpectation{present: []string{"org.springdoc:springdoc-openapi-starter-webmvc-ui"}})
		if !strings.Contains(pom, "<springdoc-openapi.version>"+springdocVersion+"</springdoc-openapi.version>") || !strings.Contains(pom, "<version>${springdoc-openapi.version}</version>") {
			t.Errorf("Springdoc version is missing:\n%s", pom)
		}
		properties := readArchiveFile(t, archive, "demo/src/main/resources/application.properties")
		for _, property := range []string{"springdoc.swagger-ui.enabled=false", "springdoc.paths-to-match=/api/**"} {
			if !strings.Contains(properties, property) {
				t.Errorf("Springdoc configuration is missing %q:\n%s", property, properties)
			}
		}
		smokeTestGeneratedProject(t, filePath, configuration.Project.BuildTool)
	})

	t.Run("Floci Spring Cloud AWS", func(t *testing.T) {
		configuration := minimalConfiguration()
		configuration.Features = []string{"spring-cloud-aws", "docker-compose"}
		filePath, archive := generateProjectArchive(t, configuration)
		pom := readArchiveFile(t, archive, "demo/pom.xml")
		assertBuildDependencies(t, "maven", pom, dependencyExpectation{present: []string{
			"org.springframework.boot:spring-boot-testcontainers",
			"io.floci:spring-boot-testcontainers-floci",
		}})
		if !strings.Contains(pom, "<spring-boot-floci-tc.version>"+flociVersion+"</spring-boot-floci-tc.version>") || !strings.Contains(pom, "<version>${spring-boot-floci-tc.version}</version>") {
			t.Errorf("Floci version is missing:\n%s", pom)
		}
		assertArchiveContains(t, archive,
			"demo/compose.yaml",
			"demo/src/test/java/com/example/demo/TestcontainersConfig.java",
			"demo/src/test/resources/floci-init/ready.d/01-seed.sh",
		)
		compose := readArchiveFile(t, archive, "demo/compose.yaml")
		for _, snippet := range []string{"floci:", "floci/floci:latest-compat", `"4566:4566"`, "AWS_DEFAULT_REGION: us-east-1"} {
			if !strings.Contains(compose, snippet) {
				t.Errorf("Floci Compose service is missing %q:\n%s", snippet, compose)
			}
		}
		flociConfig := readArchiveFile(t, archive, "demo/src/test/java/com/example/demo/TestcontainersConfig.java")
		if !strings.Contains(flociConfig, "FlociContainer") || !strings.Contains(flociConfig, "@ServiceConnection") {
			t.Errorf("Floci Testcontainers configuration is invalid:\n%s", flociConfig)
		}
		assertArchiveFileMode(t, archive, "demo/src/test/resources/floci-init/ready.d/01-seed.sh", 0o755)
		smokeTestGeneratedProject(t, filePath, configuration.Project.BuildTool)
	})

	t.Run("database and Floci share one configuration", func(t *testing.T) {
		configuration := minimalConfiguration()
		configuration.Database.Engine = "postgresql"
		configuration.Features = []string{"spring-cloud-aws"}
		filePath, archive := generateProjectArchive(t, configuration)
		config := readArchiveFile(t, archive, "demo/src/test/java/com/example/demo/TestcontainersConfig.java")
		if strings.Count(config, "class TestcontainersConfig") != 1 || !strings.Contains(config, "PostgreSQLContainer") || !strings.Contains(config, "FlociContainer") {
			t.Errorf("database and Floci are not combined in one Testcontainers configuration:\n%s", config)
		}
		assertArchiveExcludes(t, archive, "demo/src/test/java/com/example/demo/FlociTestcontainersConfig.java")
		smokeBuildGeneratedProject(t, filePath, configuration.Project.BuildTool)
	})

	containerTests := []struct {
		engine     string
		dependency string
		container  string
		image      string
	}{
		{"postgresql", "org.testcontainers:testcontainers-postgresql", "PostgreSQLContainer", "postgres:17-alpine"},
		{"mysql", "org.testcontainers:testcontainers-mysql", "MySQLContainer", "mysql:8.4"},
		{"mariadb", "org.testcontainers:testcontainers-mariadb", "MariaDBContainer", "mariadb:11.8"},
	}
	for _, test := range containerTests {
		t.Run("Testcontainers "+test.engine, func(t *testing.T) {
			configuration := minimalConfiguration()
			configuration.Database.Engine = test.engine
			filePath, archive := generateProjectArchive(t, configuration)
			pom := readArchiveFile(t, archive, "demo/pom.xml")
			assertBuildDependencies(t, "maven", pom, dependencyExpectation{present: []string{
				"org.springframework.boot:spring-boot-testcontainers",
				"org.testcontainers:testcontainers-junit-jupiter",
				test.dependency,
			}})
			config := readArchiveFile(t, archive, "demo/src/test/java/com/example/demo/TestcontainersConfig.java")
			if !strings.Contains(config, test.container) || !strings.Contains(config, test.image) || strings.Contains(config, "FlociContainer") {
				t.Errorf("%s Testcontainers configuration is invalid:\n%s", test.engine, config)
			}
			smokeBuildGeneratedProject(t, filePath, configuration.Project.BuildTool)
		})
	}
}

func assertNoArchivePrefix(t *testing.T, archive *zip.ReadCloser, prefix string) {
	t.Helper()
	for _, name := range listArchiveFiles(archive) {
		if strings.HasPrefix(name, prefix) {
			t.Errorf("archive unexpectedly contains %q beneath %q", name, prefix)
		}
	}
}

func assertDatasourceCredentialsAreSafe(t *testing.T, properties string, external bool) {
	t.Helper()
	username := propertyValue(properties, "spring.datasource.username")
	password := propertyValue(properties, "spring.datasource.password")
	if external {
		if username != "${DATABASE_USERNAME}" || password != "${DATABASE_PASSWORD}" {
			t.Errorf("external datasource credentials must use environment placeholders; username=%q password=%q", username, password)
		}
		return
	}
	if username != "" || password != "" {
		t.Errorf("embedded datasource must not contain credentials; username=%q password=%q", username, password)
	}
}

func propertyValue(properties string, key string) string {
	for _, line := range strings.Split(properties, "\n") {
		name, value, found := strings.Cut(line, "=")
		if found && name == key {
			return value
		}
	}
	return ""
}

func prefixPaths(prefix string, paths []string) []string {
	prefixed := make([]string, 0, len(paths))
	for _, path := range paths {
		prefixed = append(prefixed, prefix+path)
	}
	return prefixed
}

func assertArchiveExcludes(t *testing.T, archive *zip.ReadCloser, names ...string) {
	t.Helper()
	files := listArchiveFiles(archive)
	for _, name := range names {
		if slices.Contains(files, name) {
			t.Errorf("archive unexpectedly contains %q", name)
		}
	}
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

var gradleDependencyPattern = regexp.MustCompile(`(?m)^\s*(implementation|runtimeOnly|compileOnly|testImplementation|testRuntimeOnly)\s*\(?\s*["']([^:"']+):([^:"']+)`)

func extractGradleDependencies(t *testing.T, buildFile string) []string {
	t.Helper()
	matches := gradleDependencyPattern.FindAllStringSubmatch(buildFile, -1)
	dependencies := make([]string, 0, len(matches))
	for _, match := range matches {
		dependencies = append(dependencies, match[2]+":"+match[3])
	}
	return dependencies
}

func extractDependencyScopes(t *testing.T, buildTool string, buildFile string) map[string]string {
	t.Helper()
	if buildTool != "maven" {
		dependencies := make(map[string]string)
		for _, match := range gradleDependencyPattern.FindAllStringSubmatch(buildFile, -1) {
			dependencies[match[2]+":"+match[3]] = match[1]
		}
		return dependencies
	}

	var project struct {
		Dependencies []struct {
			GroupID    string `xml:"groupId"`
			ArtifactID string `xml:"artifactId"`
			Scope      string `xml:"scope"`
		} `xml:"dependencies>dependency"`
	}
	if err := xml.Unmarshal([]byte(buildFile), &project); err != nil {
		t.Fatalf("parse Maven dependency scopes: %v", err)
	}
	dependencies := make(map[string]string)
	for _, dependency := range project.Dependencies {
		scope := "implementation"
		if dependency.Scope == "runtime" {
			scope = "runtimeOnly"
		} else if dependency.Scope == "test" {
			scope = "testImplementation"
		}
		dependencies[dependency.GroupID+":"+dependency.ArtifactID] = scope
	}
	return dependencies
}

func mavenDependencyIsOptional(t *testing.T, pom string, groupID string, artifactID string) bool {
	t.Helper()
	var project struct {
		Dependencies []struct {
			GroupID    string `xml:"groupId"`
			ArtifactID string `xml:"artifactId"`
			Optional   bool   `xml:"optional"`
		} `xml:"dependencies>dependency"`
	}
	if err := xml.Unmarshal([]byte(pom), &project); err != nil {
		t.Fatalf("parse Maven optional dependencies: %v", err)
	}
	for _, dependency := range project.Dependencies {
		if dependency.GroupID == groupID && dependency.ArtifactID == artifactID {
			return dependency.Optional
		}
	}
	return false
}

func smokeTestGeneratedProject(t *testing.T, archivePath string, buildTool string) {
	t.Helper()
	runGeneratedProjectCommand(t, archivePath, buildTool, "test")
}

func smokeBuildGeneratedProject(t *testing.T, archivePath string, buildTool string) {
	t.Helper()
	if buildTool == "maven" {
		runGeneratedProjectCommand(t, archivePath, buildTool, "package", "-DskipTests")
		return
	}
	runGeneratedProjectCommand(t, archivePath, buildTool, "testClasses")
}

func runGeneratedProjectCommand(t *testing.T, archivePath string, buildTool string, arguments ...string) {
	t.Helper()
	projectDirectory := extractGeneratedProject(t, archivePath)

	launchers := map[string]string{
		"maven":         "./mvnw",
		"gradle-groovy": "./gradlew",
		"gradle-kotlin": "./gradlew",
	}
	launcher, ok := launchers[buildTool]
	if !ok {
		t.Fatalf("no smoke-test command for build tool %q", buildTool)
	}
	if runtime.GOOS == "windows" {
		if buildTool == "maven" {
			launcher += ".cmd"
		} else {
			launcher += ".bat"
		}
	}
	commandLine := append([]string{launcher}, arguments...)
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
	if buildTool != "maven" {
		gradleUserHome := filepath.Join(os.TempDir(), "sbgen-gradle-home")
		if err := os.MkdirAll(gradleUserHome, 0o755); err != nil {
			t.Fatalf("create isolated Gradle user home: %v", err)
		}
		command.Env = append(os.Environ(), "GRADLE_USER_HOME="+gradleUserHome)
	}
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

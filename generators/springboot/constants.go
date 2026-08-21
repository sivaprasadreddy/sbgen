package springboot

const srcMainJavaPath = "src/main/java/"
const srcMainResourcesPath = "src/main/resources/"
const srcTestJavaPath = "src/test/java/"
const srcTestResourcesPath = "src/test/resources/"

type AppType string

const (
	RestApi           AppType = "REST API"
	WebApp            AppType = "Web App"
	SpringBootAngular AppType = "Spring Boot + Angular"
)

func (a AppType) String() string {
	switch a {
	case RestApi:
		return "REST API"
	case WebApp:
		return "Web App"
	case SpringBootAngular:
		return "Spring Boot + Angular"
	default:
		return "Unknown"
	}
}

type BuildTool string

const (
	Maven  BuildTool = "Maven"
	Gradle BuildTool = "Gradle"
)

func (s BuildTool) String() string {
	return string(s)
}

type DatabaseType string

const (
	PostgreSQL DatabaseType = "PostgreSQL"
	MySQL      DatabaseType = "MySQL"
	MariaDB    DatabaseType = "MariaDB"
	H2         DatabaseType = "H2"
)

func (s DatabaseType) String() string {
	return string(s)
}

type DbMigrationTool string

const (
	None      DbMigrationTool = "None"
	Flyway    DbMigrationTool = "Flyway"
	Liquibase DbMigrationTool = "Liquibase"
)

func (s DbMigrationTool) String() string {
	return string(s)
}

const FeatureDockerComposeSupport = "Docker Compose"
const FeatureSpringModulithSupport = "Spring Modulith"
const FeatureSpringCloudAWSSupport = "Spring Cloud AWS"
const FeatureThymeleafSupport = "Thymeleaf"
const FeatureHTMXSupport = "HTMX"
const FeatureSecuritySupport = "Security"
const FeatureJwtSecuritySupport = "JWT Security"

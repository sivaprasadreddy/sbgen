package main

// option is a choice the wizard offers: its value, its name, and a line
// describing it.
type option[T comparable] struct {
	value       T
	name        string
	description string
}

var buildToolOptions = []option[BuildTool]{
	{BuildToolMaven, "Maven", "pom.xml"},
	{BuildToolGradle, "Gradle", "build.gradle"},
}

var taskRunnerOptions = []option[TaskRunner]{
	{TaskRunnerMise, "Mise", "mise.toml installs the JDK and runs tasks"},
	{TaskRunnerTaskfile, "Taskfile", "Taskfile.yml with .sdkmanrc for SDKMAN"},
}

var frontendOptions = []option[FrontendType]{
	{FrontendNone, "None", "Backend REST API only"},
	{FrontendThymeleaf, "Thymeleaf", "Server-rendered web application using Thymeleaf"},
	{FrontendJTE, "JTE", "Server-rendered web application using Java Template Engine"},
	{FrontendAngular, "Angular", "Spring Boot backend with an Angular single-page application"},
	{FrontendReact, "React", "Spring Boot backend with a React (Vite) single-page application"},
}

var databaseOptions = []option[DatabaseEngine]{
	{DatabasePostgreSQL, "PostgreSQL", "Reliable, advanced SQL database"},
	{DatabaseMySQL, "MySQL", "Popular open-source SQL database"},
	{DatabaseMariaDB, "MariaDB", "Community-developed MySQL alternative"},
}

var persistenceOptions = []option[PersistenceFramework]{
	{PersistenceSpringDataJPA, "Spring Data JPA", "Repository-based persistence with JPA"},
	{PersistenceSpringJdbcClient, "Spring JdbcClient", "Fluent JDBC-based database access"},
	{PersistenceJOOQ, "jOOQ", "Type-safe SQL query building with jOOQ"},
}

var migrationOptions = []option[MigrationTool]{
	{MigrationFlyway, "Flyway", "Versioned SQL migrations"},
	{MigrationLiquibase, "Liquibase", "Flexible database change tracking"},
}

var featureOptions = []option[Feature]{
	{FeatureSpringCloudAWS, "Spring Cloud AWS", "Integrate with AWS services using Spring"},
	{FeatureEmail, "Email", "Send transactional and application emails"},
	{FeatureRabbitMQ, "RabbitMQ", "Add messaging support with Spring AMQP"},
	{FeatureRedisCaching, "Redis Caching", "Add caching support backed by Redis"},
	{FeatureOpenTelemetry, "OpenTelemetry", "Add distributed tracing and observability"},
	{FeatureKubernetes, "Kubernetes Manifests", "Generate deployment manifests for Kubernetes"},
}

var aiModelOptions = []option[AIModel]{
	{AIModelOllama, "Ollama", "Run open models locally with Ollama"},
	{AIModelOpenAI, "OpenAI", "Use OpenAI chat models"},
	{AIModelAnthropic, "Anthropic", "Use Anthropic Claude chat models"},
}

// labels are the names the Review step and template summaries show.
var labels = map[string]string{
	"maven": "Maven", "gradle": "Gradle",
	"mise": "Mise", "taskfile": "Taskfile (with SDKMAN)",
	"none": "None (REST API)", "thymeleaf": "Thymeleaf", "jte": "JTE", "angular": "Angular", "react": "React",
	"postgresql": "PostgreSQL", "mysql": "MySQL", "mariadb": "MariaDB",
	"flyway": "Flyway", "liquibase": "Liquibase",
	"spring-data-jpa": "Spring Data JPA", "spring-jdbc-client": "Spring JdbcClient", "jooq": "jOOQ",
	"email": "Email", "spring-cloud-aws": "Spring Cloud AWS", "rabbitmq": "RabbitMQ",
	"redis-caching": "Redis Caching", "opentelemetry": "OpenTelemetry", "kubernetes": "Kubernetes Manifests",
	"ollama": "Ollama", "openai": "OpenAI", "anthropic": "Anthropic",
}

// label is the name of an option's value.
func label[T ~string](v T) string {
	if l, ok := labels[string(v)]; ok {
		return l
	}
	return string(v)
}

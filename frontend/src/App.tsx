import { FormEvent, MouseEvent, useState } from 'react'
import { Browser } from '@wailsio/runtime'
import { AppGeneratorService } from '../bindings/github.com/sivaprasadreddy/sbgen'

const authorUrl = 'https://sivalabs.in'
const repositoryUrl = 'https://github.com/sivaprasadreddy/sbgen'

// Open links in the system browser instead of navigating the app's webview.
const openExternal = (event: MouseEvent<HTMLAnchorElement>) => {
  event.preventDefault()
  void Browser.OpenURL(event.currentTarget.href)
}

const steps = ['Welcome', 'Project', 'Frontend', 'Database', 'Features', 'Review', 'Complete']

type BuildTool = 'maven' | 'gradle'
type TaskRunner = 'mise' | 'taskfile'
type Frontend = 'none' | 'thymeleaf' | 'jte' | 'angular' | 'react'
type DatabaseEngine = 'postgresql' | 'mysql' | 'mariadb'
type MigrationTool = 'flyway' | 'liquibase'
type PersistenceFramework = 'spring-data-jpa' | 'spring-jdbc-client' | 'jooq'
type Feature = 'email' | 'spring-cloud-aws' | 'rabbitmq' | 'redis-caching' | 'opentelemetry' | 'kubernetes'
type AIModel = 'ollama' | 'openai' | 'anthropic'

const isServerRendered = (frontend: Frontend) => frontend === 'thymeleaf' || frontend === 'jte'

const deriveBasePackage = (groupId: string, artifactId: string) => {
  const group = groupId.trim()
  const segment = artifactId.trim().replace(/-/g, '').toLowerCase()
  if (!group) return segment
  if (!segment) return group
  return `${group}.${segment}`
}

const featureOptions: ReadonlyArray<readonly [Feature, string, string]> = [
  ['spring-cloud-aws', 'Spring Cloud AWS', 'Integrate with AWS services using Spring'],
  ['email', 'Email', 'Send transactional and application emails'],
  ['rabbitmq', 'RabbitMQ', 'Add messaging support with Spring AMQP'],
  ['redis-caching', 'Redis Caching', 'Add caching support backed by Redis'],
  ['opentelemetry', 'OpenTelemetry', 'Add distributed tracing and observability'],
  ['kubernetes', 'Kubernetes Manifests', 'Generate deployment manifests for Kubernetes'],
]

const aiModelOptions: ReadonlyArray<readonly [AIModel, string, string]> = [
  ['ollama', 'Ollama', 'Run open models locally with Ollama'],
  ['openai', 'OpenAI', 'Use OpenAI chat models'],
  ['anthropic', 'Anthropic', 'Use Anthropic Claude chat models'],
]

const labels: Record<string, string> = {
  maven: 'Maven',
  gradle: 'Gradle',
  mise: 'Mise',
  taskfile: 'Taskfile (with SDKMAN)',
  none: 'None (REST API)',
  thymeleaf: 'Thymeleaf',
  jte: 'JTE',
  angular: 'Angular',
  react: 'React',
  postgresql: 'PostgreSQL',
  mysql: 'MySQL',
  mariadb: 'MariaDB',
  flyway: 'Flyway',
  liquibase: 'Liquibase',
  'spring-data-jpa': 'Spring Data JPA',
  'spring-jdbc-client': 'Spring JdbcClient',
  jooq: 'jOOQ',
  email: 'Email',
  'spring-cloud-aws': 'Spring Cloud AWS',
  rabbitmq: 'RabbitMQ',
  'redis-caching': 'Redis Caching',
  opentelemetry: 'OpenTelemetry',
  kubernetes: 'Kubernetes Manifests',
  ollama: 'Ollama',
  openai: 'OpenAI',
  anthropic: 'Anthropic',
}

type ProjectMetadata = {
  appName: string
  appVersion: string
  groupId: string
  artifactId: string
  packageName: string
  buildTool: BuildTool
  taskRunner: TaskRunner
}

const initialMetadata: ProjectMetadata = {
  appName: 'demo',
  appVersion: '1.0.0',
  groupId: 'dev.sivalabs',
  artifactId: 'demo',
  packageName: 'dev.sivalabs.demo',
  buildTool: 'maven',
  taskRunner: 'mise',
}

type FrontendConfiguration = {
  frontend: Frontend
  htmxSupport: boolean
}

const initialFrontendConfiguration: FrontendConfiguration = {
  frontend: 'none',
  htmxSupport: false,
}

type DatabaseConfiguration = {
  engine: DatabaseEngine
  migrationTool: MigrationTool
  persistenceFramework: PersistenceFramework
}

const initialDatabaseConfiguration: DatabaseConfiguration = {
  engine: 'postgresql',
  migrationTool: 'flyway',
  persistenceFramework: 'spring-data-jpa',
}

function App() {
  const [activeStep, setActiveStep] = useState(0)
  const [metadata, setMetadata] = useState<ProjectMetadata>(initialMetadata)
  const [savedMetadata, setSavedMetadata] = useState<ProjectMetadata | null>(null)
  const [frontend, setFrontend] = useState<FrontendConfiguration>(initialFrontendConfiguration)
  const [savedFrontend, setSavedFrontend] = useState<FrontendConfiguration | null>(null)
  const [database, setDatabase] = useState<DatabaseConfiguration>(initialDatabaseConfiguration)
  const [savedDatabase, setSavedDatabase] = useState<DatabaseConfiguration | null>(null)
  const [features, setFeatures] = useState<Feature[]>([])
  const [savedFeatures, setSavedFeatures] = useState<Feature[] | null>(null)
  const [aiModels, setAIModels] = useState<AIModel[]>([])
  const [savedAIModels, setSavedAIModels] = useState<AIModel[] | null>(null)
  const [downloadedFilePath, setDownloadedFilePath] = useState('')
  const [generationError, setGenerationError] = useState('')
  const [isGenerating, setIsGenerating] = useState(false)

  const updateMetadata = <K extends keyof ProjectMetadata>(field: K, value: ProjectMetadata[K]) => {
    setMetadata((current) => {
      const next = { ...current, [field]: value }
      // Keep the package name in sync with group/artifact IDs until the user customizes it.
      if ((field === 'groupId' || field === 'artifactId') && current.packageName === deriveBasePackage(current.groupId, current.artifactId)) {
        next.packageName = deriveBasePackage(next.groupId, next.artifactId)
      }
      return next
    })
  }

  const handleProjectSubmit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    setSavedMetadata({ ...metadata })
    setActiveStep(2)
  }

  const handleFrontendSubmit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    setSavedFrontend({ ...frontend })
    setActiveStep(3)
  }

  const updateDatabase = <K extends keyof DatabaseConfiguration>(field: K, value: DatabaseConfiguration[K]) => {
    setDatabase((current) => ({ ...current, [field]: value }))
  }

  const handleDatabaseSubmit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    setSavedDatabase({ ...database })
    setActiveStep(4)
  }

  const toggleFeature = (feature: Feature) => {
    setFeatures((current) => current.includes(feature)
      ? current.filter((selected) => selected !== feature)
      : [...current, feature])
  }

  const toggleAIModel = (model: AIModel) => {
    setAIModels((current) => current.includes(model)
      ? current.filter((selected) => selected !== model)
      : [...current, model])
  }

  const handleFeaturesSubmit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    setSavedFeatures([...features])
    setSavedAIModels(aiModelOptions.map(([value]) => value).filter((model) => aiModels.includes(model)))
    setActiveStep(5)
  }

  const handleGenerate = async () => {
    if (!savedMetadata || !savedFrontend || !savedDatabase || !savedFeatures || !savedAIModels) return

    const configuration = {
      project: savedMetadata,
      frontend: {
        type: savedFrontend.frontend,
        htmxSupport: isServerRendered(savedFrontend.frontend) && savedFrontend.htmxSupport,
      },
      database: savedDatabase,
      features: savedFeatures,
      aiModels: savedAIModels,
    }

    setIsGenerating(true)
    setGenerationError('')
    try {
      const filePath = await AppGeneratorService.GenerateProject(JSON.stringify(configuration))
      setDownloadedFilePath(filePath)
      setActiveStep(6)
    } catch (error) {
      setGenerationError(error instanceof Error ? error.message : String(error))
    } finally {
      setIsGenerating(false)
    }
  }

  const handleStartNew = () => {
    setMetadata(initialMetadata)
    setSavedMetadata(null)
    setFrontend(initialFrontendConfiguration)
    setSavedFrontend(null)
    setDatabase(initialDatabaseConfiguration)
    setSavedDatabase(null)
    setFeatures([])
    setSavedFeatures(null)
    setAIModels([])
    setSavedAIModels(null)
    setDownloadedFilePath('')
    setGenerationError('')
    setIsGenerating(false)
    setActiveStep(0)
  }

  return (
    <main className="generator-shell">
      <aside className="sidebar" aria-label="Project generation steps">
        <div className="logo-wrap">
          <img src="/sbgen.png" alt="Spring Boot Generator" className="logo" />
          <span>Spring Boot Generator</span>
        </div>
        <ol className="steps">
          {steps.map((step, index) => {
            const isComplete = index < activeStep || (index === activeStep && index === steps.length - 1 && Boolean(downloadedFilePath))
            return (
              <li className={isComplete ? 'complete' : index === activeStep ? 'active' : ''} key={step} aria-current={index === activeStep && !isComplete ? 'step' : undefined}>
                <span className="step-number">{isComplete ? '✓' : index + 1}</span><span>{step}</span>
              </li>
            )
          })}
        </ol>
        <footer className="sidebar-footer">
          <p>Author: <a href={authorUrl} onClick={openExternal}>SivaLabs</a></p>
          <p><a href={repositoryUrl} onClick={openExternal}>GitHub repository ↗</a></p>
        </footer>
      </aside>

      <section className="content-panel">
        {activeStep === 0 && (
          <div className="welcome-content" aria-labelledby="welcome-title">
            <div className="welcome-icon" aria-hidden="true"><svg viewBox="0 0 32 32"><path d="M7 16.5 13 22l12-13" /></svg></div>
            <p className="eyebrow">Welcome</p>
            <h1 id="welcome-title">Build production-ready<br />Spring Boot applications</h1>
            <p className="description">Create a thoughtfully structured Spring Boot project with sensible defaults and the dependencies your application needs—all through one simple, guided setup.</p>
            <button type="button" className="continue-button" onClick={() => setActiveStep(1)}>Get started</button>
          </div>
        )}

        {activeStep === 1 && (
          <form className="project-form" onSubmit={handleProjectSubmit}>
            <div className="form-heading">
              <p className="eyebrow">Step 2 of 7</p>
              <h1>Project Metadata</h1>
              <p>Define the coordinates, build system, and task runner for your new Spring Boot application.</p>
            </div>

            <div className="field-grid">
              <label className="field">
                <span>Application Name</span>
                <input required value={metadata.appName} onChange={(event) => updateMetadata('appName', event.target.value)} placeholder="boot-demo" autoComplete="off" />
                <small>The generated project directory name.</small>
              </label>
              <label className="field">
                <span>Application Version</span>
                <input required value={metadata.appVersion} onChange={(event) => updateMetadata('appVersion', event.target.value)} placeholder="1.0.0" autoComplete="off" />
                <small>The initial version of your application.</small>
              </label>
              <label className="field">
                <span>Group ID</span>
                <input required value={metadata.groupId} onChange={(event) => updateMetadata('groupId', event.target.value)} placeholder="com.example" autoComplete="off" />
                <small>Usually your organization’s reversed domain.</small>
              </label>
              <label className="field">
                <span>Artifact ID</span>
                <input required value={metadata.artifactId} onChange={(event) => updateMetadata('artifactId', event.target.value)} placeholder="my-application" autoComplete="off" />
                <small>The Maven/Gradle artifact name.</small>
              </label>
              <label className="field field-wide">
                <span>Package Name</span>
                <input required value={metadata.packageName} onChange={(event) => updateMetadata('packageName', event.target.value)} placeholder="com.example.demo" autoComplete="off" />
                <small>The base package for generated source code.</small>
              </label>
            </div>

            <fieldset className="build-tool">
              <legend>Build Tool</legend>
              <div className="build-options">
                {([
                  ['maven', 'Maven', 'pom.xml'],
                  ['gradle', 'Gradle', 'build.gradle'],
                ] as const).map(([value, label, file]) => (
                  <label className={metadata.buildTool === value ? 'selected' : ''} key={value}>
                    <input type="radio" name="buildTool" value={value} checked={metadata.buildTool === value} onChange={() => updateMetadata('buildTool', value)} />
                    <span><strong>{label}</strong><small>{file}</small></span>
                  </label>
                ))}
              </div>
            </fieldset>

            <fieldset className="build-tool">
              <legend>Task Runner</legend>
              <div className="build-options">
                {([
                  ['mise', 'Mise', 'mise.toml installs the JDK and runs tasks'],
                  ['taskfile', 'Taskfile', 'Taskfile.yml with .sdkmanrc for SDKMAN'],
                ] as const).map(([value, label, description]) => (
                  <label className={metadata.taskRunner === value ? 'selected' : ''} key={value}>
                    <input type="radio" name="taskRunner" value={value} checked={metadata.taskRunner === value} onChange={() => updateMetadata('taskRunner', value)} />
                    <span><strong>{label}</strong><small>{description}</small></span>
                  </label>
                ))}
              </div>
            </fieldset>

            <div className="form-actions">
              <button type="button" className="back-button" onClick={() => setActiveStep(0)}>Back</button>
              <button type="submit" className="next-button">Next <span aria-hidden="true">→</span></button>
            </div>
          </form>
        )}

        {activeStep === 2 && savedMetadata && (
          <form className="project-form frontend-form" onSubmit={handleFrontendSubmit}>
            <div className="form-heading">
              <p className="eyebrow">Step 3 of 7</p>
              <h1>Frontend</h1>
              <p>Choose how the user interface for <strong>{savedMetadata.appName}</strong> should be built.</p>
            </div>

            <fieldset className="build-tool first-option-group">
              <legend>Frontend</legend>
              <div className="build-options">
                {([
                  ['none', 'None', 'Backend REST API only'],
                  ['thymeleaf', 'Thymeleaf', 'Server-rendered web application using Thymeleaf'],
                  ['jte', 'JTE', 'Server-rendered web application using Java Template Engine'],
                  ['angular', 'Angular', 'Spring Boot backend with an Angular single-page application'],
                  ['react', 'React', 'Spring Boot backend with a React (Vite) single-page application'],
                ] as const).map(([value, label, description]) => (
                  <label className={frontend.frontend === value ? 'selected' : ''} key={value}>
                    <input type="radio" name="frontend" value={value} checked={frontend.frontend === value} onChange={() => {
                      setFrontend((current) => ({
                        ...current,
                        frontend: value,
                        htmxSupport: isServerRendered(value) && current.htmxSupport,
                      }))
                    }} />
                    <span><strong>{label}</strong><small>{description}</small></span>
                  </label>
                ))}
              </div>
            </fieldset>

            {isServerRendered(frontend.frontend) && (
              <fieldset className="build-tool">
                <legend>Additional Support</legend>
                <div className="build-options database-options">
                  <label className={frontend.htmxSupport ? 'selected' : ''}>
                    <input
                      type="checkbox"
                      name="htmxSupport"
                      checked={frontend.htmxSupport}
                      onChange={(event) => setFrontend((current) => ({ ...current, htmxSupport: event.target.checked }))}
                    />
                    <span><strong>HTMX</strong><small>Add HTMX for dynamic server-rendered pages</small></span>
                  </label>
                </div>
              </fieldset>
            )}

            <div className="form-actions">
              <button type="button" className="back-button" onClick={() => setActiveStep(1)}>Back</button>
              <button type="submit" className="next-button">Next <span aria-hidden="true">→</span></button>
            </div>
          </form>
        )}

        {activeStep === 3 && savedMetadata && savedFrontend && (
          <form className="project-form database-form" onSubmit={handleDatabaseSubmit}>
            <div className="form-heading">
              <p className="eyebrow">Step 4 of 7</p>
              <h1>Database</h1>
              <p>Choose the database, persistence framework, and schema migration tool for <strong>{savedMetadata.appName}</strong>.</p>
            </div>

            <fieldset className="build-tool first-option-group">
              <legend>Database</legend>
              <div className="build-options database-options">
                {([
                  ['postgresql', 'PostgreSQL', 'Reliable, advanced SQL database'],
                  ['mysql', 'MySQL', 'Popular open-source SQL database'],
                  ['mariadb', 'MariaDB', 'Community-developed MySQL alternative'],
                ] as const).map(([value, label, description]) => (
                  <label className={database.engine === value ? 'selected' : ''} key={value}>
                    <input type="radio" name="database" value={value} checked={database.engine === value} onChange={() => updateDatabase('engine', value)} />
                    <span><strong>{label}</strong><small>{description}</small></span>
                  </label>
                ))}
              </div>
            </fieldset>

            <fieldset className="build-tool">
              <legend>Persistence Framework</legend>
              <div className="build-options database-options">
                {([
                  ['spring-data-jpa', 'Spring Data JPA', 'Repository-based persistence with JPA'],
                  ['spring-jdbc-client', 'Spring JdbcClient', 'Fluent JDBC-based database access'],
                  ['jooq', 'jOOQ', 'Type-safe SQL query building with jOOQ'],
                ] as const).map(([value, label, description]) => (
                  <label className={database.persistenceFramework === value ? 'selected' : ''} key={value}>
                    <input type="radio" name="persistenceFramework" value={value} checked={database.persistenceFramework === value} onChange={() => updateDatabase('persistenceFramework', value)} />
                    <span><strong>{label}</strong><small>{description}</small></span>
                  </label>
                ))}
              </div>
            </fieldset>

            <fieldset className="build-tool">
              <legend>Database Migration Tool</legend>
              <div className="build-options">
                {([
                  ['flyway', 'Flyway', 'Versioned SQL migrations'],
                  ['liquibase', 'Liquibase', 'Flexible database change tracking'],
                ] as const).map(([value, label, description]) => (
                  <label className={database.migrationTool === value ? 'selected' : ''} key={value}>
                    <input type="radio" name="migrationTool" value={value} checked={database.migrationTool === value} onChange={() => updateDatabase('migrationTool', value)} />
                    <span><strong>{label}</strong><small>{description}</small></span>
                  </label>
                ))}
              </div>
            </fieldset>

            <div className="form-actions">
              <button type="button" className="back-button" onClick={() => setActiveStep(2)}>Back</button>
              <button type="submit" className="next-button">Next <span aria-hidden="true">→</span></button>
            </div>
          </form>
        )}

        {activeStep === 4 && savedDatabase && (
          <form className="project-form features-form" onSubmit={handleFeaturesSubmit}>
            <div className="form-heading">
              <p className="eyebrow">Step 5 of 7</p>
              <h1>Features</h1>
              <p>Select any additional capabilities to include in your generated application.</p>
            </div>

            <fieldset className="feature-selector">
              <legend>Optional Features</legend>
              <div className="feature-options">
                {featureOptions.map(([value, label, description]) => {
                  const selected = features.includes(value)
                  return (
                    <label className={selected ? 'selected' : ''} key={value}>
                      <input type="checkbox" value={value} checked={selected} onChange={() => toggleFeature(value)} />
                      <span><strong>{label}</strong><small>{description}</small></span>
                    </label>
                  )
                })}
              </div>
            </fieldset>

            <fieldset className="feature-selector">
              <legend>Spring AI Models (optional)</legend>
              <div className="feature-options">
                {aiModelOptions.map(([value, label, description]) => {
                  const selected = aiModels.includes(value)
                  return (
                    <label className={selected ? 'selected' : ''} key={value}>
                      <input type="checkbox" value={value} checked={selected} onChange={() => toggleAIModel(value)} />
                      <span><strong>{label}</strong><small>{description}</small></span>
                    </label>
                  )
                })}
              </div>
            </fieldset>

            <div className="form-actions">
              <button type="button" className="back-button" onClick={() => setActiveStep(3)}>Back</button>
              <button type="submit" className="next-button">Next <span aria-hidden="true">→</span></button>
            </div>
          </form>
        )}

        {activeStep === 5 && savedMetadata && savedFrontend && savedDatabase && savedFeatures && savedAIModels && (
          <div className="project-form review-form">
            <div className="form-heading">
              <p className="eyebrow">Step 6 of 7</p>
              <h1>Review</h1>
              <p>Review your selections before downloading the project configuration.</p>
            </div>

            <div className="review-grid">
              <section className="review-section">
                <h2>Project</h2>
                <dl>
                  <div><dt>Name</dt><dd>{savedMetadata.appName}</dd></div>
                  <div><dt>Version</dt><dd>{savedMetadata.appVersion}</dd></div>
                  <div><dt>Group ID</dt><dd>{savedMetadata.groupId}</dd></div>
                  <div><dt>Artifact ID</dt><dd>{savedMetadata.artifactId}</dd></div>
                  <div><dt>Package</dt><dd>{savedMetadata.packageName}</dd></div>
                  <div><dt>Build Tool</dt><dd>{labels[savedMetadata.buildTool]}</dd></div>
                  <div><dt>Task Runner</dt><dd>{labels[savedMetadata.taskRunner]}</dd></div>
                </dl>
              </section>

              <section className="review-section">
                <h2>Frontend</h2>
                <dl>
                  <div><dt>Frontend</dt><dd>{labels[savedFrontend.frontend]}</dd></div>
                  <div><dt>HTMX Support</dt><dd>{isServerRendered(savedFrontend.frontend) && savedFrontend.htmxSupport ? 'Enabled' : 'Disabled'}</dd></div>
                </dl>
              </section>

              <section className="review-section">
                <h2>Database</h2>
                <dl>
                  <div><dt>Database</dt><dd>{labels[savedDatabase.engine]}</dd></div>
                  <div><dt>Persistence</dt><dd>{labels[savedDatabase.persistenceFramework]}</dd></div>
                  <div><dt>Migration</dt><dd>{labels[savedDatabase.migrationTool]}</dd></div>
                </dl>
              </section>

              <section className="review-section review-features">
                <h2>Features</h2>
                <div className="review-tags">
                  {savedFeatures.length === 0
                    ? <span className="empty-tag">None</span>
                    : savedFeatures.map((feature) => <span key={feature}>{labels[feature]}</span>)}
                </div>
              </section>

              <section className="review-section review-features">
                <h2>Spring AI Models</h2>
                <div className="review-tags">
                  {savedAIModels.length === 0
                    ? <span className="empty-tag">None</span>
                    : savedAIModels.map((model) => <span key={model}>{labels[model]}</span>)}
                </div>
              </section>
            </div>

            {generationError && <p className="generation-error" role="alert">{generationError}</p>}
            <div className="form-actions">
              <button type="button" className="back-button" onClick={() => setActiveStep(4)} disabled={isGenerating}>Back</button>
              <button type="button" className="next-button generate-button" onClick={handleGenerate} disabled={isGenerating}>{isGenerating ? 'Generating…' : 'Generate'} <span aria-hidden="true">↓</span></button>
            </div>
          </div>
        )}

        {activeStep === 6 && downloadedFilePath && (
          <div className="complete-step">
            <div className="welcome-icon" aria-hidden="true"><svg viewBox="0 0 32 32"><path d="M7 16.5 13 22l12-13" /></svg></div>
            <p className="eyebrow">Complete</p>
            <h1>Project downloaded</h1>
            <p className="description">Your Spring Boot {savedMetadata ? labels[savedMetadata.buildTool] : ''} project was generated and saved as a ZIP archive.</p>
            <div className="download-location">
              <span>Downloaded project</span>
              <code>{downloadedFilePath}</code>
            </div>
            <button type="button" className="continue-button" onClick={handleStartNew}>Start New</button>
          </div>
        )}
      </section>
    </main>
  )
}

export default App

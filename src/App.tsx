import { FormEvent, useEffect, useRef, useState } from 'react'
// Generated from the Go code by `mygo generate`.
import {
  AppGeneratorService,
  DefaultsService,
  DialogService,
  type AIModel,
  type BuildTool,
  type ConfirmOptions,
  type DatabaseSettings,
  type Feature,
  type FrontendType as Frontend,
  type GeneratorConfiguration,
  type ProjectDefaults,
  type ProjectSettings,
  type TaskRunner,
  type Template,
  TemplateService,
} from './mygo'
import { isMyGo } from 'mygo-runtime'

const authorUrl = 'https://sivalabs.in'
const repositoryUrl = 'https://github.com/sivaprasadreddy/sbgen'

const steps = ['Welcome', 'Project', 'Frontend', 'Database', 'Features', 'Review', 'Complete']

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

type ProjectMetadata = ProjectSettings

// The defaults until the user saves their own on the Defaults page; DefaultsService has the same.
const builtInDefaults: ProjectDefaults = {
  appVersion: '1.0.0',
  groupId: 'dev.sivalabs',
  buildTool: 'maven',
  taskRunner: 'mise',
}

// The Project step's starting values for a new application.
const metadataFromDefaults = (defaults: ProjectDefaults): ProjectMetadata => ({
  appName: 'demo',
  appVersion: defaults.appVersion,
  groupId: defaults.groupId,
  artifactId: 'demo',
  packageName: deriveBasePackage(defaults.groupId, 'demo'),
  buildTool: defaults.buildTool,
  taskRunner: defaults.taskRunner,
})

// metadata with field changed. The package name follows the group and artifact IDs until the user customizes it.
const withMetadataField = <K extends keyof ProjectMetadata>(current: ProjectMetadata, field: K, value: ProjectMetadata[K]): ProjectMetadata => {
  const next = { ...current, [field]: value }
  if ((field === 'groupId' || field === 'artifactId') && current.packageName === deriveBasePackage(current.groupId, current.artifactId)) {
    next.packageName = deriveBasePackage(next.groupId, next.artifactId)
  }
  return next
}

// The Project fields the Review page can change, with their placeholders.
const reviewProjectFields: ReadonlyArray<readonly [keyof ProjectMetadata, string, string]> = [
  ['appName', 'Name', 'boot-demo'],
  ['appVersion', 'Version', '1.0.0'],
  ['groupId', 'Group ID', 'com.example'],
  ['artifactId', 'Artifact ID', 'my-application'],
  ['packageName', 'Package', 'com.example.demo'],
]

const sameMetadata = (a: ProjectMetadata, b: ProjectMetadata) =>
  (Object.keys(a) as (keyof ProjectMetadata)[]).every((key) => a[key] === b[key])

const buildToolOptions: ReadonlyArray<readonly [BuildTool, string, string]> = [
  ['maven', 'Maven', 'pom.xml'],
  ['gradle', 'Gradle', 'build.gradle'],
]

const taskRunnerOptions: ReadonlyArray<readonly [TaskRunner, string, string]> = [
  ['mise', 'Mise', 'mise.toml installs the JDK and runs tasks'],
  ['taskfile', 'Taskfile', 'Taskfile.yml with .sdkmanrc for SDKMAN'],
]

// The pages the sidebar opens in place of the current step, which they return to.
type Page = 'defaults' | 'templates'

type FrontendConfiguration = {
  frontend: Frontend
  htmxSupport: boolean
}

const initialFrontendConfiguration: FrontendConfiguration = {
  frontend: 'none',
  htmxSupport: false,
}

type DatabaseConfiguration = DatabaseSettings

const initialDatabaseConfiguration: DatabaseConfiguration = {
  engine: 'postgresql',
  migrationTool: 'flyway',
  persistenceFramework: 'spring-data-jpa',
}

// The message of a rejected Go call (its error's text), or of any other failure.
const errorMessage = (error: unknown) => error instanceof Error ? error.message : String(error)

// Asks the user to confirm with MyGo's native dialog, a sheet of the app's window. Outside a MyGo
// window (the dev server in a browser tab), or if the dialog fails, it asks with the browser's confirm().
const confirmAction = async (options: ConfirmOptions): Promise<boolean> => {
  if (isMyGo()) {
    try {
      return await DialogService.confirm(options)
    } catch (error) {
      console.error('native confirmation dialog failed', error)
    }
  }
  return window.confirm(options.detail ? `${options.message}\n\n${options.detail}` : options.message)
}

// A one-line description of a template's main choices, e.g. "Maven · JTE + HTMX · PostgreSQL · Spring Data JPA · 2 features".
const templateSummary = ({ project, frontend, database, features, aiModels }: GeneratorConfiguration) => {
  const count = (n: number, noun: string) => n > 0 ? `${n} ${noun}${n === 1 ? '' : 's'}` : ''
  return [
    labels[project.buildTool],
    frontend.type === 'none' ? 'REST API' : labels[frontend.type] + (frontend.htmxSupport ? ' + HTMX' : ''),
    labels[database.engine],
    labels[database.persistenceFramework],
    count(features.length, 'feature'),
    count(aiModels.length, 'AI model'),
  ].filter(Boolean).join(' · ')
}

function App() {
  const [activeStep, setActiveStep] = useState(0)
  const [metadata, setMetadata] = useState<ProjectMetadata>(() => metadataFromDefaults(builtInDefaults))
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
  const [templates, setTemplates] = useState<Template[]>([])
  const [templatesError, setTemplatesError] = useState('')
  // The template the current selection was loaded from or last saved as.
  const [activeTemplate, setActiveTemplate] = useState<Template | null>(null)
  const [page, setPage] = useState<Page | null>(null)
  const [projectDefaults, setProjectDefaults] = useState<ProjectDefaults>(builtInDefaults)
  const [defaultsDraft, setDefaultsDraft] = useState<ProjectDefaults>(builtInDefaults)
  const [defaultsMessage, setDefaultsMessage] = useState<{ kind: 'success' | 'error', text: string } | null>(null)
  const [isSavingDefaults, setIsSavingDefaults] = useState(false)
  const [isTemplatePanelOpen, setIsTemplatePanelOpen] = useState(false)
  const [templateName, setTemplateName] = useState('')
  const [templateMessage, setTemplateMessage] = useState<{ kind: 'success' | 'error', text: string } | null>(null)
  const [isSavingTemplate, setIsSavingTemplate] = useState(false)

  const refreshTemplates = async () => {
    try {
      setTemplates(await TemplateService.list())
      setTemplatesError('')
    } catch (error) {
      setTemplatesError(`Could not load templates: ${errorMessage(error)}`)
    }
  }

  // Moves a Project step that still shows the previous defaults to the next ones. One the user
  // changed, confirmed with Next, or filled from a template keeps its values.
  const followDefaults = (previous: ProjectDefaults, next: ProjectDefaults) => {
    if (savedMetadata || activeTemplate) return
    setMetadata((current) => sameMetadata(current, metadataFromDefaults(previous)) ? metadataFromDefaults(next) : current)
  }

  useEffect(() => {
    void refreshTemplates()
    DefaultsService.get().then((saved) => {
      setProjectDefaults(saved)
      followDefaults(builtInDefaults, saved)
    }, (error) => {
      // New projects then start from the built-in defaults.
      setDefaultsMessage({ kind: 'error', text: `Could not load defaults: ${errorMessage(error)}` })
    })
  }, [])

  const updateMetadata = <K extends keyof ProjectMetadata>(field: K, value: ProjectMetadata[K]) => {
    setMetadata((current) => withMetadataField(current, field, value))
  }

  // Changes a confirmed Project field from the Review page. Generate and Save as template use the
  // confirmed values; the Project step shows the same ones when the user goes back.
  const reviewProjectForm = useRef<HTMLFormElement>(null)
  const updateReviewMetadata = <K extends keyof ProjectMetadata>(field: K, value: ProjectMetadata[K]) => {
    if (!savedMetadata) return
    const next = withMetadataField(savedMetadata, field, value)
    setSavedMetadata(next)
    setMetadata(next)
    setGenerationError('')
  }

  // Shows the browser's message on the first empty Review field, and reports whether all are filled in.
  const reviewProjectValid = () => reviewProjectForm.current?.reportValidity() ?? true

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

  // The selection confirmed in steps 2–5, or null until all of them are.
  const currentConfiguration = (): GeneratorConfiguration | null => {
    if (!savedMetadata || !savedFrontend || !savedDatabase || !savedFeatures || !savedAIModels) return null
    return {
      project: savedMetadata,
      frontend: {
        type: savedFrontend.frontend,
        htmxSupport: isServerRendered(savedFrontend.frontend) && savedFrontend.htmxSupport,
      },
      database: savedDatabase,
      features: savedFeatures,
      aiModels: savedAIModels,
    }
  }

  // Loads a template's selection into every step, then shows the given step.
  const applyTemplate = (template: Template, step: number) => {
    const { project, frontend: frontendSettings, database: databaseSettings, features: templateFeatures, aiModels: templateAIModels } = template.configuration
    const frontendConfiguration = { frontend: frontendSettings.type, htmxSupport: frontendSettings.htmxSupport }
    const orderedAIModels = aiModelOptions.map(([value]) => value).filter((model) => templateAIModels.includes(model))
    setMetadata({ ...project })
    setSavedMetadata({ ...project })
    setFrontend(frontendConfiguration)
    setSavedFrontend(frontendConfiguration)
    setDatabase({ ...databaseSettings })
    setSavedDatabase({ ...databaseSettings })
    setFeatures([...templateFeatures])
    setSavedFeatures([...templateFeatures])
    setAIModels(orderedAIModels)
    setSavedAIModels(orderedAIModels)
    setActiveTemplate(template)
    setTemplateName(template.name)
    setIsTemplatePanelOpen(false)
    setTemplateMessage(null)
    setGenerationError('')
    setPage(null)
    setActiveStep(step)
  }

  const openPage = (next: Page) => {
    if (next === 'templates') void refreshTemplates()
    if (next === 'defaults') {
      setDefaultsDraft(projectDefaults)
      setDefaultsMessage(null)
    }
    setPage(next)
  }

  const closePage = () => {
    setPage(null)
  }

  const updateDefaultsDraft = <K extends keyof ProjectDefaults>(field: K, value: ProjectDefaults[K]) => {
    setDefaultsDraft((current) => ({ ...current, [field]: value }))
    setDefaultsMessage(null)
  }

  const handleDefaultsSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    setIsSavingDefaults(true)
    setDefaultsMessage(null)
    try {
      const saved = await DefaultsService.save(defaultsDraft)
      followDefaults(projectDefaults, saved)
      setProjectDefaults(saved)
      setDefaultsDraft(saved)
      setDefaultsMessage({ kind: 'success', text: 'Saved. New projects start with these values.' })
    } catch (error) {
      setDefaultsMessage({ kind: 'error', text: errorMessage(error) })
    } finally {
      setIsSavingDefaults(false)
    }
  }

  const resetSelection = () => {
    setMetadata(metadataFromDefaults(projectDefaults))
    setSavedMetadata(null)
    setFrontend(initialFrontendConfiguration)
    setSavedFrontend(null)
    setDatabase(initialDatabaseConfiguration)
    setSavedDatabase(null)
    setFeatures([])
    setSavedFeatures(null)
    setAIModels([])
    setSavedAIModels(null)
    setActiveTemplate(null)
    setTemplateName('')
    setIsTemplatePanelOpen(false)
    setTemplateMessage(null)
  }

  const handleGetStarted = () => {
    // Starting without a template begins from the defaults, not from a template looked at earlier.
    if (activeTemplate) resetSelection()
    setActiveStep(1)
  }

  const handleDeleteTemplate = async (template: Template) => {
    const confirmed = await confirmAction({
      message: `Delete the template “${template.name}”?`,
      detail: 'It is removed from your templates. This can’t be undone.',
      confirmLabel: 'Delete',
      destructive: true,
    })
    if (!confirmed) return
    try {
      await TemplateService.delete(template.id)
      if (activeTemplate?.id === template.id) setActiveTemplate(null)
      setTemplatesError('')
      await refreshTemplates()
    } catch (error) {
      setTemplatesError(`Could not delete “${template.name}”: ${errorMessage(error)}`)
    }
  }

  // Saves the current selection as a new template, or over activeTemplate when update is true.
  const handleSaveTemplate = async (update: boolean) => {
    const configuration = currentConfiguration()
    if (!configuration || !reviewProjectValid()) return
    setIsSavingTemplate(true)
    setTemplateMessage(null)
    try {
      const saved = await TemplateService.save({
        id: update && activeTemplate ? activeTemplate.id : '',
        name: templateName,
        configuration,
        updatedAt: new Date().toISOString(),
      })
      setActiveTemplate(saved)
      setTemplateName(saved.name)
      setTemplateMessage({ kind: 'success', text: update ? `Updated template “${saved.name}”.` : `Saved template “${saved.name}”.` })
      await refreshTemplates()
    } catch (error) {
      setTemplateMessage({ kind: 'error', text: errorMessage(error) })
    } finally {
      setIsSavingTemplate(false)
    }
  }

  const handleGenerate = async () => {
    const configuration = currentConfiguration()
    if (!configuration || !reviewProjectValid()) return

    setIsGenerating(true)
    setGenerationError('')
    try {
      const filePath = await AppGeneratorService.generateProject(configuration)
      setDownloadedFilePath(filePath)
      setActiveStep(6)
    } catch (error) {
      setGenerationError(errorMessage(error))
    } finally {
      setIsGenerating(false)
    }
  }

  // Cancel on steps 2–6: after the user confirms, drops every selection, including a loaded
  // template, and returns to Welcome.
  const handleCancel = async () => {
    const confirmed = await confirmAction({
      message: 'Discard this project’s selections?',
      detail: 'Every choice you made for this project is cleared and you return to Welcome. Your templates and defaults are kept.',
      confirmLabel: 'Discard',
      destructive: true,
    })
    if (!confirmed) return
    resetSelection()
    setGenerationError('')
    setActiveStep(0)
  }

  const handleStartNew = () => {
    resetSelection()
    setDownloadedFilePath('')
    setGenerationError('')
    setIsGenerating(false)
    setActiveStep(0)
  }

  const shownStep = page ? -1 : activeStep

  // The templates with their Update and Delete actions, on the Welcome step and the Templates page.
  const renderTemplateList = () => (
    <ul className="template-items">
      {templates.map((template) => (
        <li key={template.id}>
          <button type="button" className="template-open" onClick={() => applyTemplate(template, 5)} title={`Review “${template.name}”`}>
            <strong>{template.name}</strong>
            <small>{templateSummary(template.configuration)}</small>
          </button>
          <div className="template-actions">
            <button type="button" className="template-action" onClick={() => applyTemplate(template, 1)} title="Change this template’s selections, starting from the Project step">Update</button>
            <button type="button" className="template-action" onClick={() => void handleDeleteTemplate(template)}>Delete</button>
          </div>
        </li>
      ))}
    </ul>
  )

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
              <li className={isComplete ? 'complete' : index === shownStep ? 'active' : ''} key={step} aria-current={index === shownStep && !isComplete ? 'step' : undefined}>
                <span className="step-number">{isComplete ? '✓' : index + 1}</span><span>{step}</span>
              </li>
            )
          })}
        </ol>
        <hr className="sidebar-separator" />
        <nav aria-label="Library">
          <button
            type="button"
            className={`sidebar-link${page === 'defaults' ? ' active' : ''}`}
            aria-current={page === 'defaults' ? 'page' : undefined}
            onClick={page === 'defaults' ? closePage : () => openPage('defaults')}
            disabled={isGenerating}
            title="Set the values new projects start with"
          >
            <span className="step-number" aria-hidden="true">
              <svg viewBox="0 0 16 16"><path d="M3 4.5h10M3 11.5h10" /><circle cx="6" cy="4.5" r="1.6" /><circle cx="10.5" cy="11.5" r="1.6" /></svg>
            </span>
            <span className="sidebar-link-label">Defaults</span>
          </button>
          <button
            type="button"
            className={`sidebar-link${page === 'templates' ? ' active' : ''}`}
            aria-current={page === 'templates' ? 'page' : undefined}
            onClick={page === 'templates' ? closePage : () => openPage('templates')}
            disabled={isGenerating}
            title="Show all templates"
          >
            <span className="step-number" aria-hidden="true">
              <svg viewBox="0 0 16 16"><path d="M3 4.5h10M3 8h10M3 11.5h6" /></svg>
            </span>
            <span className="sidebar-link-label">Templates</span>
            {templates.length > 0 && <span className="sidebar-count">{templates.length}</span>}
          </button>
        </nav>
        <footer className="sidebar-footer">
          {/* target="_blank" links open in the default browser, not the app's window. */}
          <p>Author: <a href={authorUrl} target="_blank" rel="noreferrer">SivaLabs</a></p>
          <p><a href={repositoryUrl} target="_blank" rel="noreferrer">GitHub repository ↗</a></p>
        </footer>
      </aside>

      <section className="content-panel">
        {shownStep === 0 && (
          <div className="welcome-content" aria-labelledby="welcome-title">
            <div className="welcome-icon" aria-hidden="true"><svg viewBox="0 0 32 32"><path d="M7 16.5 13 22l12-13" /></svg></div>
            <p className="eyebrow">Welcome</p>
            <h1 id="welcome-title">Build production-ready<br />Spring Boot applications</h1>
            <p className="description">Create a thoughtfully structured Spring Boot project with sensible defaults and the dependencies your application needs—all through one simple, guided setup.</p>
            <button type="button" className="continue-button" onClick={handleGetStarted}>Get started</button>

            {templatesError && <p className="generation-error templates-error" role="alert">{templatesError}</p>}
            {templates.length > 0 && (
              <section className="template-list" aria-labelledby="templates-title">
                <h2 id="templates-title">Or start from a template</h2>
                {renderTemplateList()}
              </section>
            )}
          </div>
        )}

        {page === 'templates' && (
          <div className="project-form templates-page" aria-labelledby="templates-page-title">
            <div className="form-heading">
              <p className="eyebrow">{templates.length === 1 ? '1 template' : `${templates.length} templates`}</p>
              <h1 id="templates-page-title">Templates</h1>
              <p>Start a project from a saved selection, or update and delete your templates. Save new ones from the Review step.</p>
            </div>
            {templatesError && <p className="generation-error templates-error" role="alert">{templatesError}</p>}
            {templates.length > 0
              ? renderTemplateList()
              : <p className="templates-empty">No templates yet. Make your selections, then use <strong>Save as template</strong> on the Review step.</p>}
            <div className="form-actions">
              <button type="button" className="back-button" onClick={closePage}>Back to {steps[activeStep]}</button>
            </div>
          </div>
        )}

        {page === 'defaults' && (
          <form className="project-form defaults-page" aria-labelledby="defaults-page-title" onSubmit={handleDefaultsSubmit}>
            <div className="form-heading">
              <p className="eyebrow">Defaults</p>
              <h1 id="defaults-page-title">Project Defaults</h1>
              <p>New projects start with these values on the Project step. Templates keep their own.</p>
            </div>

            <div className="field-grid">
              <label className="field">
                <span>Application Version</span>
                <input required value={defaultsDraft.appVersion} onChange={(event) => updateDefaultsDraft('appVersion', event.target.value)} placeholder="1.0.0" autoComplete="off" />
                <small>The initial version of new applications.</small>
              </label>
              <label className="field">
                <span>Group ID</span>
                <input required value={defaultsDraft.groupId} onChange={(event) => updateDefaultsDraft('groupId', event.target.value)} placeholder="com.example" autoComplete="off" />
                <small>Also the start of the package name.</small>
              </label>
            </div>

            <fieldset className="build-tool">
              <legend>Build Tool</legend>
              <div className="build-options">
                {buildToolOptions.map(([value, label, file]) => (
                  <label className={defaultsDraft.buildTool === value ? 'selected' : ''} key={value}>
                    <input type="radio" name="defaultBuildTool" value={value} checked={defaultsDraft.buildTool === value} onChange={() => updateDefaultsDraft('buildTool', value)} />
                    <span><strong>{label}</strong><small>{file}</small></span>
                  </label>
                ))}
              </div>
            </fieldset>

            <fieldset className="build-tool">
              <legend>Task Runner</legend>
              <div className="build-options">
                {taskRunnerOptions.map(([value, label, description]) => (
                  <label className={defaultsDraft.taskRunner === value ? 'selected' : ''} key={value}>
                    <input type="radio" name="defaultTaskRunner" value={value} checked={defaultsDraft.taskRunner === value} onChange={() => updateDefaultsDraft('taskRunner', value)} />
                    <span><strong>{label}</strong><small>{description}</small></span>
                  </label>
                ))}
              </div>
            </fieldset>

            {defaultsMessage && <p className={`template-message defaults-message ${defaultsMessage.kind}`} role={defaultsMessage.kind === 'error' ? 'alert' : 'status'}>{defaultsMessage.text}</p>}
            <div className="form-actions">
              <button
                type="button"
                className="back-button template-toggle"
                onClick={() => {
                  setDefaultsDraft(builtInDefaults)
                  setDefaultsMessage(null)
                }}
                disabled={isSavingDefaults}
                title="Fill in SBGen's own defaults; Save to keep them"
              >
                Restore built-in values
              </button>
              <button type="button" className="back-button" onClick={closePage} disabled={isSavingDefaults}>Back to {steps[activeStep]}</button>
              <button type="submit" className="next-button" disabled={isSavingDefaults}>{isSavingDefaults ? 'Saving…' : 'Save defaults'}</button>
            </div>
          </form>
        )}

        {shownStep === 1 && (
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
                {buildToolOptions.map(([value, label, file]) => (
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
                {taskRunnerOptions.map(([value, label, description]) => (
                  <label className={metadata.taskRunner === value ? 'selected' : ''} key={value}>
                    <input type="radio" name="taskRunner" value={value} checked={metadata.taskRunner === value} onChange={() => updateMetadata('taskRunner', value)} />
                    <span><strong>{label}</strong><small>{description}</small></span>
                  </label>
                ))}
              </div>
            </fieldset>

            <div className="form-actions">
              <button type="button" className="back-button cancel-button" onClick={() => void handleCancel()} title="Clear all selections and return to Welcome">Cancel</button>
              <button type="button" className="back-button" onClick={() => setActiveStep(0)}>Back</button>
              <button type="submit" className="next-button">Next <span aria-hidden="true">→</span></button>
            </div>
          </form>
        )}

        {shownStep === 2 && savedMetadata && (
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
              <button type="button" className="back-button cancel-button" onClick={() => void handleCancel()} title="Clear all selections and return to Welcome">Cancel</button>
              <button type="button" className="back-button" onClick={() => setActiveStep(1)}>Back</button>
              <button type="submit" className="next-button">Next <span aria-hidden="true">→</span></button>
            </div>
          </form>
        )}

        {shownStep === 3 && savedMetadata && savedFrontend && (
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
              <button type="button" className="back-button cancel-button" onClick={() => void handleCancel()} title="Clear all selections and return to Welcome">Cancel</button>
              <button type="button" className="back-button" onClick={() => setActiveStep(2)}>Back</button>
              <button type="submit" className="next-button">Next <span aria-hidden="true">→</span></button>
            </div>
          </form>
        )}

        {shownStep === 4 && savedDatabase && (
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
              <button type="button" className="back-button cancel-button" onClick={() => void handleCancel()} title="Clear all selections and return to Welcome">Cancel</button>
              <button type="button" className="back-button" onClick={() => setActiveStep(3)}>Back</button>
              <button type="submit" className="next-button">Next <span aria-hidden="true">→</span></button>
            </div>
          </form>
        )}

        {shownStep === 5 && savedMetadata && savedFrontend && savedDatabase && savedFeatures && savedAIModels && (
          <div className="project-form review-form">
            <div className="form-heading">
              <p className="eyebrow">Step 6 of 7</p>
              <h1>Review</h1>
              <p>Review your selections and adjust the project’s name and coordinates before generating it.</p>
              {activeTemplate && <p className="template-origin">From template <strong>{activeTemplate.name}</strong></p>}
            </div>

            <div className="review-grid">
              <section className="review-section review-project">
                <h2>Project</h2>
                <form ref={reviewProjectForm} className="review-project-fields" onSubmit={(event) => event.preventDefault()}>
                  <dl>
                    {reviewProjectFields.map(([field, label, placeholder]) => (
                      <div key={field}>
                        <dt><label htmlFor={`review-${field}`}>{label}</label></dt>
                        <dd>
                          <input
                            id={`review-${field}`}
                            required
                            value={savedMetadata[field]}
                            onChange={(event) => updateReviewMetadata(field, event.target.value)}
                            placeholder={placeholder}
                            autoComplete="off"
                            spellCheck={false}
                          />
                        </dd>
                      </div>
                    ))}
                  </dl>
                </form>
                <dl>
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

            {isTemplatePanelOpen && (
              <form
                className="template-panel"
                onSubmit={(event) => {
                  event.preventDefault()
                  void handleSaveTemplate(Boolean(activeTemplate))
                }}
              >
                <label className="field">
                  <span>Template Name</span>
                  <input required autoFocus value={templateName} onChange={(event) => setTemplateName(event.target.value)} placeholder="e.g. JTE + PostgreSQL" autoComplete="off" />
                  <small>Saved templates are listed on the Welcome step.</small>
                </label>
                <div className="template-panel-actions">
                  {activeTemplate ? (
                    <>
                      <button type="submit" className="next-button" disabled={isSavingTemplate}>Update “{activeTemplate.name}”</button>
                      <button type="button" className="back-button" disabled={isSavingTemplate} onClick={() => void handleSaveTemplate(false)}>Save as new</button>
                    </>
                  ) : (
                    <button type="submit" className="next-button" disabled={isSavingTemplate}>Save template</button>
                  )}
                </div>
                {templateMessage && <p className={`template-message ${templateMessage.kind}`} role={templateMessage.kind === 'error' ? 'alert' : 'status'}>{templateMessage.text}</p>}
              </form>
            )}

            {generationError && <p className="generation-error" role="alert">{generationError}</p>}
            <div className="form-actions">
              <button type="button" className="back-button cancel-button" onClick={() => void handleCancel()} disabled={isGenerating} title="Clear all selections and return to Welcome">Cancel</button>
              <button
                type="button"
                className="back-button template-toggle"
                aria-expanded={isTemplatePanelOpen}
                onClick={() => {
                  setIsTemplatePanelOpen((open) => !open)
                  setTemplateMessage(null)
                }}
                disabled={isGenerating}
              >
                {isTemplatePanelOpen ? 'Close' : 'Save as template'}
              </button>
              <button
                type="button"
                className="back-button"
                onClick={() => {
                  // A message about the selection goes stale once the selection can change.
                  setIsTemplatePanelOpen(false)
                  setTemplateMessage(null)
                  setActiveStep(4)
                }}
                disabled={isGenerating}
              >
                Back
              </button>
              <button type="button" className="next-button generate-button" onClick={handleGenerate} disabled={isGenerating}>{isGenerating ? 'Generating…' : 'Generate'} <span aria-hidden="true">↓</span></button>
            </div>
          </div>
        )}

        {shownStep === 6 && downloadedFilePath && (
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

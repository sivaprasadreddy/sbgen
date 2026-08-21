import { FormEvent, useState } from 'react'
import { AppGeneratorService } from '../bindings/github.com/sivaprasadreddy/sbgen'

const steps = ['Welcome', 'Project', 'Frontend', 'Database', 'Security', 'Features', 'Review', 'Complete']

type BuildTool = 'maven' | 'gradle-groovy'
type Frontend = 'none' | 'thymeleaf' | 'angular'
type CssFramework = 'tailwind'
type DatabaseEngine = 'postgresql' | 'mysql' | 'mariadb' | 'h2'
type MigrationTool = 'none' | 'flyway' | 'liquibase'
type PersistenceFramework = 'spring-data-jpa'
type SecurityMode = 'http-session' | 'jwt-security'
type Feature = 'email' | 'docker-compose' | 'spring-modulith' | 'spring-cloud-aws'

const featureOptions: ReadonlyArray<readonly [Feature, string, string]> = [
  ['spring-modulith', 'Spring Modulith', 'Build well-structured modular Spring applications'],
  ['spring-cloud-aws', 'Spring Cloud AWS', 'Integrate with AWS services using Spring'],
  ['email', 'Email', 'Send transactional and application emails'],
  ['docker-compose', 'Docker Compose', 'Run supporting services for local development'],
]

const labels: Record<string, string> = {
  maven: 'Maven',
  'gradle-groovy': 'Gradle – Groovy',
  none: 'None',
  thymeleaf: 'Thymeleaf',
  angular: 'Angular',
  tailwind: 'Tailwind CSS',
  postgresql: 'PostgreSQL',
  mysql: 'MySQL',
  mariadb: 'MariaDB',
  h2: 'H2',
  flyway: 'Flyway',
  liquibase: 'Liquibase',
  'spring-data-jpa': 'Spring Data JPA',
  'http-session': 'HTTP Session based Security',
  'jwt-security': 'JWT Security',
  email: 'Email',
  'docker-compose': 'Docker Compose',
  'spring-modulith': 'Spring Modulith',
  'spring-cloud-aws': 'Spring Cloud AWS',
}

type ProjectMetadata = {
  groupId: string
  artifactId: string
  packageName: string
  buildTool: BuildTool
}

const initialMetadata: ProjectMetadata = {
  groupId: 'dev.sivalabs',
  artifactId: 'demo',
  packageName: 'dev.sivalabs.demo',
  buildTool: 'maven',
}

type FrontendConfiguration = {
  frontend: Frontend
  cssFramework: CssFramework
  htmxSupport: boolean
}

const initialFrontendConfiguration: FrontendConfiguration = {
  frontend: 'none',
  cssFramework: 'tailwind',
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

type SecurityConfiguration = {
  mode: SecurityMode
}

const initialSecurityConfiguration: SecurityConfiguration = {
  mode: 'jwt-security',
}

const defaultSecurityMode = (frontend: Frontend): SecurityMode =>
  frontend === 'thymeleaf' ? 'http-session' : 'jwt-security'

function App() {
  const [activeStep, setActiveStep] = useState(0)
  const [metadata, setMetadata] = useState<ProjectMetadata>(initialMetadata)
  const [savedMetadata, setSavedMetadata] = useState<ProjectMetadata | null>(null)
  const [frontend, setFrontend] = useState<FrontendConfiguration>(initialFrontendConfiguration)
  const [savedFrontend, setSavedFrontend] = useState<FrontendConfiguration | null>(null)
  const [database, setDatabase] = useState<DatabaseConfiguration>(initialDatabaseConfiguration)
  const [savedDatabase, setSavedDatabase] = useState<DatabaseConfiguration | null>(null)
  const [security, setSecurity] = useState<SecurityConfiguration>(initialSecurityConfiguration)
  const [savedSecurity, setSavedSecurity] = useState<SecurityConfiguration | null>(null)
  const [features, setFeatures] = useState<Feature[]>(['spring-modulith', 'docker-compose'])
  const [savedFeatures, setSavedFeatures] = useState<Feature[] | null>(null)
  const [downloadedFilePath, setDownloadedFilePath] = useState('')
  const [generationError, setGenerationError] = useState('')
  const [isGenerating, setIsGenerating] = useState(false)

  const updateMetadata = <K extends keyof ProjectMetadata>(field: K, value: ProjectMetadata[K]) => {
    setMetadata((current) => ({ ...current, [field]: value }))
  }

  const handleProjectSubmit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    setSavedMetadata({ ...metadata })
    setActiveStep(2)
  }

  const handleFrontendSubmit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    setSavedFrontend({ ...frontend })
    setSecurity({ mode: defaultSecurityMode(frontend.frontend) })
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

  const handleSecuritySubmit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    setSavedSecurity({ ...security })
    setActiveStep(5)
  }

  const toggleFeature = (feature: Feature) => {
    setFeatures((current) => current.includes(feature)
      ? current.filter((selected) => selected !== feature)
      : [...current, feature])
  }

  const handleFeaturesSubmit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    setSavedFeatures([...features])
    setActiveStep(6)
  }

  const handleGenerate = async () => {
    if (!savedMetadata || !savedFrontend || !savedDatabase || !savedSecurity || !savedFeatures) return

    const configuration = {
      project: savedMetadata,
      frontend: {
        type: savedFrontend.frontend,
        cssFramework: savedFrontend.frontend === 'none' ? null : savedFrontend.cssFramework,
        htmxSupport: savedFrontend.frontend === 'thymeleaf' && savedFrontend.htmxSupport,
      },
      database: savedDatabase,
      security: savedSecurity,
      features: savedFeatures,
    }

    setIsGenerating(true)
    setGenerationError('')
    try {
      const filePath = await AppGeneratorService.GenerateProject(JSON.stringify(configuration))
      setDownloadedFilePath(filePath)
      setActiveStep(7)
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
    setSecurity(initialSecurityConfiguration)
    setSavedSecurity(null)
    setFeatures(['spring-modulith', 'docker-compose'])
    setSavedFeatures(null)
    setDownloadedFilePath('')
    setGenerationError('')
    setIsGenerating(false)
    setActiveStep(0)
  }

  return (
    <main className="generator-shell">
      <aside className="sidebar" aria-label="Project generation steps">
        <div className="logo-wrap">
          <img src="/wails.png" alt="Wails" className="logo" />
          <span>Spring Boot Generator</span>
        </div>
        <ol className="steps">
          {steps.map((step, index) => (
            <li className={index === activeStep ? 'active' : index < activeStep ? 'complete' : ''} key={step} aria-current={index === activeStep ? 'step' : undefined}>
              <span className="step-number">{index < activeStep ? '✓' : index + 1}</span><span>{step}</span>
            </li>
          ))}
        </ol>
        <div className="sidebar-tools" aria-hidden="true"><span>△</span><span className="heart">♥</span><span>◔</span></div>
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
              <p className="eyebrow">Step 2 of 8</p>
              <h1>Project Metadata</h1>
              <p>Define the coordinates and build system for your new Spring Boot application.</p>
            </div>

            <div className="field-grid">
              <label className="field">
                <span>Group ID</span>
                <input required value={metadata.groupId} onChange={(event) => updateMetadata('groupId', event.target.value)} placeholder="com.example" autoComplete="off" />
                <small>Usually your organization’s reversed domain.</small>
              </label>
              <label className="field">
                <span>Artifact ID</span>
                <input required value={metadata.artifactId} onChange={(event) => updateMetadata('artifactId', event.target.value)} placeholder="my-application" autoComplete="off" />
                <small>The name of the generated project.</small>
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
                  ['gradle-groovy', 'Gradle – Groovy', 'build.gradle'],
                ] as const).map(([value, label, file]) => (
                  <label className={metadata.buildTool === value ? 'selected' : ''} key={value}>
                    <input type="radio" name="buildTool" value={value} checked={metadata.buildTool === value} onChange={() => updateMetadata('buildTool', value)} />
                    <span><strong>{label}</strong><small>{file}</small></span>
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
              <p className="eyebrow">Step 3 of 8</p>
              <h1>Frontend</h1>
              <p>Choose how the user interface for <strong>{savedMetadata.artifactId}</strong> should be built.</p>
            </div>

            <fieldset className="build-tool first-option-group">
              <legend>Frontend</legend>
              <div className="build-options">
                {([
                  ['none', 'None', 'Backend REST API only'],
                  ['thymeleaf', 'Thymeleaf', 'Server-rendered web application'],
                  ['angular', 'Angular', 'Single Page Application'],
                ] as const).map(([value, label, description]) => (
                  <label className={frontend.frontend === value ? 'selected' : ''} key={value}>
                    <input type="radio" name="frontend" value={value} checked={frontend.frontend === value} onChange={() => {
                      setFrontend((current) => ({
                        ...current,
                        frontend: value,
                        htmxSupport: value === 'thymeleaf' && current.htmxSupport,
                      }))
                      setSecurity({ mode: defaultSecurityMode(value) })
                    }} />
                    <span><strong>{label}</strong><small>{description}</small></span>
                  </label>
                ))}
              </div>
            </fieldset>

            {frontend.frontend !== 'none' && (
              <fieldset className="build-tool">
                <legend>CSS Framework</legend>
                <div className="build-options database-options">
                  {([
                    ['tailwind', 'Tailwind CSS', 'Utility-first CSS framework'],
                  ] as const).map(([value, label, description]) => (
                    <label className={frontend.cssFramework === value ? 'selected' : ''} key={value}>
                      <input type="radio" name="cssFramework" value={value} checked={frontend.cssFramework === value} onChange={() => setFrontend((current) => ({ ...current, cssFramework: value }))} />
                      <span><strong>{label}</strong><small>{description}</small></span>
                    </label>
                  ))}
                </div>
              </fieldset>
            )}

            {frontend.frontend === 'thymeleaf' && (
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
                    <span><strong>HTMX</strong><small>Add HTMX and its Spring Boot Thymeleaf integration</small></span>
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
              <p className="eyebrow">Step 4 of 8</p>
              <h1>Database</h1>
              <p>Choose the database, persistence framework, and schema migration tool for <strong>{savedMetadata.artifactId}</strong>.</p>
            </div>

            <fieldset className="build-tool first-option-group">
              <legend>Database</legend>
              <div className="build-options database-options">
                {([
                  ['postgresql', 'PostgreSQL', 'Reliable, advanced SQL database'],
                  ['mysql', 'MySQL', 'Popular open-source SQL database'],
                  ['mariadb', 'MariaDB', 'Community-developed MySQL alternative'],
                  ['h2', 'H2', 'Lightweight in-memory SQL database'],
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
                  ['none', 'None', 'No migration tool'],
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

        {activeStep === 4 && savedDatabase && savedFrontend && (
          <form className="project-form security-form" onSubmit={handleSecuritySubmit}>
            <div className="form-heading">
              <p className="eyebrow">Step 5 of 8</p>
              <h1>Security</h1>
              <p>Choose the security model that fits your selected frontend.</p>
            </div>

            <fieldset className="build-tool first-option-group">
              <legend>Security</legend>
              <div className="build-options security-options">
                {savedFrontend.frontend === 'thymeleaf' && (
                  <label className={security.mode === 'http-session' ? 'selected' : ''}>
                    <input type="radio" name="securityMode" value="http-session" checked={security.mode === 'http-session'} onChange={() => setSecurity((current) => ({ ...current, mode: 'http-session' }))} />
                    <span><strong>HTTP Session based Security</strong><small>Session-backed authentication for server-rendered pages</small></span>
                  </label>
                )}

                {(savedFrontend.frontend === 'none' || savedFrontend.frontend === 'angular') && (
                  <label className={security.mode === 'jwt-security' ? 'selected' : ''}>
                    <input type="radio" name="securityMode" value="jwt-security" checked={security.mode === 'jwt-security'} onChange={() => setSecurity({ mode: 'jwt-security' })} />
                    <span><strong>JWT Security</strong><small>Token-based security for API access</small></span>
                  </label>
                )}
              </div>
            </fieldset>

            <div className="form-actions">
              <button type="button" className="back-button" onClick={() => setActiveStep(3)}>Back</button>
              <button type="submit" className="next-button">Next <span aria-hidden="true">→</span></button>
            </div>
          </form>
        )}

        {activeStep === 5 && savedSecurity && (
          <form className="project-form features-form" onSubmit={handleFeaturesSubmit}>
            <div className="form-heading">
              <p className="eyebrow">Step 6 of 8</p>
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

            <div className="form-actions">
              <button type="button" className="back-button" onClick={() => setActiveStep(4)}>Back</button>
              <button type="submit" className="next-button">Next <span aria-hidden="true">→</span></button>
            </div>
          </form>
        )}

        {activeStep === 6 && savedMetadata && savedFrontend && savedDatabase && savedSecurity && savedFeatures && (
          <div className="project-form review-form">
            <div className="form-heading">
              <p className="eyebrow">Step 7 of 8</p>
              <h1>Review</h1>
              <p>Review your selections before downloading the project configuration.</p>
            </div>

            <div className="review-grid">
              <section className="review-section">
                <h2>Project</h2>
                <dl>
                  <div><dt>Group ID</dt><dd>{savedMetadata.groupId}</dd></div>
                  <div><dt>Artifact ID</dt><dd>{savedMetadata.artifactId}</dd></div>
                  <div><dt>Package</dt><dd>{savedMetadata.packageName}</dd></div>
                  <div><dt>Build Tool</dt><dd>{labels[savedMetadata.buildTool]}</dd></div>
                </dl>
              </section>

              <section className="review-section">
                <h2>Frontend</h2>
                <dl>
                  <div><dt>Frontend</dt><dd>{labels[savedFrontend.frontend]}</dd></div>
                  <div><dt>CSS Framework</dt><dd>{savedFrontend.frontend === 'none' ? 'Not applicable' : labels[savedFrontend.cssFramework]}</dd></div>
                  <div><dt>HTMX Support</dt><dd>{savedFrontend.frontend === 'thymeleaf' && savedFrontend.htmxSupport ? 'Enabled' : 'Disabled'}</dd></div>
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

              <section className="review-section">
                <h2>Security</h2>
                <dl>
                  <div><dt>Security</dt><dd>{labels[savedSecurity.mode]}</dd></div>
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
            </div>

            {generationError && <p className="generation-error" role="alert">{generationError}</p>}
            <div className="form-actions">
              <button type="button" className="back-button" onClick={() => setActiveStep(5)} disabled={isGenerating}>Back</button>
              <button type="button" className="next-button generate-button" onClick={handleGenerate} disabled={isGenerating}>{isGenerating ? 'Generating…' : 'Generate'} <span aria-hidden="true">↓</span></button>
            </div>
          </div>
        )}

        {activeStep === 7 && downloadedFilePath && (
          <div className="complete-step">
            <div className="welcome-icon" aria-hidden="true"><svg viewBox="0 0 32 32"><path d="M7 16.5 13 22l12-13" /></svg></div>
            <p className="eyebrow">Complete</p>
            <h1>Project downloaded</h1>
            <p className="description">Your Spring Boot Maven project was generated and saved as a ZIP archive.</p>
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

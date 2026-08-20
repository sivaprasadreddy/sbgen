import { FormEvent, useState } from 'react'

const steps = ['Welcome', 'Project', 'Database', 'Security', 'Features', 'Review', 'Complete']

type BuildTool = 'maven' | 'gradle-groovy' | 'gradle-kotlin'
type Frontend = 'none' | 'thymeleaf-tailwind' | 'angular-tailwind'
type DatabaseEngine = 'postgresql' | 'mysql' | 'mariadb' | 'h2'
type MigrationTool = 'none' | 'flyway' | 'liquibase'
type PersistenceFramework = 'spring-data-jpa' | 'spring-jdbc-client'
type SecurityMode = 'none' | 'http-session' | 'oauth-resource-server'
type Feature = 'email' | 'docker-compose' | 'spring-modulith' | 'spring-cloud-aws'

const featureOptions: ReadonlyArray<readonly [Feature, string, string]> = [
  ['spring-modulith', 'Spring Modulith', 'Build well-structured modular Spring applications'],
  ['spring-cloud-aws', 'Spring Cloud AWS', 'Integrate with AWS services using Spring'],
  ['email', 'Email', 'Send transactional and application emails'],
  ['docker-compose', 'Docker Compose', 'Run supporting services for local development'],
]

type ProjectMetadata = {
  groupId: string
  artifactId: string
  packageName: string
  buildTool: BuildTool
  frontend: Frontend
}

const initialMetadata: ProjectMetadata = {
  groupId: 'com.example',
  artifactId: 'demo',
  packageName: 'com.example.demo',
  buildTool: 'maven',
  frontend: 'none',
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
  springSession: boolean
}

const initialSecurityConfiguration: SecurityConfiguration = {
  mode: 'none',
  springSession: false,
}

function App() {
  const [activeStep, setActiveStep] = useState(0)
  const [metadata, setMetadata] = useState<ProjectMetadata>(initialMetadata)
  const [savedMetadata, setSavedMetadata] = useState<ProjectMetadata | null>(null)
  const [database, setDatabase] = useState<DatabaseConfiguration>(initialDatabaseConfiguration)
  const [savedDatabase, setSavedDatabase] = useState<DatabaseConfiguration | null>(null)
  const [security, setSecurity] = useState<SecurityConfiguration>(initialSecurityConfiguration)
  const [savedSecurity, setSavedSecurity] = useState<SecurityConfiguration | null>(null)
  const [features, setFeatures] = useState<Feature[]>(['spring-modulith', 'docker-compose'])
  const [savedFeatures, setSavedFeatures] = useState<Feature[] | null>(null)

  const updateMetadata = <K extends keyof ProjectMetadata>(field: K, value: ProjectMetadata[K]) => {
    setMetadata((current) => ({ ...current, [field]: value }))
  }

  const handleProjectSubmit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    setSavedMetadata({ ...metadata })
    setSecurity((current) => ({
      mode: metadata.frontend === 'thymeleaf-tailwind' && current.mode === 'http-session'
        ? current.mode
        : metadata.frontend === 'angular-tailwind' && current.mode === 'oauth-resource-server'
          ? current.mode
          : 'none',
      springSession: metadata.frontend === 'thymeleaf-tailwind' && current.springSession,
    }))
    setActiveStep(2)
  }

  const updateDatabase = <K extends keyof DatabaseConfiguration>(field: K, value: DatabaseConfiguration[K]) => {
    setDatabase((current) => ({ ...current, [field]: value }))
  }

  const handleDatabaseSubmit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    setSavedDatabase({ ...database })
    setActiveStep(3)
  }

  const handleSecuritySubmit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    setSavedSecurity({ ...security })
    setActiveStep(4)
  }

  const toggleFeature = (feature: Feature) => {
    setFeatures((current) => current.includes(feature)
      ? current.filter((selected) => selected !== feature)
      : [...current, feature])
  }

  const handleFeaturesSubmit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    setSavedFeatures([...features])
    setActiveStep(5)
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
              <p className="eyebrow">Step 2 of 7</p>
              <h1>Project Metadata</h1>
              <p>Define the coordinates, build system, and frontend for your new Spring Boot application.</p>
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
                  ['gradle-kotlin', 'Gradle – Kotlin', 'build.gradle.kts'],
                ] as const).map(([value, label, file]) => (
                  <label className={metadata.buildTool === value ? 'selected' : ''} key={value}>
                    <input type="radio" name="buildTool" value={value} checked={metadata.buildTool === value} onChange={() => updateMetadata('buildTool', value)} />
                    <span><strong>{label}</strong><small>{file}</small></span>
                  </label>
                ))}
              </div>
            </fieldset>

            <fieldset className="build-tool frontend-choice">
              <legend>Frontend</legend>
              <div className="build-options">
                {([
                  ['none', 'None', 'Backend only'],
                  ['thymeleaf-tailwind', 'Thymeleaf + TailwindCSS', 'Server-rendered UI'],
                  ['angular-tailwind', 'Angular + TailwindCSS', 'Single-page application'],
                ] as const).map(([value, label, description]) => (
                  <label className={metadata.frontend === value ? 'selected' : ''} key={value}>
                    <input type="radio" name="frontend" value={value} checked={metadata.frontend === value} onChange={() => updateMetadata('frontend', value)} />
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
          <form className="project-form database-form" onSubmit={handleDatabaseSubmit}>
            <div className="form-heading">
              <p className="eyebrow">Step 3 of 7</p>
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
                  ['spring-jdbc-client', 'Spring JdbcClient', 'Fluent SQL-oriented data access'],
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
              <button type="button" className="back-button" onClick={() => setActiveStep(1)}>Back</button>
              <button type="submit" className="next-button">Next <span aria-hidden="true">→</span></button>
            </div>
          </form>
        )}

        {activeStep === 3 && savedDatabase && savedMetadata && (
          <form className="project-form security-form" onSubmit={handleSecuritySubmit}>
            <div className="form-heading">
              <p className="eyebrow">Step 4 of 7</p>
              <h1>Security</h1>
              <p>Choose the security model that fits your selected frontend.</p>
            </div>

            <fieldset className="build-tool first-option-group">
              <legend>Security</legend>
              <div className="build-options security-options">
                <label className={security.mode === 'none' ? 'selected' : ''}>
                  <input type="radio" name="securityMode" value="none" checked={security.mode === 'none'} onChange={() => setSecurity({ mode: 'none', springSession: false })} />
                  <span><strong>None</strong><small>Do not configure application security</small></span>
                </label>

                {savedMetadata.frontend === 'thymeleaf-tailwind' && (
                  <label className={security.mode === 'http-session' ? 'selected' : ''}>
                    <input type="radio" name="securityMode" value="http-session" checked={security.mode === 'http-session'} onChange={() => setSecurity((current) => ({ ...current, mode: 'http-session' }))} />
                    <span><strong>HTTP Session based Security</strong><small>Session-backed authentication for server-rendered pages</small></span>
                  </label>
                )}

                {savedMetadata.frontend === 'angular-tailwind' && (
                  <label className={security.mode === 'oauth-resource-server' ? 'selected' : ''}>
                    <input type="radio" name="securityMode" value="oauth-resource-server" checked={security.mode === 'oauth-resource-server'} onChange={() => setSecurity({ mode: 'oauth-resource-server', springSession: false })} />
                    <span><strong>OAuth Resource Server</strong><small>Token-based API security for the Angular application</small></span>
                  </label>
                )}
              </div>
            </fieldset>

            {savedMetadata.frontend === 'thymeleaf-tailwind' && security.mode === 'http-session' && (
              <label className={security.springSession ? 'session-option selected' : 'session-option'}>
                <input type="checkbox" checked={security.springSession} onChange={(event) => setSecurity((current) => ({ ...current, springSession: event.target.checked }))} />
                <span><strong>Spring Session</strong><small>Opt in to persistent, shared HTTP session management.</small></span>
              </label>
            )}

            <div className="form-actions">
              <button type="button" className="back-button" onClick={() => setActiveStep(2)}>Back</button>
              <button type="submit" className="next-button">Next <span aria-hidden="true">→</span></button>
            </div>
          </form>
        )}

        {activeStep === 4 && savedSecurity && (
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

            <div className="form-actions">
              <button type="button" className="back-button" onClick={() => setActiveStep(3)}>Back</button>
              <button type="submit" className="next-button">Next <span aria-hidden="true">→</span></button>
            </div>
          </form>
        )}

        {activeStep === 5 && savedFeatures && (
          <div className="next-step-placeholder">
            <div className="welcome-icon" aria-hidden="true"><svg viewBox="0 0 32 32"><path d="M7 16.5 13 22l12-13" /></svg></div>
            <p className="eyebrow">Features saved</p>
            <h1>Review</h1>
            <p className="description">{savedFeatures.length === 0 ? 'No optional features selected.' : `${savedFeatures.length} optional feature${savedFeatures.length === 1 ? '' : 's'} selected.`} Your project is ready to review.</p>
            <button type="button" className="back-button" onClick={() => setActiveStep(4)}>Back to features</button>
          </div>
        )}
      </section>
    </main>
  )
}

export default App

import { FormEvent, useState } from 'react'

const steps = ['Welcome', 'Project', 'Dependencies', 'Configuration', 'Features', 'Review', 'Complete']

type BuildTool = 'maven' | 'gradle-groovy' | 'gradle-kotlin'

type ProjectMetadata = {
  groupId: string
  artifactId: string
  packageName: string
  buildTool: BuildTool
}

const initialMetadata: ProjectMetadata = {
  groupId: 'com.example',
  artifactId: 'demo',
  packageName: 'com.example.demo',
  buildTool: 'maven',
}

function App() {
  const [activeStep, setActiveStep] = useState(0)
  const [metadata, setMetadata] = useState<ProjectMetadata>(initialMetadata)
  const [savedMetadata, setSavedMetadata] = useState<ProjectMetadata | null>(null)

  const updateMetadata = <K extends keyof ProjectMetadata>(field: K, value: ProjectMetadata[K]) => {
    setMetadata((current) => ({ ...current, [field]: value }))
  }

  const handleProjectSubmit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    setSavedMetadata({ ...metadata })
    setActiveStep(2)
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
                  ['gradle-kotlin', 'Gradle – Kotlin', 'build.gradle.kts'],
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
          <div className="next-step-placeholder">
            <div className="welcome-icon" aria-hidden="true"><svg viewBox="0 0 32 32"><path d="M7 16.5 13 22l12-13" /></svg></div>
            <p className="eyebrow">Project saved</p>
            <h1>Choose Dependencies</h1>
            <p className="description"><strong>{savedMetadata.groupId}:{savedMetadata.artifactId}</strong> is ready. Dependency selection will be implemented in the next step.</p>
            <button type="button" className="back-button" onClick={() => setActiveStep(1)}>Back to project</button>
          </div>
        )}
      </section>
    </main>
  )
}

export default App

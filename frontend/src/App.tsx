const steps = ['Welcome', 'Project', 'Dependencies', 'Configuration', 'Features', 'Review', 'Complete']

function App() {
  return (
    <main className="generator-shell">
      <aside className="sidebar" aria-label="Project generation steps">
        <div className="logo-wrap">
          <img src="/wails.png" alt="Wails" className="logo" />
          <span>Spring Boot Generator</span>
        </div>
        <ol className="steps">
          {steps.map((step, index) => (
            <li className={index === 0 ? 'active' : ''} key={step} aria-current={index === 0 ? 'step' : undefined}>
              <span className="step-number">{index + 1}</span><span>{step}</span>
            </li>
          ))}
        </ol>
        <div className="sidebar-tools" aria-hidden="true"><span>△</span><span className="heart">♥</span><span>◔</span></div>
      </aside>

      <section className="welcome-panel" aria-labelledby="welcome-title">
        <div className="welcome-content">
          <div className="welcome-icon" aria-hidden="true"><svg viewBox="0 0 32 32"><path d="M7 16.5 13 22l12-13" /></svg></div>
          <p className="eyebrow">Welcome</p>
          <h1 id="welcome-title">Build production-ready<br />Spring Boot applications</h1>
          <p className="description">Create a thoughtfully structured Spring Boot project with sensible defaults and the dependencies your application needs—all through one simple, guided setup.</p>
          <button type="button" className="continue-button">Get started</button>
        </div>
      </section>
    </main>
  )
}

export default App

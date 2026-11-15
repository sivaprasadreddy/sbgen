# SBGen

SBGen is a macOS desktop app for generating production-ready Spring Boot applications.
A step-by-step wizard walks you through the options; the generated project is saved to your
`~/Downloads` folder as a ZIP archive.

Under the hood SBGen uses the [progen](https://github.com/sivaprasadreddy/progen) generator
as a Go library, so the projects it creates are the same as those created by the `progen` CLI.
SBGen gives you a graphical alternative to answering prompts in the terminal.

## What you can generate

| Step     | Options                                                                                                 |
|----------|---------------------------------------------------------------------------------------------------------|
| Project  | Application name and version, Group ID, Artifact ID, base package, build tool (Maven or Gradle), task runner (Mise or Taskfile with SDKMAN) |
| Frontend | None (REST API), Thymeleaf, JTE, Angular, or React; optional HTMX for Thymeleaf and JTE                   |
| Database | PostgreSQL, MySQL, or MariaDB; Spring Data JPA, Spring JdbcClient, or jOOQ; Flyway or Liquibase          |
| Features | Spring Cloud AWS, Email, RabbitMQ, Redis Caching, OpenTelemetry, Kubernetes manifests                    |
| AI       | Spring AI chat model providers: Ollama, OpenAI, Anthropic                                                |

Security is configured automatically based on the frontend: session-based login for server-rendered
frontends (Thymeleaf, JTE) and JWT for REST API, Angular, and React applications.

## Install

1. Download `sbgen-<version>-macos.zip` from the
   [Releases](https://github.com/sivaprasadreddy/sbgen/releases) page.
2. Unzip it and move `SBGen.app` to your `Applications` folder.
3. Open the app. The first time, macOS will block it — see
   [Allowing the app to run on macOS](#allowing-the-app-to-run-on-macos) below.

### Prerequisites

- macOS 12 or later (universal build for Apple Silicon and Intel).
- A JDK installed and available on your `PATH`. Generated projects target **Java 25**.
  After generating the project, SBGen formats the code by running the project's Maven or Gradle wrapper
  (`spotless:apply` / `spotlessApply`), which needs Java and an internet connection the first time
  it downloads Maven/Gradle and plugins. Generation fails if this step fails.

## Usage

1. Launch **SBGen** and click **Get started**.
2. **Project** – enter the application name, version, Group ID and Artifact ID, and pick the build tool
   and task runner (Mise by default, or Taskfile with SDKMAN).
   The package name is filled in from the Group ID and Artifact ID until you change it.
3. **Frontend** – pick a frontend, and HTMX if you chose Thymeleaf or JTE.
4. **Database** – pick the database, persistence framework, and migration tool.
5. **Features** – select optional features and Spring AI model providers.
6. **Review** – check your selections and click **Generate**.
7. The project is saved as `~/Downloads/<application-name>.zip` (a numeric suffix is added if the
   file already exists). Unzip it and follow the generated project's `README.md`.

## Allowing the app to run on macOS

SBGen releases are not signed with an Apple Developer ID or notarized by Apple, so macOS Gatekeeper
blocks the app the first time you open it with a message such as
*"Apple could not verify “SBGen” is free of malware"*. You only need to do one of the following once.

### Option 1: System Settings (recommended)

1. Try to open `SBGen.app`, then click **Done** (or **OK**) on the warning dialog.
2. Open **System Settings → Privacy & Security**.
3. Scroll down to the **Security** section. You will see *"SBGen" was blocked to protect your Mac*.
   Click **Open Anyway**.
4. Confirm with **Open Anyway** again and authenticate with your password or Touch ID.

On macOS 14 (Sonoma) and earlier you can instead Control-click (right-click) `SBGen.app` in Finder,
choose **Open**, and then click **Open** in the dialog.

### Option 2: Terminal

Remove the quarantine attribute that macOS adds to downloaded files:

```shell
xattr -dr com.apple.quarantine /Applications/SBGen.app
```

If macOS reports that the app *"is damaged and can't be opened"*, this command fixes it as well.

> Only bypass Gatekeeper for apps you trust. If you prefer, build the app from source (see below).

## Building from source

Requirements: [Go](https://go.dev/dl/) 1.27+, [Node.js](https://nodejs.org/) 24+,
the [Wails v3 CLI](https://v3.wails.io/getting-started/installation/) (`wails3`) and
[Task](https://taskfile.dev/).

```shell
git clone https://github.com/sivaprasadreddy/sbgen.git
cd sbgen

# Run in development mode with hot reload
wails3 dev

# Build an app bundle at bin/SBGen.app
wails3 task darwin:package

# Build a universal (Apple Silicon + Intel) app bundle
wails3 task darwin:package:universal

# Run Go tests
go test ./...
```

Apps you build yourself are signed locally (ad-hoc) and don't trigger the Gatekeeper warning on your own Mac.

### Project structure

- `main.go` – configures the Wails application and window.
- `app_generator.go` – `AppGeneratorService`, which validates the wizard's configuration,
  maps it to a progen `ProjectConfig`, generates the project, and zips it into `~/Downloads`.
- `frontend/` – the React + TypeScript wizard UI. `frontend/bindings/` is generated by Wails; don't edit it.
- `build/` – platform build, icon, and packaging configuration.

## Author

[SivaLabs](https://sivalabs.in)

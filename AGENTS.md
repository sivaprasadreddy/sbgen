# Repository Guidelines

## Project Structure & Module Organization

- SBGen is a [MyGo](https://mygo.run/) desktop application with a native UI: the interface is Go code using package `github.com/egoist/mygo/ui`, drawn by MyGo, with no web view, HTML or JavaScript. The Go module is `github.com/sivaprasadreddy/sbgen`. The `main` package is at the repository root (MyGo's default `main` in `mygo.json`); everything else is under `internal/`, where `app` uses `generator` and `store`, and both of those use `config`.
- `main.go` wires the services into `app.NewWizard` and opens the window with `Content: ui.View(wizard.View)` in `mygo.App.WhenReady`. It gives the wizard an `async` that runs work in a goroutine and applies the result with `Window.Update`.
- `internal/config/` holds `GeneratorConfiguration` (the user's choices: `ProjectSettings`, `FrontendSettings`, `DatabaseSettings`, features, AI models), the constants of each choice, `Validate` and `IsServerRendered`. Tests are in `config_test.go`.
- `internal/generator/` holds `generator.Service`. It validates a `config.GeneratorConfiguration`, maps it to a `progen` `springboot.ProjectConfig` (`toProjectConfig`), generates the project in a temporary directory and zips it into `~/Downloads`. Tests are in `generator_test.go`.
- `internal/store/` holds `TemplateService`, which keeps the user's templates (named configurations) in `templates.json` in the user data directory (`mygo.PathUserData`), with the three starter templates until the first change; `Save` validates with `config.Validate` and rejects duplicate names. `DefaultsService` keeps the Project step's defaults (`ProjectDefaults`: version, group ID, build tool, task runner) in `defaults.json` beside it, returning `BuiltInDefaults` until the user saves. `json.go` has the JSON read/atomic-write helpers both use.
- `internal/app/` is the window:
  - `wizard.go` holds the state and actions (`Wizard`): the selection (one `config.GeneratorConfiguration`), the step, the open sidebar page, templates, defaults, and generation. It depends on the stores through the `TemplateStore` and `DefaultsStore` interfaces. Keep logic here, out of the view, so tests can drive it.
  - `view.go` builds the screens from that state (`Wizard.View`): the sidebar, each step, the Complete screen, the Templates and Defaults pages, and the confirmation alerts (`ui.AlertDialog`).
  - `style.go` is the look: the Inter font (registered in `init`), the colors, the logo and SVG icons, and the shared building blocks (cards, option cards on `RadioBase`/`CheckboxBase`, text fields, buttons on `ButtonBase`). `options.go` lists each step's choices and the labels the Review step shows.
  - `assets/` holds the Inter font, its license, and the logo, embedded with `go:embed`.
- `mygo.json` holds the app's name, identifier, version and icons. The release workflow writes the tag's version into it before building.
- `resources/icon.png` is the app icon on every platform, and `AppIcon.icon` is the macOS 26 layered icon. Everything in `resources/` ships inside the app.
- Treat `build/` and `.mygo/` as generated output; do not edit them.

## Native UI Practices

- The view runs on the main thread, possibly several times a frame. Change state in actions (`Clicked`, `OnClick`, a widget's `Changed`), not unconditionally while building.
- Don't keep `ui.Element` values or a `*ui.Context` in the wizard or in goroutines; `go tool mygo vet .` reports it.
- Keep slow work off the main thread: generation runs through `wizard.async`, and its result is applied with `Window.Update`.
- Give a widget a key (`c.Key(...)`) where siblings would otherwise share state, as option cards in a loop.
- Style elements from state, as the cards do with `Hovered()` and the selected value; there are no style sheets.
- For alerts whose action discards data, list the action first and Cancel last: `ui.AlertDialog`'s last button is the default for Enter, and Escape dismisses.

## Build, Test, and Development Commands

- `go tool mygo dev` (or `mise run dev`): run the app, rebuilding as the Go code changes.
- `go tool mygo build` (or `mise run build`): build `SBGen.app` and a DMG into `build/<os>-<arch>/`. Add `-platform darwin/universal` for an Apple Silicon + Intel build.
- `go test -short ./...` (or `mise run test`): run the Go tests, including the view tests in `internal/app/view_test.go`, which drive the wizard with `ui.NewTester`. `-short` skips the test that generates a full project, which needs a JDK and network access.
- `go test ./...` (or `mise run all_tests`): run all Go tests.
- `SBGEN_SNAPSHOT_DIR=<dir> go test -run TestSnapshots ./internal/app`: save a picture of every screen, to check the look after a visual change.
- `go tool mygo vet <dir>`: check MyGo-specific mistakes in the Go code of one package directory (`.`, `internal/app`, …); it does not take `./...`.

## Coding Style & Naming Conventions

- Format Go code with `gofmt`; use tabs, exported identifiers in PascalCase, and local identifiers in camelCase.
- Add a choice to a step by adding its constant in `internal/config/config.go`, accepting it in `Validate`, mapping it in `toProjectConfig` (`internal/generator/generator.go`), and listing it in `internal/app/options.go`.
- Keep new code in the package it belongs to; `internal/config`, `internal/generator` and `internal/store` must not import `internal/app` or MyGo's `ui`.
- Cover new view behavior with a `ui.Tester` test in `internal/app/view_test.go`: find controls by their text or `Label`, and text fields with `AccessByRole(ui.RoleTextField, label)`.

## Commit & Pull Request Guidelines

- Use concise, imperative commit subjects, optionally scoped (for example, `ui: improve greeting validation`).
- Pull requests should explain the behavior change, list verification commands, link related issues, and include screenshots or recordings for visible UI changes.

## MyGo Documentation

- The documentation of the pinned MyGo version is in its module: `go list -m -f '{{.Dir}}' github.com/egoist/mygo`, under `docs/`, with the native UI under `docs/ui/`. Prefer it to the website when they differ.
- Online: https://mygo.egoist.dev/llms.txt and https://mygo.egoist.dev/llms-full.txt.
- `.agents/skills/` has MyGo skills for maintaining the app (`mygo-maintenance`) and writing native modules (`mygo-native-modules`).

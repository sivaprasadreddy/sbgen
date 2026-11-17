# Repository Guidelines

## Project Structure & Module Organization

- SBGen is a [MyGo](https://mygo.run/) desktop application with a native UI: the interface is Go code using package `github.com/egoist/mygo/ui`, drawn by MyGo, with no web view, HTML or JavaScript. The Go module is `github.com/sivaprasadreddy/sbgen`, all in `package main` at the repository root.
- `main.go` builds the `wizard` and opens the window with `Content: ui.View(app.view)` in `mygo.App.WhenReady`. It gives the wizard an `async` that runs work in a goroutine and applies the result with `Window.Update`.
- `wizard.go` holds the state and actions: the selection (one `GeneratorConfiguration`), the step, the open sidebar page, templates, defaults, and generation. Keep logic here, out of the view, so tests can drive it.
- `view.go` builds the screens from that state: the sidebar, each step, the Complete screen, the Templates and Defaults pages, and the confirmation alerts (`ui.AlertDialog`).
- `style.go` is the look: the Inter font (registered in `init`), the colors, the logo and SVG icons, and the shared building blocks (cards, option cards on `RadioBase`/`CheckboxBase`, text fields, buttons on `ButtonBase`). `options.go` lists each step's choices and the labels the Review step shows.
- `app_generator.go` holds `AppGeneratorService`. It validates the wizard's `GeneratorConfiguration`, maps it to a `progen` `springboot.ProjectConfig`, generates the project in a temporary directory and zips it into `~/Downloads`. Tests are in `app_generator_test.go`.
- `templates.go` holds `TemplateService`, which keeps the user's templates (named `GeneratorConfiguration`s) in `templates.json` in the user data directory (`mygo.PathUserData`), with the three starter templates until the first change. `Save` validates with `validateConfiguration` and rejects duplicate names.
- `defaults.go` holds `DefaultsService`, which keeps the Project step's defaults (`ProjectDefaults`: version, group ID, build tool, task runner) in `defaults.json` beside `templates.json`, returning the built-in defaults until the user saves. `storage.go` has the JSON read/atomic-write helpers both services use.
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
- `go test -short ./...` (or `mise run test`): run the Go tests, including the view tests in `view_test.go`, which drive the wizard with `ui.NewTester`. `-short` skips the test that generates a full project, which needs a JDK and network access.
- `go test ./...` (or `mise run all_tests`): run all Go tests.
- `SBGEN_SNAPSHOT_DIR=<dir> go test -run TestSnapshots .`: save a picture of every screen, to check the look after a visual change.
- `go tool mygo vet .`: check MyGo-specific mistakes in the Go code.

## Coding Style & Naming Conventions

- Format Go code with `gofmt`; use tabs, exported identifiers in PascalCase, and local identifiers in camelCase.
- Add a choice to a step by adding its constant in `app_generator.go`, accepting it in `validateConfiguration`, mapping it in `toProjectConfig`, and listing it in `options.go`.
- Cover new view behavior with a `ui.Tester` test in `view_test.go`: find controls by their text or `Label`, and text fields with `AccessByRole(ui.RoleTextField, label)`.

## Commit & Pull Request Guidelines

- Use concise, imperative commit subjects, optionally scoped (for example, `ui: improve greeting validation`).
- Pull requests should explain the behavior change, list verification commands, link related issues, and include screenshots or recordings for visible UI changes.

## MyGo Documentation

- The documentation of the pinned MyGo version is in its module: `go list -m -f '{{.Dir}}' github.com/egoist/mygo`, under `docs/`, with the native UI under `docs/ui/`. Prefer it to the website when they differ.
- Online: https://mygo.egoist.dev/llms.txt and https://mygo.egoist.dev/llms-full.txt.
- `.agents/skills/` has MyGo skills for maintaining the app (`mygo-maintenance`) and writing native modules (`mygo-native-modules`).

# Repository Guidelines

## Project Structure & Module Organization

- SBGen is a [MyGo](https://mygo.run/) desktop application: a Go backend with a React/TypeScript frontend in a web view. The Go module is `github.com/sivaprasadreddy/sbgen`, all in `package main` at the repository root.
- `main.go` binds the backend services (`AppGeneratorService`, `TemplateService`, `DefaultsService`, `DialogService`) with `mygo.Bind` and opens the window in `mygo.App.WhenReady`.
- `app_generator.go` holds `AppGeneratorService`. It validates the wizard's `GeneratorConfiguration`, maps it to a `progen` `springboot.ProjectConfig`, generates the project in a temporary directory and zips it into `~/Downloads`. Tests are in `app_generator_test.go`.
- `defaults.go` holds `DefaultsService`, which keeps the Project step's defaults (`ProjectDefaults`: version, group ID, build tool, task runner) in `defaults.json` beside `templates.json`, returning the built-in defaults until the user saves. `storage.go` has the JSON read/atomic-write helpers both services use.
- `dialogs.go` holds `DialogService.Confirm`, which shows `mygo.Dialog.Message` as a sheet of the calling window (`mygo.CallerWindow`). The frontend's `confirmAction` uses it, falling back to `window.confirm` outside a MyGo window. `ui.Modal` is for native UI and can't show over a web page.
- `templates.go` holds `TemplateService`, which keeps the user's templates (named `GeneratorConfiguration`s) in `templates.json` in the user data directory (`mygo.PathUserData`). `Save` validates with `validateConfiguration` and rejects duplicate names. Tests in `templates_test.go` use a temporary directory.
- `src/` is the frontend: `main.tsx`, the wizard in `App.tsx`, and `style.css`. Static assets (font, logo) are in `public/`.
- `src/mygo.ts` is the typed TypeScript client that `mygo generate` writes from the bound Go services. `mygo dev` and `mygo build` regenerate it.
- `mygo.config.ts` holds the app's name, identifier, version, icons and the frontend's dev/build commands. The version comes from `SBGEN_VERSION`, which the release workflow sets from the tag.
- `resources/icon.png` is the app icon on every platform, and `AppIcon.icon` is the macOS 26 layered icon. Everything in `resources/` ships inside the app.
- Treat `build/`, `dist/`, `.mygo/` and `node_modules/` as generated output; do not edit them.

## Calling Go from the Frontend

- Exported methods of bound services become typed functions in `src/mygo.ts`, with the method names in lower camel case. For example, `AppGeneratorService.GenerateProject` becomes `AppGeneratorService.generateProject(configuration)`.
- Structs cross as interfaces named by their `json` tags. A named string type with constants becomes a union of its values, for example `BuildTool` becomes `"maven" | "gradle"`. Add an option by adding a constant in Go, accepting it in `validateConfiguration` and mapping it in `toProjectConfig`, then regenerate the client.
- A method that returns a non-nil error rejects the promise, and the error's text is the message the UI shows.
- Links with `target="_blank"` open in the default browser. Mark window-dragging regions with the CSS property `--app-region: drag` and exclude parts of them with `no-drag`.

## Build, Test, and Development Commands

- `bun install`: install frontend dependencies and the MyGo CLI (`mygo-cli`). CI uses `bun install --frozen-lockfile`.
- `bun run dev` (or `mise run dev`): run the app with backend and frontend hot reload.
- `bun run build` (or `mise run build`): build `SBGen.app` and a DMG into `build/<os>-<arch>/`. Add `-- -platform darwin/universal` for an Apple Silicon + Intel build.
- `bun run generate`: regenerate `src/mygo.ts` without running the app.
- `bun run build:web`: type-check TypeScript and build the frontend into `dist/`.
- `go test -short ./...` (or `mise run test`): run the Go tests. `-short` skips the test that generates a full project, which needs a JDK and network access.
- `go test ./...` (or `mise run all_tests`): run all Go tests.
- `mygo vet .`: check MyGo-specific mistakes in the Go code.

## Coding Style & Naming Conventions

- Format Go code with `gofmt`; use tabs, exported identifiers in PascalCase, and local identifiers in camelCase.
- Keep services small and bind them in `main.go`. Give exported types and methods doc comments, because they become TSDoc in `src/mygo.ts`.
- In TypeScript/React, use two-space indentation, PascalCase component names, camelCase functions and variables, and `.tsx` for JSX-bearing files.
- Preserve the existing semicolon-light style unless an automated formatter is introduced.
- Never hand-edit `src/mygo.ts`; change the Go code and regenerate.

## Commit & Pull Request Guidelines

- Use concise, imperative commit subjects, optionally scoped (for example, `frontend: improve greeting validation`).
- Pull requests should explain the behavior change, list verification commands, link related issues, and include screenshots or recordings for visible UI changes.

## MyGo Documentation

- The documentation of the pinned MyGo version is in its module: `go list -m -f '{{.Dir}}' github.com/egoist/mygo`, under `docs/`. Prefer it to the website when they differ.
- Online: https://mygo.egoist.dev/llms.txt and https://mygo.egoist.dev/llms-full.txt.
- `.agents/skills/` has MyGo skills for maintaining the app (`mygo-maintenance`) and writing native modules (`mygo-native-modules`).

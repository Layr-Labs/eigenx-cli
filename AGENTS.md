# Repository Guidelines

## Project Structure & Module Organization
- `cmd/eigenx/`: CLI entrypoint (`main.go`).
- `pkg/commands/`: Subcommands and command wiring (e.g., `app`, `auth`, `version`).
- `pkg/common/`: Shared config, flags, logging, keyring, helpers.
- `pkg/hooks/`: CLI middleware (telemetry prompt, metrics).
- `internal/`: Version info and embedded templates/bindings used at build time.
- `config/`: Embedded config (`templates.yaml`, `.gitignore`).
- `templates/`: App templates for TEE projects (golang, typescript, rust, python).
- `Makefile`: Developer tasks; output binary in `bin/eigenx`.

## Build, Test, and Development Commands
- `make help` — List tasks.
- `make build` — Build CLI to `./bin/eigenx` (injects version/commit via `-ldflags`).
- `make tests` / `make tests-fast` — Run full or quick test suites.
- `make fmt` — Format Go code with `gofmt`.
- `make lint` — Run `golangci-lint` (also wired via pre-commit).
- Run locally: `./bin/eigenx --help`.

## Coding Style & Naming Conventions
- Go 1.24; keep code `gofmt`-clean and `golangci-lint`-clean.
- Packages: short, lowercase names; files `snake_case.go`.
- Exported identifiers: `CamelCase` with GoDoc-style comments.
- Prefer context-aware APIs (`ctx context.Context` first arg).
- Keep commands small and composable under `pkg/commands/...`.

## Testing Guidelines
- Frameworks: standard `testing` + `stretchr/testify`.
- Place tests alongside code as `*_test.go`; use table-driven tests where sensible.
- Run: `make tests` (CI mirrors this). Keep tests hermetic; avoid network/registry calls.

## Commit & Pull Request Guidelines
- Conventional Commits style (e.g., `feat: ...`, `fix(app): ...`, `chore(ci): ...`).
- PRs should include: clear description, rationale, screenshots/logs for UX/CLI changes, and linked issues.
- Before pushing: `make fmt lint tests` should pass locally.

## Security & Configuration Tips
- Never commit secrets. Use `.env` locally; CLI loads dotenv when present.
- Auth keys are stored via OS keyring; prefer `eigenx auth` flows over ad‑hoc files.
- Optional: set `TELEMETRY_TOKEN` when building releases to embed telemetry key.
- Assets are embedded via `go:embed` (`embeds.go`, `config/config_embeds.go`); update paths carefully.

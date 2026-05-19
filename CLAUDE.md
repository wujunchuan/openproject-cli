# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

`op` — a Go CLI tool for the [OpenProject](https://www.openproject.org/) API v3, built with [Cobra](https://github.com/spf13/cobra). Manages work packages, notifications, projects, time entries, and more via the command line.

## Commands

```bash
go build -v ./...          # Build
go test -v ./...           # Run all tests
go test -v ./components/printer/  # Run tests in a single package
go run .                   # Run without building binary
go install .               # Install as `op`
```

No Makefile or linter config exists — use standard Go toolchain directly.

## Architecture

Layered design with four top-level directories:

- **`cmd/`** — Cobra command definitions. Each subcommand group (`list`, `create`, `update`, `inspect`, `search`, `git`) has its own sub-package with a `RootCmd` variable. `tui/` is a special sub-package providing an interactive Bubble Tea TUI (`opc tui`). Commands parse flags and delegate to `components/` for logic.
- **`components/`** — Business logic:
  - `requests/` — HTTP client wrapper. `Init()` sets host/token. `Get`, `Post`, `Patch` wrap `Do()` with a spinner. `Probe()` does non-erroring GET for instance detection.
  - `resources/` — Domain-specific API interaction (e.g., `work_packages/`, `projects/`). Each resource package calls `requests` functions and returns `models`.
  - `printer/` — Output formatting via a `Printer` interface. `ConsolePrinter` for production, `TestingPrinter` for tests. Domain-specific print functions (e.g., `WorkPackages()`, `Projects()`).
  - `configuration/` — Config file reading, version struct.
  - `routes/` — URL construction helpers.
- **`dtos/`** — JSON DTOs matching the OpenProject API v3 response shapes. Each DTO has a `Convert()` method that produces the corresponding `models/` type.
- **`models/`** — Internal domain models (plain Go structs, no JSON tags).

### Data flow

```
cmd/ → components/resources/ → components/requests/ → OpenProject API
                      ↓                                    ↓
                  models/ ← dtos/ (JSON deserialization ← API response)
                      ↓
              components/printer/ → stdout
```

### Key patterns

- **Global state for HTTP client**: `requests.Init()` and `routes.Init()` set package-level vars called from `cmd/root.go`'s `init()`.
- **Printer abstraction**: All output goes through `printer.Printer` interface. Tests use `TestingPrinter` which captures output as a string.
- **Filters**: Work package list filters implement the `resources.Filter` interface (`components/resources/filters.go`). Each filter knows how to validate input and produce a `requests.Query`.
- **DTO conversion**: DTOs in `dtos/` own the conversion to `models/` via `Convert()` methods, keeping JSON concerns separate from domain logic.

## Testing

Tests use the standard `testing` package (no third-party frameworks). Printer tests initialize with `TestingPrinter` and assert on the captured string. TUI tests use Bubble Tea's `teatest` approach by calling `Update()` directly on models. To run a single test:

```bash
go test -v -run TestName ./components/printer/
go test -v ./cmd/tui/          # Run all TUI tests
```

## TUI (`opc tui`)

Interactive terminal UI built on [Bubble Tea](https://github.com/charmbracelet/bubbletea) + [lipgloss](https://github.com/charmbracelet/lipgloss) + [bubbles](https://github.com/charmbracelet/bubbles). Uses the Elm architecture: `Init` → `Update` → `View`.

### Architecture

- **`cmd/tui/app.go`** — Top-level model, view stack (`listView` ↔ `detailView`), message routing
- **`cmd/tui/list_view.go`** — Work package list with table, search, sort, pagination, filter overlay
- **`cmd/tui/detail_view.go`** — Detail view with viewport scroll, properties, activities, edit overlay
- **`cmd/tui/filter_view.go`** — Filter panel (project, status, type, assignee) loaded from API
- **`cmd/tui/edit_view.go`** — Edit overlay (type change via `work_packages.Update`)
- **`cmd/tui/styles.go`** — lipgloss style definitions
- **`cmd/tui/keymap.go`** — Key binding definitions
- **`cmd/tui/help_bar.go`** — Bottom help bar and `?` help overlay

### Key patterns

- **Async data loading**: `tea.Cmd` wraps resource calls in goroutines; results delivered as typed `tea.Msg`
- **Spinner suppression**: TUI calls `printer.SetSilent(true)` to avoid spinner/stdout conflicts
- **View stack**: `app.go` manages push/pop between list and detail views. Filter and edit are overlays, not separate views.
- **DTO expansion**: `WorkPackageDto` and `models.WorkPackage` include Priority, Project, Version, CreatedAt, UpdatedAt for the detail view

## Releases

Triggered by pushing a git tag (`git tag vX.Y.Z && git push origin vX.Y.Z`). GitHub Actions builds static binaries for linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64, and creates a GitHub Release with zip archives. Version/commit/date are injected via `-ldflags`.

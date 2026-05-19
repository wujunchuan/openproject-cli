# TUI Work Packages — Design Spec

Date: 2026-05-19

## Overview

Add an interactive TUI (Terminal User Interface) for browsing and managing work packages to the existing `openproject-cli`. The TUI is invoked via `opc tui` subcommand, built on Bubble Tea + lipgloss, and reuses the existing `components/resources/` layer for data access.

## Goals

- Efficient browsing of work packages with filtering, sorting, pagination, and search
- Rich detail view with activity history and quick edit capability
- Non-disruptive integration with existing CLI — new subcommand, minimal changes to existing code

## Non-Goals (v1)

- Other resource types (projects, notifications, time entries) — deferred to v2
- Creating new work packages from TUI — use existing `opc create` command
- File attachment management
- Offline/cached mode

## Architecture

```
cmd/tui/
  ├── tui.go           — cobra subcommand + tea.NewProgram entry
  ├── app.go           — top-level Model, view stack management
  ├── styles.go        — lipgloss style definitions
  ├── keymap.go        — global key bindings
  ├── list_view.go     — list view Model
  ├── detail_view.go   — detail view Model
  ├── filter_view.go   — filter panel Model (overlay)
  └── help_bar.go      — bottom help bar component
    ↓
components/resources/work_packages/  (reuse existing)
components/resources/projects/       (reuse for filter options)
components/resources/status/         (reuse for filter options)
components/resources/types/          (reuse for filter options)
components/requests/                 (add SetSilent mode)
components/launch/                   (reuse for browser open)
```

### View Stack

`app.go` manages a `[]view` stack. Push on enter, pop on back. Two views for v1: list and detail. Filter panel is an overlay on list, not a separate view.

### Data Flow

1. `tea.Cmd` wraps resource calls in goroutines
2. Results delivered as `tea.Msg` (typed messages)
3. Model updates state on message receipt
4. UI re-renders on state change

```
User action → Update() → tea.Cmd (goroutine) → resource function → tea.Msg → Update() → View()
```

### Requests Silent Mode

Add to `components/requests/requests.go`:

```go
var silent bool
func SetSilent(b bool) { silent = b }
```

`WithSpinner()` checks `silent` flag; when true, executes the function directly without showing spinner. TUI calls `requests.SetSilent(true)` on start, restores on exit.

## List View

### Layout

```
┌─ Work Packages ──────────────────────────────────────────────┐
│  Filter: project=frontend  status=open  assignee=me          │
│──────────────────────────────────────────────────────────────│
│  #142  ✦ Bug    Fix login redirect    New        John    ⏱  │
│  #141  ◆ Task   Add unit tests        In prog    Alice   ⏱  │
│  #140  ✦ Bug    Update deps           New        —       ⏱  │
│──────────────────────────────────────────────────────────────│
│  1-50 / 123        ← page 1 of 3 →                           │
└──────────────────────────────────────────────────────────────┘
```

### Columns

| Column | Source | Width |
|--------|--------|-------|
| ID | `wp.Id` | `#N` format |
| Type | `wp.Type` | fixed |
| Title | `wp.Subject` | fills remaining space, truncated with `…` |
| Status | `wp.Status` | fixed |
| Assignee | `wp.Assignee` | fixed, `—` if none |

### Key Bindings

| Key | Action |
|-----|--------|
| `↑`/`k` | Move selection up |
| `↓`/`j` | Move selection down |
| `enter` | Open detail view |
| `/` | Toggle inline search input |
| `f` | Open filter panel |
| `n` | Next page |
| `p` | Previous page |
| `r` | Refresh current page |
| `o` | Open selected WP in browser |
| `s` | Sort (cycle: ID → Status → Type → Assignee) |
| `?` | Show full help |
| `q` | Quit |

### Search

`/` toggles an inline text input at the bottom of the list. Typing filters the current page client-side. `enter` performs a server-side search, `esc` cancels.

### Sort

`s` cycles through sort keys: ID (default) → Status → Type → Assignee. Sorting is client-side on the current page's data. Current sort indicator shown in the header bar.

### Pagination

- Default page size: 50 (matches API default)
- `n`/`p` for next/previous page
- Page indicator in bottom bar: `1-50 / 123`

### Data Loading

- Startup: show centered spinner, load first page
- Navigation: highlight shows immediately, load on `enter`
- Page change: show spinner in list area, keep header/footer

## Detail View

### Layout

```
┌─ #142 Fix login redirect ───────────────────────────────────┐
│  Type: Bug        Status: New → In Progress                 │
│  Project: frontend    Assignee: John                        │
│  Priority: High       Created: 2026-05-18                   │
│  Version: v2.1        Updated: 2026-05-19                   │
│──────────────────────────────────────────────────────────────│
│  Description                                                 │
│  ─────────────────────────────────────────────────────────   │
│  When user logs in with SSO, the redirect URL is wrong...    │
│──────────────────────────────────────────────────────────────│
│  Activity (5)                                                │
│  ─────────────────────────────────────────────────────────   │
│  [John] 2026-05-19 10:32                                     │
│  Changed status from New to In Progress                      │
│                                                              │
│  [Alice] 2026-05-18 15:20                                    │
│  Confirmed, also seeing this on staging.                     │
│──────────────────────────────────────────────────────────────│
│  `esc` back  `e` edit  `o` browser  `r` refresh             │
└──────────────────────────────────────────────────────────────┘
```

### Sections

1. **Header**: `#ID Subject`
2. **Properties**: Two-column grid — type, status, project, assignee, priority, version, created, updated
3. **Description**: Plain text rendering (preserve line breaks, no markdown parsing)
4. **Activity**: Chronological list, newest first — author, timestamp, content

### Key Bindings

| Key | Action |
|-----|--------|
| `↑`/`↓`/`PgUp`/`PgDn` | Scroll content |
| `esc` | Back to list |
| `e` | Enter edit mode |
| `o` | Open in browser |
| `r` | Refresh data |

### Edit Mode

`e` triggers an overlay at the bottom:

```
┌─ Edit #142 ─────────────────────────────────────────────────┐
│  [S]tatus  [A]ssignee  [T]ype                               │
│                                                              │
│  Press a key to select a field...                            │
└──────────────────────────────────────────────────────────────┘
```

- Press `S`/`A`/`T` to open a selection list for that field
- Selection list uses API to fetch available options
- Selecting an option triggers `work_packages.Update()` to submit
- On success: close overlay, refresh detail data
- On failure: show error message in overlay

## Filter Panel

### Layout

```
┌─ Filters ───────────────────────────────┐
│  Project:    [frontend       ▾]         │
│  Status:     [open           ▾]         │
│  Type:       [all            ▾]         │
│  Assignee:   [me             ▾]         │
│  ──────────────────────────────────────  │
│  active: project=frontend status=open    │
│          assignee=me                     │
│                                          │
│  `tab` next field  `enter` apply         │
│  `esc` cancel  `c` clear all             │
└──────────────────────────────────────────┘
```

### Interaction

- `tab`/`shift+tab`: Navigate between fields
- `↑`/`↓`: Cycle option values in current field
- `enter`: Apply filters, close panel, refresh list
- `esc`: Cancel, restore previous filters
- `c`: Clear all filters

### Option Sources

| Field | Source | API Call |
|-------|--------|----------|
| Project | `projects.All()` | GET /api/v3/projects |
| Status | `status.All()` | GET /api/v3/statuses |
| Type | `types.All()` | GET /api/v3/types |
| Assignee | Hardcoded: `all`, `me`, `none` | — |

Options loaded on first panel open, cached for session.

## Error Handling

| Error | Behavior |
|-------|----------|
| Network error | Show error in content area + `r` to retry, don't exit |
| API 401 | Show "Authentication failed, check token", suggest `opc login` |
| API 404 | Show "Work package not found", auto-return to list |
| Terminal too small | Startup check: <80x24 shows "Terminal too small" |

## Signal Handling

| Signal | Behavior |
|--------|----------|
| `ctrl+c` | First press: show "Press again to quit". Second press: exit |
| `ctrl+l` | Force redraw (clear terminal artifacts) |

## Dependencies

New dependencies to add to `go.mod`:

- `github.com/charmbracelet/bubbletea` — TUI framework
- `github.com/charmbracelet/lipgloss` — styling
- `github.com/charmbracelet/bubbles` — reusable components (textinput, viewport, spinner)

## DTO Expansion

The current `WorkPackageDto` and `models.WorkPackage` only capture: Id, Subject, Type, Assignee, Status, Description, LockVersion. For the TUI detail view, expand the DTO to include fields already present in the API response:

```go
// dtos/work_package.go — add to WorkPackageDto
Priority    *LinkDto   `json:"priority,omitempty"`    // in _links
Project     *LinkDto   `json:"project,omitempty"`     // already in _links
Version     *LinkDto   `json:"version,omitempty"`     // in _links
CreatedAt   string     `json:"createdAt,omitempty"`
UpdatedAt   string     `json:"updatedAt,omitempty"`
```

```go
// models/work_package.go — expand WorkPackage struct
Priority    string
Project     string
Version     string
CreatedAt   string
UpdatedAt   string
```

Update `Convert()` to populate the new fields from `dto.Links.Priority.Title`, `dto.Links.Project.Title`, `dto.Links.Version.Title`, `dto.CreatedAt`, `dto.UpdatedAt`.

## File Changes Summary

| File | Change |
|------|--------|
| `cmd/root.go` | Register `tui` subcommand |
| `cmd/tui/*.go` | New files — all TUI code |
| `components/requests/requests.go` | Add `SetSilent()` + silent check in `WithSpinner()` |
| `dtos/work_package.go` | Add Priority, Version, CreatedAt, UpdatedAt fields + links |
| `models/work_package.go` | Add Priority, Project, Version, CreatedAt, UpdatedAt fields |

## Testing

- Unit tests for `app.go` state transitions (view push/pop, message handling)
- Unit tests for filter logic (option selection produces correct query)
- Integration tests using `tea.Model` directly (no terminal needed, Bubble Tea supports this)
- Manual testing for visual layout and keyboard navigation

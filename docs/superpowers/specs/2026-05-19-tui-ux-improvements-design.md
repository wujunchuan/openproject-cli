# TUI UX Improvements — Design Spec

## Overview

Six UX improvements to the interactive TUI (`opc tui`): faster navigation, clipboard copy, filter popups, enhanced assignee filtering, vim-style keys, and updated help bars.

## 1. `l` Key to Enter Detail from List

**Files:** `cmd/tui/list_view.go`, `cmd/tui/keymap.go`

Add `l` as a parallel binding to `enter` for opening the detail view. Only active when search overlay is not open.

**Help bar:** `enter/l open`

## 2. `c` Key to Copy Work Package ID

**Files:** `cmd/tui/list_view.go`, `cmd/tui/detail_view.go`, `components/configuration/util.go`

**Format:** `{prefix}#{id}` (e.g. `OP#107`)

**Config:** Extend config file to support key-value pairs. Add `OP_CLI_COPY_PREFIX` env var or `copy_prefix` config key (default: `OP`).

**Dependency:** `github.com/atotto/clipboard` for cross-platform clipboard access.

**Behavior:**
- List view: `c` copies ID of the selected item
- Detail view: `c` copies ID of the current work package
- Brief confirmation via `copyConfirmMsg` with ~1.5s timeout

## 3. Filter Popup Overlays

**Files:** `cmd/tui/filter_view.go`, `cmd/tui/styles.go`

**State machine:**
```
filterFieldMode (browse fields) ↔ filterPopupMode (select from list)
```

When pressing `enter` on a focused field:
- Open a centered popup overlay showing all options as a scrollable list
- Navigate with `j/k` or `up/down`
- Select with `enter` → updates field value, closes popup
- Cancel with `esc` → returns to field browse mode

Pattern matches existing `edit_view.go` overlay (editChooseValue state).

## 4. Assignee: [all] / [me] / [selected]

**Files:** `cmd/tui/filter_view.go`, `components/resources/users/functions.go`

**Modes:**
- `[all]` — no assignee filter
- `[me]` — filter by current user
- `[selected]` — show selected user's name, open user list popup on `enter`

**h/l toggle:** In field browse mode, `h`/`l` cycles: `selected → me → all → selected`

**User list popup:** Load all users from API via `users.All()` (if available), show in scrollable popup. If no `All()` function exists, use the search API.

## 5. hjkl Navigation in Filter Page

**Files:** `cmd/tui/filter_view.go`

**Field browse mode:**
- `h` = previous field (like `shift+tab`)
- `l` = next field (like `tab`)
- `j` = next value (like `down`)
- `k` = previous value (like `up`)

**Popup mode:**
- `j/k` = scroll list
- `h/l` = no action (lateral nav disabled in popup)

## 6. Help Bar Updates

| View | Help Text |
|------|-----------|
| List | `j/k move  enter/l open  c copy  / search  f filter  ? help  q quit` |
| Detail | `esc back  c copy  e edit  o browser  r refresh  ? help` |
| Filter (fields) | `h/l field  j/k select  enter popup  esc cancel  c clear  ? help` |
| Filter (popup) | `j/k move  enter select  esc cancel` |

Help overlay (`?`) also updated with new bindings.

## Dependencies

- `github.com/atotto/clipboard` — cross-platform clipboard access

## Testing

- List view: test `l` opens detail, `c` returns copy command
- Detail view: test `c` returns copy command
- Filter view: test popup open/close/selection, hjkl navigation, assignee mode cycling
- Config: test prefix reading with default fallback

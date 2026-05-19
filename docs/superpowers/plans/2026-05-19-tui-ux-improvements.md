# TUI UX Improvements Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [`) syntax for tracking.

**Goal:** Six UX improvements to `opc tui`: `l` key detail navigation, `c` key clipboard copy, filter popup overlays, enhanced assignee filtering with h/l toggle, hjkl vim-style navigation, and updated help bars.

**Architecture:** Incremental changes to existing Bubble Tea views. Filter popup reuses the overlay pattern from `edit_view.go`. Clipboard adds a new dependency. Config adds an env var for copy prefix. Users resource adds an `All()` function.

**Tech Stack:** Go, Bubble Tea, lipgloss, github.com/atotto/clipboard

---

## File Map

| File | Action | Purpose |
|------|--------|---------|
| `cmd/tui/list_view.go` | Modify | Add `l` key, `c` copy, update help bar |
| `cmd/tui/detail_view.go` | Modify | Add `c` copy, update help bar/overlay |
| `cmd/tui/filter_view.go` | Modify | Popup overlays, hjkl nav, assignee modes |
| `cmd/tui/app.go` | Modify | Handle copyConfirmMsg, propagate to views |
| `components/configuration/util.go` | Modify | Add ReadConfigValue for copy_prefix |
| `components/resources/users/functions.go` | Modify | Add All() function |
| `cmd/tui/list_view_test.go` | Modify | Tests for `l` key and `c` copy |
| `cmd/tui/detail_view_test.go` | Modify | Tests for `c` copy |
| `cmd/tui/filter_view_test.go` | Modify | Tests for popup, hjkl, assignee modes |

---

## Task 1: Add clipboard dependency

- [ ] **Step 1: Add clipboard package**

```bash
go get github.com/atotto/clipboard
```

- [ ] **Step 2: Verify build**

```bash
go build ./...
```

- [ ] **Step 3: Commit**

```bash
git add go.mod go.sum
git commit -m "deps: add atotto/clipboard for copy-to-clipboard support"
```

---

## Task 2: Add `All()` function to users resource

**Files:**
- Modify: `components/resources/users/functions.go`
- Modify: `components/resources/users/functions.go` (no test file exists for this package)

- [ ] **Step 1: Add All() function**

In `components/resources/users/functions.go`, add the following function at the end of the file:

```go
func All() ([]*models.User, error) {
	query := requests.NewPaginatedQuery(-1, nil)
	response, err := requests.Get(paths.Principals(), &query)
	if err != nil {
		return nil, err
	}

	userCollection := parser.Parse[dtos.UserCollectionDto](response)
	return userCollection.Convert(), nil
}
```

- [ ] **Step 2: Verify build**

```bash
go build ./components/resources/users/...
```

- [ ] **Step 3: Commit**

```bash
git add components/resources/users/functions.go
git commit -m "feat(users): add All() function to load all users"
```

---

## Task 3: Add copy_prefix configuration support

**Files:**
- Modify: `components/configuration/util.go`

- [ ] **Step 1: Add ReadConfigValue function**

Add the following to `components/configuration/util.go`:

```go
func ReadConfigValue(key string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return ""
}
```

- [ ] **Step 2: Add CopyPrefix helper**

Add to `components/configuration/util.go`:

```go
func CopyPrefix() string {
	prefix := ReadConfigValue("OP_CLI_COPY_PREFIX")
	if prefix == "" {
		return "OP"
	}
	return prefix
}
```

- [ ] **Step 3: Verify build**

```bash
go build ./components/configuration/...
```

- [ ] **Step 4: Commit**

```bash
git add components/configuration/util.go
git commit -m "feat(config): add CopyPrefix() with OP_CLI_COPY_PREFIX env var support"
```

---

## Task 4: Add `l` key to open detail from list view

**Files:**
- Modify: `cmd/tui/list_view.go`
- Modify: `cmd/tui/list_view_test.go`

- [ ] **Step 1: Add test for `l` key opens detail**

In `cmd/tui/list_view_test.go`, append:

```go
func TestListModelLOpensDetail(t *testing.T) {
	m := newListModel()
	wp := &models.WorkPackage{Id: 99, Subject: "LKey"}
	m.SetWorkPackages(&models.WorkPackageCollection{
		Items: []*models.WorkPackage{wp},
	})

	_, cmd := m.Update(keyMsg("l"))
	if cmd == nil {
		t.Fatal("l should return a command")
	}

	msg := cmd()
	detailMsg, ok := msg.(openDetailMsg)
	if !ok {
		t.Fatalf("expected openDetailMsg, got %T", msg)
	}
	if detailMsg.wp.Id != 99 {
		t.Fatalf("expected wp id 99, got %d", detailMsg.wp.Id)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test -v -run TestListModelLOpensDetail ./cmd/tui/
```

Expected: FAIL (l key not handled)

- [ ] **Step 3: Add `l` key handling in list_view.go**

In the `Update` method of `listModel`, inside the `tea.KeyMsg` switch, add `"l"` alongside the existing `"enter"` case. Change the existing case from:

```go
case "enter":
```

to:

```go
case "enter", "l":
```

- [ ] **Step 4: Run test to verify it passes**

```bash
go test -v -run TestListModelLOpensDetail ./cmd/tui/
```

- [ ] **Step 5: Update help bar text**

In the `View()` method of `listModel`, change the help text from:

```go
keys := []string{"j/k", "move", "enter", "open", "/", "search", "f", "filter", "?", "help", "q", "quit"}
```

to:

```go
keys := []string{"j/k", "move", "enter/l", "open", "c", "copy", "/", "search", "f", "filter", "?", "help", "q", "quit"}
```

- [ ] **Step 6: Run all list view tests**

```bash
go test -v ./cmd/tui/
```

- [ ] **Step 7: Commit**

```bash
git add cmd/tui/list_view.go cmd/tui/list_view_test.go
git commit -m "feat(tui): add 'l' key to open detail from list view"
```

---

## Task 5: Add `c` key to copy work package ID (list view)

**Files:**
- Modify: `cmd/tui/list_view.go`
- Modify: `cmd/tui/app.go`
- Modify: `cmd/tui/list_view_test.go`

- [ ] **Step 1: Add copyConfirmMsg to app.go**

In `cmd/tui/app.go`, add a new message type alongside the existing messages:

```go
type copyConfirmMsg struct{}
```

In the `App.Update` method, handle this message in the `tea.KeyMsg` switch — after the existing key handling, add:

```go
case copyConfirmMsg:
	// copy confirmation handled in sub-views
	return a, nil
```

Also add a `copyConfirm` field and a timer:

```go
type App struct {
	state          viewState
	list           *listModel
	detail         *detailModel
	width          int
	height         int
	err            error
	quitting       bool
	ctrlCPressed   bool
	ctrlCTime      time.Time
	copyConfirm    bool
}
```

In `App.Update`, handle `copyConfirmMsg`:

```go
case copyConfirmMsg:
	a.copyConfirm = true
	return a, tea.Tick(1500*time.Millisecond, func(t time.Time) tea.Msg {
		return copyClearMsg{}
	})
case copyClearMsg:
	a.copyConfirm = false
	return a, nil
```

Add the clear message type:

```go
type copyClearMsg struct{}
```

In `App.View()`, after the error display block, add:

```go
if a.copyConfirm {
	content += "\n" + helpStyle.Render("  copied!")
}
```

- [ ] **Step 2: Add clipboard copy test in list_view_test.go**

```go
func TestListModelCCopiesId(t *testing.T) {
	m := newListModel()
	wp := &models.WorkPackage{Id: 107, Subject: "Copy"}
	m.SetWorkPackages(&models.WorkPackageCollection{
		Items: []*models.WorkPackage{wp},
	})

	_, cmd := m.Update(keyMsg("c"))
	if cmd == nil {
		t.Fatal("c should return a command")
	}

	msg := cmd()
	copyMsg, ok := msg.(copyIdMsg)
	if !ok {
		t.Fatalf("expected copyIdMsg, got %T", msg)
	}
	if copyMsg.id != 107 {
		t.Fatalf("expected id 107, got %d", copyMsg.id)
	}
}
```

- [ ] **Step 3: Add copyIdMsg and handle `c` key in list_view.go**

Add to `list_view.go` (at the bottom, near other message types):

```go
type copyIdMsg struct {
	id uint64
}
```

In the `Update` method, add a `"c"` case in the `tea.KeyMsg` switch (inside the `!m.searchActive && !m.filterOverlay` guard):

```go
case "c":
	if !m.searchActive && !m.filterOverlay && len(m.items) > 0 {
		selected := m.items[m.selected]
		return m, func() tea.Msg {
			return copyIdMsg{id: selected.Id}
		}
	}
```

- [ ] **Step 4: Handle copyIdMsg in app.go**

In `App.Update`, add handling for `copyIdMsg`:

```go
case copyIdMsg:
	prefix := configuration.CopyPrefix()
	text := fmt.Sprintf("%s#%d", prefix, msg.id)
	if err := clipboard.WriteAll(text); err == nil {
		return a, func() tea.Msg { return copyConfirmMsg{} }
	}
	return a, nil
```

Add imports to `app.go`: `"github.com/atotto/clipboard"` and `"github.com/opf/openproject-cli/components/configuration"`.

- [ ] **Step 5: Run all tests**

```bash
go test -v ./cmd/tui/
```

- [ ] **Step 6: Commit**

```bash
git add cmd/tui/list_view.go cmd/tui/list_view_test.go cmd/tui/app.go
git commit -m "feat(tui): add 'c' key to copy work package ID from list view"
```

---

## Task 6: Add `c` key to copy work package ID (detail view)

**Files:**
- Modify: `cmd/tui/detail_view.go`
- Modify: `cmd/tui/detail_view_test.go`

- [ ] **Step 1: Add test for `c` key in detail view**

In `cmd/tui/detail_view_test.go`, append:

```go
func TestDetailModelCCopiesId(t *testing.T) {
	wp := &models.WorkPackage{Id: 200, Subject: "CopyDetail"}
	m := newDetailModel(wp, 80, 24)

	_, cmd := m.Update(keyMsg("c"))
	if cmd == nil {
		t.Fatal("c should return a command")
	}

	msg := cmd()
	copyMsg, ok := msg.(copyIdMsg)
	if !ok {
		t.Fatalf("expected copyIdMsg, got %T", msg)
	}
	if copyMsg.id != 200 {
		t.Fatalf("expected id 200, got %d", copyMsg.id)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test -v -run TestDetailModelCCopiesId ./cmd/tui/
```

- [ ] **Step 3: Add `c` key handling in detail_view.go**

In the `Update` method of `detailModel`, in the `tea.KeyMsg` switch block (alongside `"e"`, `"o"`, `"r"`), add:

```go
case "c":
	return m, func() tea.Msg {
		return copyIdMsg{id: m.wp.Id}
	}
```

- [ ] **Step 4: Update detail help bar and help overlay**

In `detailView.View()`, change the footer from:

```go
footer := helpStyle.Render("  esc back  e edit  o browser  r refresh  ? help")
```

to:

```go
footer := helpStyle.Render("  esc back  c copy  e edit  o browser  r refresh  ? help")
```

In the help overlay bindings, add `c copy` after `esc back`:

```go
return helpOverlay("Detail — Key Bindings", [][2]string{
	{"↑ / ↓", "scroll content"},
	{"PgUp / PgDn", "scroll page"},
	{"esc", "back to list"},
	{"c", "copy ID to clipboard"},
	{"e", "edit (change type)"},
	{"o", "open in browser"},
	{"r", "refresh"},
	{"?", "toggle this help"},
}, m.width)
```

- [ ] **Step 5: Run all tests**

```bash
go test -v ./cmd/tui/
```

- [ ] **Step 6: Commit**

```bash
git add cmd/tui/detail_view.go cmd/tui/detail_view_test.go
git commit -m "feat(tui): add 'c' key to copy work package ID from detail view"
```

---

## Task 7: Add hjkl navigation and popup infrastructure to filter view

**Files:**
- Modify: `cmd/tui/filter_view.go`
- Modify: `cmd/tui/filter_view_test.go`

- [ ] **Step 1: Add filter popup state and model**

In `cmd/tui/filter_view.go`, add a new state enum and popup model:

```go
type filterState int

const (
	filterBrowseFields filterState = iota
	filterPopup
)
```

Update `filterModel` struct:

```go
type filterModel struct {
	fields       []filterField
	activeField  int
	visible      bool
	originalOpts map[work_packages.FilterOption]string
	showHelp     bool
	state        filterState
	popupItems   []string
	popupIndex   int
}
```

- [ ] **Step 2: Add hjkl navigation test**

In `cmd/tui/filter_view_test.go`, append:

```go
func TestFilterModelHJKLNavigation(t *testing.T) {
	m := newFilterModel()

	if m.activeField != 0 {
		t.Fatalf("expected activeField=0, got %d", m.activeField)
	}

	// j = next field (down)
	m.Update(keyMsg("j"))
	if m.activeField != 1 {
		t.Fatalf("expected activeField=1 after j, got %d", m.activeField)
	}

	// k = previous field (up)
	m.Update(keyMsg("k"))
	if m.activeField != 0 {
		t.Fatalf("expected activeField=0 after k, got %d", m.activeField)
	}

	// l = next value (right)
	m.activeField = 1 // Status: ["all", "open", "closed"]
	m.Update(keyMsg("l"))
	if m.fields[1].current != 1 {
		t.Fatalf("expected current=1 after l, got %d", m.fields[1].current)
	}

	// h = previous value (left)
	m.Update(keyMsg("h"))
	if m.fields[1].current != 0 {
		t.Fatalf("expected current=0 after h, got %d", m.fields[1].current)
	}
}
```

- [ ] **Step 3: Implement hjkl in Update method**

In `filterModel.Update`, add hjkl handling. Change the existing switch to:

```go
func (m *filterModel) Update(msg tea.Msg) tea.Cmd {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return nil
	}

	if m.state == filterPopup {
		return m.updatePopup(keyMsg)
	}

	// filterBrowseFields
	switch keyMsg.String() {
	case "?":
		m.showHelp = !m.showHelp
	case "tab", "down", "j":
		if !m.showHelp {
			m.activeField = (m.activeField + 1) % len(m.fields)
		}
	case "shift+tab", "up", "k":
		if !m.showHelp {
			m.activeField--
			if m.activeField < 0 {
				m.activeField = len(m.fields) - 1
			}
		}
	case "right", "l":
		if !m.showHelp {
			field := &m.fields[m.activeField]
			if field.current < len(field.options)-1 {
				field.current++
			}
		}
	case "left", "h":
		if !m.showHelp {
			field := &m.fields[m.activeField]
			if field.current > 0 {
				field.current--
			}
		}
	case "c":
		if !m.showHelp {
			for i := range m.fields {
				m.fields[i].current = 0
			}
		}
	case "enter":
		if !m.showHelp {
			m.openPopup()
		}
	}
	return nil
}
```

- [ ] **Step 4: Add openPopup and updatePopup methods**

```go
func (m *filterModel) openPopup() {
	field := m.fields[m.activeField]
	m.popupItems = field.options
	m.popupIndex = field.current
	m.state = filterPopup
}

func (m *filterModel) updatePopup(keyMsg tea.KeyMsg) tea.Cmd {
	switch keyMsg.String() {
	case "up", "k":
		if m.popupIndex > 0 {
			m.popupIndex--
		}
	case "down", "j":
		if m.popupIndex < len(m.popupItems)-1 {
			m.popupIndex++
		}
	case "enter":
		m.fields[m.activeField].current = m.popupIndex
		m.state = filterBrowseFields
		m.popupItems = nil
	case "esc":
		m.state = filterBrowseFields
		m.popupItems = nil
	}
	return nil
}
```

- [ ] **Step 5: Update View to render popup**

Change `filterModel.View()` to:

```go
func (m *filterModel) View() string {
	if m.showHelp {
		return helpOverlay("Filter — Key Bindings", [][2]string{
			{"j / k", "next / previous field"},
			{"h / l", "previous / next value"},
			{"tab / shift+tab", "next / previous field"},
			{"← / →", "change value"},
			{"enter", "open popup"},
			{"esc", "cancel / close popup"},
			{"c", "clear all filters"},
			{"?", "toggle this help"},
		}, 60)
	}

	if m.state == filterPopup {
		return m.popupView()
	}

	var b strings.Builder
	b.WriteString(titleStyle.Render("Filters"))
	b.WriteString("\n\n")

	for i, field := range m.fields {
		prefix := "  "
		if i == m.activeField {
			prefix = "▸ "
		}

		value := field.options[field.current]
		line := fmt.Sprintf("%s%-10s [%s]", prefix, field.name+":", value)

		if i == m.activeField {
			b.WriteString(selectedItemStyle.Render(line))
		} else {
			b.WriteString(normalItemStyle.Render(line))
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(helpStyle.Render("  j/k field  h/l value  enter popup  esc cancel  c clear  ? help"))
	return b.String()
}
```

- [ ] **Step 6: Add popupView method**

```go
func (m *filterModel) popupView() string {
	var b strings.Builder
	b.WriteString("\n")
	field := m.fields[m.activeField]
	b.WriteString(titleStyle.Render(field.name))
	b.WriteString("\n\n")

	for i, item := range m.popupItems {
		prefix := "  "
		if i == m.popupIndex {
			prefix = "▸ "
			b.WriteString(selectedItemStyle.Render(prefix + item))
		} else {
			b.WriteString(normalItemStyle.Render(prefix + item))
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(helpStyle.Render("  j/k move  enter select  esc cancel"))
	return b.String()
}
```

- [ ] **Step 7: Run all tests**

```bash
go test -v -run TestFilter ./cmd/tui/
```

- [ ] **Step 8: Commit**

```bash
git add cmd/tui/filter_view.go cmd/tui/filter_view_test.go
git commit -m "feat(tui): add hjkl navigation and popup overlay to filter view"
```

---

## Task 8: Enhance Assignee filter with [all]/[me]/[selected] and h/l toggle

**Files:**
- Modify: `cmd/tui/filter_view.go`
- Modify: `cmd/tui/filter_view_test.go`

Note: With the corrected navigation model, h/l changes values for ALL fields. The assignee toggle works naturally since h/l cycles through `all → me → <user1> → <user2> → ...`. No special assignee-specific logic is needed beyond loading users into the options.

- [ ] **Step 1: Update newFilterModel to load users for assignee**

Update `newFilterModel` default:

```go
func newFilterModel() *filterModel {
	return &filterModel{
		fields: []filterField{
			{name: "Project", options: []string{"all"}},
			{name: "Status", options: []string{"all", "open", "closed"}},
			{name: "Type", options: []string{"all"}},
			{name: "Assignee", options: []string{"all", "me"}},
		},
		originalOpts: make(map[work_packages.FilterOption]string),
	}
}
```

- [ ] **Step 2: Update loadOptions to load users for assignee**

In `loadOptions()`, add user loading after the existing type loading:

```go
// Load users for assignee
if us, err := users.All(); err == nil {
	m.fields[3].options = []string{"all", "me"}
	for _, u := range us {
		m.fields[3].options = append(m.fields[3].options, u.Name)
	}
}
```

Add import: `"github.com/opf/openproject-cli/components/resources/users"`.

- [ ] **Step 3: Add assignee value cycling test**

In `cmd/tui/filter_view_test.go`, append:

```go
func TestFilterAssigneeValueCycling(t *testing.T) {
	m := newFilterModel()
	m.fields[3].options = []string{"all", "me", "John Smith", "Jane Doe"}
	m.fields[3].current = 0 // all
	m.activeField = 3

	// l on Assignee: all -> me
	m.Update(keyMsg("l"))
	if m.fields[3].current != 1 {
		t.Fatalf("expected current=1 (me), got %d", m.fields[3].current)
	}

	// l on Assignee: me -> John Smith
	m.Update(keyMsg("l"))
	if m.fields[3].current != 2 {
		t.Fatalf("expected current=2 (John Smith), got %d", m.fields[3].current)
	}

	// h on Assignee: John Smith -> me
	m.Update(keyMsg("h"))
	if m.fields[3].current != 1 {
		t.Fatalf("expected current=1 (me), got %d", m.fields[3].current)
	}

	// h on Assignee: me -> all
	m.Update(keyMsg("h"))
	if m.fields[3].current != 0 {
		t.Fatalf("expected current=0 (all), got %d", m.fields[3].current)
	}

	// h at all (should stay at 0)
	m.Update(keyMsg("h"))
	if m.fields[3].current != 0 {
		t.Fatalf("expected current=0 (clamped), got %d", m.fields[3].current)
	}
}
```

- [ ] **Step 4: Verify FilterOptions handles assignee correctly**

The existing `FilterOptions()` method already maps field values to filter options. For Assignee, the value "me" maps to `work_packages.Assignee = "me"`, and a user name maps to that name. This already works correctly with the current code.

- [ ] **Step 5: Run all filter tests**

```bash
go test -v -run TestFilter ./cmd/tui/
```

- [ ] **Step 6: Commit**

```bash
git add cmd/tui/filter_view.go cmd/tui/filter_view_test.go
git commit -m "feat(tui): enhance assignee filter with all/me/selected toggle via h/l"
```

---

## Task 9: Ensure list view help bar is complete

**Files:**
- Modify: `cmd/tui/list_view.go`

- [ ] **Step 1: Verify help bar text**

The list view help bar was already updated in Task 4 to:

```go
keys := []string{"j/k", "move", "enter/l", "open", "c", "copy", "/", "search", "f", "filter", "?", "help", "q", "quit"}
```

Also update the help overlay in `listView.View()` to include `l` and `c`:

```go
return helpOverlay("List — Key Bindings", [][2]string{
	{"j / k", "move selection"},
	{"enter / l", "open detail"},
	{"c", "copy ID to clipboard"},
	{"/", "search"},
	{"f", "filter"},
	{"s", "cycle sort"},
	{"r", "refresh"},
	{"?", "toggle this help"},
	{"q", "quit"},
}, m.width)
```

- [ ] **Step 2: Verify build and tests**

```bash
go build ./...
go test -v ./cmd/tui/
```

- [ ] **Step 3: Commit if there were changes**

```bash
git add cmd/tui/list_view.go
git commit -m "fix(tui): complete list view help bar and overlay with all key bindings"
```

---

## Task 10: Final verification

- [ ] **Step 1: Build and run all tests**

```bash
go build ./...
go test -v ./...
```

- [ ] **Step 2: Manual smoke test (optional)**

```bash
go run . tui
```

Verify:
- `l` opens detail from list
- `c` copies `OP#<id>` in both list and detail views
- Filter view: `h/j/k/l` navigate fields and values
- Filter view: `enter` opens popup, `j/k` scrolls, `enter` selects, `esc` cancels
- Assignee: `h/l` toggles between all → me → selected
- Help bars show correct bindings
- `?` help overlays show all bindings

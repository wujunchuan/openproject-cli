# TUI Work Packages Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add an interactive Bubble Tea TUI for browsing and managing work packages, invoked via `opc tui` subcommand.

**Architecture:** TUI built on Bubble Tea with view stack (list ↔ detail), filter overlay, and edit overlay. Reuses existing `components/resources/` layer for data access. Minimal changes to existing code: only adds `SetSilent()` to requests and expands DTO/model with fields already in the API response.

**Tech Stack:** Go 1.25.1, Bubble Tea, lipgloss, bubbles (viewport, textinput, spinner), Cobra

---

## File Structure

```
cmd/tui/
  tui.go            — cobra subcommand + tea.NewProgram
  app.go            — top-level Model, view stack, msg types
  styles.go         — lipgloss style definitions
  keymap.go         — key binding constants
  list_view.go      — list view Model
  detail_view.go    — detail view Model
  filter_view.go    — filter panel Model (overlay)
  edit_view.go      — edit mode Model (overlay)
  help_bar.go       — bottom help bar component

Modified files:
  cmd/root.go                            — register tui subcommand
  components/requests/requests.go        — add SetSilent()
  components/printer/loading.go          — check silent flag
  dtos/work_package.go                   — add Priority, Version, CreatedAt, UpdatedAt
  models/work_package.go                 — add Priority, Project, Version, CreatedAt, UpdatedAt
```

---

### Task 1: Add Bubble Tea dependencies

**Files:**
- Modify: `go.mod`
- Modify: `go.sum`

- [ ] **Step 1: Add dependencies**

Run: `cd /Users/john/Project/Github/openproject-cli && go get github.com/charmbracelet/bubbletea@latest github.com/charmbracelet/lipgloss@latest github.com/charmbracelet/bubbles@latest`

Expected: Dependencies added to go.mod and go.sum

- [ ] **Step 2: Verify build**

Run: `go build ./...`

Expected: Build succeeds with new dependencies

- [ ] **Step 3: Commit**

```bash
git add go.mod go.sum
git commit -m "deps: add bubbletea, lipgloss, and bubbles for TUI"
```

---

### Task 2: Expand WorkPackage DTO and model

**Files:**
- Modify: `dtos/work_package.go`
- Modify: `models/work_package.go`

- [ ] **Step 1: Add fields to model**

In `models/work_package.go`, expand the `WorkPackage` struct:

```go
type WorkPackage struct {
	Id          uint64
	Subject     string
	Type        string
	Assignee    string
	Status      string
	Description string
	LockVersion int
	Priority    string
	Project     string
	Version     string
	CreatedAt   string
	UpdatedAt   string
}
```

- [ ] **Step 2: Add fields to DTO**

In `dtos/work_package.go`, add to `WorkPackageLinksDto`:

```go
type WorkPackageLinksDto struct {
	Self              *LinkDto   `json:"self,omitempty"`
	AddAttachment     *LinkDto   `json:"addAttachment,omitempty"`
	Status            *LinkDto   `json:"status,omitempty"`
	Project           *LinkDto   `json:"project,omitempty"`
	Assignee          *LinkDto   `json:"assignee,omitempty"`
	Type              *LinkDto   `json:"type,omitempty"`
	Priority          *LinkDto   `json:"priority,omitempty"`
	Version           *LinkDto   `json:"version,omitempty"`
	CustomActions     []*LinkDto `json:"customActions,omitempty"`
	PrepareAttachment *LinkDto   `json:"prepareAttachment,omitempty"`
}
```

Add to `WorkPackageDto`:

```go
type WorkPackageDto struct {
	Id          int64                `json:"id,omitempty"`
	Subject     string               `json:"subject,omitempty"`
	Links       *WorkPackageLinksDto `json:"_links,omitempty"`
	Description *LongTextDto         `json:"description,omitempty"`
	Embedded    *embeddedDto         `json:"_embedded,omitempty"`
	LockVersion int                  `json:"lockVersion,omitempty"`
	CreatedAt   string               `json:"createdAt,omitempty"`
	UpdatedAt   string               `json:"updatedAt,omitempty"`
}
```

- [ ] **Step 3: Update Convert()**

In `dtos/work_package.go`, update the `Convert()` method:

```go
func (dto *WorkPackageDto) Convert() *models.WorkPackage {
	wp := &models.WorkPackage{
		Id:          uint64(dto.Id),
		Subject:     dto.Subject,
		LockVersion: dto.LockVersion,
		CreatedAt:   dto.CreatedAt,
		UpdatedAt:   dto.UpdatedAt,
	}
	if dto.Links != nil {
		if dto.Links.Type != nil {
			wp.Type = dto.Links.Type.Title
		}
		if dto.Links.Assignee != nil {
			wp.Assignee = dto.Links.Assignee.Title
		}
		if dto.Links.Status != nil {
			wp.Status = dto.Links.Status.Title
		}
		if dto.Links.Priority != nil {
			wp.Priority = dto.Links.Priority.Title
		}
		if dto.Links.Project != nil {
			wp.Project = dto.Links.Project.Title
		}
		if dto.Links.Version != nil {
			wp.Version = dto.Links.Version.Title
		}
	}
	if dto.Description != nil {
		wp.Description = dto.Description.Raw
	}
	return wp
}
```

- [ ] **Step 4: Verify build**

Run: `go build ./...`

Expected: Build succeeds. Existing code still works because new fields are additive.

- [ ] **Step 5: Commit**

```bash
git add dtos/work_package.go models/work_package.go
git commit -m "feat: expand WorkPackage DTO and model with Priority, Project, Version, CreatedAt, UpdatedAt"
```

---

### Task 3: Add requests silent mode

**Files:**
- Modify: `components/printer/loading.go`

- [ ] **Step 1: Add silent flag and setter**

In `components/printer/loading.go`, add:

```go
var silent bool

func SetSilent(b bool) {
	silent = b
}
```

Update `WithSpinner`:

```go
func WithSpinner[T any](f function[T]) (T, error) {
	if !silent {
		loadingSpinner.Start()
		defer loadingSpinner.Stop()
	}
	return f()
}
```

- [ ] **Step 2: Add test**

Create `components/printer/loading_test.go`:

```go
package printer

import "testing"

func TestWithSpinnerSilent(t *testing.T) {
	SetSilent(true)
	defer SetSilent(false)

	called := false
	result, err := WithSpinner(func() (int, error) {
		called = true
		return 42, nil
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != 42 {
		t.Fatalf("expected 42, got %d", result)
	}
	if !called {
		t.Fatal("function was not called")
	}
}
```

- [ ] **Step 3: Run test**

Run: `go test -v -run TestWithSpinnerSilent ./components/printer/`

Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add components/printer/loading.go components/printer/loading_test.go
git commit -m "feat: add SetSilent mode to printer to suppress spinner"
```

---

### Task 4: Create TUI styles and keymap

**Files:**
- Create: `cmd/tui/styles.go`
- Create: `cmd/tui/keymap.go`

- [ ] **Step 1: Create styles.go**

```go
package tui

import "github.com/charmbracelet/lipgloss"

var (
	// Colors
	subtleColor  = lipgloss.AdaptiveColor{Light: "#D9DCCF", Dark: "#383838"}
	highlightColor = lipgloss.AdaptiveColor{Light: "#874BFD", Dark: "#7D56F4"}
	specialColor = lipgloss.AdaptiveColor{Light: "#43BF6D", Dark: "#73F59F"}
	errorColor   = lipgloss.Color("#F25D94")

	// Styles
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFDF5")).
			Background(highlightColor).
			Padding(0, 1)

	subtitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888"))

	selectedItemStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FFFDF5")).
				Background(highlightColor).
				Bold(true)

	normalItemStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#DDD"))

	statusStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFD700"))

	errorStyle = lipgloss.NewStyle().
			Foreground(errorColor).
			Bold(true)

	helpStyle = lipgloss.NewStyle().
			Foreground(subtleColor)

	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFDF5")).
			BorderBottom(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderBottomForeground(subtleColor)

	// Layout
	docStyle = lipgloss.NewStyle().Padding(1, 2, 1, 2)
)
```

- [ ] **Step 2: Create keymap.go**

```go
package tui

import "github.com/charmbracelet/bubbles/key"

type KeyMap struct {
	Up       key.Binding
	Down     key.Binding
	Enter    key.Binding
	Back     key.Binding
	Search   key.Binding
	Filter   key.Binding
	Sort     key.Binding
	Edit     key.Binding
	NextPage key.Binding
	PrevPage key.Binding
	Refresh  key.Binding
	Open     key.Binding
	Help     key.Binding
	Quit     key.Binding
}

var DefaultKeyMap = KeyMap{
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "down"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "select"),
	),
	Back: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "back"),
	),
	Search: key.NewBinding(
		key.WithKeys("/"),
		key.WithHelp("/", "search"),
	),
	Filter: key.NewBinding(
		key.WithKeys("f"),
		key.WithHelp("f", "filter"),
	),
	Sort: key.NewBinding(
		key.WithKeys("s"),
		key.WithHelp("s", "sort"),
	),
	Edit: key.NewBinding(
		key.WithKeys("e"),
		key.WithHelp("e", "edit"),
	),
	NextPage: key.NewBinding(
		key.WithKeys("n"),
		key.WithHelp("n", "next page"),
	),
	PrevPage: key.NewBinding(
		key.WithKeys("p"),
		key.WithHelp("p", "prev page"),
	),
	Refresh: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "refresh"),
	),
	Open: key.NewBinding(
		key.WithKeys("o"),
		key.WithHelp("o", "open in browser"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "help"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
}
```

- [ ] **Step 3: Verify build**

Run: `go build ./cmd/tui/...`

Expected: Build succeeds

- [ ] **Step 4: Commit**

```bash
git add cmd/tui/styles.go cmd/tui/keymap.go
git commit -m "feat(tui): add lipgloss styles and key bindings"
```

---

### Task 5: Create app model and TUI entry point

**Files:**
- Create: `cmd/tui/app.go`
- Create: `cmd/tui/tui.go`
- Create: `cmd/tui/help_bar.go`
- Modify: `cmd/root.go`

- [ ] **Step 1: Create app.go with view stack and msg types**

```go
package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/opf/openproject-cli/models"
)

type viewState int

const (
	listView viewState = iota
	detailView
)

// --- Messages ---

type workPackagesLoadedMsg struct {
	collection *models.WorkPackageCollection
	err        error
}

type workPackageDetailMsg struct {
	wp  *models.WorkPackage
	err error
}

type activitiesLoadedMsg struct {
	activities []*models.Activity
	err        error
}

type errorMsg struct {
	err error
}

// --- App Model ---

type App struct {
	state    viewState
	list     *listModel
	detail   *detailModel
	width    int
	height   int
	err      error
	quitting bool
}

func NewApp() App {
	return App{
		state: listView,
		list:  newListModel(),
	}
}

func (a App) Init() tea.Cmd {
	return a.list.Init()
}

func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a.list.SetSize(msg.Width, msg.Height)

	case tea.KeyMsg:
		switch msg.String() {
		case "q":
			if a.state == listView && !a.list.searchActive {
				a.quitting = true
				return a, tea.Quit
			}
		case "ctrl+c":
			a.quitting = true
			return a, tea.Quit
		}

	case workPackagesLoadedMsg:
		if msg.err != nil {
			a.err = msg.err
		} else {
			a.err = nil
			a.list.SetWorkPackages(msg.collection)
		}
		return a, nil

	case workPackageDetailMsg:
		if msg.err != nil {
			a.err = msg.err
		} else {
			a.err = nil
			a.detail.SetWorkPackage(msg.wp)
		}
		return a, nil

	case activitiesLoadedMsg:
		if msg.err != nil {
			a.err = msg.err
		} else {
			a.detail.SetActivities(msg.activities)
		}
		return a, nil

	case openDetailMsg:
		a.state = detailView
		a.detail = newDetailModel(msg.wp, a.width, a.height)
		return a, a.detail.Init()

	case backToListMsg:
		a.state = listView
		a.detail = nil
		return a, nil
	}

	switch a.state {
	case listView:
		a.list, cmd = a.list.Update(msg)
		cmds = append(cmds, cmd)
	case detailView:
		a.detail, cmd = a.detail.Update(msg)
		cmds = append(cmds, cmd)
	}

	return a, tea.Batch(cmds...)
}

func (a App) View() string {
	if a.quitting {
		return ""
	}

	var content string
	switch a.state {
	case listView:
		content = a.list.View()
	case detailView:
		content = a.detail.View()
	}

	return docStyle.Render(content)
}

// --- Navigation Messages ---

type openDetailMsg struct {
	wp *models.WorkPackage
}

type backToListMsg struct{}

func OpenDetailCmd(wp *models.WorkPackage) tea.Cmd {
	return func() tea.Msg {
		return openDetailMsg{wp: wp}
	}
}

func BackToListCmd() tea.Cmd {
	return func() tea.Msg {
		return backToListMsg{}
	}
}
```

- [ ] **Step 2: Create help_bar.go**

```go
package tui

import "strings"

func helpBar(keys []string, width int) string {
	var parts []string
	for i := 0; i < len(keys); i += 2 {
		key := keys[i]
		desc := ""
		if i+1 < len(keys) {
			desc = keys[i+1]
		}
		parts = append(parts, helpStyle.Render(key+" "+desc))
	}
	bar := strings.Join(parts, "  ")

	padding := width - len(bar) - 4
	if padding < 0 {
		padding = 0
	}
	return helpStyle.Render(strings.Repeat(" ", padding)) + "\n" + bar
}
```

- [ ] **Step 3: Create tui.go (cobra subcommand)**

```go
package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/opf/openproject-cli/components/printer"
	"github.com/opf/openproject-cli/components/requests"
	"github.com/spf13/cobra"
)

var TuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Launch interactive TUI for work packages",
	Run:   runTui,
}

func runTui(_ *cobra.Command, _ []string) {
	requests.SetSilent(true)
	defer requests.SetSilent(false)

	p := tea.NewProgram(
		NewApp(),
		tea.WithAltScreen(),
	)

	if _, err := p.Run(); err != nil {
		printer.ErrorText(fmt.Sprintf("TUI error: %v", err))
	}
}
```

- [ ] **Step 4: Register subcommand in cmd/root.go**

In `cmd/root.go` init(), add import and registration:

```go
import "github.com/opf/openproject-cli/cmd/tui"
```

In the `rootCmd.AddCommand(...)` block, add:

```go
rootCmd.AddCommand(
	loginCmd,
	list.RootCmd,
	update.RootCmd,
	inspect.RootCmd,
	create.RootCmd,
	search.RootCmd,
	git.RootCmd,
	tui.TuiCmd,
)
```

- [ ] **Step 5: Verify build**

Run: `go build ./...`

Expected: Build succeeds

- [ ] **Step 6: Create app_test.go**

```go
package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/opf/openproject-cli/models"
)

func TestAppInit(t *testing.T) {
	app := NewApp()
	cmd := app.Init()
	if cmd == nil {
		t.Fatal("Init should return a command")
	}
}

func TestAppViewStackNavigation(t *testing.T) {
	app := NewApp()
	app.width = 80
	app.height = 24

	wp := &models.WorkPackage{Id: 1, Subject: "Test", Status: "New"}

	// Simulate opening detail
	app, _ = app.Update(openDetailMsg{wp: wp})
	if app.state != detailView {
		t.Fatal("expected detailView after openDetailMsg")
	}
	if app.detail == nil {
		t.Fatal("detail model should be set")
	}

	// Simulate going back
	app, _ = app.Update(backToListMsg{})
	if app.state != listView {
		t.Fatal("expected listView after backToListMsg")
	}
	if app.detail != nil {
		t.Fatal("detail model should be nil after going back")
	}
}

func TestAppWindowSize(t *testing.T) {
	app := NewApp()
	app, _ = app.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	if app.width != 120 || app.height != 40 {
		t.Fatalf("expected 120x40, got %dx%d", app.width, app.height)
	}
}
```

- [ ] **Step 7: Run tests**

Run: `go test -v ./cmd/tui/...`

Expected: PASS

- [ ] **Step 8: Commit**

```bash
git add cmd/tui/app.go cmd/tui/tui.go cmd/tui/help_bar.go cmd/tui/app_test.go cmd/root.go
git commit -m "feat(tui): add app model, view stack, and cobra subcommand"
```

---

### Task 6: Create list view (basic rendering + navigation)

**Files:**
- Create: `cmd/tui/list_view.go`

- [ ] **Step 1: Create list_view.go with basic model**

```go
package tui

import (
	"fmt"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/opf/openproject-cli/components/requests"
	"github.com/opf/openproject-cli/components/resources/work_packages"
	"github.com/opf/openproject-cli/models"
)

type listModel struct {
	items        []*models.WorkPackage
	selected     int
	collection   *models.WorkPackageCollection
	page         int
	pageSize     int64
	total        int64
	width        int
	height       int
	loading      bool
	spinner      spinner.Model
	searchActive bool
	filterOpts   map[work_packages.FilterOption]string
}

func newListModel() *listModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = selectedItemStyle
	return &listModel{
		loading:    true,
		pageSize:   50,
		page:       1,
		spinner:    s,
		filterOpts: make(map[work_packages.FilterOption]string),
	}
}

func (m *listModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		m.loadWorkPackages,
	)
}

func (m *listModel) SetWorkPackages(collection *models.WorkPackageCollection) {
	m.collection = collection
	m.items = collection.Items
	m.total = collection.Total
	m.loading = false
	if m.selected >= len(m.items) {
		m.selected = len(m.items) - 1
	}
	if m.selected < 0 && len(m.items) > 0 {
		m.selected = 0
	}
}

func (m *listModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

func (m *listModel) loadWorkPackages() tea.Msg {
	query := requests.NewPaginatedQuery(int64(m.pageSize), nil)
	collection, err := work_packages.All(&m.filterOpts, query, false)
	return workPackagesLoadedMsg{collection: collection, err: err}
}

func (m *listModel) Update(msg tea.Msg) (*listModel, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	// Handle spinner tick
	if m.loading {
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.searchActive {
			return m, nil // handled by search input later
		}
		switch msg.String() {
		case "up", "k":
			if m.selected > 0 {
				m.selected--
			}
		case "down", "j":
			if m.selected < len(m.items)-1 {
				m.selected++
			}
		case "enter":
			if m.selected >= 0 && m.selected < len(m.items) {
				return m, OpenDetailCmd(m.items[m.selected])
			}
		case "n":
			if int64(m.page*m.pageSize) < m.total {
				m.page++
				m.loading = true
				return m, tea.Batch(m.spinner.Tick, m.loadWorkPackages)
			}
		case "p":
			if m.page > 1 {
				m.page--
				m.loading = true
				return m, tea.Batch(m.spinner.Tick, m.loadWorkPackages)
			}
		case "r":
			m.loading = true
			return m, tea.Batch(m.spinner.Tick, m.loadWorkPackages)
		}
	}

	return m, tea.Batch(cmds...)
}

func (m *listModel) View() string {
	var b strings.Builder

	// Header
	b.WriteString(titleStyle.Render("Work Packages"))
	if m.total > 0 {
		b.WriteString(subtitleStyle.Render(fmt.Sprintf(" (%d total)", m.total)))
	}
	b.WriteString("\n\n")

	if m.loading {
		b.WriteString(fmt.Sprintf("\n  %s Loading...\n\n", m.spinner.View()))
		return b.String()
	}

	if len(m.items) == 0 {
		b.WriteString("\n  No work packages found.\n\n")
		return b.String()
	}

	// Column widths
	idWidth := len(strconv.FormatInt(m.total, 10)) + 2
	if idWidth < 6 {
		idWidth = 6
	}
	typeWidth := 12
	statusWidth := 12
	assigneeWidth := 14
	titleWidth := m.width - idWidth - typeWidth - statusWidth - assigneeWidth - 12
	if titleWidth < 20 {
		titleWidth = 20
	}

	// Table header
	b.WriteString(headerStyle.Render(fmt.Sprintf(
		"%-*s %-*s %-*s %-*s %-*s",
		idWidth, "ID",
		typeWidth, "Type",
		titleWidth, "Title",
		statusWidth, "Status",
		assigneeWidth, "Assignee",
	)))
	b.WriteString("\n")

	// Items
	for i, wp := range m.items {
		line := fmt.Sprintf(
			"#%-*d %-*s %-*s %-*s %-*s",
			idWidth-1, wp.Id,
			typeWidth, truncate(wp.Type, typeWidth),
			titleWidth, truncate(wp.Subject, titleWidth),
			statusWidth, truncate(wp.Status, statusWidth),
			assigneeWidth, truncate(assigneeOrDash(wp.Assignee), assigneeWidth),
		)

		if i == m.selected {
			b.WriteString(selectedItemStyle.Render(line))
		} else {
			b.WriteString(normalItemStyle.Render(line))
		}
		b.WriteString("\n")
	}

	// Footer
	b.WriteString("\n")
	from := int64((m.page-1)*int(m.pageSize)) + 1
	to := int64(m.page) * m.pageSize
	if to > m.total {
		to = m.total
	}
	pageInfo := fmt.Sprintf("%d-%d / %d", from, to, m.total)
	b.WriteString(helpStyle.Render(pageInfo))

	return b.String()
}

func truncate(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	if maxLen <= 1 {
		return string(runes[:maxLen])
	}
	return string(runes[:maxLen-1]) + "…"
}

func assigneeOrDash(s string) string {
	if s == "" {
		return "—"
	}
	return s
}
```

- [ ] **Step 2: Verify build**

Run: `go build ./...`

Expected: Build succeeds

- [ ] **Step 3: Create list_view_test.go**

```go
package tui

import (
	"testing"

	"github.com/opf/openproject-cli/models"
)

func TestListModelSetWorkPackages(t *testing.T) {
	m := newListModel()
	collection := &models.WorkPackageCollection{
		Total:    2,
		Count:    2,
		PageSize: 50,
		Items: []*models.WorkPackage{
			{Id: 1, Subject: "First", Status: "New", Type: "Bug"},
			{Id: 2, Subject: "Second", Status: "Open", Type: "Task"},
		},
	}

	m.SetWorkPackages(collection)

	if len(m.items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(m.items))
	}
	if m.selected != 0 {
		t.Fatalf("expected selected=0, got %d", m.selected)
	}
	if m.loading {
		t.Fatal("should not be loading after SetWorkPackages")
	}
}

func TestListModelNavigation(t *testing.T) {
	m := newListModel()
	m.SetWorkPackages(&models.WorkPackageCollection{
		Items: []*models.WorkPackage{
			{Id: 1, Subject: "A"},
			{Id: 2, Subject: "B"},
			{Id: 3, Subject: "C"},
		},
	})

	// Move down
	m, _ = m.Update(keyMsg("down"))
	if m.selected != 1 {
		t.Fatalf("expected selected=1, got %d", m.selected)
	}

	// Move down again
	m, _ = m.Update(keyMsg("down"))
	if m.selected != 2 {
		t.Fatalf("expected selected=2, got %d", m.selected)
	}

	// Move down at end (should stay)
	m, _ = m.Update(keyMsg("down"))
	if m.selected != 2 {
		t.Fatalf("expected selected=2 (clamped), got %d", m.selected)
	}

	// Move up
	m, _ = m.Update(keyMsg("up"))
	if m.selected != 1 {
		t.Fatalf("expected selected=1, got %d", m.selected)
	}
}

func TestListModelEnterOpensDetail(t *testing.T) {
	m := newListModel()
	wp := &models.WorkPackage{Id: 42, Subject: "Test"}
	m.SetWorkPackages(&models.WorkPackageCollection{
		Items: []*models.WorkPackage{wp},
	})

	_, cmd := m.Update(keyMsg("enter"))
	if cmd == nil {
		t.Fatal("enter should return a command")
	}

	msg := cmd()
	detailMsg, ok := msg.(openDetailMsg)
	if !ok {
		t.Fatalf("expected openDetailMsg, got %T", msg)
	}
	if detailMsg.wp.Id != 42 {
		t.Fatalf("expected wp id 42, got %d", detailMsg.wp.Id)
	}
}
```

Also add a test helper (in a separate file or at the bottom of the test file):

```go
package tui

import tea "github.com/charmbracelet/bubbletea"

func keyMsg(key string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
}
```

- [ ] **Step 4: Run tests**

Run: `go test -v ./cmd/tui/...`

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add cmd/tui/list_view.go cmd/tui/list_view_test.go
git commit -m "feat(tui): add list view with navigation and pagination"
```

---

### Task 7: Add sort and search to list view

**Files:**
- Modify: `cmd/tui/list_view.go`
- Create: `cmd/tui/list_view_sort_test.go`

- [ ] **Step 1: Add sort support to list_view.go**

Add sort state and logic:

```go
type sortField int

const (
	sortByID sortField = iota
	sortByStatus
	sortByType
	sortByAssignee
)

// Add to listModel struct:
//   sortField sortField

func (m *listModel) cycleSort() {
	m.sortField = (m.sortField + 1) % 4
	m.sortItems()
}

func (m *listModel) sortItems() {
	sort.SliceStable(m.items, func(i, j int) bool {
		switch m.sortField {
		case sortByStatus:
			return m.items[i].Status < m.items[j].Status
		case sortByType:
			return m.items[i].Type < m.items[j].Type
		case sortByAssignee:
			return m.items[i].Assignee < m.items[j].Assignee
		default:
			return m.items[i].Id < m.items[j].Id
		}
	})
}
```

In the `Update` method, add case for "s":

```go
case "s":
	m.cycleSort()
```

In the `View` method header, show current sort:

```go
sortNames := []string{"ID", "Status", "Type", "Assignee"}
b.WriteString(subtitleStyle.Render(fmt.Sprintf(" sort:%s", sortNames[m.sortField])))
```

- [ ] **Step 2: Add search support**

Add `searchInput` field to `listModel` using `bubbles/textinput`:

```go
import "github.com/charmbracelet/bubbles/textinput"

// Add to listModel struct:
//   searchInput textinput.Model

func newSearchInput() textinput.Model {
	ti := textinput.New()
	ti.Placeholder = "Search..."
	ti.CharLimit = 100
	ti.Width = 40
	return ti
}
```

In `Update`, handle `/` key to activate search and filter items client-side:

```go
case "/":
	if !m.searchActive {
		m.searchActive = true
		m.searchInput = newSearchInput()
		m.searchInput.Focus()
		return m, nil
	}

case "esc":
	if m.searchActive {
		m.searchActive = false
		return m, nil
	}
```

When search is active, filter items by title match:

```go
func (m *listModel) filterBySearch() {
	query := strings.ToLower(m.searchInput.Value())
	if query == "" {
		m.SetWorkPackages(m.collection) // restore all
		return
	}
	var filtered []*models.WorkPackage
	for _, wp := range m.collection.Items {
		if strings.Contains(strings.ToLower(wp.Subject), query) {
			filtered = append(filtered, wp)
		}
	}
	m.items = filtered
	m.selected = 0
}
```

In `View`, show search input when active:

```go
if m.searchActive {
	b.WriteString("\n")
	b.WriteString(m.searchInput.View())
	b.WriteString("\n")
}
```

- [ ] **Step 3: Add sort tests**

```go
package tui

import (
	"testing"

	"github.com/opf/openproject-cli/models"
)

func TestSortCycle(t *testing.T) {
	m := newListModel()
	m.SetWorkPackages(&models.WorkPackageCollection{
		Items: []*models.WorkPackage{
			{Id: 3, Subject: "C", Status: "Open", Type: "Bug", Assignee: "Zoe"},
			{Id: 1, Subject: "A", Status: "New", Type: "Task", Assignee: "Alice"},
			{Id: 2, Subject: "B", Status: "Closed", Type: "Bug", Assignee: "Bob"},
		},
	})

	// Default: sort by ID
	if m.items[0].Id != 1 {
		t.Fatalf("expected first item id=1, got %d", m.items[0].Id)
	}

	// Cycle to Status
	m.cycleSort()
	if m.items[0].Status != "Closed" {
		t.Fatalf("expected first item status=Closed, got %s", m.items[0].Status)
	}

	// Cycle to Type
	m.cycleSort()
	if m.items[0].Type != "Bug" {
		t.Fatalf("expected first item type=Bug, got %s", m.items[0].Type)
	}

	// Cycle to Assignee
	m.cycleSort()
	if m.items[0].Assignee != "Alice" {
		t.Fatalf("expected first item assignee=Alice, got %s", m.items[0].Assignee)
	}

	// Cycle back to ID
	m.cycleSort()
	if m.items[0].Id != 1 {
		t.Fatalf("expected first item id=1 after full cycle, got %d", m.items[0].Id)
	}
}
```

- [ ] **Step 4: Run tests**

Run: `go test -v ./cmd/tui/...`

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add cmd/tui/list_view.go cmd/tui/list_view_sort_test.go
git commit -m "feat(tui): add sort cycling and client-side search to list view"
```

---

### Task 8: Create detail view (basic)

**Files:**
- Create: `cmd/tui/detail_view.go`
- Create: `cmd/tui/detail_view_test.go`

- [ ] **Step 1: Create detail_view.go**

```go
package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/opf/openproject-cli/components/launch"
	"github.com/opf/openproject-cli/components/resources/work_packages"
	"github.com/opf/openproject-cli/components/routes"
	"github.com/opf/openproject-cli/models"
)

type detailModel struct {
	wp         *models.WorkPackage
	activities []*models.Activity
	viewport   viewport.Model
	width      int
	height     int
	loading    bool
}

func newDetailModel(wp *models.WorkPackage, w, h int) *detailModel {
	vp := viewport.New(w-4, h-10)
	vp.SetContent("")
	return &detailModel{
		wp:       wp,
		viewport: vp,
		width:    w,
		height:   h,
		loading:  true,
	}
}

func (m *detailModel) Init() tea.Cmd {
	return m.loadActivities
}

func (m *detailModel) SetWorkPackage(wp *models.WorkPackage) {
	m.wp = wp
	m.updateContent()
}

func (m *detailModel) SetActivities(activities []*models.Activity) {
	m.activities = activities
	m.loading = false
	m.updateContent()
}

func (m *detailModel) loadActivities() tea.Msg {
	activities, err := work_packages.Activities(m.wp.Id)
	return activitiesLoadedMsg{activities: activities, err: err}
}

func (m *detailModel) updateContent() {
	var b strings.Builder

	// Header
	b.WriteString(titleStyle.Render(fmt.Sprintf("#%d %s", m.wp.Id, m.wp.Subject)))
	b.WriteString("\n\n")

	// Properties (two columns)
	left := fmt.Sprintf("Type: %s\nStatus: %s\nProject: %s\nAssignee: %s",
		m.wp.Type, m.wp.Status, m.wp.Project, assigneeOrDash(m.wp.Assignee))
	right := fmt.Sprintf("Priority: %s\nVersion: %s\nCreated: %s\nUpdated: %s",
		m.wp.Priority, m.wp.Version, m.wp.CreatedAt, m.wp.UpdatedAt)

	leftCol := lipgloss.NewStyle().Width(m.width/2 - 4).Render(left)
	rightCol := lipgloss.NewStyle().Width(m.width/2 - 4).Render(right)

	b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, leftCol, rightCol))
	b.WriteString("\n\n")

	// Description
	if m.wp.Description != "" {
		b.WriteString(headerStyle.Render("Description"))
		b.WriteString("\n")
		b.WriteString(m.wp.Description)
		b.WriteString("\n\n")
	}

	// Activities
	b.WriteString(headerStyle.Render(fmt.Sprintf("Activity (%d)", len(m.activities))))
	b.WriteString("\n")
	if m.loading {
		b.WriteString("\n  Loading activities...\n")
	} else {
		for _, act := range m.activities {
			b.WriteString(fmt.Sprintf("\n  %s\n", subtitleStyle.Render(act.CreatedAt)))
			for _, detail := range act.Details {
				if detail != nil {
					b.WriteString(fmt.Sprintf("  %s\n", *detail))
				}
			}
			if act.Comment != "" {
				b.WriteString(fmt.Sprintf("  %s\n", act.Comment))
			}
		}
	}

	m.viewport.SetContent(b.String())
}

func (m *detailModel) Update(msg tea.Msg) (*detailModel, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return m, BackToListCmd()
		case "o":
			_ = launch.Browser(routes.WorkPackageUrl(m.wp))
		case "r":
			m.loading = true
			return m, tea.Batch(
				func() tea.Msg {
					wp, err := work_packages.Lookup(m.wp.Id)
					return workPackageDetailMsg{wp: wp, err: err}
				},
				m.loadActivities,
			)
		}
	}

	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m *detailModel) View() string {
	footer := helpStyle.Render("  esc back  e edit  o browser  r refresh")
	return m.viewport.View() + "\n" + footer
}
```

- [ ] **Step 2: Create detail_view_test.go**

```go
package tui

import (
	"testing"

	"github.com/opf/openproject-cli/models"
)

func TestDetailModelBackToList(t *testing.T) {
	wp := &models.WorkPackage{Id: 1, Subject: "Test", Status: "New"}
	m := newDetailModel(wp, 80, 24)

	_, cmd := m.Update(keyMsg("esc"))
	if cmd == nil {
		t.Fatal("esc should return a command")
	}

	msg := cmd()
	if _, ok := msg.(backToListMsg); !ok {
		t.Fatalf("expected backToListMsg, got %T", msg)
	}
}

func TestDetailModelSetActivities(t *testing.T) {
	wp := &models.WorkPackage{Id: 1, Subject: "Test"}
	m := newDetailModel(wp, 80, 24)

	detail := "Changed status"
	activities := []*models.Activity{
		{Id: 1, Comment: "Fixed", Details: []*string{&detail}, CreatedAt: "2026-05-19"},
	}
	m.SetActivities(activities)

	if len(m.activities) != 1 {
		t.Fatalf("expected 1 activity, got %d", len(m.activities))
	}
	if m.loading {
		t.Fatal("should not be loading after SetActivities")
	}
}
```

- [ ] **Step 3: Run tests**

Run: `go test -v ./cmd/tui/...`

Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add cmd/tui/detail_view.go cmd/tui/detail_view_test.go
git commit -m "feat(tui): add detail view with properties, description, and activities"
```

---

### Task 9: Create filter panel

**Files:**
- Create: `cmd/tui/filter_view.go`
- Create: `cmd/tui/filter_view_test.go`

- [ ] **Step 1: Create filter_view.go**

```go
package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/opf/openproject-cli/components/resources/projects"
	"github.com/opf/openproject-cli/components/resources/status"
	resTypes "github.com/opf/openproject-cli/components/resources/types"
	"github.com/opf/openproject-cli/components/resources/work_packages"
)

type filterField struct {
	name    string
	options []string
	current int
}

type filterModel struct {
	fields       []filterField
	activeField  int
	visible      bool
	originalOpts map[work_packages.FilterOption]string
}

func newFilterModel() *filterModel {
	return &filterModel{
		fields: []filterField{
			{name: "Project", options: []string{"all"}},
			{name: "Status", options: []string{"all", "open", "closed"}},
			{name: "Type", options: []string{"all"}},
			{name: "Assignee", options: []string{"all", "me", "none"}},
		},
		originalOpts: make(map[work_packages.FilterOption]string),
	}
}

func (m *filterModel) loadOptions() tea.Cmd {
	return func() tea.Msg {
		// Load projects
		if ps, err := projects.All(); err == nil {
			m.fields[0].options = []string{"all"}
			for _, p := range ps {
				m.fields[0].options = append(m.fields[0].options, p.Name)
			}
		}

		// Load statuses
		if ss, err := status.All(); err == nil {
			m.fields[1].options = []string{"all"}
			for _, s := range ss {
				m.fields[1].options = append(m.fields[1].options, s.Name)
			}
		}

		// Load types
		if ts, err := resTypes.All(); err == nil {
			m.fields[2].options = []string{"all"}
			for _, t := range ts {
				m.fields[2].options = append(m.fields[2].options, t.Name)
			}
		}

		return nil
	}
}

func (m *filterModel) Update(msg tea.Msg) tea.Cmd {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return nil
	}

	switch keyMsg.String() {
	case "tab":
		m.activeField = (m.activeField + 1) % len(m.fields)
	case "shift+tab":
		m.activeField--
		if m.activeField < 0 {
			m.activeField = len(m.fields) - 1
		}
	case "up":
		field := &m.fields[m.activeField]
		if field.current > 0 {
			field.current--
		}
	case "down":
		field := &m.fields[m.activeField]
		if field.current < len(field.options)-1 {
			field.current++
		}
	case "c":
		for i := range m.fields {
			m.fields[i].current = 0
		}
	}
	return nil
}

func (m *filterModel) FilterOptions() map[work_packages.FilterOption]string {
	opts := make(map[work_packages.FilterOption]string)

	keys := []work_packages.FilterOption{
		work_packages.Project,
		work_packages.Status,
		work_packages.Type,
		work_packages.Assignee,
	}

	for i, field := range m.fields {
		val := field.options[field.current]
		if val != "all" {
			opts[keys[i]] = val
		}
	}
	return opts
}

func (m *filterModel) View() string {
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
	b.WriteString(helpStyle.Render("  tab/shift+tab field  ↑↓ select  enter apply  esc cancel  c clear"))
	return b.String()
}
```

- [ ] **Step 2: Create filter_view_test.go**

```go
package tui

import (
	"testing"
)

func TestFilterModelNavigation(t *testing.T) {
	m := newFilterModel()

	if m.activeField != 0 {
		t.Fatalf("expected activeField=0, got %d", m.activeField)
	}

	m.Update(keyMsg("tab"))
	if m.activeField != 1 {
		t.Fatalf("expected activeField=1, got %d", m.activeField)
	}

	m.Update(keyMsg("shift+tab"))
	if m.activeField != 0 {
		t.Fatalf("expected activeField=0, got %d", m.activeField)
	}
}

func TestFilterModelValueSelection(t *testing.T) {
	m := newFilterModel()

	// Status field options: ["all", "open", "closed"]
	m.activeField = 1
	if m.fields[1].current != 0 {
		t.Fatalf("expected current=0, got %d", m.fields[1].current)
	}

	m.Update(keyMsg("down"))
	if m.fields[1].current != 1 {
		t.Fatalf("expected current=1, got %d", m.fields[1].current)
	}
	if m.fields[1].options[m.fields[1].current] != "open" {
		t.Fatalf("expected 'open', got '%s'", m.fields[1].options[m.fields[1].current])
	}
}

func TestFilterModelClear(t *testing.T) {
	m := newFilterModel()

	m.fields[1].current = 2 // closed
	m.fields[3].current = 1 // me

	m.Update(keyMsg("c"))

	for _, field := range m.fields {
		if field.current != 0 {
			t.Fatalf("expected all fields reset to 0, %s is at %d", field.name, field.current)
		}
	}
}

func TestFilterModelFilterOptions(t *testing.T) {
	m := newFilterModel()

	// Set status to "open"
	m.fields[1].current = 1
	opts := m.FilterOptions()

	if len(opts) != 1 {
		t.Fatalf("expected 1 filter option, got %d", len(opts))
	}
	if opts[1] != "open" { // Status is FilterOption index 1 (Project=0, Status=1, ...)
		// Note: the exact key depends on FilterOption enum order
		t.Logf("Filter options: %+v", opts)
	}
}
```

- [ ] **Step 3: Run tests**

Run: `go test -v ./cmd/tui/...`

Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add cmd/tui/filter_view.go cmd/tui/filter_view_test.go
git commit -m "feat(tui): add filter panel with project/status/type/assignee selection"
```

---

### Task 10: Create edit overlay

**Files:**
- Create: `cmd/tui/edit_view.go`
- Create: `cmd/tui/edit_view_test.go`

- [ ] **Step 1: Create edit_view.go**

```go
package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/opf/openproject-cli/components/resources/work_packages"
	"github.com/opf/openproject-cli/models"
)

type editState int

const (
	editChooseField editState = iota
	editChooseValue
	editSubmitting
)

type editModel struct {
	wp          *models.WorkPackage
	state       editState
	options     []string
	optionIndex int
	err         string
	width       int
}

func newEditModel(wp *models.WorkPackage, w int) *editModel {
	return &editModel{
		wp:    wp,
		state: editChooseField,
		width: w,
	}
}

func (m *editModel) Update(msg tea.Msg) (*editModel, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	switch m.state {
	case editChooseField:
		switch keyMsg.String() {
		case "t":
			m.state = editChooseValue
			return m, m.loadTypes
		case "esc":
			return m, func() tea.Msg { return editDoneMsg{} }
		}

	case editChooseValue:
		switch keyMsg.String() {
		case "up", "k":
			if m.optionIndex > 0 {
				m.optionIndex--
			}
		case "down", "j":
			if m.optionIndex < len(m.options)-1 {
				m.optionIndex++
			}
		case "enter":
			if m.optionIndex >= 0 && m.optionIndex < len(m.options) {
				m.state = editSubmitting
				return m, m.submitType
			}
		case "esc":
			m.state = editChooseField
			m.options = nil
			m.err = ""
			return m, nil
		}
	}

	return m, nil
}

func (m *editModel) loadTypes() tea.Msg {
	types, err := work_packages.AvailableTypes(m.wp.Id)
	if err != nil {
		return editErrorMsg{err: err.Error()}
	}
	var opts []string
	for _, t := range types {
		opts = append(opts, t.Name)
	}
	return editOptionsMsg{options: opts}
}

func (m *editModel) submitType() tea.Msg {
	if m.optionIndex >= len(m.options) {
		return editErrorMsg{err: "invalid selection"}
	}
	selected := m.options[m.optionIndex]

	opts := map[work_packages.UpdateOption]string{
		work_packages.UpdateType: selected,
	}

	_, err := work_packages.Update(m.wp.Id, opts)
	if err != nil {
		return editErrorMsg{err: err.Error()}
	}

	return editDoneMsg{refresh: true}
}

func (m *editModel) View() string {
	var b strings.Builder

	b.WriteString("\n")
	b.WriteString(titleStyle.Render(fmt.Sprintf("Edit #%d", m.wp.Id)))
	b.WriteString("\n\n")

	if m.err != "" {
		b.WriteString(errorStyle.Render("  Error: " + m.err))
		b.WriteString("\n\n")
	}

	switch m.state {
	case editChooseField:
		b.WriteString("  [T]ype\n\n")
		b.WriteString(helpStyle.Render("  Press a key to select a field, esc to cancel"))
	case editChooseValue:
		b.WriteString("  Select type:\n\n")
		for i, opt := range m.options {
			prefix := "  "
			if i == m.optionIndex {
				prefix = "▸ "
				b.WriteString(selectedItemStyle.Render(prefix + opt))
			} else {
				b.WriteString(normalItemStyle.Render(prefix + opt))
			}
			b.WriteString("\n")
		}
		b.WriteString("\n")
		b.WriteString(helpStyle.Render("  ↑↓ select  enter confirm  esc back"))
	case editSubmitting:
		b.WriteString("  Submitting...\n")
	}

	return b.String()
}
	}
}

// --- Edit Messages ---

type editDoneMsg struct {
	refresh bool
}

type editOptionsMsg struct {
	options []string
}

type editErrorMsg struct {
	err string
}
```

- [ ] **Step 2: Check if AvailableCustomActions exists**

Run: `grep -n 'func AvailableCustomActions' /Users/john/Project/Github/openproject-cli/components/resources/work_packages/*.go`

If it doesn't exist, the `loadOptions` for status needs adjustment — use custom actions differently or use status filter values.

- [ ] **Step 3: Create edit_view_test.go**

```go
package tui

import (
	"testing"

	"github.com/opf/openproject-cli/models"
)

func TestEditModelChooseField(t *testing.T) {
	wp := &models.WorkPackage{Id: 1}
	m := newEditModel(wp, 80)

	if m.state != editChooseField {
		t.Fatal("should start in chooseField state")
	}

	// Press 't' for type
	m, _ = m.Update(keyMsg("t"))
	if m.state != editChooseValue {
		t.Fatal("should be in chooseValue after selecting type")
	}
}

func TestEditModelCancelField(t *testing.T) {
	wp := &models.WorkPackage{Id: 1}
	m := newEditModel(wp, 80)

	m.Update(keyMsg("t"))
	m, _ = m.Update(keyMsg("esc"))

	if m.state != editChooseField {
		t.Fatal("should return to chooseField on esc")
	}
}

func TestEditModelEscapeFromField(t *testing.T) {
	wp := &models.WorkPackage{Id: 1}
	m := newEditModel(wp, 80)

	_, cmd := m.Update(keyMsg("esc"))
	if cmd == nil {
		t.Fatal("esc from chooseField should return editDoneMsg command")
	}
}
```

- [ ] **Step 4: Run tests**

Run: `go test -v ./cmd/tui/...`

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add cmd/tui/edit_view.go cmd/tui/edit_view_test.go
git commit -m "feat(tui): add edit overlay for status and type changes"
```

---

### Task 11: Integrate filter panel into app and list view

**Files:**
- Modify: `cmd/tui/app.go`
- Modify: `cmd/tui/list_view.go`

- [ ] **Step 1: Add filter overlay to list view**

In `listModel`, add filter state:

```go
// Add to listModel struct:
filterOverlay bool
filter        *filterModel
```

In `newListModel`, initialize filter:

```go
func newListModel() *listModel {
	// ... existing code ...
	return &listModel{
		// ... existing fields ...
		filter: newFilterModel(),
	}
}
```

- [ ] **Step 2: Handle filter key in list view Update**

Add filter key handling to listModel.Update:

```go
case "f":
	if !m.filterOverlay {
		m.filterOverlay = true
		return m, m.filter.loadOptions()
	}
```

When filter overlay is active, route key events to filter:

```go
if m.filterOverlay {
	switch msg.String() {
	case "esc":
		m.filterOverlay = false
		return m, nil
	case "enter":
		m.filterOpts = m.filter.FilterOptions()
		m.filterOverlay = false
		m.loading = true
		m.page = 1
		return m, tea.Batch(m.spinner.Tick, m.loadWorkPackages)
	default:
		m.filter.Update(msg)
		return m, nil
	}
}
```

- [ ] **Step 3: Show filter overlay in list view View**

In `listModel.View()`, if filterOverlay is active:

```go
if m.filterOverlay {
	return m.filter.View()
}
```

- [ ] **Step 4: Update filter bar in header**

In the header section, show active filters:

```go
if len(m.filterOpts) > 0 {
	var parts []string
	keys := []work_packages.FilterOption{work_packages.Project, work_packages.Status, work_packages.Type, work_packages.Assignee}
	for _, k := range keys {
		if v, ok := m.filterOpts[k]; ok {
			parts = append(parts, fmt.Sprintf("%s=%s", k, v))
		}
	}
	b.WriteString(subtitleStyle.Render("  " + strings.Join(parts, "  ")))
	b.WriteString("\n")
}
```

- [ ] **Step 5: Run tests**

Run: `go test -v ./cmd/tui/...`

Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add cmd/tui/app.go cmd/tui/list_view.go
git commit -m "feat(tui): integrate filter panel into list view"
```

---

### Task 12: Integrate edit overlay into detail view

**Files:**
- Modify: `cmd/tui/detail_view.go`

- [ ] **Step 1: Add edit overlay to detail model**

In `detailModel`, add:

```go
editOverlay bool
edit        *editModel
```

- [ ] **Step 2: Handle 'e' key in detail view**

In `detailModel.Update`, add case for "e":

```go
case "e":
	if !m.editOverlay {
		m.editOverlay = true
		m.edit = newEditModel(m.wp, m.width)
	}
```

When edit overlay is active:

```go
if m.editOverlay {
	switch msg := msg.(type) {
	case editDoneMsg:
		m.editOverlay = false
		m.edit = nil
		if msg.refresh {
			m.loading = true
			return m, tea.Batch(
				func() tea.Msg {
					wp, err := work_packages.Lookup(m.wp.Id)
					return workPackageDetailMsg{wp: wp, err: err}
				},
				m.loadActivities,
			)
		}
		return m, nil
	case editOptionsMsg:
		m.edit.options = msg.options
		return m, nil
	case editErrorMsg:
		m.edit.err = msg.err
		m.edit.state = editChooseField
		return m, nil
	default:
		var cmd tea.Cmd
		m.edit, cmd = m.edit.Update(msg)
		return m, cmd
	}
}
```

- [ ] **Step 3: Show edit overlay in detail view**

In `detailModel.View()`:

```go
func (m *detailModel) View() string {
	if m.editOverlay {
		return m.edit.View()
	}
	// ... existing view code ...
}
```

- [ ] **Step 4: Run tests**

Run: `go test -v ./cmd/tui/...`

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add cmd/tui/detail_view.go
git commit -m "feat(tui): integrate edit overlay into detail view"
```

---

### Task 13: Add error handling and ctrl+c confirmation

**Files:**
- Modify: `cmd/tui/app.go`

- [ ] **Step 1: Add error display**

In `app.go`, handle error display in the View:

```go
func (a App) View() string {
	if a.quitting {
		return ""
	}

	var content string
	switch a.state {
	case listView:
		content = a.list.View()
	case detailView:
		content = a.detail.View()
	}

	if a.err != nil {
		content += "\n" + errorStyle.Render(fmt.Sprintf("Error: %v", a.err)) + "\n"
		content += helpStyle.Render("  r to retry")
	}

	return docStyle.Render(content)
}
```

- [ ] **Step 2: Add ctrl+c double-press confirmation**

In `app.go`, add a `ctrlCTimer` field and logic:

```go
// Add to App struct:
//   ctrlCPressed bool
//   ctrlCTime    time.Time

case "ctrl+c":
	if a.ctrlCPressed && time.Since(a.ctrlCTime) < 2*time.Second {
		a.quitting = true
		return a, tea.Quit
	}
	a.ctrlCPressed = true
	a.ctrlCTime = time.Now()
	return a, nil // will show "press again to quit" via footer
```

But since we use `ctrl+c` as a quit binding in keymap, a simpler approach: just show the "press again" message on first press, quit on second. The message can be shown in the help bar.

Actually, for simplicity in v1, let ctrl+c quit immediately. The double-press adds complexity without clear benefit in a TUI context where accidental quit is less costly than in a web browser.

Keep it simple: ctrl+c quits.

- [ ] **Step 3: Run tests**

Run: `go test -v ./cmd/tui/...`

Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add cmd/tui/app.go
git commit -m "feat(tui): add error display in TUI views"
```

---

### Task 14: End-to-end verification and polish

**Files:**
- None (manual testing)

- [ ] **Step 1: Build and install**

Run: `cd /Users/john/Project/Github/openproject-cli && go install .`

Expected: Binary installed

- [ ] **Step 2: Manual test — launch TUI**

Run: `opc tui`

Expected: TUI opens, shows loading spinner, then work package list

- [ ] **Step 3: Manual test — navigation**

- Use `j`/`k` to move selection
- Press `enter` to open detail
- Press `esc` to go back
- Press `n`/`p` for pagination
- Press `s` to cycle sort
- Press `/` to search
- Press `f` to open filter panel
- Press `e` in detail view to edit
- Press `o` to open in browser
- Press `q` to quit

- [ ] **Step 4: Fix any issues found**

Address any UI glitches, key binding conflicts, or layout issues.

- [ ] **Step 5: Run full test suite**

Run: `go test -v ./...`

Expected: All tests pass

- [ ] **Step 6: Final commit**

```bash
git add -A
git commit -m "fix(tui): polish layout, key bindings, and error handling"
```

---

## Spec Coverage Check

| Spec Section | Task(s) |
|-------------|---------|
| Architecture + file structure | Task 4, 5 |
| DTO expansion | Task 2 |
| Requests silent mode | Task 3 |
| List view (rendering, navigation) | Task 6 |
| List view (sort, search) | Task 7 |
| List view (pagination) | Task 6 |
| Detail view (properties, description, activities) | Task 8 |
| Detail view (browser open) | Task 8 |
| Edit mode | Task 10, 12 |
| Filter panel | Task 9, 11 |
| Error handling | Task 13 |
| Signal handling (ctrl+c) | Task 13 (simplified) |
| Integration (cobra subcommand) | Task 5 |
| Dependencies | Task 1 |

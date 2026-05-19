# TUI Tree View Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a tree view to the TUI list showing work package parent-child hierarchy, toggled with `t`.

**Architecture:** Parse parent link from OpenProject API v3 into `ParentId` on the model. Build a client-side tree from the current page's flat list. Render with Unicode line characters (`├─`, `└─`, `│`). Toggle between flat table and tree mode.

**Tech Stack:** Go, Bubble Tea (tea), lipgloss, standard `testing` package

---

## File Structure

| File | Responsibility |
|------|----------------|
| `models/work_package.go` | Add `ParentId uint64` field |
| `dtos/work_package.go` | Add `Parent *LinkDto`, parse parent ID in `Convert()` |
| `cmd/tui/tree.go` | **New** — `treeNode`, `buildTree`, `flatten`, line rendering helpers |
| `cmd/tui/tree_test.go` | **New** — tests for buildTree, flatten, rendering |
| `cmd/tui/keymap.go` | Add `Tree`, `Expand`, `Collapse` key bindings |
| `cmd/tui/list_view.go` | Add `treeMode`, tree state, toggle logic, tree-aware view |
| `cmd/tui/list_view_test.go` | Add tests for tree toggle and tree navigation |

---

### Task 1: Add ParentId to Model and DTO

**Files:**
- Modify: `models/work_package.go`
- Modify: `dtos/work_package.go`
- Test: `dtos/work_package_test.go` (verify if exists, add test)

- [ ] **Step 1: Add ParentId to WorkPackage model**

In `models/work_package.go`, add `ParentId uint64` to the `WorkPackage` struct:

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
	StartDate   string
	DueDate     string
	ParentId    uint64
}
```

- [ ] **Step 2: Add Parent link to WorkPackageLinksDto**

In `dtos/work_package.go`, add `Parent *LinkDto` to `WorkPackageLinksDto`:

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
	Parent            *LinkDto   `json:"parent,omitempty"`
	CustomActions     []*LinkDto `json:"customActions,omitempty"`
	PrepareAttachment *LinkDto   `json:"prepareAttachment,omitempty"`
}
```

- [ ] **Step 3: Parse ParentId in Convert()**

In `dtos/work_package.go`, add parent parsing to `Convert()`. The OpenProject API parent href looks like `/api/v3/work_packages/42`. Add an import for `strconv` and `path`:

```go
import (
	"strconv"
	"path"
	"github.com/opf/openproject-cli/models"
)
```

Add parsing logic at the end of `Convert()`, before `return wp`:

```go
if dto.Links != nil && dto.Links.Parent != nil && dto.Links.Parent.Href != "" {
	if id, err := strconv.ParseUint(path.Base(dto.Links.Parent.Href), 10, 64); err == nil {
		wp.ParentId = id
	}
}
```

- [ ] **Step 4: Run tests to verify**

Run: `go test -v ./models/ ./dtos/`
Expected: PASS (existing tests should still pass since ParentId defaults to 0)

- [ ] **Step 5: Commit**

```bash
git add models/work_package.go dtos/work_package.go
git commit -m "feat: add ParentId to work package model and DTO"
```

---

### Task 2: Create tree data structure

**Files:**
- Create: `cmd/tui/tree.go`

- [ ] **Step 1: Create tree.go with treeNode and buildTree**

```go
package tui

import (
	"sort"
	"strings"

	"github.com/opf/openproject-cli/models"
)

// treeNode wraps a WorkPackage with tree hierarchy info.
type treeNode struct {
	item     *models.WorkPackage
	children []*treeNode
	expanded bool
	depth    int
}

// hasChildren reports whether this node has child nodes.
func (n *treeNode) hasChildren() bool {
	return len(n.children) > 0
}

// sortFunc is a comparator for sorting tree nodes within sibling groups.
type sortFunc func(a, b *treeNode) bool

// byID is the default sort comparator (by work package ID).
var byID sortFunc = func(a, b *treeNode) bool { return a.item.Id < b.item.Id }

// buildTree constructs a tree from a flat list of work packages.
// Items whose parent is also in the list are nested under it;
// items whose parent is absent become top-level roots.
// Siblings are sorted using the provided comparator (nil = byID).
func buildTree(items []*models.WorkPackage, cmp sortFunc) []*treeNode {
	if cmp == nil {
		cmp = byID
	}
	// Create nodes
	nodeMap := make(map[uint64]*treeNode, len(items))
	for _, wp := range items {
		nodeMap[wp.Id] = &treeNode{
			item:     wp,
			expanded: true,
		}
	}

	// Build parent-child relationships
	var roots []*treeNode
	for _, node := range nodeMap {
		if node.item.ParentId != 0 {
			if parent, ok := nodeMap[node.item.ParentId]; ok {
				parent.children = append(parent.children, node)
				continue
			}
		}
		roots = append(roots, node)
	}

	// Sort roots and children using comparator
	sortNodes := func(nodes []*treeNode) {
		sort.SliceStable(nodes, func(i, j int) bool {
			return cmp(nodes[i], nodes[j])
		})
	}
	sortNodes(roots)
	for _, node := range nodeMap {
		sortNodes(node.children)
	}

	// Set depths via BFS
	queue := make([]*treeNode, len(roots))
	copy(queue, roots)
	for _, n := range roots {
		n.depth = 0
	}
	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]
		for _, child := range node.children {
			child.depth = node.depth + 1
			queue = append(queue, child)
		}
	}

	return roots
}

// flatten returns a DFS traversal of visible nodes (respecting expanded state).
func flatten(roots []*treeNode) []*treeNode {
	var result []*treeNode
	var walk func(nodes []*treeNode)
	walk = func(nodes []*treeNode) {
		for _, n := range nodes {
			result = append(result, n)
			if n.expanded && len(n.children) > 0 {
				walk(n.children)
			}
		}
	}
	walk(roots)
	return result
}

// treeNodeAt returns the node at the given index in the flattened list, or nil.
func treeNodeAt(flat []*treeNode, index int) *treeNode {
	if index < 0 || index >= len(flat) {
		return nil
	}
	return flat[index]
}

// treeLinePrefix builds the Unicode prefix for a tree node line.
// Example outputs:
//   depth=0: ""         (root node, no prefix)
//   depth=1, isLast=false, ancestorsLast=[false]: "├─ "
//   depth=1, isLast=true,  ancestorsLast=[false]: "└─ "
//   depth=2, isLast=true,  ancestorsLast=[false,true]: "│  └─ "
//   depth=2, isLast=false, ancestorsLast=[false,false]: "│  ├─ "
func treeLinePrefix(depth int, isLast bool, ancestorsLast []bool) string {
	if depth == 0 {
		return ""
	}

	var b strings.Builder
	for i := 0; i < depth-1; i++ {
		if i < len(ancestorsLast) && ancestorsLast[i] {
			b.WriteString("   ")
		} else {
			b.WriteString("│  ")
		}
	}
	if isLast {
		b.WriteString("└─ ")
	} else {
		b.WriteString("├─ ")
	}
	return b.String()
}

// computeAncestorsLast computes, for each ancestor of a node at the given
// flat index, whether that ancestor is the last child of its parent.
func computeAncestorsLast(flat []*treeNode, index int) []bool {
	if index <= 0 {
		return nil
	}

	node := flat[index]
	var ancestorsLast []bool

	// Walk backwards through the flat list to find ancestors
	for i := index - 1; i >= 0; i-- {
		candidate := flat[i]
		if candidate.depth < node.depth {
			// candidate is an ancestor
			// Check if candidate is the last child of its parent
			isLast := true
			if candidate.depth > 0 {
				// Find candidate's parent and check if candidate is last
				for j := i + 1; j < len(flat); j++ {
					if flat[j].depth == candidate.depth {
						// Another sibling at same depth comes after
						isLast = false
						break
					}
					if flat[j].depth < candidate.depth {
						break
					}
				}
			} else {
				// Root level: check if there's another root after this one
				for j := i + 1; j < len(flat); j++ {
					if flat[j].depth == 0 {
						isLast = false
						break
					}
				}
			}
			ancestorsLast = append([]bool{isLast}, ancestorsLast...)
			node = candidate
		}
	}

	return ancestorsLast
}
```

- [ ] **Step 2: Commit**

```bash
git add cmd/tui/tree.go
git commit -m "feat(tui): add tree data structure with buildTree and flatten"
```

---

### Task 3: Write tree tests

**Files:**
- Create: `cmd/tui/tree_test.go`

- [ ] **Step 1: Write tests for buildTree, flatten, and treeLinePrefix**

```go
package tui

import (
	"strings"
	"testing"

	"github.com/opf/openproject-cli/models"
)

func TestBuildTreeFlatList(t *testing.T) {
	items := []*models.WorkPackage{
		{Id: 1, Subject: "A"},
		{Id: 2, Subject: "B"},
		{Id: 3, Subject: "C"},
	}
	roots := buildTree(items, nil)
	if len(roots) != 3 {
		t.Fatalf("expected 3 roots, got %d", len(roots))
	}
	for _, r := range roots {
		if r.depth != 0 {
			t.Fatalf("expected depth 0, got %d", r.depth)
		}
	}
}

func TestBuildTreeParentChild(t *testing.T) {
	items := []*models.WorkPackage{
		{Id: 1, Subject: "Parent"},
		{Id: 2, Subject: "Child A", ParentId: 1},
		{Id: 3, Subject: "Child B", ParentId: 1},
		{Id: 4, Subject: "Grandchild", ParentId: 2},
		{Id: 5, Subject: "Orphan"},
	}
	roots := buildTree(items, nil)

	if len(roots) != 2 {
		t.Fatalf("expected 2 roots, got %d", len(roots))
	}
	if roots[0].item.Id != 1 {
		t.Fatalf("expected first root id=1, got %d", roots[0].item.Id)
	}
	if len(roots[0].children) != 2 {
		t.Fatalf("expected root 1 to have 2 children, got %d", len(roots[0].children))
	}
	if len(roots[0].children[0].children) != 1 {
		t.Fatalf("expected child 2 to have 1 grandchild, got %d", len(roots[0].children[0].children))
	}
	if roots[0].children[0].children[0].depth != 2 {
		t.Fatalf("expected grandchild depth=2, got %d", roots[0].children[0].children[0].depth)
	}
}

func TestBuildTreeMissingParent(t *testing.T) {
	items := []*models.WorkPackage{
		{Id: 3, Subject: "Child", ParentId: 99},
		{Id: 5, Subject: "Root"},
	}
	roots := buildTree(items, nil)
	if len(roots) != 2 {
		t.Fatalf("expected 2 roots (orphaned child becomes root), got %d", len(roots))
	}
}

func TestFlattenExpanded(t *testing.T) {
	items := []*models.WorkPackage{
		{Id: 1, Subject: "Parent"},
		{Id: 2, Subject: "Child", ParentId: 1},
	}
	roots := buildTree(items, nil)
	flat := flatten(roots)
	if len(flat) != 2 {
		t.Fatalf("expected 2 visible nodes, got %d", len(flat))
	}
}

func TestFlattenCollapsed(t *testing.T) {
	items := []*models.WorkPackage{
		{Id: 1, Subject: "Parent"},
		{Id: 2, Subject: "Child", ParentId: 1},
	}
	roots := buildTree(items, nil)
	roots[0].expanded = false
	flat := flatten(roots)
	if len(flat) != 1 {
		t.Fatalf("expected 1 visible node (collapsed), got %d", len(flat))
	}
}

func TestTreeLinePrefix(t *testing.T) {
	tests := []struct {
		depth        int
		isLast       bool
		ancestorsLast []bool
		want         string
	}{
		{0, false, nil, ""},
		{0, true, nil, ""},
		{1, false, []bool{false}, "├─ "},
		{1, true, []bool{false}, "└─ "},
		{2, true, []bool{false, false}, "│  └─ "},
		{2, false, []bool{false, true}, "   ├─ "},
		{3, true, []bool{false, false, true}, "│     └─ "},
	}
	for _, tt := range tests {
		got := treeLinePrefix(tt.depth, tt.isLast, tt.ancestorsLast)
		if got != tt.want {
			t.Errorf("treeLinePrefix(%d, %v, %v) = %q, want %q",
				tt.depth, tt.isLast, tt.ancestorsLast, got, tt.want)
		}
	}
}

func TestTreeSortedById(t *testing.T) {
	items := []*models.WorkPackage{
		{Id: 3, Subject: "C", ParentId: 1},
		{Id: 1, Subject: "A"},
		{Id: 2, Subject: "B", ParentId: 1},
	}
	roots := buildTree(items, nil)
	if len(roots) != 1 {
		t.Fatalf("expected 1 root, got %d", len(roots))
	}
	if roots[0].children[0].item.Id != 2 {
		t.Fatalf("expected first child id=2, got %d", roots[0].children[0].item.Id)
	}
	if roots[0].children[1].item.Id != 3 {
		t.Fatalf("expected second child id=3, got %d", roots[0].children[1].item.Id)
	}
}

func TestBuildTreeEmpty(t *testing.T) {
	roots := buildTree(nil, nil)
	if len(roots) != 0 {
		t.Fatalf("expected 0 roots for nil input, got %d", len(roots))
	}
}

func TestBuildTreeSorting(t *testing.T) {
	items := []*models.WorkPackage{
		{Id: 5, Subject: "E"},
		{Id: 2, Subject: "B"},
		{Id: 8, Subject: "H"},
	}
	roots := buildTree(items, nil)
	if roots[0].item.Id != 2 {
		t.Fatalf("expected first root id=2, got %d", roots[0].item.Id)
	}
	if roots[1].item.Id != 5 {
		t.Fatalf("expected second root id=5, got %d", roots[1].item.Id)
	}
	if roots[2].item.Id != 8 {
		t.Fatalf("expected third root id=8, got %d", roots[2].item.Id)
	}
}

// Helper: render the tree as a multi-line string for visual testing
func renderTree(flat []*treeNode) string {
	var lines []string
	for i, node := range flat {
		ancLast := computeAncestorsLast(flat, i)
		isLast := true
		// Check if this node is the last child
		for j := i + 1; j < len(flat); j++ {
			if flat[j].depth == node.depth {
				isLast = false
				break
			}
			if flat[j].depth < node.depth {
				break
			}
		}
		prefix := treeLinePrefix(node.depth, isLast, ancLast)
		lines = append(lines, prefix+node.item.Subject)
	}
	return strings.Join(lines, "\n")
}

func TestRenderTreeBasic(t *testing.T) {
	items := []*models.WorkPackage{
		{Id: 1, Subject: "Feature A"},
		{Id: 2, Subject: "Subtask 1", ParentId: 1},
		{Id: 3, Subject: "Detail", ParentId: 2},
		{Id: 4, Subject: "Subtask 2", ParentId: 1},
		{Id: 5, Subject: "Feature B"},
	}
	roots := buildTree(items, nil)
	flat := flatten(roots)
	rendered := renderTree(flat)

	expected := "Feature A\n├─ Subtask 1\n│  └─ Detail\n└─ Subtask 2\nFeature B"
	if rendered != expected {
		t.Errorf("unexpected tree rendering:\n%s\n--- expected ---\n%s", rendered, expected)
	}
}
```

- [ ] **Step 2: Run tests to verify**

Run: `go test -v ./cmd/tui/ -run TestBuildTree -count=1 && go test -v ./cmd/tui/ -run TestFlatten -count=1 && go test -v ./cmd/tui/ -run TestTreeLinePrefix -count=1 && go test -v ./cmd/tui/ -run TestRenderTree -count=1`
Expected: all PASS

- [ ] **Step 3: Commit**

```bash
git add cmd/tui/tree_test.go
git commit -m "test(tui): add tree data structure tests"
```

---

### Task 4: Add tree key bindings

**Files:**
- Modify: `cmd/tui/keymap.go`

- [ ] **Step 1: Add Tree, Expand, Collapse key bindings to KeyMap**

Add three new fields to `KeyMap` struct:

```go
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
	Tree     key.Binding
	Expand   key.Binding
	Collapse key.Binding
}
```

Add to `DefaultKeyMap`:

```go
Tree: key.NewBinding(
	key.WithKeys("t"),
	key.WithHelp("t", "tree toggle"),
),
Expand: key.NewBinding(
	key.WithKeys("right", ">"),
	key.WithHelp(">/→", "expand"),
),
Collapse: key.NewBinding(
	key.WithKeys("left", "<"),
	key.WithHelp("</←", "collapse"),
),
```

- [ ] **Step 2: Commit**

```bash
git add cmd/tui/keymap.go
git commit -m "feat(tui): add tree toggle, expand, collapse key bindings"
```

---

### Task 5: Integrate tree mode into list view

**Files:**
- Modify: `cmd/tui/list_view.go`

- [ ] **Step 1: Add tree fields to listModel**

Add to the `listModel` struct:

```go
type listModel struct {
	items         []*models.WorkPackage
	selected      int
	collection    *models.WorkPackageCollection
	page          int
	pageSize      int64
	total         int64
	width         int
	height        int
	loading       bool
	spinner       spinner.Model
	searchActive  bool
	searchInput   string
	filterOpts    map[work_packages.FilterOption]string
	sortField     sortField
	filter        *filterModel
	filterOverlay bool
	showHelp      bool
	treeMode      bool
	treeRoots     []*treeNode
	flatNodes     []*treeNode
}
```

- [ ] **Step 2: Add buildTreeFromItems helper to rebuild tree after data changes**

Add method:

```go
func (m *listModel) buildTreeFromItems() {
	cmp := m.treeSortFunc()
	m.treeRoots = buildTree(m.items, cmp)
	m.flatNodes = flatten(m.treeRoots)
	if m.selected >= len(m.flatNodes) {
		m.selected = len(m.flatNodes) - 1
	}
	if m.selected < 0 && len(m.flatNodes) > 0 {
		m.selected = 0
	}
}

// treeSortFunc returns a sort comparator based on the current sortField.
func (m *listModel) treeSortFunc() sortFunc {
	switch m.sortField {
	case sortByStatus:
		return func(a, b *treeNode) bool { return a.item.Status < b.item.Status }
	case sortByType:
		return func(a, b *treeNode) bool { return a.item.Type < b.item.Type }
	case sortByAssignee:
		return func(a, b *treeNode) bool { return a.item.Assignee < b.item.Assignee }
	default:
		return byID
	}
}
```

- [ ] **Step 3: Call buildTreeFromItems in SetWorkPackages**

After `m.sortItems()`, add:

```go
if m.treeMode {
	m.buildTreeFromItems()
}
```

- [ ] **Step 4: Handle tree key bindings in Update**

In the `case tea.KeyMsg:` handler, after the existing `switch msg.String()` block, add a new block for tree mode keys. Insert these cases BEFORE the existing `switch msg.String()`:

```go
// Tree mode keys (active when not in search/filter/help)
if !m.filterOverlay && !m.searchActive && !m.showHelp {
	if msg.String() == "t" {
		m.treeMode = !m.treeMode
		if m.treeMode {
			m.buildTreeFromItems()
		} else {
			// Sync selected back to items index
			if m.selected >= 0 && m.selected < len(m.flatNodes) {
				wp := m.flatNodes[m.selected].item
				for i, item := range m.items {
					if item.Id == wp.Id {
						m.selected = i
						break
					}
				}
			}
		}
		return m, nil
	}

	if m.treeMode {
		switch msg.String() {
		case "right", ">":
			if m.selected >= 0 && m.selected < len(m.flatNodes) {
				node := m.flatNodes[m.selected]
				if node.hasChildren() && !node.expanded {
					node.expanded = true
					m.flatNodes = flatten(m.treeRoots)
				}
			}
			return m, nil
		case "left", "<":
			if m.selected >= 0 && m.selected < len(m.flatNodes) {
				node := m.flatNodes[m.selected]
				if node.hasChildren() && node.expanded {
					node.expanded = false
					m.flatNodes = flatten(m.treeRoots)
					if m.selected >= len(m.flatNodes) {
						m.selected = len(m.flatNodes) - 1
					}
				}
			}
			return m, nil
		case "enter":
			if m.selected >= 0 && m.selected < len(m.flatNodes) {
				node := m.flatNodes[m.selected]
				if node.hasChildren() {
					node.expanded = !node.expanded
					m.flatNodes = flatten(m.treeRoots)
					if m.selected >= len(m.flatNodes) {
						m.selected = len(m.flatNodes) - 1
					}
					return m, nil
				}
				return m, OpenDetailCmd(node.item)
			}
		case "up", "k":
			if m.selected > 0 {
				m.selected--
			}
			return m, nil
		case "down", "j":
			if m.selected < len(m.flatNodes)-1 {
				m.selected++
			}
			return m, nil
		case "s":
			// Sort in tree mode: cycle sort, sort items, then rebuild tree.
			// Sorting applies within sibling groups (parent-child preserved).
			m.cycleSort()
			m.buildTreeFromItems()
			return m, nil
		case "r":
			m.loading = true
			return m, tea.Batch(m.spinner.Tick, m.loadWorkPackages)
		}
	}
}
```

- [ ] **Step 5: Run existing tests to verify no regressions**

Run: `go test -v ./cmd/tui/`
Expected: PASS (existing tests should still work since treeMode defaults to false)

- [ ] **Step 6: Commit**

```bash
git add cmd/tui/list_view.go
git commit -m "feat(tui): integrate tree mode toggle and navigation into list view"
```

---

### Task 6: Update list view rendering for tree mode

**Files:**
- Modify: `cmd/tui/list_view.go` (View method)

- [ ] **Step 1: Add tree rendering branch to View()**

In the `View()` method, after the loading/empty checks and before the existing column/header rendering, add a tree mode branch:

```go
// Tree mode rendering
if m.treeMode {
	return m.treeView()
}
```

Add the `treeView()` method to `listModel`:

```go
func (m *listModel) treeView() string {
	var b strings.Builder

	// Header
	b.WriteString(titleStyle.Render("Work Packages"))
	if m.total > 0 {
		b.WriteString(subtitleStyle.Render(fmt.Sprintf(" (%d total)", m.total)))
	}
	b.WriteString(subtitleStyle.Render("  [tree]"))
	b.WriteString("\n")

	// Active filters
	if len(m.filterOpts) > 0 {
		var parts []string
		keys := []work_packages.FilterOption{work_packages.Project, work_packages.Status, work_packages.Type, work_packages.Assignee}
		for _, k := range keys {
			if v, ok := m.filterOpts[k]; ok {
				parts = append(parts, fmt.Sprintf("%s=%s", k, v))
			}
		}
		b.WriteString(subtitleStyle.Render("  Filter: " + strings.Join(parts, "  ")))
		b.WriteString("\n")
	}

	b.WriteString("\n")

	if len(m.flatNodes) == 0 {
		b.WriteString("\n  No work packages found.\n\n")
		return b.String()
	}

	// Column widths (account for indent)
	maxDepth := 0
	for _, n := range m.flatNodes {
		if n.depth > maxDepth {
			maxDepth = n.depth
		}
	}
	indentWidth := maxDepth*4 + 2 // 4 chars per depth level + margin

	idWidth := len(strconv.FormatInt(m.total, 10)) + 2
	if idWidth < 6 {
		idWidth = 6
	}
	typeWidth := 12
	statusWidth := 12
	assigneeWidth := 14
	titleWidth := m.width - idWidth - typeWidth - statusWidth - assigneeWidth - indentWidth - 12
	if titleWidth < 20 {
		titleWidth = 20
	}

	// Table header
	headerLine := strings.Repeat(" ", indentWidth) +
		padRight("ID", idWidth) + " " +
		padRight("Type", typeWidth) + " " +
		padRight("Title", titleWidth) + " " +
		padRight("Status", statusWidth) + " " +
		padRight("Assignee", assigneeWidth)
	b.WriteString(headerStyle.Render(headerLine))
	b.WriteString("\n\n")

	// Render each visible node
	for i, node := range m.flatNodes {
		// Determine if this node is the last child of its parent
		isLast := true
		for j := i + 1; j < len(m.flatNodes); j++ {
			if m.flatNodes[j].depth == node.depth {
				isLast = false
				break
			}
			if m.flatNodes[j].depth < node.depth {
				break
			}
		}

		ancLast := computeAncestorsLast(m.flatNodes, i)
		prefix := treeLinePrefix(node.depth, isLast, ancLast)

		// Expand/collapse marker
		marker := "  "
		if node.hasChildren() {
			if node.expanded {
				marker = "▼ "
			} else {
				marker = "▶ "
			}
		}

		wp := node.item
		idStr := fmt.Sprintf("#%d", wp.Id)
		line := marker + padRight(idStr, idWidth-2) + " " +
			padRight(truncate(wp.Type, typeWidth), typeWidth) + " " +
			padRight(truncate(wp.Subject, titleWidth), titleWidth) + " " +
			padRight(truncate(wp.Status, statusWidth), statusWidth) + " " +
			padRight(truncate(assigneeOrDash(wp.Assignee), assigneeWidth), assigneeWidth)

		fullLine := prefix + line

		if i == m.selected {
			b.WriteString(selectedItemStyle.Render(fullLine))
		} else {
			b.WriteString(normalItemStyle.Render(fullLine))
		}
		b.WriteString("\n")
	}

	// Footer
	b.WriteString("\n")
	pageInfo := fmt.Sprintf("%d-%d / %d",
		int64(m.page-1)*m.pageSize+1,
		min(int64(m.page)*m.pageSize, m.total),
		m.total)
	b.WriteString(helpStyle.Render(pageInfo))

	// Help bar
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("  ↑↓ move  enter open/expand  >/< expand/collapse  t list  / search  f filter  n/p page  ? help  q quit"))

	return b.String()
}
```

- [ ] **Step 2: Update mouse handler for tree mode**

In `mouseEventToRow`, account for tree mode offset. The existing method works for flat list. For tree mode, the offset is the same (title + filter bar + blank + header), so no change needed as long as `m.items` vs `m.flatNodes` count is handled. Update the method:

```go
func (m *listModel) mouseEventToRow(y int) int {
	offset := 4
	if len(m.filterOpts) > 0 {
		offset++
	}
	row := y - offset

	itemCount := len(m.items)
	if m.treeMode {
		itemCount = len(m.flatNodes)
	}

	if row < 0 || row >= itemCount {
		return -1
	}
	return row
}
```

And in the mouse handler, when treeMode is on, get the selected wp from `m.flatNodes[row].item` instead of `m.items[row]`.

- [ ] **Step 3: Run tests**

Run: `go test -v ./cmd/tui/`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add cmd/tui/list_view.go
git commit -m "feat(tui): add tree mode rendering with Unicode line characters"
```

---

### Task 7: Update help overlay for tree mode

**Files:**
- Modify: `cmd/tui/list_view.go` (View method, help overlay section)

- [ ] **Step 1: Update help overlay to show tree bindings when tree mode is active**

In the `View()` method, update the `if m.showHelp` block to show tree-specific help when in tree mode:

```go
if m.showHelp {
	bindings := [][2]string{
		{"↑ / k", "move up"},
		{"↓ / j", "move down"},
		{"enter", "open detail view"},
		{"/", "search (client-side)"},
		{"f", "open filter panel"},
		{"s", "cycle sort (ID → Status → Type → Assignee)"},
		{"n", "next page"},
		{"p", "previous page"},
		{"r", "refresh"},
		{"o", "open in browser"},
		{"?", "toggle this help"},
		{"q", "quit"},
	}
	if m.treeMode {
		bindings = append([][2]string{
			{"t", "toggle tree/list mode"},
			{"enter", "expand/collapse or open detail"},
			{"> / →", "expand node"},
			{"< / ←", "collapse node"},
		}, bindings...)
	} else {
		bindings = append([][2]string{
			{"t", "toggle tree/list mode"},
		}, bindings...)
	}
	return helpOverlay("List — Key Bindings", bindings, m.width)
}
```

- [ ] **Step 2: Run tests**

Run: `go test -v ./cmd/tui/`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add cmd/tui/list_view.go
git commit -m "feat(tui): update help overlay with tree mode key bindings"
```

---

### Task 8: Add integration tests for tree mode

**Files:**
- Modify: `cmd/tui/list_view_test.go`

- [ ] **Step 1: Add test for tree toggle**

```go
func TestTreeToggle(t *testing.T) {
	m := newListModel()
	m.SetWorkPackages(&models.WorkPackageCollection{
		Total: 3,
		Items: []*models.WorkPackage{
			{Id: 1, Subject: "Parent"},
			{Id: 2, Subject: "Child", ParentId: 1},
			{Id: 3, Subject: "Root"},
		},
	})

	if m.treeMode {
		t.Fatal("treeMode should be false initially")
	}

	// Toggle to tree mode
	m, _ = m.Update(keyMsg("t"))
	if !m.treeMode {
		t.Fatal("treeMode should be true after pressing t")
	}
	if len(m.flatNodes) != 2 {
		t.Fatalf("expected 2 visible nodes (Parent+Child, Root), got %d", len(m.flatNodes))
	}

	// Toggle back to list mode
	m, _ = m.Update(keyMsg("t"))
	if m.treeMode {
		t.Fatal("treeMode should be false after pressing t again")
	}
}
```

- [ ] **Step 2: Add test for expand/collapse**

```go
func TestTreeExpandCollapse(t *testing.T) {
	m := newListModel()
	m.SetWorkPackages(&models.WorkPackageCollection{
		Total: 2,
		Items: []*models.WorkPackage{
			{Id: 1, Subject: "Parent"},
			{Id: 2, Subject: "Child", ParentId: 1},
		},
	})

	// Enter tree mode
	m, _ = m.Update(keyMsg("t"))

	// Default: expanded, should see 2 nodes
	if len(m.flatNodes) != 2 {
		t.Fatalf("expected 2 visible nodes, got %d", len(m.flatNodes))
	}

	// Collapse the parent
	m, _ = m.Update(keyMsg("<"))
	if len(m.flatNodes) != 1 {
		t.Fatalf("expected 1 visible node after collapse, got %d", len(m.flatNodes))
	}

	// Expand the parent
	m, _ = m.Update(keyMsg(">"))
	if len(m.flatNodes) != 2 {
		t.Fatalf("expected 2 visible nodes after expand, got %d", len(m.flatNodes))
	}
}
```

- [ ] **Step 3: Add test for tree mode enter behavior**

```go
func TestTreeEnterOnLeafOpensDetail(t *testing.T) {
	m := newListModel()
	m.SetWorkPackages(&models.WorkPackageCollection{
		Total: 2,
		Items: []*models.WorkPackage{
			{Id: 1, Subject: "Parent"},
			{Id: 2, Subject: "Child", ParentId: 1},
		},
	})

	m, _ = m.Update(keyMsg("t"))
	// Move to child
	m, _ = m.Update(keyMsg("down"))
	// Enter on leaf should open detail
	_, cmd := m.Update(keyMsg("enter"))
	if cmd == nil {
		t.Fatal("enter on leaf node should return a command")
	}
	msg := cmd()
	detailMsg, ok := msg.(openDetailMsg)
	if !ok {
		t.Fatalf("expected openDetailMsg, got %T", msg)
	}
	if detailMsg.wp.Id != 2 {
		t.Fatalf("expected wp id 2, got %d", detailMsg.wp.Id)
	}
}

func TestTreeEnterOnParentTogglesExpand(t *testing.T) {
	m := newListModel()
	m.SetWorkPackages(&models.WorkPackageCollection{
		Total: 2,
		Items: []*models.WorkPackage{
			{Id: 1, Subject: "Parent"},
			{Id: 2, Subject: "Child", ParentId: 1},
		},
	})

	m, _ = m.Update(keyMsg("t"))
	// Enter on parent should toggle collapse
	m, _ = m.Update(keyMsg("enter"))
	if len(m.flatNodes) != 1 {
		t.Fatalf("expected 1 node after collapsing, got %d", len(m.flatNodes))
	}
}
```

- [ ] **Step 4: Run all tests**

Run: `go test -v ./cmd/tui/`
Expected: all PASS

- [ ] **Step 5: Commit**

```bash
git add cmd/tui/list_view_test.go
git commit -m "test(tui): add tree mode toggle, expand/collapse, and enter tests"
```

---

### Task 9: Build and run full test suite

- [ ] **Step 1: Build the project**

Run: `go build -v ./...`
Expected: no errors

- [ ] **Step 2: Run all tests**

Run: `go test -v ./...`
Expected: all PASS

- [ ] **Step 3: Smoke test in terminal**

Run: `go run . tui`
Manually verify:
1. List shows flat table (default mode)
2. Press `t` — switches to tree mode with `[tree]` indicator
3. If items have parent-child relationships, see Unicode tree lines
4. Press `>` / `<` to expand/collapse
5. Press `enter` on parent toggles expand, on leaf opens detail
6. Press `t` again — returns to flat list, selection preserved
7. Press `?` — help shows tree bindings

- [ ] **Step 4: Final commit (if any fixes needed)**

```bash
git add -A
git commit -m "fix(tui): polish tree view after smoke testing"
```

---

## Self-Review

**Spec coverage:**
- ParentId in model/DTO — Task 1
- Tree data structure (buildTree, flatten, sortFunc) — Task 2
- Tree rendering (Unicode lines) — Task 6
- Toggle with `t` — Task 5
- Expand/collapse with `>/<` — Task 5
- Enter behavior (context-dependent) — Task 5
- Sort within sibling groups — Task 5 (treeSortFunc + cycleSort rebuilds tree)
- Help overlay update — Task 7
- Tests — Tasks 3, 8

**Placeholder scan:** None found. All steps contain exact code, commands, and expected output.

**Type consistency:** `treeNode`, `sortFunc`, `byID` defined in Task 2, used consistently in Tasks 3-6. `buildTree` signature has `sortFunc` parameter, all callers pass `nil` or a comparator. `treeMode`, `treeRoots`, `flatNodes` fields referenced correctly throughout.

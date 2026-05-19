# TUI Tree View Design

## Summary

Add a tree view to the TUI list, displaying work packages in a hierarchical parent-child structure. Users can toggle between flat list and tree mode with `t`. Tree nodes use Unicode line characters (`├─`, `└─`, `│`) for visual hierarchy.

## Approach

Client-side tree building from the current page's data. Items whose parent is also on the current page are nested under it; items whose parent is absent appear as top-level nodes. No extra API calls beyond ensuring the `parent` link is included.

## Data Layer

### Model — `models/work_package.go`

Add field:

```go
ParentId uint64
```

### DTO — `dtos/work_package.go`

- Add `Parent *LinkDto` to `WorkPackageLinksDto`
- In `Convert()`, parse `Parent.Href` to extract the numeric ID and set `ParentId`

### API Request

OpenProject API v3 returns `_links.parent` by default. Verify with existing `select` parameter in `components/resources/work_packages/all.go`. If parent is not included, add it to the select or expand params.

## Tree Data Structure

New file: `cmd/tui/tree.go`

```go
type treeNode struct {
    item     *models.WorkPackage
    children []*treeNode
    expanded bool
    depth    int
}
```

### `buildTree(items []*models.WorkPackage) []*treeNode`

1. Create `id → treeNode` map from all items
2. For each item: if `ParentId != 0` and parent exists in map, append to parent's `children`; otherwise add to roots
3. Sort roots and each node's children by `Id`
4. Calculate `depth` via BFS from roots

### `flatten(roots []*treeNode) []*treeNode`

DFS traversal. For each node: emit it, then if `expanded`, recurse into children. Returns ordered list of visible nodes with correct `depth`.

## List View Changes — `cmd/tui/list_view.go`

### New Fields

```go
treeMode  bool
treeRoots []*treeNode
flatNodes []*treeNode
```

### Key Bindings

| Key | Action |
|-----|--------|
| `t` | Toggle tree/list mode. Tree mode calls `buildTree` + `flatten`. |
| `>` / `right` | Expand selected node (if has children) |
| `<` / `left` | Collapse selected node (if has children) |
| `enter` | If node has children: toggle expand. Otherwise: open detail view. |

### View Rendering (tree mode)

```
#1  Epic        Feature A      In Progress  Alice
├─▶ #3  Task     Subtask 1      New          Bob
│  └─ #5  Sub-task Detail work  New          —
└─ #4  Task       Subtask 2      In Progress  Carol
▼ #2  Epic        Feature B      New          Dave
├─ #6  Task       Work item      New          —
└─ #7  Task       Another item   New          Eve
```

- Indentation: `depth * 2` spaces
- Lines: `├─` for non-last child, `└─` for last child, `│` for vertical continuation from ancestors
- Expand marker: `▶` collapsed (has children), `▼` expanded
- Columns after indent: ID, Type, Title, Status, Assignee (same as flat mode)

### Mouse

Click on `▶`/`▼` area toggles expand/collapse for that row.

### Help Bar

Tree mode shows: `t tree  >/< expand/collapse  enter open  / search  f filter  s sort  n/p page  ? help`

List mode shows existing bar unchanged.

## Edge Cases

### Sorting

- Tree mode: sort applies within each sibling group (does not cross parent-child boundaries)
- After sorting, rebuild tree

### Search

- Tree mode search: match nodes, auto-expand ancestors of matched nodes, collapse non-matching subtrees
- Preserve tree structure in search results

### Pagination

- Tree built only from current page data
- No change to pagination logic

### Empty Tree

- If no item has a parent in the current page, tree mode visually equals list mode (all items as roots)

## Files Changed

| File | Change |
|------|--------|
| `models/work_package.go` | Add `ParentId` field |
| `dtos/work_package.go` | Add `Parent` link, parse in `Convert()` |
| `components/resources/work_packages/all.go` | Ensure parent link included in request |
| `cmd/tui/tree.go` | **New** — `treeNode`, `buildTree`, `flatten`, line rendering helpers |
| `cmd/tui/list_view.go` | Add `treeMode`, toggle logic, tree-aware key handling and view |
| `cmd/tui/keymap.go` | Add tree key bindings |

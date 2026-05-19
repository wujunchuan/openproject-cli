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
		{2, false, []bool{false, true}, "│  ├─ "},
		{3, true, []bool{false, false, true}, "│  │  └─ "},
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

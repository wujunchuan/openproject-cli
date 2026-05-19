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
//
//	depth=0: ""         (root node, no prefix)
//	depth=1, isLast=false, ancestorsLast=[false]: "├─ "
//	depth=1, isLast=true,  ancestorsLast=[false]: "└─ "
//	depth=2, isLast=true,  ancestorsLast=[false,true]: "│  └─ "
//	depth=2, isLast=false, ancestorsLast=[false,false]: "│  ├─ "
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

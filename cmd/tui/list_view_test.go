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
	if len(m.flatNodes) != 3 {
		t.Fatalf("expected 3 visible nodes (Parent, Child, Root), got %d", len(m.flatNodes))
	}

	// Toggle back to list mode
	m, _ = m.Update(keyMsg("t"))
	if m.treeMode {
		t.Fatal("treeMode should be false after pressing t again")
	}
}

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

func TestTreeEnterOnParentOpensDetail(t *testing.T) {
	m := newListModel()
	m.SetWorkPackages(&models.WorkPackageCollection{
		Total: 2,
		Items: []*models.WorkPackage{
			{Id: 1, Subject: "Parent"},
			{Id: 2, Subject: "Child", ParentId: 1},
		},
	})

	m, _ = m.Update(keyMsg("t"))
	// Enter on parent should open detail (not toggle expand)
	m, cmd := m.Update(keyMsg("enter"))
	if cmd == nil {
		t.Fatal("expected detail command on enter")
	}
	msg := cmd()
	detailMsg, ok := msg.(openDetailMsg)
	if !ok {
		t.Fatalf("expected openDetailMsg, got %T", msg)
	}
	if detailMsg.wp.Id != 1 {
		t.Fatalf("expected wp id 1, got %d", detailMsg.wp.Id)
	}
	// flatNodes should be unchanged (not collapsed)
	if len(m.flatNodes) != 2 {
		t.Fatalf("expected 2 nodes (unchanged), got %d", len(m.flatNodes))
	}
}

func TestTreeLeftCollapsesParent(t *testing.T) {
	m := newListModel()
	m.SetWorkPackages(&models.WorkPackageCollection{
		Total: 2,
		Items: []*models.WorkPackage{
			{Id: 1, Subject: "Parent"},
			{Id: 2, Subject: "Child", ParentId: 1},
		},
	})

	m, _ = m.Update(keyMsg("t"))
	// < on parent should collapse
	m, _ = m.Update(keyMsg("<"))
	if len(m.flatNodes) != 1 {
		t.Fatalf("expected 1 node after collapsing, got %d", len(m.flatNodes))
	}
}

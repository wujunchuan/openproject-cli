package tui

import (
	"strings"
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

func TestDetailModelSetFamilyTree(t *testing.T) {
	parent := &models.WorkPackage{Id: 1, Subject: "Parent Task"}
	m := newDetailModel(&models.WorkPackage{Id: 2, Subject: "Child 1", ParentId: 1}, 80, 24)
	children := []*models.WorkPackage{
		{Id: 2, Subject: "Child 1", Status: "Open", ParentId: 1},
		{Id: 3, Subject: "Child 2", Status: "Closed", ParentId: 1},
	}
	m.SetFamilyTree(parent, children)

	if len(m.familyRoots) != 1 {
		t.Fatalf("expected 1 root, got %d", len(m.familyRoots))
	}
	if m.familyRoots[0].item.Id != 1 {
		t.Fatalf("expected root id 1, got %d", m.familyRoots[0].item.Id)
	}
	// parent (depth 0) + 2 children (depth 1) = 3 flat
	if len(m.familyFlat) != 3 {
		t.Fatalf("expected 3 flat nodes, got %d", len(m.familyFlat))
	}
}

func TestDetailModelSetFamilyTreeEmpty(t *testing.T) {
	m := newDetailModel(&models.WorkPackage{Id: 1, Subject: "No Children"}, 80, 24)
	m.SetFamilyTree(nil, nil)

	if len(m.familyRoots) != 0 {
		t.Fatalf("expected 0 roots, got %d", len(m.familyRoots))
	}
}

func TestDetailModelFamilyTreeRendersTree(t *testing.T) {
	parent := &models.WorkPackage{Id: 10, Subject: "Parent WP"}
	m := newDetailModel(&models.WorkPackage{Id: 11, Subject: "Current WP", ParentId: 10}, 80, 24)
	children := []*models.WorkPackage{
		{Id: 11, Subject: "Current WP", Status: "Open", ParentId: 10},
		{Id: 12, Subject: "Sibling WP", Status: "Closed", ParentId: 10},
	}
	m.SetFamilyTree(parent, children)

	content := m.viewport.View()
	if !strings.Contains(content, "Children") {
		t.Fatal("expected 'Children' header in view")
	}
	if !strings.Contains(content, "#10 Parent WP") {
		t.Fatal("expected '#10 Parent WP' in view")
	}
	if !strings.Contains(content, "#11 Current WP") {
		t.Fatal("expected '#11 Current WP' in view")
	}
	if !strings.Contains(content, "#12 Sibling WP") {
		t.Fatal("expected '#12 Sibling WP' in view")
	}
	// Current WP should have marker
	if !strings.Contains(content, "←") {
		t.Fatal("expected '←' marker on current WP")
	}
}

func TestDetailModelFamilyTreeCursorInit(t *testing.T) {
	parent := &models.WorkPackage{Id: 10, Subject: "Parent WP"}
	m := newDetailModel(&models.WorkPackage{Id: 12, Subject: "Child 2", ParentId: 10}, 80, 24)
	children := []*models.WorkPackage{
		{Id: 11, Subject: "Child 1", Status: "Open", ParentId: 10},
		{Id: 12, Subject: "Child 2", Status: "Open", ParentId: 10},
		{Id: 13, Subject: "Child 3", Status: "Open", ParentId: 10},
	}
	m.SetFamilyTree(parent, children)

	// Cursor should be on current WP (id=12, index 2 in flat: 0=parent, 1=child1, 2=child2)
	if m.familyCursor != 2 {
		t.Fatalf("expected cursor=2 (current WP), got %d", m.familyCursor)
	}
}

func TestDetailModelFamilyTreeNavigation(t *testing.T) {
	parent := &models.WorkPackage{Id: 10, Subject: "Parent WP"}
	m := newDetailModel(&models.WorkPackage{Id: 11, Subject: "Child 1", ParentId: 10}, 80, 24)
	children := []*models.WorkPackage{
		{Id: 11, Subject: "Child 1", Status: "Open", ParentId: 10},
		{Id: 12, Subject: "Child 2", Status: "Open", ParentId: 10},
		{Id: 13, Subject: "Child 3", Status: "Open", ParentId: 10},
	}
	m.SetFamilyTree(parent, children)

	// Initial: cursor on current WP (index 1)
	if m.familyCursor != 1 {
		t.Fatalf("expected cursor=1, got %d", m.familyCursor)
	}

	// Focus the tree
	m.Update(keyMsg("tab"))
	if !m.familyFocused {
		t.Fatal("expected familyFocused=true after tab")
	}

	// j moves down
	m.Update(keyMsg("j"))
	if m.familyCursor != 2 {
		t.Fatalf("expected cursor=2 after j, got %d", m.familyCursor)
	}

	// j moves down
	m.Update(keyMsg("j"))
	if m.familyCursor != 3 {
		t.Fatalf("expected cursor=3 after j, got %d", m.familyCursor)
	}

	// j at end should not move
	m.Update(keyMsg("j"))
	if m.familyCursor != 3 {
		t.Fatalf("expected cursor=3 (end), got %d", m.familyCursor)
	}

	// k moves up
	m.Update(keyMsg("k"))
	if m.familyCursor != 2 {
		t.Fatalf("expected cursor=2 after k, got %d", m.familyCursor)
	}
}

func TestDetailModelFamilyTreeEnterOpensDetail(t *testing.T) {
	parent := &models.WorkPackage{Id: 10, Subject: "Parent WP"}
	current := &models.WorkPackage{Id: 11, Subject: "Child 1", ParentId: 10}
	m := newDetailModel(current, 80, 24)
	children := []*models.WorkPackage{
		{Id: 11, Subject: "Child 1", Status: "Open", ParentId: 10},
		{Id: 12, Subject: "Child 2", Status: "Closed", ParentId: 10},
	}
	m.SetFamilyTree(parent, children)

	// Focus and move to child 2
	m.Update(keyMsg("tab"))
	m.Update(keyMsg("j"))

	// l should open detail
	_, cmd := m.Update(keyMsg("l"))
	if cmd == nil {
		t.Fatal("l should return a command")
	}
	msg := cmd()
	openMsg, ok := msg.(openDetailMsg)
	if !ok {
		t.Fatalf("expected openDetailMsg, got %T", msg)
	}
	if openMsg.wp.Id != 12 {
		t.Fatalf("expected id 12, got %d", openMsg.wp.Id)
	}
}

func TestDetailModelFamilyTreeEnterOnCurrentWPNoop(t *testing.T) {
	parent := &models.WorkPackage{Id: 10, Subject: "Parent WP"}
	m := newDetailModel(&models.WorkPackage{Id: 11, Subject: "Child 1", ParentId: 10}, 80, 24)
	children := []*models.WorkPackage{
		{Id: 11, Subject: "Child 1", Status: "Open", ParentId: 10},
		{Id: 12, Subject: "Child 2", Status: "Open", ParentId: 10},
	}
	m.SetFamilyTree(parent, children)

	// Focus — cursor is on current WP (index 1)
	m.Update(keyMsg("tab"))
	_, cmd := m.Update(keyMsg("l"))
	if cmd != nil {
		t.Fatal("l on current WP should return nil command")
	}
}

func TestDetailModelFamilyTreeEnterOnParentNoop(t *testing.T) {
	parent := &models.WorkPackage{Id: 10, Subject: "Parent WP"}
	m := newDetailModel(&models.WorkPackage{Id: 11, Subject: "Child 1", ParentId: 10}, 80, 24)
	children := []*models.WorkPackage{
		{Id: 11, Subject: "Child 1", Status: "Open", ParentId: 10},
	}
	m.SetFamilyTree(parent, children)

	// Focus, move cursor to parent (index 0)
	m.Update(keyMsg("tab"))
	m.Update(keyMsg("k"))

	// l on parent (has children) should do nothing
	_, cmd := m.Update(keyMsg("l"))
	if cmd != nil {
		t.Fatal("l on parent node should return nil command")
	}
}

func TestDetailModelFamilyTreeTabToggleFocus(t *testing.T) {
	parent := &models.WorkPackage{Id: 10, Subject: "Parent WP"}
	m := newDetailModel(&models.WorkPackage{Id: 11, Subject: "Child 1", ParentId: 10}, 80, 24)
	children := []*models.WorkPackage{
		{Id: 11, Subject: "Child 1", Status: "Open", ParentId: 10},
		{Id: 12, Subject: "Child 2", Status: "Open", ParentId: 10},
	}
	m.SetFamilyTree(parent, children)

	// Initially not focused
	if m.familyFocused {
		t.Fatal("expected familyFocused=false initially")
	}

	// Tab focuses
	m.Update(keyMsg("tab"))
	if !m.familyFocused {
		t.Fatal("expected familyFocused=true after tab")
	}

	// Tab unfocuses
	m.Update(keyMsg("tab"))
	if m.familyFocused {
		t.Fatal("expected familyFocused=false after second tab")
	}
}

func TestDetailModelFamilyTreeUnfocusedJNoop(t *testing.T) {
	parent := &models.WorkPackage{Id: 10, Subject: "Parent WP"}
	m := newDetailModel(&models.WorkPackage{Id: 11, Subject: "Child 1", ParentId: 10}, 80, 24)
	children := []*models.WorkPackage{
		{Id: 11, Subject: "Child 1", Status: "Open", ParentId: 10},
		{Id: 12, Subject: "Child 2", Status: "Open", ParentId: 10},
	}
	m.SetFamilyTree(parent, children)

	cursor := m.familyCursor
	// Without focus, j should not change cursor
	m.Update(keyMsg("j"))
	if m.familyCursor != cursor {
		t.Fatalf("j without focus should not move cursor, expected %d got %d", cursor, m.familyCursor)
	}
}

func TestDetailModelTabWithoutTreeNoop(t *testing.T) {
	m := newDetailModel(&models.WorkPackage{Id: 1, Subject: "No Children"}, 80, 24)
	// No family tree set

	m.Update(keyMsg("tab"))
	if m.familyFocused {
		t.Fatal("tab without tree should not focus")
	}
}

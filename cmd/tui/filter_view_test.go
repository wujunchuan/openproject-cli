package tui

import (
	"testing"

	"github.com/opf/openproject-cli/components/resources/work_packages"
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

	// Status field: open popup and select via enter
	m.activeField = 1
	m.Update(keyMsg("enter")) // open popup
	if m.state != filterPopup {
		t.Fatal("expected filterPopup state")
	}

	m.Update(keyMsg("j")) // move to "open"
	m.Update(keyMsg("enter")) // select
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
		if len(field.selected) != 0 {
			t.Fatalf("expected field %s selected cleared", field.name)
		}
	}
}

func TestFilterModelJKNavigation(t *testing.T) {
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
}

func TestFilterPopupOpenAndSelect(t *testing.T) {
	m := newFilterModel()
	m.activeField = 1 // Status: ["all", "open", "closed"]

	// Open popup
	m.Update(keyMsg("enter"))
	if m.state != filterPopup {
		t.Fatal("expected filterPopup state after enter")
	}
	if len(m.popupItems) != 3 {
		t.Fatalf("expected 3 popup items, got %d", len(m.popupItems))
	}

	// Navigate down in popup
	m.Update(keyMsg("j"))
	if m.popupIndex != 1 {
		t.Fatalf("expected popupIndex=1, got %d", m.popupIndex)
	}

	// Select (single-select mode: enter selects)
	m.Update(keyMsg("enter"))
	if m.state != filterBrowseFields {
		t.Fatal("expected filterBrowseFields after enter")
	}
	if m.fields[1].current != 1 {
		t.Fatalf("expected field current=1, got %d", m.fields[1].current)
	}
}

func TestFilterPopupCancel(t *testing.T) {
	m := newFilterModel()
	m.activeField = 1
	original := m.fields[1].current

	m.Update(keyMsg("enter"))
	m.Update(keyMsg("j"))  // navigate away
	m.Update(keyMsg("esc")) // cancel

	if m.state != filterBrowseFields {
		t.Fatal("expected filterBrowseFields after esc")
	}
	if m.fields[1].current != original {
		t.Fatalf("expected field unchanged, got %d", m.fields[1].current)
	}
}

func TestFilterMultiSelect(t *testing.T) {
	m := newFilterModel()
	m.fields[1].options = []string{"all", "open", "closed", "in progress"}
	m.fields[1].ids = []string{"", "open", "closed", "7"}
	m.fields[1].multi = true
	m.activeField = 1

	// Open popup
	m.Update(keyMsg("enter"))

	// Move to "open" and toggle with space
	m.Update(keyMsg("j")) // popupIndex=1 (open)
	m.Update(keyMsg(" ")) // toggle

	if !m.fields[1].selected[1] {
		t.Fatal("expected 'open' to be selected")
	}

	// Move to "in progress" and toggle
	m.Update(keyMsg("j")) // popupIndex=2 (closed)
	m.Update(keyMsg("j")) // popupIndex=3 (in progress)
	m.Update(keyMsg(" ")) // toggle

	if !m.fields[1].selected[3] {
		t.Fatal("expected 'in progress' to be selected")
	}
	if !m.fields[1].selected[1] {
		t.Fatal("expected 'open' to still be selected")
	}

	// Confirm
	m.Update(keyMsg("enter"))

	opts := m.FilterOptions()
	if opts[work_packages.Status] != "7,open" && opts[work_packages.Status] != "open,7" {
		t.Fatalf("expected '7,open' or 'open,7', got '%s'", opts[work_packages.Status])
	}
}

func TestFilterMultiSelectNumberShortcut(t *testing.T) {
	m := newFilterModel()
	m.fields[1].options = []string{"all", "open", "closed", "in progress"}
	m.fields[1].ids = []string{"", "open", "closed", "7"}
	m.fields[1].multi = true
	m.activeField = 1

	// Open popup
	m.Update(keyMsg("enter"))

	// Press "2" to quick-toggle "open" (index 1)
	m.Update(keyMsg("2"))
	if !m.fields[1].selected[1] {
		t.Fatal("expected 'open' selected via number shortcut")
	}

	// Press "4" to quick-toggle "in progress" (index 3)
	m.Update(keyMsg("4"))
	if !m.fields[1].selected[3] {
		t.Fatal("expected 'in progress' selected via number shortcut")
	}
}

func TestFilterMultiSelectAllClearsSelection(t *testing.T) {
	m := newFilterModel()
	m.fields[1].options = []string{"all", "open", "closed"}
	m.fields[1].ids = []string{"", "open", "closed"}
	m.fields[1].multi = true
	m.activeField = 1

	// Open popup, select "open"
	m.Update(keyMsg("enter"))
	m.Update(keyMsg("j"))
	m.Update(keyMsg(" ")) // toggle open

	if !m.fields[1].selected[1] {
		t.Fatal("expected 'open' selected")
	}

	// Toggle "all" (index 0) - should clear all
	m.Update(keyMsg("up")) // go to "all"
	m.Update(keyMsg(" ")) // toggle all

	if len(m.fields[1].selected) != 0 {
		t.Fatalf("expected no selection after toggling 'all', got %v", m.fields[1].selected)
	}
}

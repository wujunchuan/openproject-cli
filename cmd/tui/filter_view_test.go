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

func TestFilterModelHJKLNavigation(t *testing.T) {
	m := newFilterModel()

	if m.activeField != 0 {
		t.Fatalf("expected activeField=0, got %d", m.activeField)
	}

	// l = next field
	m.Update(keyMsg("l"))
	if m.activeField != 1 {
		t.Fatalf("expected activeField=1 after l, got %d", m.activeField)
	}

	// h = previous field
	m.Update(keyMsg("h"))
	if m.activeField != 0 {
		t.Fatalf("expected activeField=0 after h, got %d", m.activeField)
	}

	// j = next value
	m.activeField = 1 // Status: ["all", "open", "closed"]
	m.Update(keyMsg("j"))
	if m.fields[1].current != 1 {
		t.Fatalf("expected current=1 after j, got %d", m.fields[1].current)
	}

	// k = previous value
	m.Update(keyMsg("k"))
	if m.fields[1].current != 0 {
		t.Fatalf("expected current=0 after k, got %d", m.fields[1].current)
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

	// Select
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

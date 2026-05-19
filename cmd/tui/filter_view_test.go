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

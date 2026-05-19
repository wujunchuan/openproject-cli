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

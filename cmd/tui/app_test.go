package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/opf/openproject-cli/models"
)

func updateApp(app App, msg tea.Msg) App {
	m, _ := app.Update(msg)
	return m.(App)
}

func TestAppInit(t *testing.T) {
	app := NewApp()
	cmd := app.Init()
	if cmd == nil {
		t.Fatal("Init should return a command")
	}
}

func TestAppViewStackNavigation(t *testing.T) {
	app := NewApp()
	app.width = 80
	app.height = 24

	wp := &models.WorkPackage{Id: 1, Subject: "Test", Status: "New"}

	// Simulate opening detail
	app = updateApp(app, openDetailMsg{wp: wp})
	if app.state != detailView {
		t.Fatal("expected detailView after openDetailMsg")
	}
	if app.detail == nil {
		t.Fatal("detail model should be set")
	}

	// Simulate going back
	app = updateApp(app, backToListMsg{})
	if app.state != listView {
		t.Fatal("expected listView after backToListMsg")
	}
	if app.detail != nil {
		t.Fatal("detail model should be nil after going back")
	}
}

func TestAppWindowSize(t *testing.T) {
	app := NewApp()
	app = updateApp(app, tea.WindowSizeMsg{Width: 120, Height: 40})
	if app.width != 120 || app.height != 40 {
		t.Fatalf("expected 120x40, got %dx%d", app.width, app.height)
	}
}

func keyMsg(key string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
}

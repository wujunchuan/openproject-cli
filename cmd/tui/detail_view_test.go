package tui

import (
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

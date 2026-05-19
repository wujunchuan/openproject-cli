package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/opf/openproject-cli/components/launch"
	"github.com/opf/openproject-cli/components/resources/work_packages"
	"github.com/opf/openproject-cli/components/routes"
	"github.com/opf/openproject-cli/models"
)

type detailModel struct {
	wp          *models.WorkPackage
	activities  []*models.Activity
	viewport    viewport.Model
	width       int
	height      int
	loading     bool
	editOverlay bool
	edit        *editModel
	showHelp    bool
}

func newDetailModel(wp *models.WorkPackage, w, h int) *detailModel {
	vp := viewport.New(w-4, h-10)
	vp.SetContent("")
	return &detailModel{
		wp:       wp,
		viewport: vp,
		width:    w,
		height:   h,
		loading:  true,
	}
}

func (m *detailModel) Init() tea.Cmd {
	return m.loadActivities
}

func (m *detailModel) SetWorkPackage(wp *models.WorkPackage) {
	m.wp = wp
	m.updateContent()
}

func (m *detailModel) SetActivities(activities []*models.Activity) {
	m.activities = activities
	m.loading = false
	m.updateContent()
}

func (m *detailModel) loadActivities() tea.Msg {
	activities, err := work_packages.Activities(m.wp.Id)
	return activitiesLoadedMsg{activities: activities, err: err}
}

func (m *detailModel) updateContent() {
	var b strings.Builder

	// Header
	b.WriteString(titleStyle.Render(fmt.Sprintf("#%d %s", m.wp.Id, m.wp.Subject)))
	b.WriteString("\n\n")

	// Properties (two columns)
	left := fmt.Sprintf("Type: %s\nStatus: %s\nProject: %s\nAssignee: %s",
		m.wp.Type, m.wp.Status, m.wp.Project, assigneeOrDash(m.wp.Assignee))
	right := fmt.Sprintf("Priority: %s\nVersion: %s\nStart: %s\nDue: %s\nCreated: %s\nUpdated: %s",
		m.wp.Priority, m.wp.Version,
		dateOrDash(m.wp.StartDate), dateOrDash(m.wp.DueDate),
		m.wp.CreatedAt, m.wp.UpdatedAt)

	leftCol := lipgloss.NewStyle().Width(m.width/2 - 4).Render(left)
	rightCol := lipgloss.NewStyle().Width(m.width/2 - 4).Render(right)

	b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, leftCol, rightCol))
	b.WriteString("\n\n")

	// Description
	if m.wp.Description != "" {
		b.WriteString(headerStyle.Render("Description"))
		b.WriteString("\n")
		b.WriteString(m.wp.Description)
		b.WriteString("\n\n")
	}

	// Activities
	b.WriteString(headerStyle.Render(fmt.Sprintf("Activity (%d)", len(m.activities))))
	b.WriteString("\n")
	if m.loading {
		b.WriteString("\n  Loading activities...\n")
	} else {
		for _, act := range m.activities {
			b.WriteString(fmt.Sprintf("\n  %s\n", subtitleStyle.Render(act.CreatedAt)))
			for _, detail := range act.Details {
				if detail != nil {
					b.WriteString(fmt.Sprintf("  %s\n", *detail))
				}
			}
			if act.Comment != "" {
				b.WriteString(fmt.Sprintf("  %s\n", act.Comment))
			}
		}
	}

	m.viewport.SetContent(b.String())
}

func (m *detailModel) Update(msg tea.Msg) (*detailModel, tea.Cmd) {
	// Handle edit overlay messages first
	if m.editOverlay {
		switch msg := msg.(type) {
		case editDoneMsg:
			m.editOverlay = false
			m.edit = nil
			if msg.refresh {
				m.loading = true
				return m, tea.Batch(
					func() tea.Msg {
						wp, err := work_packages.Lookup(m.wp.Id)
						return workPackageDetailMsg{wp: wp, err: err}
					},
					m.loadActivities,
				)
			}
			return m, nil
		case editOptionsMsg:
			m.edit.options = msg.options
			return m, nil
		case editErrorMsg:
			m.edit.err = msg.err
			m.edit.state = editChooseField
			return m, nil
		}

		var cmd tea.Cmd
		m.edit, cmd = m.edit.Update(msg)
		return m, cmd
	}

	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "?":
			m.showHelp = !m.showHelp
		case "esc":
			if m.showHelp {
				m.showHelp = false
			} else {
				return m, BackToListCmd()
			}
		case "e":
			m.editOverlay = true
			m.edit = newEditModel(m.wp, m.width)
			return m, nil
		case "o":
			_ = launch.Browser(routes.WorkPackageUrl(m.wp))
		case "r":
			m.loading = true
			return m, tea.Batch(
				func() tea.Msg {
					wp, err := work_packages.Lookup(m.wp.Id)
					return workPackageDetailMsg{wp: wp, err: err}
				},
				m.loadActivities,
			)
		}
	}

	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m *detailModel) View() string {
	if m.showHelp {
		return helpOverlay("Detail — Key Bindings", [][2]string{
			{"↑ / ↓", "scroll content"},
			{"PgUp / PgDn", "scroll page"},
			{"esc", "back to list"},
			{"e", "edit (type, start/due date)"},
			{"o", "open in browser"},
			{"r", "refresh"},
			{"?", "toggle this help"},
		}, m.width)
	}
	if m.editOverlay {
		return m.edit.View()
	}
	footer := helpStyle.Render("  esc back  e edit  o browser  r refresh  ? help")
	return m.viewport.View() + "\n" + footer
}

func dateOrDash(s string) string {
	if s == "" {
		return "—"
	}
	return s
}

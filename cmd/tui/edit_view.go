package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/opf/openproject-cli/components/resources/work_packages"
	"github.com/opf/openproject-cli/models"
)

type editState int

const (
	editChooseField editState = iota
	editChooseValue
	editSubmitting
)

type editModel struct {
	wp          *models.WorkPackage
	state       editState
	options     []string
	optionIndex int
	err         string
	width       int
}

func newEditModel(wp *models.WorkPackage, w int) *editModel {
	return &editModel{
		wp:    wp,
		state: editChooseField,
		width: w,
	}
}

func (m *editModel) Update(msg tea.Msg) (*editModel, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	switch m.state {
	case editChooseField:
		switch keyMsg.String() {
		case "t":
			m.state = editChooseValue
			return m, m.loadTypes
		case "esc":
			return m, func() tea.Msg { return editDoneMsg{} }
		}

	case editChooseValue:
		switch keyMsg.String() {
		case "up", "k":
			if m.optionIndex > 0 {
				m.optionIndex--
			}
		case "down", "j":
			if m.optionIndex < len(m.options)-1 {
				m.optionIndex++
			}
		case "enter":
			if m.optionIndex >= 0 && m.optionIndex < len(m.options) {
				m.state = editSubmitting
				return m, m.submitType
			}
		case "esc":
			m.state = editChooseField
			m.options = nil
			m.err = ""
			return m, nil
		}
	}

	return m, nil
}

func (m *editModel) loadTypes() tea.Msg {
	types, err := work_packages.AvailableTypes(m.wp.Id)
	if err != nil {
		return editErrorMsg{err: err.Error()}
	}
	var opts []string
	for _, t := range types {
		opts = append(opts, t.Name)
	}
	return editOptionsMsg{options: opts}
}

func (m *editModel) submitType() tea.Msg {
	if m.optionIndex >= len(m.options) {
		return editErrorMsg{err: "invalid selection"}
	}
	selected := m.options[m.optionIndex]

	opts := map[work_packages.UpdateOption]string{
		work_packages.UpdateType: selected,
	}

	_, err := work_packages.Update(m.wp.Id, opts)
	if err != nil {
		return editErrorMsg{err: err.Error()}
	}

	return editDoneMsg{refresh: true}
}

func (m *editModel) View() string {
	var b strings.Builder

	b.WriteString("\n")
	b.WriteString(titleStyle.Render(fmt.Sprintf("Edit #%d", m.wp.Id)))
	b.WriteString("\n\n")

	if m.err != "" {
		b.WriteString(errorStyle.Render("  Error: " + m.err))
		b.WriteString("\n\n")
	}

	switch m.state {
	case editChooseField:
		b.WriteString("  [T]ype\n\n")
		b.WriteString(helpStyle.Render("  Press a key to select a field, esc to cancel"))
	case editChooseValue:
		b.WriteString("  Select type:\n\n")
		for i, opt := range m.options {
			prefix := "  "
			if i == m.optionIndex {
				prefix = "▸ "
				b.WriteString(selectedItemStyle.Render(prefix + opt))
			} else {
				b.WriteString(normalItemStyle.Render(prefix + opt))
			}
			b.WriteString("\n")
		}
		b.WriteString("\n")
		b.WriteString(helpStyle.Render("  ↑↓ select  enter confirm  esc back"))
	case editSubmitting:
		b.WriteString("  Submitting...\n")
	}

	return b.String()
}

// --- Edit Messages ---

type editDoneMsg struct {
	refresh bool
}

type editOptionsMsg struct {
	options []string
}

type editErrorMsg struct {
	err string
}

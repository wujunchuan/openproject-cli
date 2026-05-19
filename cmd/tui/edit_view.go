package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/opf/openproject-cli/components/resources/status"
	"github.com/opf/openproject-cli/components/resources/work_packages"
	"github.com/opf/openproject-cli/models"
)

type editState int

const (
	editChooseField editState = iota
	editChooseValue
	editTextInput
	editSubmitting
)

type editModel struct {
	wp          *models.WorkPackage
	state       editState
	options     []string
	optionIndex int
	err         string
	width       int
	activeField string
	textInput   textinput.Model
}

func newEditModel(wp *models.WorkPackage, w int) *editModel {
	ti := textinput.New()
	ti.Placeholder = "YYYY-MM-DD"
	ti.CharLimit = 10
	ti.Width = 30
	return &editModel{
		wp:        wp,
		state:     editChooseField,
		width:     w,
		textInput: ti,
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
			m.activeField = "type"
			return m, m.loadTypes
		case "u":
			m.state = editChooseValue
			m.activeField = "status"
			return m, m.loadStatuses
		case "s":
			m.state = editTextInput
			m.activeField = "startDate"
			m.textInput.SetValue(m.wp.StartDate)
			m.textInput.Focus()
			return m, nil
		case "d":
			m.state = editTextInput
			m.activeField = "dueDate"
			m.textInput.SetValue(m.wp.DueDate)
			m.textInput.Focus()
			return m, nil
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
				switch m.activeField {
				case "status":
					return m, m.submitStatus
				default:
					return m, m.submitType
				}
			}
		case "esc":
			m.state = editChooseField
			m.options = nil
			m.err = ""
			return m, nil
		}

	case editTextInput:
		switch keyMsg.String() {
		case "esc":
			m.state = editChooseField
			m.textInput.Blur()
			return m, nil
		case "enter":
			m.state = editSubmitting
			m.textInput.Blur()
			return m, m.submitDate
		}
		var cmd tea.Cmd
		m.textInput, cmd = m.textInput.Update(msg)
		return m, cmd
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

func (m *editModel) loadStatuses() tea.Msg {
	statuses, err := status.All()
	if err != nil {
		return editErrorMsg{err: err.Error()}
	}
	var opts []string
	for _, s := range statuses {
		opts = append(opts, s.Name)
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

func (m *editModel) submitStatus() tea.Msg {
	if m.optionIndex >= len(m.options) {
		return editErrorMsg{err: "invalid selection"}
	}
	selected := m.options[m.optionIndex]

	opts := map[work_packages.UpdateOption]string{
		work_packages.UpdateStatus: selected,
	}

	_, err := work_packages.Update(m.wp.Id, opts)
	if err != nil {
		return editErrorMsg{err: err.Error()}
	}

	return editDoneMsg{refresh: true}
}

func (m *editModel) submitDate() tea.Msg {
	value := m.textInput.Value()
	var updateOpt work_packages.UpdateOption
	switch m.activeField {
	case "startDate":
		updateOpt = work_packages.UpdateStartDate
	case "dueDate":
		updateOpt = work_packages.UpdateDueDate
	default:
		return editErrorMsg{err: "unknown field"}
	}
	opts := map[work_packages.UpdateOption]string{updateOpt: value}
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
		b.WriteString("  [T]ype\n")
		b.WriteString("  [U]status\n")
		b.WriteString("  [S]tart date\n")
		b.WriteString("  [D]ue date\n\n")
		b.WriteString(helpStyle.Render("  Press a key to select a field, esc to cancel"))
	case editChooseValue:
		b.WriteString(fmt.Sprintf("  Select %s:\n\n", fieldLabel(m.activeField)))
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
	case editTextInput:
		b.WriteString(fmt.Sprintf("  %s:\n\n", fieldLabel(m.activeField)))
		b.WriteString("  " + m.textInput.View() + "\n\n")
		b.WriteString(helpStyle.Render("  enter confirm  esc cancel"))
	case editSubmitting:
		b.WriteString("  Submitting...\n")
	}

	return b.String()
}

func fieldLabel(field string) string {
	switch field {
	case "type":
		return "type"
	case "status":
		return "status"
	case "startDate":
		return "Start date"
	case "dueDate":
		return "Due date"
	default:
		return field
	}
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

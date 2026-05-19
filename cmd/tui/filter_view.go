package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/opf/openproject-cli/components/configuration"
	"github.com/opf/openproject-cli/components/resources/projects"
	"github.com/opf/openproject-cli/components/resources/status"
	resTypes "github.com/opf/openproject-cli/components/resources/types"
	"github.com/opf/openproject-cli/components/resources/users"
	"github.com/opf/openproject-cli/components/resources/work_packages"
)

type filterState int

const (
	filterBrowseFields filterState = iota
	filterPopup
)

type filterApplyMsg struct{}

type filterField struct {
	name    string
	options []string
	current int
}

type filterModel struct {
	fields       []filterField
	activeField  int
	visible      bool
	originalOpts map[work_packages.FilterOption]string
	showHelp     bool
	state        filterState
	popupItems   []string
	popupIndex   int
}

func newFilterModel() *filterModel {
	return &filterModel{
		fields: []filterField{
			{name: "Project", options: []string{"all"}},
			{name: "Status", options: []string{"all", "open", "closed"}},
			{name: "Type", options: []string{"all"}},
			{name: "Assignee", options: []string{"all", "me"}},
		},
		originalOpts: make(map[work_packages.FilterOption]string),
	}
}

func (m *filterModel) loadOptions() tea.Cmd {
	return func() tea.Msg {
		// Load projects
		if ps, err := projects.All(); err == nil {
			m.fields[0].options = []string{"all"}
			for _, p := range ps {
				m.fields[0].options = append(m.fields[0].options, p.Name)
			}
		}

		// Load statuses
		if ss, err := status.All(); err == nil {
			m.fields[1].options = []string{"all"}
			for _, s := range ss {
				m.fields[1].options = append(m.fields[1].options, s.Name)
			}
		}

		// Load types
		if ts, err := resTypes.All(); err == nil {
			m.fields[2].options = []string{"all"}
			for _, t := range ts {
				m.fields[2].options = append(m.fields[2].options, t.Name)
			}
		}

		// Load users for assignee
		if us, err := users.All(); err == nil {
			m.fields[3].options = []string{"all", "me"}
			for _, u := range us {
				m.fields[3].options = append(m.fields[3].options, u.Name)
			}
		}

		return nil
	}
}

func (m *filterModel) Update(msg tea.Msg) tea.Cmd {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return nil
	}

	if m.state == filterPopup {
		return m.updatePopup(keyMsg)
	}

	// filterBrowseFields
	switch keyMsg.String() {
	case "?":
		m.showHelp = !m.showHelp
	case "tab", "down", "j":
		if !m.showHelp {
			m.activeField = (m.activeField + 1) % len(m.fields)
		}
	case "shift+tab", "up", "k":
		if !m.showHelp {
			m.activeField--
			if m.activeField < 0 {
				m.activeField = len(m.fields) - 1
			}
		}
	case "right", "l":
		if !m.showHelp {
			field := &m.fields[m.activeField]
			if field.current < len(field.options)-1 {
				field.current++
			}
		}
	case "left", "h":
		if !m.showHelp {
			field := &m.fields[m.activeField]
			if field.current > 0 {
				field.current--
			}
		}
	case "c":
		if !m.showHelp {
			for i := range m.fields {
				m.fields[i].current = 0
			}
		}
	case "enter":
		if !m.showHelp {
			m.openPopup()
		}
	case "a":
		if !m.showHelp {
			return func() tea.Msg { return filterApplyMsg{} }
		}
	}
	return nil
}

func (m *filterModel) isInPopup() bool {
	return m.state == filterPopup
}

func (m *filterModel) openPopup() {
	field := m.fields[m.activeField]
	m.popupItems = field.options
	m.popupIndex = field.current
	m.state = filterPopup
}

func (m *filterModel) updatePopup(keyMsg tea.KeyMsg) tea.Cmd {
	switch keyMsg.String() {
	case "up", "k":
		if m.popupIndex > 0 {
			m.popupIndex--
		}
	case "down", "j":
		if m.popupIndex < len(m.popupItems)-1 {
			m.popupIndex++
		}
	case "enter":
		m.fields[m.activeField].current = m.popupIndex
		m.state = filterBrowseFields
		m.popupItems = nil
	case "esc", "q":
		m.state = filterBrowseFields
		m.popupItems = nil
	}
	return nil
}

func (m *filterModel) popupView() string {
	var b strings.Builder
	b.WriteString("\n")
	field := m.fields[m.activeField]
	b.WriteString(titleStyle.Render(field.name))
	b.WriteString("\n\n")

	for i, item := range m.popupItems {
		prefix := "  "
		if i == m.popupIndex {
			prefix = "▸ "
			b.WriteString(selectedItemStyle.Render(prefix + item))
		} else {
			b.WriteString(normalItemStyle.Render(prefix + item))
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(helpStyle.Render("  j/k move  enter select  esc cancel"))
	return b.String()
}

func (m *filterModel) FilterOptions() map[work_packages.FilterOption]string {
	opts := make(map[work_packages.FilterOption]string)

	keys := []work_packages.FilterOption{
		work_packages.Project,
		work_packages.Status,
		work_packages.Type,
		work_packages.Assignee,
	}

	for i, field := range m.fields {
		val := field.options[field.current]
		if val != "all" {
			opts[keys[i]] = val
		}
	}
	return opts
}

func (m *filterModel) View() string {
	if m.showHelp {
		return helpOverlay("Filter — Key Bindings", [][2]string{
			{"j / k", "next / previous field"},
			{"h / l", "previous / next value"},
			{"tab / shift+tab", "next / previous field"},
			{"← / →", "change value"},
			{"enter", "open popup"},
			{"a", "apply filters"},
			{"esc", "cancel / close popup"},
			{"c", "clear all filters"},
			{"?", "toggle this help"},
		}, 60)
	}

	if m.state == filterPopup {
		return m.popupView()
	}

	var b strings.Builder
	b.WriteString(titleStyle.Render("Filters"))
	b.WriteString("\n\n")

	for i, field := range m.fields {
		prefix := "  "
		if i == m.activeField {
			prefix = "▸ "
		}

		value := field.options[field.current]
		line := fmt.Sprintf("%s%-10s [%s]", prefix, field.name+":", value)

		if i == m.activeField {
			b.WriteString(selectedItemStyle.Render(line))
		} else {
			b.WriteString(normalItemStyle.Render(line))
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(helpStyle.Render("  j/k field  h/l value  enter popup  a apply  esc cancel  c clear  ? help"))
	return b.String()
}

func (m *filterModel) toFilterState() configuration.FilterState {
	return configuration.FilterState{
		Project:  m.fields[0].options[m.fields[0].current],
		Status:   m.fields[1].options[m.fields[1].current],
		Type:     m.fields[2].options[m.fields[2].current],
		Assignee: m.fields[3].options[m.fields[3].current],
	}
}

func (m *filterModel) setFromState(fs configuration.FilterState) {
	vals := []string{fs.Project, fs.Status, fs.Type, fs.Assignee}
	for i, field := range m.fields {
		val := vals[i]
		if val == "" {
			val = "all"
		}
		for j, opt := range field.options {
			if opt == val {
				m.fields[i].current = j
				break
			}
		}
	}
}

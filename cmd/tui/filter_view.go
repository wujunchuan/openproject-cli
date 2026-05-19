package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/opf/openproject-cli/components/resources/projects"
	"github.com/opf/openproject-cli/components/resources/status"
	resTypes "github.com/opf/openproject-cli/components/resources/types"
	"github.com/opf/openproject-cli/components/resources/work_packages"
)

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
}

func newFilterModel() *filterModel {
	return &filterModel{
		fields: []filterField{
			{name: "Project", options: []string{"all"}},
			{name: "Status", options: []string{"all", "open", "closed"}},
			{name: "Type", options: []string{"all"}},
			{name: "Assignee", options: []string{"all", "me", "none"}},
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

		return nil
	}
}

func (m *filterModel) Update(msg tea.Msg) tea.Cmd {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return nil
	}

	switch keyMsg.String() {
	case "?":
		m.showHelp = !m.showHelp
	case "tab":
		if !m.showHelp {
			m.activeField = (m.activeField + 1) % len(m.fields)
		}
	case "shift+tab":
		if !m.showHelp {
			m.activeField--
			if m.activeField < 0 {
				m.activeField = len(m.fields) - 1
			}
		}
	case "up":
		if !m.showHelp {
			field := &m.fields[m.activeField]
			if field.current > 0 {
				field.current--
			}
		}
	case "down":
		if !m.showHelp {
			field := &m.fields[m.activeField]
			if field.current < len(field.options)-1 {
				field.current++
			}
		}
	case "c":
		if !m.showHelp {
			for i := range m.fields {
				m.fields[i].current = 0
			}
		}
	}
	return nil
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
			{"tab", "next field"},
			{"shift+tab", "previous field"},
			{"↑ / ↓", "cycle value"},
			{"enter", "apply filters"},
			{"esc", "cancel"},
			{"c", "clear all filters"},
			{"?", "toggle this help"},
		}, 60)
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
	b.WriteString(helpStyle.Render("  tab/shift+tab field  ↑↓ select  enter apply  esc cancel  c clear  ? help"))
	return b.String()
}

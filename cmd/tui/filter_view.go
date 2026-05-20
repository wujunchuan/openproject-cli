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
	name     string
	options  []string
	ids      []string
	current  int
	selected map[int]bool // multi-select: set of selected indices
	multi    bool         // whether this field supports multi-select
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
			{name: "Project", options: []string{"all"}, ids: []string{""}, selected: map[int]bool{}, multi: false},
			{name: "Status", options: []string{"all", "open", "closed"}, ids: []string{"", "open", "closed"}, selected: map[int]bool{}, multi: true},
			{name: "Type", options: []string{"all"}, ids: []string{""}, selected: map[int]bool{}, multi: true},
			{name: "Assignee", options: []string{"all", "me"}, ids: []string{"", "me"}, selected: map[int]bool{}, multi: true},
		},
		originalOpts: make(map[work_packages.FilterOption]string),
	}
}

func (m *filterModel) loadOptions() tea.Cmd {
	return func() tea.Msg {
		statusColors := make(map[string]string)

		// Load projects
		if ps, err := projects.All(); err == nil {
			m.fields[0].options = []string{"all"}
			m.fields[0].ids = []string{""}
			for _, p := range ps {
				m.fields[0].options = append(m.fields[0].options, p.Name)
				m.fields[0].ids = append(m.fields[0].ids, fmt.Sprintf("%d", p.Id))
			}
		}

		// Load statuses
		if ss, err := status.All(); err == nil {
			m.fields[1].options = []string{"all"}
			m.fields[1].ids = []string{""}
			for _, s := range ss {
				m.fields[1].options = append(m.fields[1].options, s.Name)
				m.fields[1].ids = append(m.fields[1].ids, fmt.Sprintf("%d", s.Id))
				if s.Color != "" {
					statusColors[s.Name] = s.Color
				}
			}
		}

		// Load types
		if ts, err := resTypes.All(); err == nil {
			m.fields[2].options = []string{"all"}
			m.fields[2].ids = []string{""}
			for _, t := range ts {
				m.fields[2].options = append(m.fields[2].options, t.Name)
				m.fields[2].ids = append(m.fields[2].ids, fmt.Sprintf("%d", t.Id))
			}
		}

		// Load users for assignee
		if us, err := users.All(); err == nil {
			m.fields[3].options = []string{"all", "me"}
			m.fields[3].ids = []string{"", "me"}
			for _, u := range us {
				m.fields[3].options = append(m.fields[3].options, u.Name)
				m.fields[3].ids = append(m.fields[3].ids, fmt.Sprintf("%d", u.Id))
			}
		}

		return statusColorsLoadedMsg{colors: statusColors}
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
	case "c":
		if !m.showHelp {
			for i := range m.fields {
				m.fields[i].current = 0
				m.fields[i].selected = map[int]bool{}
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
	field := &m.fields[m.activeField]

	switch keyMsg.String() {
	case "up", "k":
		if m.popupIndex > 0 {
			m.popupIndex--
		}
	case "down", "j":
		if m.popupIndex < len(m.popupItems)-1 {
			m.popupIndex++
		}
	case " ":
		if field.multi {
			if m.popupIndex == 0 {
				// "all" selected → clear all
				field.selected = map[int]bool{}
			} else {
				delete(field.selected, 0) // remove "all" when selecting specific
				if field.selected[m.popupIndex] {
					delete(field.selected, m.popupIndex)
					if len(field.selected) == 0 {
						field.selected[0] = true // fall back to "all"
					}
				} else {
					field.selected[m.popupIndex] = true
				}
			}
		}
	case "enter":
		if field.multi {
			// Space handles multi-select; enter just confirms
			field.current = m.popupIndex
		} else {
			// Single-select: enter selects
			field.current = m.popupIndex
			if m.popupIndex == 0 {
				field.selected = map[int]bool{}
			} else {
				field.selected = map[int]bool{m.popupIndex: true}
			}
		}
		m.state = filterBrowseFields
		m.popupItems = nil
	case "esc", "q":
		m.state = filterBrowseFields
		m.popupItems = nil
	default:
		// Number shortcuts: 1-9 for quick toggle/select
		if len(keyMsg.String()) == 1 {
			ch := keyMsg.String()[0]
			if ch >= '1' && ch <= '9' {
				idx := int(ch - '1')
				if idx < len(m.popupItems) {
					if field.multi {
						m.popupIndex = idx
						if idx == 0 {
							field.selected = map[int]bool{}
						} else {
							delete(field.selected, 0)
							if field.selected[idx] {
								delete(field.selected, idx)
								if len(field.selected) == 0 {
									field.selected[0] = true
								}
							} else {
								field.selected[idx] = true
							}
						}
					} else {
						field.current = idx
						if idx == 0 {
							field.selected = map[int]bool{}
						} else {
							field.selected = map[int]bool{idx: true}
						}
						m.state = filterBrowseFields
						m.popupItems = nil
					}
				}
			}
		}
	}
	return nil
}

func (m *filterModel) popupView() string {
	var b strings.Builder
	b.WriteString("\n")
	field := m.fields[m.activeField]
	b.WriteString(titleStyle.Render(field.name))
	if field.multi {
		b.WriteString(subtitleStyle.Render(" (multi-select)"))
	}
	b.WriteString("\n\n")

	for i, item := range m.popupItems {
		cursor := "  "
		if i == m.popupIndex {
			cursor = "▸ "
		}

		checked := "  "
		if field.selected[i] {
			checked = "[●]"
		} else if !field.multi {
			if i == field.current {
				checked = "[●]"
			} else {
				checked = "[ ]"
			}
		} else {
			checked = "[ ]"
		}

		line := fmt.Sprintf("%s%s %s", cursor, checked, item)
		if i == m.popupIndex {
			b.WriteString(selectedItemStyle.Render(line))
		} else {
			b.WriteString(normalItemStyle.Render(line))
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")
	if field.multi {
		b.WriteString(helpStyle.Render("  j/k move  space toggle  enter confirm  1-9 quick toggle  esc cancel"))
	} else {
		b.WriteString(helpStyle.Render("  j/k move  enter select  1-9 quick pick  esc cancel"))
	}
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
		if field.multi {
			// Multi-select: collect all selected IDs
			var ids []string
			for idx := range field.selected {
				if idx > 0 && idx < len(field.ids) && field.ids[idx] != "" {
					ids = append(ids, field.ids[idx])
				}
			}
			if len(ids) > 0 {
				opts[keys[i]] = strings.Join(ids, ",")
			}
		} else {
			// Single-select
			if field.current > 0 && field.current < len(field.options) {
				if len(field.ids) > field.current && field.ids[field.current] != "" {
					opts[keys[i]] = field.ids[field.current]
				} else {
					opts[keys[i]] = field.options[field.current]
				}
			}
		}
	}
	return opts
}

func (m *filterModel) FilterDisplayNames() map[work_packages.FilterOption]string {
	names := make(map[work_packages.FilterOption]string)
	keys := []work_packages.FilterOption{
		work_packages.Project,
		work_packages.Status,
		work_packages.Type,
		work_packages.Assignee,
	}
	for i, field := range m.fields {
		display := m.selectedDisplay(&field)
		if display != "all" {
			names[keys[i]] = display
		}
	}
	return names
}

func (m *filterModel) selectedDisplay(field *filterField) string {
	if field.multi {
		if len(field.selected) == 0 || field.selected[0] {
			return "all"
		}
		var names []string
		for idx := range field.selected {
			if idx < len(field.options) {
				names = append(names, field.options[idx])
			}
		}
		return strings.Join(names, ", ")
	}
	return field.options[field.current]
}

func (m *filterModel) View() string {
	if m.showHelp {
		return helpOverlay("Filter — Key Bindings", [][2]string{
			{"j / k", "next / previous field"},
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

		value := m.selectedDisplay(&field)
		line := fmt.Sprintf("%s%-10s [%s]", prefix, field.name+":", value)

		if i == m.activeField {
			b.WriteString(selectedItemStyle.Render(line))
		} else {
			b.WriteString(normalItemStyle.Render(line))
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(helpStyle.Render("  j/k field  enter popup  a apply  esc cancel  c clear  ? help"))
	return b.String()
}

func (m *filterModel) toFilterState() configuration.FilterState {
	return configuration.FilterState{
		Project:  m.fields[0].options[m.fields[0].current],
		Status:   m.selectedNames(&m.fields[1]),
		Type:     m.selectedNames(&m.fields[2]),
		Assignee: m.selectedNames(&m.fields[3]),
	}
}

func (m *filterModel) selectedNames(field *filterField) string {
	if !field.multi || len(field.selected) == 0 || field.selected[0] {
		return field.options[field.current]
	}
	var names []string
	for idx := range field.selected {
		if idx < len(field.options) {
			names = append(names, field.options[idx])
		}
	}
	return strings.Join(names, ",")
}

func (m *filterModel) setFromState(fs configuration.FilterState) {
	// Project (single-select)
	m.setSingleField(0, fs.Project)

	// Status, Type, Assignee (multi-select)
	m.setMultiField(1, fs.Status)
	m.setMultiField(2, fs.Type)
	m.setMultiField(3, fs.Assignee)
}

func (m *filterModel) setSingleField(idx int, val string) {
	if val == "" {
		val = "all"
	}
	field := &m.fields[idx]
	for j, opt := range field.options {
		if opt == val {
			field.current = j
			if j > 0 {
				field.selected = map[int]bool{j: true}
			}
			return
		}
	}
}

func (m *filterModel) setMultiField(idx int, val string) {
	field := &m.fields[idx]
	if val == "" || val == "all" {
		field.current = 0
		field.selected = map[int]bool{}
		return
	}

	parts := strings.Split(val, ",")
	field.selected = map[int]bool{}
	for _, part := range parts {
		part = strings.TrimSpace(part)
		for j, opt := range field.options {
			if opt == part {
				field.selected[j] = true
				field.current = j // cursor to last selected
				break
			}
		}
	}
	if len(field.selected) == 0 {
		field.selected[0] = true
	}
}

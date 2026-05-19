package tui

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	runewidth "github.com/mattn/go-runewidth"
	"github.com/opf/openproject-cli/components/configuration"
	"github.com/opf/openproject-cli/components/launch"
	"github.com/opf/openproject-cli/components/requests"
	"github.com/opf/openproject-cli/components/resources/work_packages"
	"github.com/opf/openproject-cli/components/routes"
	"github.com/opf/openproject-cli/models"
)

type listModel struct {
	items         []*models.WorkPackage
	selected      int
	collection    *models.WorkPackageCollection
	page          int
	pageSize      int64
	total         int64
	width         int
	height        int
	loading       bool
	spinner       spinner.Model
	searchActive  bool
	searchInput   string
	filterOpts    map[work_packages.FilterOption]string
	sortField     sortField
	filter        *filterModel
	filterOverlay bool
	showHelp      bool
}

func newListModel() *listModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = selectedItemStyle

	fm := newFilterModel()
	savedFilters, _ := configuration.LoadFilters()
	fm.setFromState(savedFilters)

	return &listModel{
		loading:    true,
		pageSize:   50,
		page:       1,
		spinner:    s,
		filterOpts: fm.FilterOptions(),
		filter:     fm,
	}
}

func (m *listModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		m.loadWorkPackages,
	)
}

func (m *listModel) SetWorkPackages(collection *models.WorkPackageCollection) {
	m.collection = collection
	m.items = collection.Items
	m.total = collection.Total
	m.loading = false
	if m.selected >= len(m.items) {
		m.selected = len(m.items) - 1
	}
	if m.selected < 0 && len(m.items) > 0 {
		m.selected = 0
	}
	m.sortItems()
}

func (m *listModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

func (m *listModel) loadWorkPackages() tea.Msg {
	query := requests.NewPaginatedQuery(int(m.pageSize), nil)
	collection, err := work_packages.All(&m.filterOpts, query, false)
	return workPackagesLoadedMsg{collection: collection, err: err}
}

func (m *listModel) Update(msg tea.Msg) (*listModel, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	// Handle spinner tick
	if m.loading {
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)
	}

	switch msg := msg.(type) {
	case tea.MouseMsg:
		if msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonLeft {
			if !m.filterOverlay && !m.searchActive && !m.showHelp {
				row := m.mouseEventToRow(msg.Y)
				if row >= 0 && row < len(m.items) {
					m.selected = row
				}
			}
		}
	case tea.KeyMsg:
		// Filter overlay handling
		if m.filterOverlay {
			switch msg.String() {
			case "esc":
				if m.filter.showHelp {
					m.filter.showHelp = false
				} else {
					m.filterOverlay = false
				}
				return m, nil
			case "?":
				m.filter.Update(msg)
				return m, nil
			case "enter":
				if !m.filter.showHelp {
					m.filterOpts = m.filter.FilterOptions()
					configuration.SaveFilters(m.filter.toFilterState())
					m.filterOverlay = false
					m.loading = true
					m.page = 1
					return m, tea.Batch(m.spinner.Tick, m.loadWorkPackages)
				}
				return m, nil
			default:
				m.filter.Update(msg)
				return m, nil
			}
		}

		if m.searchActive {
			switch msg.String() {
			case "esc":
				m.searchActive = false
				m.searchInput = ""
				// Restore from collection
				if m.collection != nil {
					m.items = m.collection.Items
				}
				return m, nil
			case "enter":
				m.searchActive = false
				m.filterBySearch()
				return m, nil
			case "backspace":
				if len(m.searchInput) > 0 {
					m.searchInput = m.searchInput[:len(m.searchInput)-1]
				}
			default:
				if len(msg.String()) == 1 {
					m.searchInput += msg.String()
				}
			}
			return m, nil
		}

		switch msg.String() {
		case "?":
			m.showHelp = !m.showHelp
		case "up", "k":
			if m.selected > 0 {
				m.selected--
			}
		case "down", "j":
			if m.selected < len(m.items)-1 {
				m.selected++
			}
		case "enter":
			if m.selected >= 0 && m.selected < len(m.items) {
				return m, OpenDetailCmd(m.items[m.selected])
			}
		case "/":
			m.searchActive = true
			m.searchInput = ""
		case "n":
			if int64(m.page)*m.pageSize < m.total {
				m.page++
				m.loading = true
				return m, tea.Batch(m.spinner.Tick, m.loadWorkPackages)
			}
		case "p":
			if m.page > 1 {
				m.page--
				m.loading = true
				return m, tea.Batch(m.spinner.Tick, m.loadWorkPackages)
			}
		case "r":
			m.loading = true
			return m, tea.Batch(m.spinner.Tick, m.loadWorkPackages)
		case "s":
			m.cycleSort()
		case "f":
			m.filterOverlay = true
			return m, m.filter.loadOptions()
		case "o":
			if m.selected >= 0 && m.selected < len(m.items) {
				_ = launch.Browser(routes.WorkPackageUrl(m.items[m.selected]))
			}
		}
	}

	return m, tea.Batch(cmds...)
}

func (m *listModel) filterBySearch() {
	query := strings.ToLower(m.searchInput)
	if query == "" {
		if m.collection != nil {
			m.items = m.collection.Items
		}
		return
	}
	var filtered []*models.WorkPackage
	src := m.collection.Items
	if m.collection == nil {
		src = m.items
	}
	for _, wp := range src {
		if strings.Contains(strings.ToLower(wp.Subject), query) {
			filtered = append(filtered, wp)
		}
	}
	m.items = filtered
	m.selected = 0
}

// --- Sort ---

type sortField int

const (
	sortByID sortField = iota
	sortByStatus
	sortByType
	sortByAssignee
)

func (m *listModel) cycleSort() {
	m.sortField = (m.sortField + 1) % 4
	m.sortItems()
}

func (m *listModel) sortItems() {
	if len(m.items) == 0 {
		return
	}
	sort.SliceStable(m.items, func(i, j int) bool {
		switch m.sortField {
		case sortByStatus:
			return m.items[i].Status < m.items[j].Status
		case sortByType:
			return m.items[i].Type < m.items[j].Type
		case sortByAssignee:
			return m.items[i].Assignee < m.items[j].Assignee
		default:
			return m.items[i].Id < m.items[j].Id
		}
	})
}

func (m *listModel) View() string {
	if m.showHelp {
		return helpOverlay("List — Key Bindings", [][2]string{
			{"↑ / k", "move up"},
			{"↓ / j", "move down"},
			{"enter", "open detail view"},
			{"/", "search (client-side)"},
			{"f", "open filter panel"},
			{"s", "cycle sort (ID → Status → Type → Assignee)"},
			{"n", "next page"},
			{"p", "previous page"},
			{"r", "refresh"},
			{"o", "open in browser"},
			{"?", "toggle this help"},
			{"q", "quit"},
		}, m.width)
	}

	if m.filterOverlay {
		return m.filter.View()
	}

	var b strings.Builder

	// Header
	b.WriteString(titleStyle.Render("Work Packages"))
	if m.total > 0 {
		b.WriteString(subtitleStyle.Render(fmt.Sprintf(" (%d total)", m.total)))
	}
	sortNames := []string{"ID", "Status", "Type", "Assignee"}
	b.WriteString(subtitleStyle.Render(fmt.Sprintf("  sort:%s", sortNames[m.sortField])))
	b.WriteString("\n")

	// Active filters
	if len(m.filterOpts) > 0 {
		var parts []string
		keys := []work_packages.FilterOption{work_packages.Project, work_packages.Status, work_packages.Type, work_packages.Assignee}
		for _, k := range keys {
			if v, ok := m.filterOpts[k]; ok {
				parts = append(parts, fmt.Sprintf("%s=%s", k, v))
			}
		}
		b.WriteString(subtitleStyle.Render("  Filter: " + strings.Join(parts, "  ")))
		b.WriteString("\n")
	}

	b.WriteString("\n")

	if m.loading {
		b.WriteString(fmt.Sprintf("\n  %s Loading...\n\n", m.spinner.View()))
		return b.String()
	}

	if len(m.items) == 0 {
		b.WriteString("\n  No work packages found.\n\n")
		return b.String()
	}

	// Column widths
	idWidth := len(strconv.FormatInt(m.total, 10)) + 2
	if idWidth < 6 {
		idWidth = 6
	}
	typeWidth := 12
	statusWidth := 12
	assigneeWidth := 14
	titleWidth := m.width - idWidth - typeWidth - statusWidth - assigneeWidth - 12
	if titleWidth < 20 {
		titleWidth = 20
	}

		// Table header
		headerLine := padRight("ID", idWidth) + " " +
			padRight("Type", typeWidth) + " " +
			padRight("Title", titleWidth) + " " +
			padRight("Status", statusWidth) + " " +
			padRight("Assignee", assigneeWidth)
		b.WriteString(headerStyle.Render(headerLine))
		b.WriteString("\n")
	b.WriteString("\n")

	// Items
	for i, wp := range m.items {
		idStr := fmt.Sprintf("#%d", wp.Id)
		line := padRight(idStr, idWidth) + " " +
			padRight(truncate(wp.Type, typeWidth), typeWidth) + " " +
			padRight(truncate(wp.Subject, titleWidth), titleWidth) + " " +
			padRight(truncate(wp.Status, statusWidth), statusWidth) + " " +
			padRight(truncate(assigneeOrDash(wp.Assignee), assigneeWidth), assigneeWidth)

		if i == m.selected {
			b.WriteString(selectedItemStyle.Render(line))
		} else {
			b.WriteString(normalItemStyle.Render(line))
		}
		b.WriteString("\n")
	}

	// Footer
	b.WriteString("\n")
	from := int64(m.page-1)*m.pageSize + 1
	to := int64(m.page) * m.pageSize
	if to > m.total {
		to = m.total
	}
	pageInfo := fmt.Sprintf("%d-%d / %d", from, to, m.total)
	b.WriteString(helpStyle.Render(pageInfo))

	// Search bar
	if m.searchActive {
		b.WriteString("\n\n")
		b.WriteString("/" + m.searchInput)
		b.WriteString("_")
	}

	// Help bar
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("  ↑↓ move  enter select  / search  f filter  s sort  n/p page  r refresh  ? help  q quit"))

	return b.String()
}

func truncate(s string, maxDisplayWidth int) string {
	if runewidth.StringWidth(s) <= maxDisplayWidth {
		return s
	}
	if maxDisplayWidth <= 1 {
		return runewidth.Truncate(s, maxDisplayWidth, "")
	}
	return runewidth.Truncate(s, maxDisplayWidth, "…")
}

func padRight(s string, width int) string {
	w := runewidth.StringWidth(s)
	if w >= width {
		return runewidth.Truncate(s, width, "")
	}
	return s + strings.Repeat(" ", width-w)
}

func assigneeOrDash(s string) string {
	if s == "" {
		return "—"
	}
	return s
}

func (m *listModel) mouseEventToRow(y int) int {
	// docStyle top padding: 1 line
	// title line: 1 line
	// filter bar: 0 or 1 line
	// blank line: 1 line
	// table header: 1 line
	offset := 4 // top pad + title + blank + header
	if len(m.filterOpts) > 0 {
		offset++ // filter bar
	}
	row := y - offset
	if row < 0 || row >= len(m.items) {
		return -1
	}
	return row
}

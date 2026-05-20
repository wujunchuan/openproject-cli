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
	"github.com/opf/openproject-cli/components/resources/status"
	"github.com/opf/openproject-cli/components/resources/work_packages"
	"github.com/opf/openproject-cli/components/routes"
	"github.com/opf/openproject-cli/models"
)

type listModel struct {
	items              []*models.WorkPackage
	selected           int
	scrollOffset       int
	collection         *models.WorkPackageCollection
	page               int
	pageSize           int64
	total              int64
	width              int
	height             int
	loading            bool
	spinner            spinner.Model
	searchActive       bool
	searchInput        string
	filterOpts         map[work_packages.FilterOption]string
	filterDisplayNames map[work_packages.FilterOption]string
	sortField          sortField
	filter             *filterModel
	filterOverlay      bool
	showHelp           bool
	statusColors       map[string]string
	treeMode           bool
	treeRoots          []*treeNode
	flatNodes          []*treeNode
}

func newListModel() *listModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = selectedItemStyle

	fm := newFilterModel()
	savedFilters, _ := configuration.LoadFilters()
	fm.setFromState(savedFilters)

	return &listModel{
		loading:            true,
		pageSize:           50,
		page:               1,
		spinner:            s,
		filterOpts:         fm.FilterOptions(),
		filterDisplayNames: fm.FilterDisplayNames(),
		filter:             fm,
	}
}

func (m *listModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		m.loadWorkPackages,
		m.loadStatusColors,
	)
}

func (m *listModel) SetWorkPackages(collection *models.WorkPackageCollection) {
	m.collection = collection
	m.items = collection.Items
	m.total = collection.Total
	m.loading = false
	m.scrollOffset = 0
	if m.selected >= len(m.items) {
		m.selected = len(m.items) - 1
	}
	if m.selected < 0 && len(m.items) > 0 {
		m.selected = 0
	}
	m.sortItems()
	if m.treeMode {
		m.buildTreeFromItems()
	}
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

func (m *listModel) loadStatusColors() tea.Msg {
	colors := make(map[string]string)
	if ss, err := status.All(); err == nil {
		for _, s := range ss {
			if s.Color != "" {
				colors[s.Name] = s.Color
			}
		}
	}
	return statusColorsLoadedMsg{colors: colors}
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
				if m.treeMode {
					if row >= 0 && row < len(m.flatNodes) {
						m.selected = row
					}
				} else {
					if row >= 0 && row < len(m.items) {
						m.selected = row
					}
				}
			}
		}
	case filterApplyMsg:
		m.filterOpts = m.filter.FilterOptions()
		m.filterDisplayNames = m.filter.FilterDisplayNames()
		configuration.SaveFilters(m.filter.toFilterState())
		m.filterOverlay = false
		m.loading = true
		m.page = 1
		return m, tea.Batch(m.spinner.Tick, m.loadWorkPackages)
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
				cmd := m.filter.Update(msg)
				return m, cmd
			case "enter":
				cmd := m.filter.Update(msg)
				return m, cmd
			default:
				cmd := m.filter.Update(msg)
				return m, cmd
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

		// Tree mode keys (active when not in search/filter/help)
		if !m.filterOverlay && !m.searchActive && !m.showHelp {
			if msg.String() == "t" {
				m.treeMode = !m.treeMode
				if m.treeMode {
					m.buildTreeFromItems()
				} else {
					// Sync selected back to items index
					if m.selected >= 0 && m.selected < len(m.flatNodes) {
						wp := m.flatNodes[m.selected].item
						for i, item := range m.items {
							if item.Id == wp.Id {
								m.selected = i
								break
							}
						}
					}
				}
				return m, nil
			}

			if m.treeMode {
				switch msg.String() {
				case "right", ">":
					if m.selected >= 0 && m.selected < len(m.flatNodes) {
						node := m.flatNodes[m.selected]
						if node.hasChildren() && !node.expanded {
							node.expanded = true
							m.flatNodes = flatten(m.treeRoots)
						}
					}
					return m, nil
				case "left", "<":
					if m.selected >= 0 && m.selected < len(m.flatNodes) {
						node := m.flatNodes[m.selected]
						if node.hasChildren() && node.expanded {
							node.expanded = false
							m.flatNodes = flatten(m.treeRoots)
							if m.selected >= len(m.flatNodes) {
								m.selected = len(m.flatNodes) - 1
							}
						}
					}
					return m, nil
				case "enter", "l":
					if m.selected >= 0 && m.selected < len(m.flatNodes) {
						return m, OpenDetailCmd(m.flatNodes[m.selected].item)
					}
					return m, nil
				case "up", "k":
					if m.selected > 0 {
						m.selected--
						m.ensureVisible()
					}
					return m, nil
				case "down", "j":
					if m.selected < len(m.flatNodes)-1 {
						m.selected++
						m.ensureVisible()
					}
					return m, nil
				case "s":
					// Sort in tree mode: cycle sort, sort items, then rebuild tree.
					// Sorting applies within sibling groups (parent-child preserved).
					m.cycleSort()
					m.buildTreeFromItems()
					return m, nil
				case "r":
					m.loading = true
					return m, tea.Batch(m.spinner.Tick, m.loadWorkPackages)
				}
			}
		}

		switch msg.String() {
		case "?":
			m.showHelp = !m.showHelp
		case "up", "k":
			if m.selected > 0 {
				m.selected--
				m.ensureVisible()
			}
		case "down", "j":
			if m.selected < len(m.items)-1 {
				m.selected++
				m.ensureVisible()
			}
		case "enter", "l":
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
			if m.treeMode {
				if m.selected >= 0 && m.selected < len(m.flatNodes) {
					_ = launch.Browser(routes.WorkPackageUrl(m.flatNodes[m.selected].item))
				}
			} else if m.selected >= 0 && m.selected < len(m.items) {
				_ = launch.Browser(routes.WorkPackageUrl(m.items[m.selected]))
			}
		case "c":
			if !m.searchActive && !m.filterOverlay {
				var selected *models.WorkPackage
				if m.treeMode {
					if m.selected >= 0 && m.selected < len(m.flatNodes) {
						selected = m.flatNodes[m.selected].item
					}
				} else if m.selected >= 0 && m.selected < len(m.items) {
					selected = m.items[m.selected]
				}
				if selected != nil {
					return m, func() tea.Msg {
						return copyIdMsg{id: selected.Id}
					}
				}
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
		m.scrollOffset = 0
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
	m.scrollOffset = 0
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

func (m *listModel) buildTreeFromItems() {
	cmp := m.treeSortFunc()
	m.treeRoots = buildTree(m.items, cmp)
	m.flatNodes = flatten(m.treeRoots)
	m.scrollOffset = 0
	if m.selected >= len(m.flatNodes) {
		m.selected = len(m.flatNodes) - 1
	}
	if m.selected < 0 && len(m.flatNodes) > 0 {
		m.selected = 0
	}
}

func (m *listModel) treeSortFunc() sortFunc {
	switch m.sortField {
	case sortByStatus:
		return func(a, b *treeNode) bool { return a.item.Status < b.item.Status }
	case sortByType:
		return func(a, b *treeNode) bool { return a.item.Type < b.item.Type }
	case sortByAssignee:
		return func(a, b *treeNode) bool { return a.item.Assignee < b.item.Assignee }
	default:
		return byID
	}
}

func (m *listModel) View() string {
	if m.showHelp {
		bindings := [][2]string{
			{"j / k", "move selection"},
			{"enter / l", "open detail"},
			{"c", "copy ID to clipboard"},
			{"/", "search"},
			{"f", "filter"},
			{"s", "cycle sort"},
			{"r", "refresh"},
			{"?", "toggle this help"},
			{"q", "quit"},
		}
		if m.treeMode {
			bindings = append([][2]string{
				{"t", "toggle tree/list mode"},
				{"enter / l", "open detail"},
				{"> / →", "expand node"},
				{"< / ←", "collapse node"},
			}, bindings...)
		} else {
			bindings = append([][2]string{
				{"t", "toggle tree/list mode"},
			}, bindings...)
		}
		return helpOverlay("List — Key Bindings", bindings, m.width)
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

	// Tree mode rendering
	if m.treeMode {
		return m.treeView()
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

	// Items (viewport: only render visible rows)
	v := m.visibleItemRows()
	end := m.scrollOffset + v
	if end > len(m.items) {
		end = len(m.items)
	}
	for i := m.scrollOffset; i < end; i++ {
		wp := m.items[i]
		idStr := fmt.Sprintf("#%d", wp.Id)
		colID := padRight(idStr, idWidth)
		colType := padRight(truncate(wp.Type, typeWidth), typeWidth)
		colTitle := padRight(truncate(wp.Subject, titleWidth), titleWidth)
		colStatus := padRight(truncate(wp.Status, statusWidth), statusWidth)
		colAssignee := padRight(truncate(assigneeOrDash(wp.Assignee), assigneeWidth), assigneeWidth)

		if i == m.selected {
			b.WriteString(selectedItemStyle.Render(colID) + " " +
				selectedItemStyle.Render(colType) + " " +
				selectedItemStyle.Render(colTitle) + " " +
				selectedItemStyle.Render(colStatus) + " " +
				selectedItemStyle.Render(colAssignee))
		} else {
			if hex, ok := m.statusColors[wp.Status]; ok {
				colStatus = statusColorStyle(hex).Render(colStatus)
			}
			b.WriteString(normalItemStyle.Render(colID) + " " +
				normalItemStyle.Render(colType) + " " +
				normalItemStyle.Render(colTitle) + " " +
				colStatus + " " +
				normalItemStyle.Render(colAssignee))
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
	keys := []string{"j/k", "move", "enter/l", "open", "c", "copy", "/", "search", "f", "filter", "?", "help", "q", "quit"}
	b.WriteString(helpStyle.Render("  " + strings.Join(keys, " ")))

	return b.String()
}

func (m *listModel) treeView() string {
	var b strings.Builder

	// Header
	b.WriteString(titleStyle.Render("Work Packages"))
	if m.total > 0 {
		b.WriteString(subtitleStyle.Render(fmt.Sprintf(" (%d total)", m.total)))
	}
	b.WriteString(subtitleStyle.Render("  [tree]"))
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

	if len(m.flatNodes) == 0 {
		b.WriteString("\n  No work packages found.\n\n")
		return b.String()
	}

	// Column widths (account for indent)
	maxDepth := 0
	for _, n := range m.flatNodes {
		if n.depth > maxDepth {
			maxDepth = n.depth
		}
	}
	indentWidth := maxDepth*4 + 2 // 4 chars per depth level + margin

	idWidth := len(strconv.FormatInt(m.total, 10)) + 2
	if idWidth < 6 {
		idWidth = 6
	}
	typeWidth := 12
	statusWidth := 12
	assigneeWidth := 14
	titleWidth := m.width - idWidth - typeWidth - statusWidth - assigneeWidth - indentWidth - 12
	if titleWidth < 20 {
		titleWidth = 20
	}

	// Table header
	headerLine := strings.Repeat(" ", indentWidth) +
		padRight("ID", idWidth) + " " +
		padRight("Type", typeWidth) + " " +
		padRight("Title", titleWidth) + " " +
		padRight("Status", statusWidth) + " " +
		padRight("Assignee", assigneeWidth)
	b.WriteString(headerStyle.Render(headerLine))
	b.WriteString("\n\n")

	// Render each visible node (viewport clipping)
	v := m.visibleItemRows()
	start := m.scrollOffset
	end := m.scrollOffset + v
	if end > len(m.flatNodes) {
		end = len(m.flatNodes)
	}
	for i := start; i < end; i++ {
		node := m.flatNodes[i]
		// Determine if this node is the last child of its parent
		isLast := true
		for j := i + 1; j < len(m.flatNodes); j++ {
			if m.flatNodes[j].depth == node.depth {
				isLast = false
				break
			}
			if m.flatNodes[j].depth < node.depth {
				break
			}
		}

		ancLast := computeAncestorsLast(m.flatNodes, i)
		prefix := treeLinePrefix(node.depth, isLast, ancLast)

		// Expand/collapse marker
		marker := "  "
		if node.hasChildren() {
			if node.expanded {
				marker = "▼ "
			} else {
				marker = "▶ "
			}
		}

		wp := node.item
		idStr := fmt.Sprintf("#%d", wp.Id)
		colID := padRight(idStr, idWidth-2)
		colType := padRight(truncate(wp.Type, typeWidth), typeWidth)
		colTitle := padRight(truncate(wp.Subject, titleWidth), titleWidth)
		colStatus := padRight(truncate(wp.Status, statusWidth), statusWidth)
		colAssignee := padRight(truncate(assigneeOrDash(wp.Assignee), assigneeWidth), assigneeWidth)

		if i == m.selected {
			fullLine := marker + selectedItemStyle.Render(colID) + " " +
				selectedItemStyle.Render(colType) + " " +
				selectedItemStyle.Render(colTitle) + " " +
				selectedItemStyle.Render(colStatus) + " " +
				selectedItemStyle.Render(colAssignee)
			b.WriteString(selectedItemStyle.Render(prefix) + fullLine)
		} else {
			if hex, ok := m.statusColors[wp.Status]; ok {
				colStatus = statusColorStyle(hex).Render(colStatus)
			}
			fullLine := marker + normalItemStyle.Render(colID) + " " +
				normalItemStyle.Render(colType) + " " +
				normalItemStyle.Render(colTitle) + " " +
				colStatus + " " +
				normalItemStyle.Render(colAssignee)
			b.WriteString(normalItemStyle.Render(prefix) + fullLine)
		}
		b.WriteString("\n")
	}

	// Footer
	b.WriteString("\n")
	pageInfo := fmt.Sprintf("%d-%d / %d",
		int64(m.page-1)*m.pageSize+1,
		min(int64(m.page)*m.pageSize, m.total),
		m.total)
	b.WriteString(helpStyle.Render(pageInfo))

	// Help bar
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("  ↑↓ move  enter/l detail  >/< expand/collapse  t list  / search  f filter  n/p page  ? help  q quit"))

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
	offset := m.headerLines()
	row := y - offset + m.scrollOffset
	itemCount := len(m.items)
	if m.treeMode {
		itemCount = len(m.flatNodes)
	}
	if row < 0 || row >= itemCount {
		return -1
	}
	return row
}

// headerLines returns the number of lines before the first item row.
func (m *listModel) headerLines() int {
	lines := 1 // title
	if len(m.filterOpts) > 0 {
		lines++ // filter bar
	}
	lines += 1 // blank line
	lines += 2 // table header + bottom border
	return lines
}

// footerLines returns the number of lines after the last item row.
func (m *listModel) footerLines() int {
	lines := 1 // blank line before footer
	lines += 1 // page info
	lines += 1 // help bar
	if m.searchActive {
		lines += 2 // search bar
	}
	return lines
}

// visibleItemRows returns how many item rows fit in the terminal.
func (m *listModel) visibleItemRows() int {
	// docStyle top+bottom padding = 2
	v := m.height - 2 - m.headerLines() - m.footerLines()
	if v < 1 {
		return 1
	}
	return v
}

// ensureVisible adjusts scrollOffset so the selected item is visible.
func (m *listModel) ensureVisible() {
	v := m.visibleItemRows()
	if m.selected < m.scrollOffset {
		m.scrollOffset = m.selected
	} else if m.selected >= m.scrollOffset+v {
		m.scrollOffset = m.selected - v + 1
	}
}

type copyIdMsg struct {
	id uint64
}

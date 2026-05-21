package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/opf/openproject-cli/components/configuration"
	"github.com/opf/openproject-cli/components/launch"
	"github.com/opf/openproject-cli/components/resources/work_packages"
	"github.com/opf/openproject-cli/components/routes"
	"github.com/opf/openproject-cli/models"
)

type detailModel struct {
	wp            *models.WorkPackage
	activities    []*models.Activity
	viewport      viewport.Model
	width         int
	height        int
	loading       bool
	editOverlay   bool
	edit          *editModel
	showHelp      bool
	familyRoots   []*treeNode
	familyFlat    []*treeNode
	familyCursor  int  // selected node index in familyFlat (-1 = none)
	familyFocused bool // whether the children tree has focus
}

func newDetailModel(wp *models.WorkPackage, w, h int) *detailModel {
	vp := viewport.New(w-4, h-10)
	vp.SetContent("")
	return &detailModel{
		wp:           wp,
		viewport:     vp,
		width:        w,
		height:       h,
		loading:      true,
		familyCursor: -1,
	}
}

func (m *detailModel) Init() tea.Cmd {
	return tea.Batch(m.loadActivities, m.loadFamilyTree)
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

func (m *detailModel) loadFamilyTree() tea.Msg {
	if m.wp.ParentId != 0 {
		// Current WP has a parent — fetch parent + all siblings
		parent, err := work_packages.Lookup(m.wp.ParentId)
		if err != nil {
			return familyTreeLoadedMsg{}
		}
		siblings, err := work_packages.Children(m.wp.ParentId)
		if err != nil {
			return familyTreeLoadedMsg{}
		}
		return familyTreeLoadedMsg{parent: parent, children: siblings}
	}

	// No parent — fetch children of current WP
	children, err := work_packages.Children(m.wp.Id)
	if err != nil {
		return familyTreeLoadedMsg{}
	}
	return familyTreeLoadedMsg{parent: m.wp, children: children}
}

func (m *detailModel) SetFamilyTree(parent *models.WorkPackage, children []*models.WorkPackage) {
	m.familyRoots = nil
	m.familyCursor = -1
	if parent == nil || len(children) == 0 {
		m.familyFlat = nil
		m.updateContent()
		return
	}

	root := &treeNode{
		item:     parent,
		expanded: true,
		depth:    0,
	}
	sort.Slice(children, func(i, j int) bool { return children[i].Id < children[j].Id })
	for _, child := range children {
		root.children = append(root.children, &treeNode{
			item:     child,
			expanded: true,
			depth:    1,
		})
	}
	m.familyRoots = []*treeNode{root}
	m.familyFlat = flatten(m.familyRoots)

	// Position cursor on current WP if found, otherwise first child
	for i, node := range m.familyFlat {
		if node.item.Id == m.wp.Id && node.depth > 0 {
			m.familyCursor = i
			break
		}
	}
	if m.familyCursor < 0 && len(m.familyFlat) > 1 {
		m.familyCursor = 1
	}
	m.updateContent()
}

func (m *detailModel) updateContent() {
	var b strings.Builder

	// Header
	b.WriteString(titleStyle.Render(fmt.Sprintf("#%d %s", m.wp.Id, m.wp.Subject)))
	b.WriteString("\n\n")

	// Properties (two columns)
	statusStr := m.wp.Status
	if m.wp.StatusColor != "" {
		statusStr = statusColorStyle(m.wp.StatusColor).Render(m.wp.Status)
	}
	left := fmt.Sprintf("Type: %s\nStatus: %s\nProject: %s\nAssignee: %s",
		m.wp.Type, statusStr, m.wp.Project, assigneeOrDash(m.wp.Assignee))
	right := fmt.Sprintf("Priority: %s\nVersion: %s\nStart: %s\nDue: %s\nCreated: %s\nUpdated: %s",
		m.wp.Priority, m.wp.Version,
		dateOrDash(m.wp.StartDate), dateOrDash(m.wp.DueDate),
		configuration.FormatTime(m.wp.CreatedAt), configuration.FormatTime(m.wp.UpdatedAt))

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

	// Children (tree)
	if len(m.familyRoots) > 0 {
		childrenLabel := "Children"
		if m.familyFocused {
			childrenLabel = "Children [focused]"
		}
		b.WriteString(headerStyle.Render(childrenLabel))
		b.WriteString("\n")
		for i, node := range m.familyFlat {
			ancestorsLast := computeAncestorsLast(m.familyFlat, i)
			isLast := true
			if i+1 < len(m.familyFlat) {
				for j := i + 1; j < len(m.familyFlat); j++ {
					if m.familyFlat[j].depth == node.depth {
						isLast = false
						break
					}
					if m.familyFlat[j].depth < node.depth {
						break
					}
				}
			}
			prefix := treeLinePrefix(node.depth, isLast, ancestorsLast)

			var line string
			if node.hasChildren() {
				line = fmt.Sprintf("%s▼ #%d %s", prefix, node.item.Id, node.item.Subject)
			} else {
				statusStr := node.item.Status
				if node.item.StatusColor != "" {
					statusStr = statusColorStyle(node.item.StatusColor).Render(node.item.Status)
				}
				marker := ""
				if node.item.Id == m.wp.Id {
					marker = " ←"
				}
				line = fmt.Sprintf("%s#%d %s [%s]%s", prefix, node.item.Id, node.item.Subject, statusStr, marker)
			}

			if i == m.familyCursor {
				b.WriteString(selectedItemStyle.Render(line))
			} else {
				b.WriteString(line)
			}
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	// Activities
	b.WriteString(headerStyle.Render(fmt.Sprintf("Activity (%d)", len(m.activities))))
	b.WriteString("\n")
	if m.loading {
		b.WriteString("\n  Loading activities...\n")
	} else {
		for _, act := range m.activities {
			b.WriteString(fmt.Sprintf("\n  %s\n", subtitleStyle.Render(configuration.FormatTime(act.CreatedAt))))
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
					m.loadFamilyTree,
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
		case "c":
			return m, func() tea.Msg {
				return copyIdMsg{id: m.wp.Id}
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
				m.loadFamilyTree,
			)
		case "tab":
			if len(m.familyFlat) > 0 {
				m.familyFocused = !m.familyFocused
				m.updateContent()
				if m.familyFocused {
					m.scrollFamilyCursor()
				}
				return m, nil
			}
		case "j", "down":
			if m.familyFocused && m.familyCursor >= 0 && m.familyCursor < len(m.familyFlat)-1 {
				m.familyCursor++
				m.updateContent()
				m.scrollFamilyCursor()
				return m, nil
			}
		case "k", "up":
			if m.familyFocused && m.familyCursor > 0 {
				m.familyCursor--
				m.updateContent()
				m.scrollFamilyCursor()
				return m, nil
			}
		case "l", "enter":
			if m.familyFocused && m.familyCursor >= 0 && m.familyCursor < len(m.familyFlat) {
				node := m.familyFlat[m.familyCursor]
				if !node.hasChildren() && node.item.Id != m.wp.Id {
					return m, OpenDetailCmd(node.item)
				}
			}
		}
	}

	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

// scrollFamilyCursor scrolls the viewport so the selected family tree node is visible.
func (m *detailModel) scrollFamilyCursor() {
	if m.familyCursor < 0 || m.familyCursor >= len(m.familyFlat) {
		return
	}

	// Count lines before the Children section
	linesBefore := 2 // header + blank
	// Properties: max(leftLines, rightLines) lines
	leftLines := strings.Count("Type: \nStatus: \nProject: \nAssignee: ", "\n") + 1
	rightLines := strings.Count("Priority: \nVersion: \nStart: \nDue: \nCreated: \nUpdated: ", "\n") + 1
	if leftLines > rightLines {
		linesBefore += leftLines
	} else {
		linesBefore += rightLines
	}
	linesBefore++ // blank after properties
	if m.wp.Description != "" {
		linesBefore += 2 + strings.Count(m.wp.Description, "\n") + 1 + 1
	}
	linesBefore++ // "Children" header

	targetLine := linesBefore + m.familyCursor
	visibleLines := m.viewport.VisibleLineCount()
	currentTop := m.viewport.YOffset

	if targetLine < currentTop {
		m.viewport.SetYOffset(targetLine)
	} else if targetLine >= currentTop+visibleLines {
		m.viewport.SetYOffset(targetLine - visibleLines + 1)
	}
}

func (m *detailModel) View() string {
	if m.showHelp {
		bindings := [][2]string{
			{"tab", "focus / unfocus children tree"},
			{"j / k", "select child item (when focused)"},
			{"l / enter", "open selected item (when focused)"},
			{"↑ / ↓", "scroll content"},
			{"PgUp / PgDn", "scroll page"},
			{"esc", "back to list"},
			{"c", "copy ID to clipboard"},
			{"e", "edit (type, status, dates)"},
			{"o", "open in browser"},
			{"r", "refresh"},
			{"?", "toggle this help"},
		}
		return helpOverlay("Detail — Key Bindings", bindings, m.width)
	}
	if m.editOverlay {
		return m.edit.View()
	}
	footer := helpStyle.Render("  tab focus tree  esc back  c copy  e edit  o browser  r refresh  ? help")
	return m.viewport.View() + "\n" + footer
}

func dateOrDash(s string) string {
	if s == "" {
		return "—"
	}
	return s
}

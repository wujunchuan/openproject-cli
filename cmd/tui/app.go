package tui

import (
	"fmt"
	"time"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/opf/openproject-cli/components/configuration"
	"github.com/opf/openproject-cli/models"
)

type viewState int

const (
	listView viewState = iota
	detailView
)

// --- Messages ---

type workPackagesLoadedMsg struct {
	collection *models.WorkPackageCollection
	err        error
}

type workPackageDetailMsg struct {
	wp  *models.WorkPackage
	err error
}

type activitiesLoadedMsg struct {
	activities []*models.Activity
	err        error
}

type errorMsg struct {
	err error
}

type copyConfirmMsg struct{}
type copyClearMsg struct{}

// --- App Model ---

type App struct {
	state        viewState
	list         *listModel
	detail       *detailModel
	width        int
	height       int
	err          error
	quitting     bool
	ctrlCPressed bool
	ctrlCTime    time.Time
	copyConfirm  bool
}

func NewApp() App {
	return App{
		state: listView,
		list:  newListModel(),
	}
}

func (a App) Init() tea.Cmd {
	return a.list.Init()
}

func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a.list.SetSize(msg.Width, msg.Height)

	case tea.KeyMsg:
		switch msg.String() {
		case "q":
			if a.state == listView && !a.list.searchActive && !a.list.filterOverlay {
				a.quitting = true
				return a, tea.Quit
			}
		case "ctrl+c":
			a.quitting = true
			return a, tea.Quit
		}


		case tea.MouseMsg:
			switch a.state {
			case listView:
				a.list, cmd = a.list.Update(msg)
				cmds = append(cmds, cmd)
			}
			return a, tea.Batch(cmds...)
	case workPackagesLoadedMsg:
		if msg.err != nil {
			a.err = msg.err
		} else {
			a.err = nil
			a.list.SetWorkPackages(msg.collection)
		}
		return a, nil

	case workPackageDetailMsg:
		if msg.err != nil {
			a.err = msg.err
		} else {
			a.err = nil
			if a.detail != nil {
				a.detail.SetWorkPackage(msg.wp)
			}
		}
		return a, nil

	case activitiesLoadedMsg:
		if msg.err != nil {
			a.err = msg.err
		} else {
			if a.detail != nil {
				a.detail.SetActivities(msg.activities)
			}
		}
		return a, nil

	case openDetailMsg:
		a.state = detailView
		a.detail = newDetailModel(msg.wp, a.width, a.height)
		return a, a.detail.Init()

	case backToListMsg:
		a.state = listView
		a.detail = nil
		return a, nil

	case copyIdMsg:
		prefix := configuration.CopyPrefix()
		text := fmt.Sprintf("%s#%d", prefix, msg.id)
		if err := clipboard.WriteAll(text); err == nil {
			return a, func() tea.Msg { return copyConfirmMsg{} }
		}
		return a, nil

	case copyConfirmMsg:
		a.copyConfirm = true
		return a, tea.Tick(1500*time.Millisecond, func(t time.Time) tea.Msg {
			return copyClearMsg{}
		})

	case copyClearMsg:
		a.copyConfirm = false
		return a, nil
	}

	switch a.state {
	case listView:
		a.list, cmd = a.list.Update(msg)
		cmds = append(cmds, cmd)
	case detailView:
		a.detail, cmd = a.detail.Update(msg)
		cmds = append(cmds, cmd)
	}

	return a, tea.Batch(cmds...)
}

func (a App) View() string {
	if a.quitting {
		return ""
	}

	var content string
	switch a.state {
	case listView:
		content = a.list.View()
	case detailView:
		content = a.detail.View()
	}

	if a.err != nil {
		content += "\n" + errorStyle.Render(fmt.Sprintf("Error: %v", a.err)) + "\n"
		content += helpStyle.Render("  r to retry")
	}

	if a.copyConfirm {
		content += "\n" + helpStyle.Render("  copied!")
	}

	return docStyle.Render(content)
}

// --- Navigation Messages ---

type openDetailMsg struct {
	wp *models.WorkPackage
}

type backToListMsg struct{}

func OpenDetailCmd(wp *models.WorkPackage) tea.Cmd {
	return func() tea.Msg {
		return openDetailMsg{wp: wp}
	}
}

func BackToListCmd() tea.Cmd {
	return func() tea.Msg {
		return backToListMsg{}
	}
}


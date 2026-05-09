package tui

import (
	"context"
	"fmt"
	"strings"

	appcore "aeswibon.com/github/gitopsctl/internal/core/app"
	clustercore "aeswibon.com/github/gitopsctl/internal/core/cluster"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type viewState int

const (
	appsView viewState = iota
	clustersView
)

type item struct {
	title, desc string
	status      string
}

func (i item) Title() string       { return i.title }
func (i item) Description() string { return i.desc }
func (i item) FilterValue() string { return i.title }

type Model struct {
	state         viewState
	appList       list.Model
	clusterList   list.Model
	spinner       spinner.Model
	width, height int
	loading       bool
	err           error
	client        *apiClient
	ctx           context.Context
	cancel        context.CancelFunc
	confirmMsg    string
	confirmAction func()
}

func NewModel(apiURL string) Model {
	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, false, false, true).
		BorderForeground(accentColor).
		Foreground(accentColor).
		Padding(0, 0, 0, 1)

	appL := list.New([]list.Item{}, delegate, 0, 0)
	appL.SetShowTitle(false)
	appL.SetShowStatusBar(false)
	appL.KeyMap.Quit.Unbind()

	clusterL := list.New([]list.Item{}, delegate, 0, 0)
	clusterL.SetShowTitle(false)
	clusterL.SetShowStatusBar(false)
	clusterL.KeyMap.Quit.Unbind()

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(primaryColor)

	ctx, cancel := context.WithCancel(context.Background())

	return Model{
		state:       appsView,
		appList:     appL,
		clusterList: clusterL,
		spinner:     s,
		loading:     true,
		client:      newAPIClient(apiURL),
		ctx:         ctx,
		cancel:      cancel,
	}
}

type (
	appsLoadedMsg     []appcore.Application
	clustersLoadedMsg []clustercore.Cluster
	errorMsg          error
)

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.fetchApps(),
		m.fetchClusters(),
		m.client.listenForEvents(m.ctx),
		m.spinner.Tick,
	)
}

func (m Model) fetchApps() tea.Cmd {
	return func() tea.Msg {
		apps, err := m.client.getApplications()
		if err != nil {
			return errorMsg(err)
		}
		return appsLoadedMsg(apps)
	}
}

func (m Model) fetchClusters() tea.Cmd {
	return func() tea.Msg {
		clusters, err := m.client.getClusters()
		if err != nil {
			return errorMsg(err)
		}
		return clustersLoadedMsg(clusters)
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case appsLoadedMsg:
		m.loading = false
		m.err = nil
		items := make([]list.Item, len(msg))
		for i, a := range msg {
			clName := a.ClusterName
			if clName == "" {
				clName = "N/A"
			}
			desc := fmt.Sprintf("Cluster: %s | Repo: %s", clName, a.RepoURL)
			items[i] = item{title: a.Name, desc: desc, status: a.Status}
		}
		m.appList.SetItems(items)

	case clustersLoadedMsg:
		m.loading = false
		m.err = nil
		items := make([]list.Item, len(msg))
		for i, c := range msg {
			desc := fmt.Sprintf("Message: %s", c.Message)
			items[i] = item{title: c.Name, desc: desc, status: c.Status}
		}
		m.clusterList.SetItems(items)

	case errorMsg:
		m.loading = false
		m.err = msg

	case eventReceivedMsg:
		cmds = append(cmds, m.fetchApps(), m.fetchClusters(), m.client.listenForEvents(m.ctx))

	case tea.KeyMsg:
		if m.confirmMsg != "" {
			switch msg.String() {
			case "y", "Y":
				if m.confirmAction != nil {
					m.confirmAction()
				}
				m.confirmMsg = ""
				m.confirmAction = nil
			case "n", "N", "esc":
				m.confirmMsg = ""
				m.confirmAction = nil
			}
			return m, nil
		}

		switch msg.String() {
		case "ctrl+c", "q":
			m.cancel()
			return m, tea.Quit
		case "tab":
			if m.state == appsView {
				m.state = clustersView
			} else {
				m.state = appsView
			}
		case "r":
			m.loading = true
			cmds = append(cmds, m.fetchApps(), m.fetchClusters())
		case "s":
			if m.state == appsView {
				if it := m.appList.SelectedItem(); it != nil {
					appName := it.(item).title
					m.confirmMsg = fmt.Sprintf("Trigger sync for %s? (y/n)", appName)
					m.confirmAction = func() { _ = m.client.syncApp(appName) }
				}
			}
		case "c":
			if m.state == clustersView {
				if it := m.clusterList.SelectedItem(); it != nil {
					clusterName := it.(item).title
					m.confirmMsg = fmt.Sprintf("Trigger check for %s? (y/n)", clusterName)
					m.confirmAction = func() { _ = m.client.checkCluster(clusterName) }
				}
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.appList.SetSize(msg.Width-8, msg.Height-12)
		m.clusterList.SetSize(msg.Width-8, msg.Height-12)

	case spinner.TickMsg:
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)
	}

	if m.confirmMsg == "" {
		if m.state == appsView {
			m.appList, cmd = m.appList.Update(msg)
			cmds = append(cmds, cmd)
		} else {
			m.clusterList, cmd = m.clusterList.Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	return m, tea.Batch(cmds...)
}

func renderBadge(status string) string {
	var style lipgloss.Style
	switch status {
	case "Synced", "Active":
		style = BadgeStyle.Background(secondaryColor)
	case "Error":
		style = BadgeStyle.Background(errorColor)
	case "Syncing", "Pending":
		style = BadgeStyle.Background(warningColor)
	default:
		style = BadgeStyle.Background(inactiveColor)
	}
	return style.Render(" " + strings.ToUpper(status) + " ")
}

func (m Model) View() string {
	var s strings.Builder

	// Header
	header := TitleStyle.Render(" GitOpsCTL Dashboard ")
	if m.loading {
		header += " " + m.spinner.View()
	}
	s.WriteString(header + "\n\n")

	// Error or Confirmation
	if m.err != nil {
		s.WriteString(StatusErrorStyle.Render(fmt.Sprintf("!! %v", m.err)) + "\n\n")
	}

	if m.confirmMsg != "" {
		s.WriteString(lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(accentColor).Padding(0, 2).Render(m.confirmMsg) + "\n\n")
	}

	// Tabs
	var appsTab, clustersTab string
	if m.state == appsView {
		appsTab = HeaderStyle.Render(" APPLICATIONS ")
		clustersTab = lipgloss.NewStyle().Foreground(inactiveColor).Render(" CLUSTERS ")
	} else {
		appsTab = lipgloss.NewStyle().Foreground(inactiveColor).Render(" APPLICATIONS ")
		clustersTab = HeaderStyle.Render(" CLUSTERS ")
	}
	s.WriteString(fmt.Sprintf("%s | %s\n\n", appsTab, clustersTab))

	// Content
	var content string
	if m.state == appsView {
		content = m.appList.View()
		if it := m.appList.SelectedItem(); it != nil {
			content += "\n" + DetailPaneStyle.Render(fmt.Sprintf("%s %s\n%s", renderBadge(it.(item).status), KeyStyle.Render(it.(item).title), it.(item).desc))
		}
	} else {
		content = m.clusterList.View()
		if it := m.clusterList.SelectedItem(); it != nil {
			content += "\n" + DetailPaneStyle.Render(fmt.Sprintf("%s %s\n%s", renderBadge(it.(item).status), KeyStyle.Render(it.(item).title), it.(item).desc))
		}
	}
	s.WriteString(content)

	s.WriteString(HelpStyle.Render("\ntab: switch view • r: refresh • s: sync • c: check • q: quit"))

	return MainContentStyle.
		Border(lipgloss.DoubleBorder()).
		BorderForeground(primaryColor).
		Width(m.width - 2).
		Height(m.height - 2).
		Render(s.String())
}

func Run(apiURL string) error {
	p := tea.NewProgram(NewModel(apiURL), tea.WithAltScreen())
	_, err := p.Run()
	return err
}

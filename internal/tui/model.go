package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type viewState int

const (
	appsView viewState = iota
	clustersView
)

// ── Model ─────────────────────────────────────────────────────────────────────

type Model struct {
	state         viewState
	apps          []AppResponse
	clusters      []ClusterResponse
	appCursor     int
	clusterCursor int
	spinner       spinner.Model
	width, height int
	loading       bool
	err           error
	client        *apiClient
	ctx           context.Context
	cancel        context.CancelFunc
	confirmMsg    string
	confirmAction func()
	statusMsg     string
	statusUntil   time.Time
}

func NewModel(apiURL string) Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(accent)

	ctx, cancel := context.WithCancel(context.Background())
	return Model{
		loading: true,
		spinner: s,
		client:  newAPIClient(apiURL),
		ctx:     ctx,
		cancel:  cancel,
	}
}

// ── Messages ──────────────────────────────────────────────────────────────────

type (
	appsLoadedMsg     []AppResponse
	clustersLoadedMsg []ClusterResponse
	errorMsg          error
)

// ── Init ──────────────────────────────────────────────────────────────────────

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

// ── Update ────────────────────────────────────────────────────────────────────

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	// Clear transient status message
	if !m.statusUntil.IsZero() && time.Now().After(m.statusUntil) {
		m.statusMsg = ""
		m.statusUntil = time.Time{}
	}

	switch msg := msg.(type) {
	case appsLoadedMsg:
		m.loading = false
		m.err = nil
		m.apps = msg
		if m.appCursor >= len(m.apps) {
			m.appCursor = max(0, len(m.apps)-1)
		}

	case clustersLoadedMsg:
		m.loading = false
		m.err = nil
		m.clusters = msg
		if m.clusterCursor >= len(m.clusters) {
			m.clusterCursor = max(0, len(m.clusters)-1)
		}

	case errorMsg:
		m.loading = false
		m.err = msg

	case eventReceivedMsg:
		cmds = append(cmds, m.fetchApps(), m.fetchClusters(), m.client.listenForEvents(m.ctx))

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)

	case tea.KeyMsg:
		if m.confirmMsg != "" {
			switch msg.String() {
			case "y", "Y":
				if m.confirmAction != nil {
					m.confirmAction()
					m.statusMsg = "✓ Action dispatched"
					m.statusUntil = time.Now().Add(3 * time.Second)
				}
				m.confirmMsg = ""
				m.confirmAction = nil
			case "n", "N", "esc":
				m.confirmMsg = ""
				m.confirmAction = nil
			}
			return m, tea.Batch(cmds...)
		}

		switch msg.String() {
		case "ctrl+c", "q":
			m.cancel()
			return m, tea.Quit
		case "tab", "shift+tab":
			if m.state == appsView {
				m.state = clustersView
			} else {
				m.state = appsView
			}
		case "up", "k":
			if m.state == appsView && m.appCursor > 0 {
				m.appCursor--
			} else if m.state == clustersView && m.clusterCursor > 0 {
				m.clusterCursor--
			}
		case "down", "j":
			if m.state == appsView && m.appCursor < len(m.apps)-1 {
				m.appCursor++
			} else if m.state == clustersView && m.clusterCursor < len(m.clusters)-1 {
				m.clusterCursor++
			}
		case "r":
			m.loading = true
			cmds = append(cmds, m.fetchApps(), m.fetchClusters())
		case "s":
			if m.state == appsView && len(m.apps) > 0 {
				name := m.apps[m.appCursor].Name
				m.confirmMsg = fmt.Sprintf("Sync  %q ?  (y/n)", name)
				m.confirmAction = func() { _ = m.client.syncApp(name) }
			}
		case "c":
			if m.state == clustersView && len(m.clusters) > 0 {
				name := m.clusters[m.clusterCursor].Name
				m.confirmMsg = fmt.Sprintf("Health-check  %q ?  (y/n)", name)
				m.confirmAction = func() { _ = m.client.checkCluster(name) }
			}
		case "u":
			if m.state == appsView && len(m.apps) > 0 {
				name := m.apps[m.appCursor].Name
				m.confirmMsg = fmt.Sprintf("Unregister application  %q ?  (y/n)", name)
				m.confirmAction = func() { _ = m.client.unregisterApp(name) }
			} else if m.state == clustersView && len(m.clusters) > 0 {
				name := m.clusters[m.clusterCursor].Name
				m.confirmMsg = fmt.Sprintf("Unregister cluster  %q ?  (y/n)", name)
				m.confirmAction = func() { _ = m.client.unregisterCluster(name) }
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}

	return m, tea.Batch(cmds...)
}

// ── View ──────────────────────────────────────────────────────────────────────

func (m Model) View() string {
	if m.width == 0 {
		return ""
	}

	var out strings.Builder

	// ── Header ──────────────────────────────────────────────────────────────
	logo := HeaderStyle.Render("⎈ GitOpsCTL")
	spin := ""
	if m.loading {
		spin = "  " + m.spinner.View()
	}
	status := ""
	if m.statusMsg != "" {
		status = "  " + ChipSynced.Render(m.statusMsg)
	}
	header := lipgloss.JoinHorizontal(lipgloss.Bottom,
		logo, spin, status,
		lipgloss.NewStyle().Foreground(subtle).Render(
			lipgloss.PlaceHorizontal(m.width-lipgloss.Width(logo)-lipgloss.Width(spin)-lipgloss.Width(status)-4, lipgloss.Right,
				VersionStyle.Render("gitopsctl · dashboard"),
			),
		),
	)
	out.WriteString(header + "\n")

	// ── Tab bar ─────────────────────────────────────────────────────────────
	appsTab := InactiveTab.Render("Applications")
	clTab := InactiveTab.Render("Clusters")
	if m.state == appsView {
		appsTab = ActiveTab.Render("Applications")
	} else {
		clTab = ActiveTab.Render("Clusters")
	}
	out.WriteString(TabBar.Width(m.width-2).Render(appsTab+clTab) + "\n\n")

	// ── Error / Confirm ─────────────────────────────────────────────────────
	if m.err != nil {
		out.WriteString(ErrStyle.Render("  ✗ "+m.err.Error()) + "\n\n")
	}
	if m.confirmMsg != "" {
		out.WriteString(ConfirmStyle.Render("  ? "+m.confirmMsg) + "\n\n")
	}

	// ── Main columns ────────────────────────────────────────────────────────
	listW := m.width / 2
	detailW := m.width - listW - 4
	listH := m.height - 9 // rows below header/tabs/help

	var listContent, detailContent string

	if m.state == appsView {
		listContent = m.renderAppList(listW-4, listH)
		detailContent = m.renderAppDetail(detailW - 4)
	} else {
		listContent = m.renderClusterList(listW-4, listH)
		detailContent = m.renderClusterDetail(detailW - 4)
	}

	leftPanel := ListPanelActive.Width(listW - 2).Height(listH).Render(listContent)
	rightPanel := DetailPanel.Width(detailW - 2).Height(listH).Render(detailContent)
	columns := lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, "  ", rightPanel)
	out.WriteString(columns + "\n")

	// ── Help bar ────────────────────────────────────────────────────────────
	out.WriteString(m.renderHelp())

	return out.String()
}

// ── List renderers ────────────────────────────────────────────────────────────

func (m Model) renderAppList(w, h int) string {
	if len(m.apps) == 0 {
		return ItemMeta.Render("\n  No applications registered.")
	}
	maxVisible := max(1, h-2)
	start := 0
	if m.appCursor >= maxVisible {
		start = m.appCursor - maxVisible + 1
	}
	var rows []string
	for i := start; i < len(m.apps) && i < start+maxVisible; i++ {
		a := m.apps[i]
		cl := a.ClusterName
		if cl == "" {
			cl = "—"
		}
		name := ItemName.Render(a.Name)
		meta := ItemMeta.Render(fmt.Sprintf("  %s · %s", cl, a.Interval))
		chip := StatusChip(a.Status)
		line := fmt.Sprintf("%s  %s\n%s", name, chip, meta)
		if i == m.appCursor {
			line = lipgloss.NewStyle().
				Foreground(accent).
				Border(lipgloss.NormalBorder(), false, false, false, true).
				BorderForeground(accent).
				PaddingLeft(1).
				Render(line)
		} else {
			line = lipgloss.NewStyle().PaddingLeft(2).Render(line)
		}
		rows = append(rows, line)
	}
	return strings.Join(rows, "\n")
}

func (m Model) renderClusterList(w, h int) string {
	if len(m.clusters) == 0 {
		return ItemMeta.Render("\n  No clusters registered.")
	}
	maxVisible := max(1, h-2)
	start := 0
	if m.clusterCursor >= maxVisible {
		start = m.clusterCursor - maxVisible + 1
	}
	var rows []string
	for i := start; i < len(m.clusters) && i < start+maxVisible; i++ {
		c := m.clusters[i]
		name := ItemName.Render(c.Name)
		chip := StatusChip(c.Status)
		reg := ItemMeta.Render("  Registered " + c.RegisteredAt.Format("02 Jan 15:04"))
		line := fmt.Sprintf("%s  %s\n%s", name, chip, reg)
		if i == m.clusterCursor {
			line = lipgloss.NewStyle().
				Foreground(accent).
				Border(lipgloss.NormalBorder(), false, false, false, true).
				BorderForeground(accent).
				PaddingLeft(1).
				Render(line)
		} else {
			line = lipgloss.NewStyle().PaddingLeft(2).Render(line)
		}
		rows = append(rows, line)
	}
	return strings.Join(rows, "\n")
}

// ── Detail renderers ──────────────────────────────────────────────────────────

func kv(label, value string) string {
	return DetailLabel.Render(label) + DetailValue.Render(value)
}

func (m Model) renderAppDetail(w int) string {
	if len(m.apps) == 0 {
		return lipgloss.NewStyle().Foreground(subtle).Render("Select an application")
	}
	a := m.apps[m.appCursor]

	hash := a.LastSyncedGitHash
	if len(hash) > 7 {
		hash = hash[:7]
	}
	if hash == "" {
		hash = "—"
	}
	cl := a.ClusterName
	if cl == "" {
		cl = "—"
	}
	failures := fmt.Sprintf("%d", a.ConsecutiveFailures)
	msg := a.Message
	if msg == "" {
		msg = "—"
	}
	if len(msg) > w-20 && w > 20 {
		msg = msg[:w-20] + "…"
	}

	var b strings.Builder
	b.WriteString(DetailTitle.Render(a.Name) + "\n")
	b.WriteString(StatusChip(a.Status) + "\n\n")
	b.WriteString(kv("Repo", a.RepoURL) + "\n")
	b.WriteString(kv("Branch", a.Branch) + "\n")
	b.WriteString(kv("Path", a.Path) + "\n")
	b.WriteString(kv("Cluster", cl) + "\n")
	b.WriteString(kv("Interval", a.Interval) + "\n")
	b.WriteString(kv("Last Synced", hash) + "\n")
	b.WriteString(kv("Failures", failures) + "\n")
	b.WriteString("\n" + DetailLabel.Render("Message") + "\n")
	b.WriteString(DetailValue.Italic(true).Render(msg))
	return b.String()
}

func (m Model) renderClusterDetail(w int) string {
	if len(m.clusters) == 0 {
		return lipgloss.NewStyle().Foreground(subtle).Render("Select a cluster")
	}
	c := m.clusters[m.clusterCursor]

	reg := c.RegisteredAt.Format("02 Jan 2006 15:04")
	checked := "—"
	if !c.LastCheckedAt.IsZero() {
		checked = c.LastCheckedAt.Format("02 Jan 2006 15:04")
	}
	msg := c.Message
	if msg == "" {
		msg = "—"
	}

	var b strings.Builder
	b.WriteString(DetailTitle.Render(c.Name) + "\n")
	b.WriteString(StatusChip(c.Status) + "\n\n")
	b.WriteString(kv("Kubeconfig", c.KubeconfigPath) + "\n")
	b.WriteString(kv("Registered", reg) + "\n")
	b.WriteString(kv("Last Checked", checked) + "\n")
	b.WriteString("\n" + DetailLabel.Render("Message") + "\n")
	b.WriteString(DetailValue.Italic(true).Render(msg))
	return b.String()
}

// ── Help ──────────────────────────────────────────────────────────────────────

func (m Model) renderHelp() string {
	type binding struct{ key, desc string }
	bindings := []binding{
		{"↑/↓", "navigate"},
		{"tab", "switch view"},
		{"r", "refresh"},
	}
	if m.state == appsView {
		bindings = append(bindings, binding{"s", "sync"}, binding{"u", "unregister"})
	} else {
		bindings = append(bindings, binding{"c", "check"}, binding{"u", "unregister"})
	}
	bindings = append(bindings, binding{"q", "quit"})

	var parts []string
	for _, b := range bindings {
		parts = append(parts, HelpKey.Render(b.key)+" "+HelpDesc.Render(b.desc))
	}
	return HelpSep.Render("  ") + strings.Join(parts, HelpSep.Render("  ·  "))
}

// ── helpers ───────────────────────────────────────────────────────────────────

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// ── Run ───────────────────────────────────────────────────────────────────────

func Run(apiURL string) error {
	p := tea.NewProgram(NewModel(apiURL), tea.WithAltScreen())
	_, err := p.Run()
	return err
}

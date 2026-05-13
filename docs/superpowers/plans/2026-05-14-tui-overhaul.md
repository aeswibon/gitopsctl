# TUI Overhaul Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Overhaul the GitOpsCTL TUI to a high-density, K9s-inspired dashboard with Vim-style navigation and a pure black dark theme.

**Architecture:** Component-based refactor of `internal/tui`. Decompose the monolithic model view into specialized renderers for Header, Table, Detail, and Footer. Implement a state machine for view management and a command mode for aliases.

**Tech Stack:** Go, Bubble Tea, Lipgloss, lipgloss.Table.

---

### Task 1: Update Theme and Styles
**Files:**
- Modify: `/Volumes/Seagate/developer/personal/gitopsctl/internal/tui/styles.go`

- [ ] **Step 1: Update palette to K9s-Dark (Pure Black)**
```go
	bg        = lipgloss.Color("#000000")
	fg        = lipgloss.Color("#FFFFFF")
	subtle    = lipgloss.Color("#333333")
	accent    = lipgloss.Color("#00FFFF") // Cyan
	green     = lipgloss.Color("#00FF00")
	red       = lipgloss.Color("#FF0000")
	yellow    = lipgloss.Color("#FFFF00")
```

- [ ] **Step 2: Update base styles**
```go
	Base = lipgloss.NewStyle().Background(bg).Foreground(fg)
	DetailLabel = lipgloss.NewStyle().Foreground(accent).Width(18)
```

- [ ] **Step 3: Commit**
```bash
git add internal/tui/styles.go
git commit -m "style: update TUI palette to K9s high-contrast dark"
```

### Task 2: Decompose TUI into Components (Header)
**Files:**
- Create: `/Volumes/Seagate/developer/personal/gitopsctl/internal/tui/header.go`

- [ ] **Step 1: Implement Header component**
```go
package tui

import (
	"fmt"
	"strings"
	"github.com/charmbracelet/lipgloss"
)

func (m Model) renderHeader() string {
	logo := HeaderStyle.Render("⎈ GitOpsCTL")

	// Calculate vitals
	var synced, drifted, errors int
	for _, a := range m.apps {
		switch a.Status {
		case "Synced": synced++
		case "OutOfSync", "Drifted": drifted++
		case "Error": errors++
		}
	}

	vitals := []string{
		kvCyan("APPS", fmt.Sprintf("%d", len(m.apps))),
		kvGreen("SYNCED", fmt.Sprintf("%d", synced)),
		kvYellow("DRIFTED", fmt.Sprintf("%d", drifted)),
		kvRed("ERRORS", fmt.Sprintf("%d", errors)),
		kvCyan("CLUSTERS", fmt.Sprintf("%d", len(m.clusters))),
	}

	return lipgloss.JoinHorizontal(lipgloss.Top,
		logo,
		"  ",
		strings.Join(vitals, "  "),
	)
}

func kvCyan(k, v string) string { return lipgloss.NewStyle().Foreground(accent).Render(k) + ": " + v }
func kvGreen(k, v string) string { return lipgloss.NewStyle().Foreground(accent).Render(k) + ": " + lipgloss.NewStyle().Foreground(green).Render(v) }
func kvYellow(k, v string) string { return lipgloss.NewStyle().Foreground(accent).Render(k) + ": " + lipgloss.NewStyle().Foreground(yellow).Render(v) }
func kvRed(k, v string) string { return lipgloss.NewStyle().Foreground(accent).Render(k) + ": " + lipgloss.NewStyle().Foreground(red).Render(v) }
```

- [ ] **Step 2: Commit**
```bash
git add internal/tui/header.go
git commit -m "feat: add TUI header component with vitals"
```

### Task 3: Implement Full-Width Table View
**Files:**
- Create: `/Volumes/Seagate/developer/personal/gitopsctl/internal/tui/table.go`
- Modify: `/Volumes/Seagate/developer/personal/gitopsctl/internal/tui/model.go`

- [ ] **Step 1: Implement Table rendering using lipgloss.Table**
```go
package tui

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
)

func (m Model) renderTable(width, height int) string {
	var rows [][]string
	var headers []string

	if m.state == appsView {
		headers = []string{"NAME", "STATUS", "HEALTH", "REPO", "LAST SYNC"}
		for i, a := range m.filteredApps() {
			row := []string{a.Name, a.Status, "Healthy", a.RepoURL, "2m ago"} // Simplified for now
			rows = append(rows, row)
		}
	} else {
		headers = []string{"NAME", "STATUS", "VERSION", "KUBECONFIG"}
		for i, c := range m.filteredClusters() {
			row := []string{c.Name, c.Status, "v1.28.0", c.KubeconfigPath}
			rows = append(rows, row)
		}
	}

	t := table.New().
		Border(lipgloss.NormalBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(subtle)).
		Headers(headers...).
		Rows(rows...).
		Width(width)

	return t.Render()
}
```

- [ ] **Step 2: Commit**
```bash
git add internal/tui/table.go
git commit -m "feat: implement full-width table view"
```

### Task 4: Advanced Vim Navigation and Command Mode
**Files:**
- Modify: `/Volumes/Seagate/developer/personal/gitopsctl/internal/tui/model.go`

- [ ] **Step 1: Add command mode state to Model struct**
```go
type Model struct {
    // ... existing ...
    commandText string
    commandMode bool // active when ':' is pressed
}
```

- [ ] **Step 2: Update Update() to handle gg, G, and : commands**
```go
case "g":
    // Check for double 'g'
    // ... logic ...
case "G":
    if m.state == appsView { m.appCursor = len(m.filteredApps()) - 1 }
case ":":
    m.commandMode = true
    m.commandText = ""
```

- [ ] **Step 3: Commit**
```bash
git add internal/tui/model.go
git commit -m "feat: add advanced Vim keys and command mode"
```

### Task 5: Finalize Footer and Activity View
**Files:**
- Create: `/Volumes/Seagate/developer/personal/gitopsctl/internal/tui/footer.go`

- [ ] **Step 1: Implement Footer with Command Line**
```go
func (m Model) renderFooter() string {
    if m.commandMode {
        return lipgloss.NewStyle().Foreground(green).Render(": " + m.commandText + "█")
    }
    // Render keybindings
    return HelpDesc.Render("? Help  :a Apps  :c Clusters  / Filter  d Delete")
}
```

- [ ] **Step 2: Integrate everything into View()**
- [ ] **Step 3: Commit**
```bash
git add internal/tui/
git commit -m "feat: finalize TUI overhaul"
```

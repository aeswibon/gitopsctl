# Design Spec: K9s-Inspired TUI Enhancement for GitOpsCTL

## 1. Overview
Enhance the GitOpsCTL Terminal User Interface (TUI) to provide a high-density, high-contrast, and keyboard-driven experience inspired by K9s. The goal is to move from a basic split-screen dashboard to a professional, full-width resource management tool with advanced Vim capabilities.

## 2. UI Architecture
The TUI will be reorganized into a three-tier layout:

### 2.1 Header (Vitals)
- **Background**: Pure Black (`#000000`).
- **Content**:
  - Left: App Name and Version.
  - Right: "Vitals" block with live counts:
    - `APPS`: Total number of applications.
    - `SYNCED`: Count of apps in 'Synced' state (Green).
    - `DRIFTED`: Count of apps in 'Drifted' or 'OutOfSync' state (Yellow).
    - `ERRORS`: Count of apps in 'Error' state (Red).
    - `CLUSTERS`: Total number of registered clusters.
- **Style**: High-contrast labels (Cyan) and values (White/Colored).

### 2.2 Main View (Resource Table)
- **Layout**: Full-width table using `lipgloss.Table`.
- **Modes**:
  - **Applications View**: Columns: `NAME`, `STATUS`, `HEALTH`, `REPO`, `BRANCH`, `LAST SYNC`.
  - **Clusters View**: Columns: `NAME`, `STATUS`, `VERSION`, `KUBECONFIG`, `LAST CHECKED`.
  - **Activity View**: Full-width log stream.
  - **Detail View**: Full-page drill-down for a specific app or cluster (triggered by `Enter`).

### 2.3 Footer (Command & Shortcuts)
- **Command Line**: Prefix `:` with an active cursor. Used for view switching and commands.
- **Shortcuts Bar**: Quick reference for context-sensitive keys (e.g., `? Help`, `s Sync`, `d Delete`).

## 3. Navigation & Interaction (Vim-Style)
- **Movement**:
  - `j`/`k`: Move selection down/up.
  - `gg`: Jump to top of the list.
  - `G`: Jump to bottom of the list.
  - `ctrl+u`/`ctrl+d`: Page up/down.
- **Views & Commands**:
  - `:a`: Switch to Applications view.
  - `:c`: Switch to Clusters view.
  - `:l`: Switch to Activity view.
  - `/`: Enter filter mode (live search).
  - `Enter`: Open Detail View for the selected resource.
  - `Esc`: Back to previous view or clear filter/command.
- **Actions**:
  - `s`: Trigger 'Sync' for the selected application.
  - `c`: Trigger 'Health Check' for the selected cluster.
  - `d`: Trigger 'Unregister' with a `y/n` confirmation.
  - `a`: Trigger 'Approve' for manual sync apps.

## 4. Theme (K9s-Dark)
- **Palette**:
  - `BG`: `#000000` (Pure Black)
  - `FG`: `#FFFFFF` (White)
  - `Label`: `#00FFFF` (Cyan)
  - `Border`: `#333333` (Dark Grey)
  - `Selection`: `#7C6AF7` (Violet background or bold border)
  - `Success`: `#00FF00` (Green)
  - `Error`: `#FF0000` (Red)
  - `Warning`: `#FFFF00` (Yellow)

## 5. Technical Implementation
- **Framework**: Bubble Tea with Lipgloss.
- **Components**: Decompose `internal/tui/model.go` logic into smaller functional units:
  - `Header`: Renders the vitals.
  - `Table`: Generic table component for resources.
  - `Detail`: Detail page renderer.
  - `Command`: Command line handler.
- **State Management**:
  - Track `activeView` (apps, clusters, logs, detail).
  - Track `commandMode` (idle, command, filter).
  - Use a `cursor` index for each view to preserve position when switching.

## 6. Success Criteria
- [ ] User can switch between apps and clusters using `:a` and `:c`.
- [ ] Vim keys (`gg`, `G`, `j`, `k`) work as expected in list views.
- [ ] The theme is pure black with high-contrast colored text.
- [ ] 'Enter' drills down into a full-page detail view.
- [ ] All existing actions (Sync, Unregister) are accessible via the new interface.

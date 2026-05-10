# Terminal UI (Dashboard)

GitOpsCTL features a premium Terminal User Interface (TUI) built with Bubble Tea and Lipgloss. It provides a real-time overview of your GitOps state without leaving your terminal.

## Opening the Dashboard

```bash
gitopsctl dashboard --api-url http://localhost:8080
```

## Key Views

### 1. Applications View
Displays a list of all registered applications, their current status (Synced, OutOfSync, Error, etc.), the last synced Git hash, and health indicators.

### 2. Clusters View
Displays the status of target clusters, including connectivity health and server version information.

### 3. Activity Logs
A live stream of integration events, including sync starts, successes, failures, and health changes.

## Interactions

| Key | Action |
|-----|--------|
| `tab` / `shift+tab` | Switch between Applications, Clusters, and Activity views. |
| `up` / `down` | Navigate through lists. |
| `s` | Manually trigger a synchronization for the selected application. |
| `a` | Approve a pending commit for manual sync policy. |
| `u` | Unregister the selected application. |
| `r` | Force refresh data. |
| `/` | Filter the list of applications or clusters. |
| `esc` | Clear filter or cancel action. |
| `q` / `ctrl+c` | Exit the dashboard. |

## Real-time Updates

The dashboard uses Server-Sent Events (SSE) to receive live updates from the controller. If the connection is lost, the dashboard will automatically attempt to reconnect.

## Visual Indicators

- **Synced** (Green): Git state matches cluster state.
- **OutOfSync** (Yellow): New commits are available in Git but not yet applied.
- **Error** (Red): A failure occurred during synchronization or connection.
- **Healthy** (Green Check): All underlying Kubernetes resources are ready.
- **Progressing** (Blue): Resources are being created or updated (e.g., rolling update).
- **Degraded** (Red Cross): One or more resources are in an unhealthy state (e.g., CrashLoopBackOff).

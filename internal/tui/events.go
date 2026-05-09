package tui

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func (c *apiClient) listenForEvents(ctx context.Context) tea.Cmd {
	return func() tea.Msg {
		req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/api/v1/events", c.baseURL), nil)
		if err != nil {
			return errorMsg(err)
		}

		resp, err := c.sseClient.Do(req)

		if err != nil {
			return errorMsg(err)
		}
		defer func() { _ = resp.Body.Close() }()

		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "data:") {
				// We got an event! Trigger a refresh.
				// In a more complex TUI, we would parse the JSON and update only the relevant item.
				// For now, a full refresh is safer and easier.
				return eventReceivedMsg{}
			}
		}

		if err := scanner.Err(); err != nil {
			return errorMsg(err)
		}
		return nil
	}
}

type eventReceivedMsg struct{}

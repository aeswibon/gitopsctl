package tui

import (
	"bufio"
	"context"
	"encoding/json"
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
				data := strings.TrimPrefix(line, "data:")
				var ev Event
				if err := json.Unmarshal([]byte(data), &ev); err == nil {
					return eventReceivedMsg{Event: ev}
				}
				return eventReceivedMsg{}
			}
		}

		if err := scanner.Err(); err != nil {
			return errorMsg(err)
		}
		return sseDisconnectedMsg{}
	}
}

type sseDisconnectedMsg struct{}

type eventReceivedMsg struct {
	Event Event
}

package utils

import (
	"fmt"
	"testing"

	"go.uber.org/zap"
)

type mockRenderable struct {
	name string
}

func (m *mockRenderable) ToTableHeaders(details bool) []string {
	return []string{"NAME"}
}
func (m *mockRenderable) ToTableRow(details bool) []string {
	return []string{m.name}
}
func (m *mockRenderable) ToJSONMap() map[string]any {
	return map[string]any{"name": m.name}
}
func (m *mockRenderable) ToYAMLString() string {
	return fmt.Sprintf("name: %s", m.name)
}

func TestRunListCommand(t *testing.T) {
	logger := zap.NewNop()
	opts := ListOptions{OutputFormat: "table", SortBy: "name"}

	loadFunc := func() ([]Renderable, error) {
		return []Renderable{&mockRenderable{name: "item1"}}, nil
	}
	filterFunc := func(items []Renderable, filter string) []Renderable {
		return items
	}
	sortFunc := func(items []Renderable, sortBy string) {}
	emptyFunc := func(filter string) error { return nil }

	err := RunListCommand(logger, opts, loadFunc, filterFunc, sortFunc, emptyFunc)
	if err != nil {
		t.Errorf("RunListCommand() error = %v", err)
	}
}

func TestRenderFunctions(t *testing.T) {
	items := []Renderable{&mockRenderable{name: "test"}}

	if err := RenderTable(items, false, false); err != nil {
		t.Errorf("RenderTable() error = %v", err)
	}
	if err := RenderJSON(items); err != nil {
		t.Errorf("RenderJSON() error = %v", err)
	}
	if err := RenderYAML(items); err != nil {
		t.Errorf("RenderYAML() error = %v", err)
	}
}

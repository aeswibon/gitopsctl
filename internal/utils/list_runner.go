package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"go.uber.org/zap"
)

// Renderable is an interface that defines methods for rendering items in different formats.
type Renderable interface {
	ToTableHeaders(details bool) []string
	ToTableRow(details bool) []string
	ToJSONMap() map[string]any
	ToYAMLString() string
}

// RunListCommand executes a list command with the provided options.
func RunListCommand(
	logger *zap.Logger,
	opts ListOptions,
	loadFunc func() ([]Renderable, error),
	filterFunc func([]Renderable, string) []Renderable,
	sortFunc func([]Renderable, string),
	emptyMessageFunc func(string) error,
) error {
	items, err := loadFunc()
	if err != nil {
		// If loadFunc returns an error and also indicates no items,
		// it's likely a controlled empty state rather than a critical error.
		// Check for specific "no items registered" error from loadFunc.
		if strings.Contains(err.Error(), "no ") {
			return emptyMessageFunc(opts.StatusFilter)
		}
		return err
	}

	filteredItems := filterFunc(items, opts.StatusFilter)
	if len(filteredItems) == 0 {
		logger.Info("No items found matching the specified criteria.")
		return emptyMessageFunc(opts.StatusFilter)
	}

	sortFunc(filteredItems, opts.SortBy)

	switch strings.ToLower(opts.OutputFormat) {
	case "json":
		return RenderJSON(filteredItems)
	case "yaml":
		return RenderYAML(filteredItems)
	default:
		return RenderTable(filteredItems, opts.NoHeader, opts.ShowDetails)
	}
}

// RenderTable renders items as a table.
func RenderTable(items []Renderable, noHeader bool, showDetails bool) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', tabwriter.TabIndent)
	defer w.Flush()

	if !noHeader && len(items) > 0 {
		fmt.Fprintln(w, strings.Join(items[0].ToTableHeaders(showDetails), "\t"))
		fmt.Fprintln(w, strings.Join(generateSeparator(items[0].ToTableHeaders(showDetails)), "\t"))
	}

	for _, item := range items {
		fmt.Fprintln(w, strings.Join(item.ToTableRow(showDetails), "\t"))
	}
	return nil
}

func generateSeparator(headers []string) []string {
	separators := make([]string, len(headers))
	for i, header := range headers {
		separators[i] = strings.Repeat("-", len(header))
	}
	return separators
}

// RenderJSON renders items as JSON.
func RenderJSON(items []Renderable) error {
	var jsonItems []map[string]any
	for _, item := range items {
		jsonItems = append(jsonItems, item.ToJSONMap())
	}
	response := map[string]any{
		"items": jsonItems,
		"total": len(jsonItems),
	}
	jsonData, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	fmt.Println(string(jsonData))
	return nil
}

// RenderYAML renders items as YAML.
func RenderYAML(items []Renderable) error {
	fmt.Println("items:")
	for _, item := range items {
		// Indent each line of the YAML string
		lines := strings.Split(item.ToYAMLString(), "\n")
		for _, line := range lines {
			fmt.Printf("  %s\n", line)
		}
	}
	fmt.Printf("total: %d\n", len(items))
	return nil
}

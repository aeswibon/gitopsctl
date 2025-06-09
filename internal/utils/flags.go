package utils

import "github.com/spf13/cobra"

// ListOptions holds the options for listing resources.
// It includes output format, header visibility, detail level, status filter, and sorting options.
type ListOptions struct {
	OutputFormat string
	NoHeader     bool
	ShowDetails  bool
	StatusFilter string
	SortBy       string
}

// AddListFlags adds common flags for listing commands to the provided Cobra command.
// It includes flags for output format, header visibility, detail level, status filter, and sorting options.
func AddListFlags(cmd *cobra.Command, opts *ListOptions, defaultSort string) {
	cmd.Flags().StringVarP(&opts.OutputFormat, "output", "o", "table", "Output format: table, json, yaml")
	cmd.Flags().BoolVar(&opts.NoHeader, "no-header", false, "Hide table headers")
	cmd.Flags().BoolVar(&opts.ShowDetails, "details", false, "Show additional details")
	cmd.Flags().StringVar(&opts.StatusFilter, "status", "all", "Filter by status: all, active, inactive, error, pending")
	cmd.Flags().StringVar(&opts.SortBy, "sort-by", defaultSort, "Sort by: name, status, registered")

	cmd.RegisterFlagCompletionFunc("output", func(_ *cobra.Command, _ []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"table", "json", "yaml"}, cobra.ShellCompDirectiveDefault
	})
	cmd.RegisterFlagCompletionFunc("status", func(_ *cobra.Command, _ []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"all", "active", "inactive", "error", "pending"}, cobra.ShellCompDirectiveDefault
	})
}

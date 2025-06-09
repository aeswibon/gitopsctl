package cmd

import "github.com/spf13/cobra"

var appGroup = &cobra.Group{
	ID:    "appGroup",
	Title: "Manage Applications",
}

var clusterGroup = &cobra.Group{
	ID:    "clusterGroup",
	Title: "Manage Clusters",
}

package main

import (
	"github.com/spf13/cobra"
)

var (
	clearCmd = &cobra.Command{
		Use:   "clear",
		Short: "clears all images",
		RunE: func(cmd *cobra.Command, args []string) error {
			return d.Clear()
		},
	}
)

func init() {
	RootCmd.AddCommand(clearCmd)
}

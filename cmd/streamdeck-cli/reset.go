package main

import (
	"github.com/spf13/cobra"
)

var (
	resetCmd = &cobra.Command{
		Use:   "reset",
		Short: "resets the device, clears all images and shows the default logo",
		RunE: func(cmd *cobra.Command, args []string) error {
			return d.Reset()
		},
	}
)

func init() {
	RootCmd.AddCommand(resetCmd)
}

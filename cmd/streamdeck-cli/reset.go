package main

import (
	"github.com/muesli/coral"
)

var (
	resetCmd = &coral.Command{
		Use:   "reset",
		Short: "resets the device, clears all images and shows the default logo",
		RunE: func(cmd *coral.Command, args []string) error {
			return d.Reset()
		},
	}
)

func init() {
	RootCmd.AddCommand(resetCmd)
}

package main

import (
	"github.com/muesli/coral"
)

var (
	clearCmd = &coral.Command{
		Use:   "clear",
		Short: "clears all images",
		RunE: func(cmd *coral.Command, args []string) error {
			return d.Clear()
		},
	}
)

func init() {
	RootCmd.AddCommand(clearCmd)
}

package main

import (
	"fmt"
	"strconv"

	"github.com/muesli/coral"
)

var (
	brightnessCmd = &coral.Command{
		Use:   "brightness <percentage>",
		Short: "controls the brightness of the keys (in percent)",
		RunE: func(cmd *coral.Command, args []string) error {
			if len(args) < 1 {
				return fmt.Errorf("brightness requires a percentage")
			}

			brightness, err := strconv.ParseInt(args[0], 10, 8)
			if err != nil {
				return fmt.Errorf("supplied parameter is not a valid number")
			}
			return d.SetBrightness(uint8(brightness))
		},
	}
)

func init() {
	RootCmd.AddCommand(brightnessCmd)
}

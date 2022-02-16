package main

import (
	"fmt"

	"github.com/muesli/coral"
	"github.com/muesli/streamdeck"
)

var (
	devicesCmd = &coral.Command{
		Use:   "devices",
		Short: "devices lists all available Stream Deck devices",
		RunE: func(cmd *coral.Command, args []string) error {
			_ = d.Close()

			devs, err := streamdeck.Devices()
			if err != nil {
				return fmt.Errorf("no Stream Deck devices found: %s", err)
			}
			if len(devs) == 0 {
				return fmt.Errorf("no Stream Deck devices found")
			}

			fmt.Printf("Found %d devices:\n", len(devs))

			for _, d := range devs {
				if err := d.Open(); err != nil {
					return fmt.Errorf("can't open device %s: %s", d.ID, err)
				}

				ver, err := d.FirmwareVersion()
				if err != nil {
					return fmt.Errorf("can't retrieve device info: %s", err)
				}
				fmt.Printf("Serial %s with %d keys (ID: %s, firmware %s)\n",
					d.Serial, d.Keys, d.ID, ver)

				_ = d.Close()
			}

			return nil
		},
	}
)

func init() {
	RootCmd.AddCommand(devicesCmd)
}

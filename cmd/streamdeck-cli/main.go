package main

import (
	"fmt"
	"os"

	"github.com/muesli/coral"
	"github.com/muesli/streamdeck"
)

var (
	// RootCmd is the core command used for cli-arg parsing.
	RootCmd = &coral.Command{
		Use:                "streamdeck-cli",
		Short:              "streamdeck-cli lets you control your Elgato Stream Deck",
		SilenceErrors:      true,
		SilenceUsage:       true,
		PersistentPreRunE:  initStreamDeck,
		PersistentPostRunE: closeStreamDeck,
	}

	d streamdeck.Device
)

func closeStreamDeck(cmd *coral.Command, args []string) error {
	return d.Close()
}

func initStreamDeck(cmd *coral.Command, args []string) error {
	devs, err := streamdeck.Devices()
	if err != nil {
		return fmt.Errorf("no Stream Deck devices found: %s", err)
	}
	if len(devs) == 0 {
		return fmt.Errorf("no Stream Deck devices found")
	}
	d = devs[0]

	if err := d.Open(); err != nil {
		return fmt.Errorf("can't open device: %s", err)
	}

	/*
		ver, err := d.FirmwareVersion()
		if err != nil {
			return fmt.Errorf("can't retrieve device info: %s", err)
		}
		fmt.Printf("Found device with serial %s (firmware %s)\n",
			d.Serial, ver)
	*/

	return nil
}

func main() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

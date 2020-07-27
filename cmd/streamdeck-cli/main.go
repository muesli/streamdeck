package main

import (
	"fmt"
	"log"
	"os"

	"github.com/unix-streamdeck/streamdeck"
	"github.com/spf13/cobra"
)

var (
	// RootCmd is the core command used for cli-arg parsing.
	RootCmd = &cobra.Command{
		Use:           "streamdeck-cli",
		Short:         "streamdeck-cli lets you control your Elgato Stream Deck",
		SilenceErrors: true,
		SilenceUsage:  true,
	}

	d streamdeck.Device
)

func main() {
	devs, err := streamdeck.Devices()
	if err != nil {
		panic(err)
	}
	if len(devs) == 0 {
		log.Fatalln("No Stream Deck devices found.")
	}
	d = devs[0]

	err = d.Open()
	if err != nil {
		panic(err)
	}
	defer d.Close()

	ver, err := d.FirmwareVersion()
	if err != nil {
		panic(err)
	}
	fmt.Printf("Found device with serial %s (firmware %s)\n",
		d.Serial, ver)

	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
}

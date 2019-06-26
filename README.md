# streamdeck

An application and Go library to control your Elgato Stream Deck on Linux

## Installation

Make sure you have a working Go environment (Go 1.9 or higher is required).
See the [install instructions](http://golang.org/doc/install.html).

To install streamdeck, simply run:

    go get github.com/muesli/streamdeck

## Configuration

On Linux you need to set up some udev rules to be able to access the device as a
regular user. Edit `/etc/udev/rules.d/99-streamdeck.rules` and add these lines:

```
SUBSYSTEM=="usb", ATTRS{idVendor}=="0fd9", ATTRS{idProduct}=="0060", MODE:="666", GROUP="plugdev"
SUBSYSTEM=="usb", ATTRS{idVendor}=="0fd9", ATTRS{idProduct}=="0063", MODE:="666", GROUP="plugdev"
SUBSYSTEM=="usb", ATTRS{idVendor}=="0fd9", ATTRS{idProduct}=="006c", MODE:="666", GROUP="plugdev"
```

Make sure your user is part of the `plugdev` group and reload the rules with
`sudo udevadm control --reload-rules`. Unplug and replug the device and you
should be good to go.

## Development

[![GoDoc](https://godoc.org/github.com/golang/gddo?status.svg)](https://godoc.org/github.com/muesli/streamdeck)
[![Build Status](https://travis-ci.org/muesli/streamdeck.svg?branch=master)](https://travis-ci.org/muesli/streamdeck)
[![Go ReportCard](http://goreportcard.com/badge/muesli/streamdeck)](http://goreportcard.com/report/muesli/streamdeck)

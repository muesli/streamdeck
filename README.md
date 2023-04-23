# streamdeck

[![Latest Release](https://img.shields.io/github/release/muesli/streamdeck.svg?style=for-the-badge)](https://github.com/muesli/streamdeck/releases)
[![Software License](https://img.shields.io/badge/license-MIT-brightgreen.svg?style=for-the-badge)](/LICENSE)
[![Build Status](https://img.shields.io/github/actions/workflow/status/muesli/streamdeck/build.yml?branch=master&style=for-the-badge)](https://github.com/muesli/streamdeck/actions)
[![Go ReportCard](https://goreportcard.com/badge/github.com/muesli/streamdeck?style=for-the-badge)](https://goreportcard.com/report/muesli/streamdeck)
[![Go Doc](https://img.shields.io/badge/godoc-reference-blue.svg?style=for-the-badge)](https://pkg.go.dev/github.com/muesli/streamdeck)

A CLI application and Go library to control your Elgato Stream Deck on Linux.

If you're looking for a complete Linux service to control your StreamDeck, check
out [Deckmaster](https://github.com/muesli/deckmaster), which is based on this
library.

## Installation

Make sure you have a working Go environment (Go 1.12 or higher is required).
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
SUBSYSTEM=="usb", ATTRS{idVendor}=="0fd9", ATTRS{idProduct}=="006d", MODE:="666", GROUP="plugdev"
SUBSYSTEM=="usb", ATTRS{idVendor}=="0fd9", ATTRS{idProduct}=="0080", MODE:="666", GROUP="plugdev"
SUBSYSTEM=="usb", ATTRS{idVendor}=="0fd9", ATTRS{idProduct}=="0090", MODE:="666", GROUP="plugdev"
SUBSYSTEM=="usb", ATTRS{idVendor}=="0fd9", ATTRS{idProduct}=="0084", MODE:="666", GROUP="plugdev"
```

Make sure your user is part of the `plugdev` group and reload the rules with
`sudo udevadm control --reload-rules`. Unplug and replug the device and you
should be good to go.

## Usage

Control the brightness, in percent between 0 and 100:

```
streamdeck-cli brightness 50
```

Set an image on the first key (from the top-left):

```
streamdeck-cli image 0 image.png
```

Clear all images:

```
streamdeck-cli clear
```

Reset the device:

```
streamdeck-cli reset
```

## Stream Deck Plus

At the current state, the knobs and touch screen usage will be transformed to "normal button keys indexes".

### Normal Buttons

The 8 "normal" buttons have the key index 0 - 7 (top left to bottom right)

### Touch Screen

The touch screen is divided into four horizontal segments (matching the numbers of knobs):
Segment index is from 0 - 3  (from left to right)

The key indexes for touch screen usages are:

Segment 0
- Short touch: 20
- Long touch:  24

Segment 1
- Short touch: 21
- Long touch:  25

Segment 2
- Short touch: 22
- Long touch:  26

Segment 3
- Short touch: 23
- Long touch:  27

All touch screen presses are not "holdable" like normal buttons.

Swiping is mapped to:
- From left to right: 28
- From right to left: 29

### Knobs

The knobs usages will be mapped to key indexes as following (left to right):

Knob 1
- Press:       8 (holdable)
- Left turn:  12
- Right turn: 16

Knob 2
- Press:       9 (holdable)
- Left turn:  13
- Right turn: 17

Knob 3
- Press:      10 (holdable)
- Left turn:  14
- Right turn: 18

Knob 4
- Press:      11 (holdable)
- Left turn:  15
- Right turn: 19

## Feedback

Got some feedback or suggestions? Please open an issue or drop me a note!

* [Twitter](https://twitter.com/mueslix)
* [The Fediverse](https://mastodon.social/@fribbledom)

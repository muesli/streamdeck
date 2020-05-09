package streamdeck

import (
	"fmt"
	"image"
	"image/color"

	"github.com/karalabe/hid"
	"golang.org/x/image/draw"
)

const (
	VID_ELGATO          = 0x0fd9
	PID_STREAMDECK      = 0x0060
	PID_STREAMDECK_V2   = 0x006d
	PID_STREAMDECK_MINI = 0x0063
	PID_STREAMDECK_XL   = 0x006c
)

// Device represents a single Stream Deck device
type Device struct {
	ID     string
	Serial string

	Columns uint8
	Rows    uint8
	Pixels  uint

	state []byte

	device *hid.Device
	info   hid.DeviceInfo

	newHardware func(device *hid.Device) hardware
	hardware    hardware
}

// Key holds the current status of a key on the device
type Key struct {
	Index   uint8
	Pressed bool
}

// Devices returns all attached Stream Decks
func Devices() ([]Device, error) {
	dd := []Device{}

	devs := hid.Enumerate(VID_ELGATO, 0)
	for _, d := range devs {
		if d.VendorID == VID_ELGATO && d.ProductID == PID_STREAMDECK {
			dev := Device{
				ID:          d.Path,
				Serial:      d.Serial,
				Columns:     5,
				Rows:        3,
				Pixels:      72,
				state:       make([]byte, 5*3), // Columns * Rows
				info:        d,
				newHardware: newClassicHardware,
			}

			dd = append(dd, dev)
		}
		if d.VendorID == VID_ELGATO && d.ProductID == PID_STREAMDECK_MINI {
			dev := Device{
				ID:          d.Path,
				Serial:      d.Serial,
				Columns:     3,
				Rows:        2,
				Pixels:      80,
				state:       make([]byte, 3*2), // Columns * Rows
				info:        d,
				newHardware: newMiniHardware,
			}

			dd = append(dd, dev)
		}

		/*
			if d.VendorID == VID_ELGATO && d.ProductID == PID_STREAMDECK_V2 {
				dev := Device{
					ID:         d.Path,
					Serial:     d.Serial,
					Columns:    5,
					Rows:       3,
					Pixels:     72,
					state:      make([]byte, 5*3), // Columns * Rows
					info:       d,
					newHardware: newXLHardware,
				}

				dd = append(dd, dev)
			}
		*/

		if d.VendorID == VID_ELGATO && d.ProductID == PID_STREAMDECK_XL {
			dev := Device{
				ID:          d.Path,
				Serial:      d.Serial,
				Columns:     8,
				Rows:        4,
				Pixels:      96,
				state:       make([]byte, 8*4), // Columns * Rows
				info:        d,
				newHardware: newXLHardware,
			}

			dd = append(dd, dev)
		}
	}

	return dd, nil
}

// Open the device for input/output. This must be called before trying to
// communicate with the device
func (d *Device) Open() error {
	var err error
	d.device, err = d.info.Open()
	if err != nil {
		return err
	}
	d.hardware = d.newHardware(d.device)
	return nil
}

// Close the connection with the device
func (d Device) Close() error {
	return d.device.Close()
}

// FirmwareVersion returns the firmware version of the device
func (d Device) FirmwareVersion() (string, error) {
	return d.hardware.FirmwareVersion()
}

// Resets the Stream Deck, clears all button images and shows the standby image
func (d Device) Reset() error {
	return d.hardware.Reset()
}

// Clears the Stream Deck, setting a black image on all buttons
func (d Device) Clear() error {
	img := image.NewRGBA(image.Rect(0, 0, int(d.Pixels), int(d.Pixels)))
	draw.Draw(img, img.Bounds(), image.NewUniform(color.RGBA{0, 0, 0, 255}), image.ZP, draw.Src)
	for i := uint8(0); i <= d.Columns*d.Rows; i++ {
		err := d.SetImage(i, img)
		if err != nil {
			fmt.Println(err)
			return err
		}
	}

	return nil
}

// ReadKeys returns a channel, which it will use to emit key presses/releases
func (d Device) ReadKeys() (chan Key, error) {
	kch := make(chan Key)
	b := make([]byte, len(d.state))
	go func() {
		for {
			copy(d.state, b)

			err := d.hardware.ReadKeyState(b)
			if err != nil {
				close(kch)
				return
			}

			for i := 0; i < len(b); i++ {
				if b[i] != d.state[i] {
					kch <- Key{
						Index:   d.translateKeyIndex(uint8(i)),
						Pressed: b[i] == 1,
					}
				}
			}
		}
	}()

	return kch, nil
}

// SetBrightness sets the background lighting brightness from 0 to 100 percent
func (d Device) SetBrightness(percent uint8) error {
	if percent > 100 {
		percent = 100
	}
	return d.hardware.SetBrightness(percent)
}

// SetImage sets the image of a button on the Stream Deck. The provided image
// needs to be in the correct resolution for the device. The index starts with
// 0 being the top-left button.
func (d Device) SetImage(index uint8, img image.Image) error {
	if img.Bounds().Dy() != int(d.Pixels) ||
		img.Bounds().Dx() != int(d.Pixels) {
		return fmt.Errorf("supplied image has wrong dimensions, expected %[1]dx%[1]d pixels", d.Pixels)
	}

	imageData, err := d.hardware.GetImageData(img)
	if err != nil {
		return fmt.Errorf("cannot get image data: %v", err)
	}

	data := make([]byte, d.hardware.ImagePageSize())

	var page int
	var lastPage bool
	for !lastPage {
		var payload []byte
		payload, lastPage = imageData.Page(page)
		header := d.hardware.GetImagePageHeader(page, d.translateKeyIndex(index), len(payload), lastPage)

		copy(data, header)
		copy(data[len(header):], payload)

		_, err := d.device.Write(data)
		if err != nil {
			return fmt.Errorf("cannot write image page %d of %d (%d image bytes) %d bytes: %v", page, imageData.PageCount(), imageData.Length(), len(data), err)
		}

		page++
	}

	return nil
}

func (d Device) translateKeyIndex(index uint8) uint8 {
	keyCol := index % d.Columns
	return (index - keyCol) + (d.Columns - 1) - keyCol
}

// toRGBA converts an image.Image to an image.RGBA
func toRGBA(img image.Image) *image.RGBA {
	switch img.(type) {
	case *image.RGBA:
		return img.(*image.RGBA)
	}
	out := image.NewRGBA(img.Bounds())
	draw.Copy(out, image.Pt(0, 0), img, img.Bounds(), draw.Src, nil)
	return out
}

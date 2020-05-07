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

var (
	c_ELGATO_FIRMWARE   = []byte{0x04}
	c_ELGATO_RESET      = []byte{0x0b, 0x63}
	c_ELGATO_BRIGHTNESS = []byte{0x05, 0x55, 0xaa, 0xd1, 0x01}
)

// Device represents a single Stream Deck device
type Device struct {
	ID     string
	Serial string

	Columns uint8
	Rows    uint8
	Pixels  uint

	startPage   uint8
	pageLength  uint
	state       []byte
	stateOffset int

	device *hid.Device
	info   hid.DeviceInfo
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
				startPage:   1,
				pageLength:  7803,
				state:       make([]byte, 5*3+1), // Columns * Rows + 1
				stateOffset: 1,
				info:        d,
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
				startPage:   1,
				pageLength:  1008,
				state:       make([]byte, 3*2+1), // Columns * Rows + 1
				stateOffset: 1,
				info:        d,
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
					startPage:  1,
					pageLength: 7803,
					state:      make([]byte, 5*3+1), // Columns * Rows + 1
					info:       d,
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
				startPage:   1,
				state:       make([]byte, 4+8*4), // 4 + Columns * Rows
				stateOffset: 4,
				info:        d,
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
	return err
}

// Close the connection with the device
func (d Device) Close() error {
	return d.device.Close()
}

// FirmwareVersion returns the firmware version of the device
func (d Device) FirmwareVersion() (string, error) {
	b := make([]byte, 17)
	copy(b, c_ELGATO_FIRMWARE)

	_, err := d.device.GetFeatureReport(b)
	if err != nil {
		return "", err
	}

	return string(b[5:]), nil
}

// Resets the Stream Deck, clears all button images and shows the standby image
func (d Device) Reset() error {
	b := make([]byte, 17)
	copy(b, c_ELGATO_RESET)

	_, err := d.device.SendFeatureReport(b)
	if err != nil {
		return err
	}

	return nil
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
			_, err := d.device.Read(b)
			if err != nil {
				close(kch)
				return
			}

			for i := d.stateOffset; i < len(b); i++ {
				if b[i] != d.state[i] {
					kch <- Key{
						Index:   d.translateKeyIndex(uint8(i - d.stateOffset)),
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
	b := make([]byte, 17)
	copy(b, c_ELGATO_BRIGHTNESS)

	if percent > 100 {
		percent = 100
	}
	b[len(c_ELGATO_BRIGHTNESS)] = percent

	_, err := d.device.SendFeatureReport(b)
	if err != nil {
		return err
	}

	return nil
}

// SetImage sets the image of a button on the Stream Deck. The provided image
// needs to be in the correct resolution for the device. The index starts with
// 0 being the top-left button.
func (d Device) SetImage(index uint8, img image.Image) error {
	rgba := toRGBA(img)
	if rgba.Bounds().Dy() != int(d.Pixels) ||
		rgba.Bounds().Dx() != int(d.Pixels) {
		return fmt.Errorf("supplied image has wrong dimensions, expected %[1]dx%[1]d pixels",
			d.Pixels)
	}

	b := []byte{
		0x42, 0x4d, 0xf6, 0x3c, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x36, 0x00, 0x00, 0x00, 0x28, 0x00,
		0x00, 0x00, 0x48, 0x00, 0x00, 0x00, 0x48, 0x00,
		0x00, 0x00, 0x01, 0x00, 0x18, 0x00, 0x00, 0x00,
		0x00, 0x00, 0xc0, 0x3c, 0x00, 0x00, 0xc4, 0x0e,
		0x00, 0x00, 0xc4, 0x0e, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	}

	for y := 0; y < rgba.Bounds().Dy(); y++ {
		// flip image horizontally
		for x := rgba.Bounds().Dx() - 1; x >= 0; x-- {
			c := rgba.RGBAAt(x, y)
			b = append(b, c.B)
			b = append(b, c.G)
			b = append(b, c.R)
		}
	}

	var page uint8
	rem := uint(len(b))

	var key uint8
	for rem > 0 {
		sent := uint(page) * d.pageLength

		// this reports length
		l := rem
		if l > d.pageLength {
			l = d.pageLength
		}

		// last page?
		if l == rem {
			key = 1
		}

		payload := []byte{
			0x02, 0x01, page + d.startPage, 0x00, key, d.translateKeyIndex(index) + 1, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		}
		payload = append(payload, b[sent:sent+l]...)

		_, err := d.device.Write(payload)
		if err != nil {
			return err
		}

		rem = rem - l
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

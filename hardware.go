package streamdeck

import (
	"bytes"
	"image"
	"image/jpeg"

	"github.com/karalabe/hid"
	"golang.org/x/image/draw"
)

// hardware encapsulates the different protocols and behavior of the different hardware variants.
type hardware interface {
	FirmwareVersion() (string, error)
	Reset() error
	SetBrightness(percent uint8) error
	ReadKeyState(state []byte) error
	GetImageData(img image.Image) (*imageData, error)
	GetImagePageHeader(pageIndex int, keyIndex uint8, payloadLength int, lastPage bool) []byte
	ImagePageSize() int
}

// setup and data used by the different hardware variants.
type setup struct {
	device              *hid.Device
	keyBuffer           []byte
	featureReportSize   int
	imagePageSize       int
	imagePageHeaderSize int
}

// getFeatureReport from the device without worries about the correct payload size.
func (s setup) getFeatureReport(payload ...byte) ([]byte, error) {
	b := make([]byte, s.featureReportSize)
	copy(b, payload)
	_, err := s.device.GetFeatureReport(b)
	if err != nil {
		return nil, err
	}
	return b, nil
}

// sendFeatureReport to the device without worries about the correct payload size.
func (s setup) sendFeatureReport(payload ...byte) error {
	b := make([]byte, s.featureReportSize)
	copy(b, payload)
	_, err := s.device.SendFeatureReport(b)
	return err
}

/*
	Stream Deck, Stream Deck Mini
*/

// classicHardware implements the protocol and behavior used by the original Stream Deck and the Stream Deck Mini.
type classicHardware struct {
	setup
}

// newClassicHardware creates the hardware instance for use with an original Stream Deck.
func newClassicHardware(device *hid.Device) hardware {
	return &classicHardware{
		setup: setup{
			device:              device,
			keyBuffer:           make([]byte, 1+5*3), // Offset + Columns * Rows
			featureReportSize:   17,
			imagePageSize:       8192,
			imagePageHeaderSize: 16,
		},
	}
}

// newMiniHardware create the hardware instance for use with a Stream Deck Mini.
func newMiniHardware(device *hid.Device) hardware {
	return &classicHardware{
		setup: setup{
			device:              device,
			keyBuffer:           make([]byte, 1+3*2), // Offset + Columns * Rows
			featureReportSize:   17,
			imagePageSize:       1024,
			imagePageHeaderSize: 16,
		},
	}
}

func (h *classicHardware) FirmwareVersion() (string, error) {
	result, err := h.getFeatureReport(0x04)
	if err != nil {
		return "", err
	}
	return string(result[5:]), nil
}

func (h *classicHardware) Reset() error {
	return h.sendFeatureReport(0x0B, 0x63)
}

func (h *classicHardware) SetBrightness(percent uint8) error {
	return h.sendFeatureReport(0x05, 0x55, 0xaa, 0xd1, 0x01, percent)
}

func (h *classicHardware) ReadKeyState(state []byte) error {
	_, err := h.device.Read(h.keyBuffer)
	if err != nil {
		return err
	}
	copy(state, h.keyBuffer[1:])
	return nil
}

func (h *classicHardware) GetImageData(img image.Image) (*imageData, error) {
	rgba := toRGBA(img)

	// this is a BMP file header followed by a BPM bitmap info header
	// find more information here: https://en.wikipedia.org/wiki/BMP_file_format
	header := []byte{
		0x42, 0x4d, 0xf6, 0x3c, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x36, 0x00, 0x00, 0x00, 0x28, 0x00,
		0x00, 0x00, 0x48, 0x00, 0x00, 0x00, 0x48, 0x00,
		0x00, 0x00, 0x01, 0x00, 0x18, 0x00, 0x00, 0x00,
		0x00, 0x00, 0xc0, 0x3c, 0x00, 0x00, 0xc4, 0x0e,
		0x00, 0x00, 0xc4, 0x0e, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	}

	buffer := make([]byte, len(header)+rgba.Bounds().Dx()*rgba.Bounds().Dy()*3)
	copy(buffer, header)

	i := len(header)
	for y := 0; y < rgba.Bounds().Dy(); y++ {
		// flip image horizontally
		for x := rgba.Bounds().Dx() - 1; x >= 0; x-- {
			c := rgba.RGBAAt(x, y)
			buffer[i] = c.B
			buffer[i+1] = c.G
			buffer[i+2] = c.R
			i += 3
		}
	}
	return &imageData{
		image:    buffer,
		pageSize: h.imagePageSize - h.imagePageHeaderSize,
	}, nil
}

func (h *classicHardware) GetImagePageHeader(pageIndex int, keyIndex uint8, payloadLength int, lastPage bool) []byte {
	var lastPageByte byte
	if lastPage {
		lastPageByte = 0
	}
	return []byte{
		0x02, 0x01,
		byte(pageIndex + 1), 0x00,
		lastPageByte,
		keyIndex + 1,
	}
}

func (h *classicHardware) ImagePageSize() int {
	return h.imagePageSize
}

/*
	Stream Deck XL
*/

// xlHardware implements the protocol and behavior of the Stream Deck XL.
type xlHardware struct {
	setup
}

// newXLHardware creates the hardware instance for use with a Stream Deck XL.
func newXLHardware(device *hid.Device) hardware {
	return &xlHardware{
		setup: setup{
			device:              device,
			keyBuffer:           make([]byte, 4+8*4), // Offset + Columns * Rows
			featureReportSize:   32,
			imagePageSize:       1024,
			imagePageHeaderSize: 8,
		},
	}
}
func (h *xlHardware) FirmwareVersion() (string, error) {
	result, err := h.getFeatureReport(0x05)
	if err != nil {
		return "", err
	}
	return string(result[6:]), nil
}

func (h *xlHardware) Reset() error {
	return h.sendFeatureReport(0x03, 0x02)
}

func (h *xlHardware) SetBrightness(percent uint8) error {
	return h.sendFeatureReport(0x03, 0x08, percent)
}

func (h *xlHardware) ReadKeyState(state []byte) error {
	_, err := h.device.Read(h.keyBuffer)
	if err != nil {
		return err
	}
	copy(state, h.keyBuffer[4:])
	return nil
}

func (h *xlHardware) GetImageData(img image.Image) (*imageData, error) {
	// flip image horizontally and vertically
	flipped := image.NewRGBA(img.Bounds())
	draw.Copy(flipped, image.ZP, img, img.Bounds(), draw.Src, nil)
	for y := 0; y < flipped.Bounds().Dy()/2; y++ {
		yy := flipped.Bounds().Max.Y - y - 1
		for x := 0; x < flipped.Bounds().Dx(); x++ {
			xx := flipped.Bounds().Max.X - x - 1

			c := flipped.RGBAAt(x, y)
			flipped.SetRGBA(x, y, flipped.RGBAAt(xx, yy))
			flipped.SetRGBA(xx, yy, c)
		}
	}

	buffer := bytes.NewBuffer([]byte{})
	err := jpeg.Encode(buffer, flipped, nil)
	if err != nil {
		return nil, err
	}
	return &imageData{
		image:    buffer.Bytes(),
		pageSize: h.imagePageSize - h.imagePageHeaderSize,
	}, nil
}

func (h *xlHardware) GetImagePageHeader(pageIndex int, keyIndex uint8, payloadLength int, lastPage bool) []byte {
	var lastPageByte byte
	if lastPage {
		lastPageByte = 1
	}
	return []byte{
		0x02, 0x07, keyIndex, lastPageByte,
		byte(payloadLength), byte(payloadLength >> 8),
		byte(pageIndex), byte(pageIndex >> 8),
	}
}

func (h *xlHardware) ImagePageSize() int {
	return h.imagePageSize
}

/*
	Image Data
*/

// imageData allows to access raw image data in a byte array through pages of a given size.
type imageData struct {
	image    []byte
	pageSize int
}

// Page returns the page with the given index and an indication if this is the last page.
func (d imageData) Page(pageIndex int) ([]byte, bool) {
	offset := pageIndex * d.pageSize
	if offset >= len(d.image) {
		return []byte{}, true
	}

	length := d.pageLength(pageIndex)
	if offset+length > len(d.image) {
		length = len(d.image) - offset
	}

	return d.image[offset : offset+length], pageIndex == d.PageCount()-1
}

func (d imageData) pageLength(pageIndex int) int {
	remaining := len(d.image) - (pageIndex * d.pageSize)
	if remaining > d.pageSize {
		return d.pageSize
	}
	if remaining > 0 {
		return remaining
	}
	return 0
}

// PageCount returns the total number of pages.
func (d imageData) PageCount() int {
	count := len(d.image) / d.pageSize
	if len(d.image)%d.pageSize != 0 {
		return count + 1
	}
	return count
}

// Length of the raw image data in bytes.
func (d imageData) Length() int {
	return len(d.image)
}

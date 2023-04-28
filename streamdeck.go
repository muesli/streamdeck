package streamdeck

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"math"
	"sync"
	"time"

	"github.com/karalabe/hid"
	"golang.org/x/image/draw"
)

const (
	// 30 fps fade animation.
	fadeDelay = time.Second / 30
)

// Stream Deck Vendor & Product IDs.
//
//nolint:revive
const (
	VID_ELGATO              = 0x0fd9
	PID_STREAMDECK          = 0x0060
	PID_STREAMDECK_V2       = 0x006d
	PID_STREAMDECK_MK2      = 0x0080
	PID_STREAMDECK_MINI     = 0x0063
	PID_STREAMDECK_MINI_MK2 = 0x0090
	PID_STREAMDECK_XL       = 0x006c
	PID_STREAMDECK_PLUS     = 0x0084

	INPUT_TYPE_ID_BUTTON = uint8(0)
	INPUT_TYPE_ID_TOUCH  = uint8(2)
	INPUT_TYPE_ID_KNOB   = uint8(3)

	INPUT_KNOB_USAGE_PRESS  = uint8(0)
	INPUT_KNOB_USAGE_DIAL   = uint8(1)
	INPUT_KNOB_STATE_OFFSET = uint8(5)

	INPUT_TOUCH_USAGE_SHORT = uint8(1)
	INPUT_TOUCH_USAGE_LONG  = uint8(2)
	INPUT_TOUCH_USAGE_SWIPE = uint8(3)

	INPUT_POSITION_TYPE_ID        = uint8(1)
	INPUT_POSITION_KNOB_USAGE_ID  = uint8(4)
	INPUT_POSITION_TOUCH_USAGE_ID = uint8(4)
	INPUT_POSITION_TOUCH_X_ID     = uint8(6)
	INPUT_POSITION_TOUCH_Y_ID     = uint8(8)
	INPUT_POSITION_TOUCH_X2_ID    = uint8(10)
	INPUT_POSITION_TOUCH_Y2_ID    = uint8(12)
)

// Firmware command IDs.
//
//nolint:revive
var (
	c_REV1_FIRMWARE   = []byte{0x04}
	c_REV1_RESET      = []byte{0x0b, 0x63}
	c_REV1_BRIGHTNESS = []byte{0x05, 0x55, 0xaa, 0xd1, 0x01}

	c_REV2_FIRMWARE   = []byte{0x05}
	c_REV2_RESET      = []byte{0x03, 0x02}
	c_REV2_BRIGHTNESS = []byte{0x03, 0x08}
)

// Device represents a single Stream Deck device.
type Device struct {
	ID     string
	Serial string

	Columns uint8
	Rows    uint8
	Keys    uint8
	Pixels  uint
	DPI     uint
	Padding uint

	ScreenWidth          uint
	ScreenHeight         uint
	ScreenVerticalDPI    uint
	ScreenHorizontalDPI  uint
	ScreenSegmentsAmount uint8

	Knobs uint8

	featureReportSize    int
	firmwareOffset       int
	keyStateOffset       int
	translateKeyIndex    func(index, columns uint8) uint8
	readKeys             func(*Device) (chan Key, error)
	imagePageSize        int
	imagePageHeaderSize  int
	flipImage            func(image.Image) image.Image
	toImageFormat        func(image.Image) ([]byte, error)
	imagePageHeader      func(pageIndex int, keyIndex uint8, payloadLength int, lastPage bool) []byte
	screenPageSize       int
	screenPageHeaderSize int
	screenPageHeader     func(page int, position image.Point, width uint, height uint, payloadLength int, lastPage bool) []byte

	getFirmwareCommand   []byte
	resetCommand         []byte
	setBrightnessCommand []byte

	keyState []byte

	device *hid.Device
	info   hid.DeviceInfo

	lastActionTime time.Time
	asleep         bool
	sleepCancel    context.CancelFunc
	sleepMutex     *sync.RWMutex
	fadeDuration   time.Duration

	brightness         uint8
	preSleepBrightness uint8
}

// Key holds the current status of a key on the device.
type Key struct {
	Index    uint8
	Pressed  bool
	Holdable bool
}

// Devices returns all attached Stream Decks.
func Devices() ([]Device, error) {
	dd := []Device{}

	devs := hid.Enumerate(VID_ELGATO, 0)
	for _, d := range devs {
		var dev Device

		switch {
		case d.VendorID == VID_ELGATO && d.ProductID == PID_STREAMDECK:
			dev = Device{
				ID:                   d.Path,
				Serial:               d.Serial,
				Columns:              5,
				Rows:                 3,
				Keys:                 15,
				Pixels:               72,
				DPI:                  124,
				Padding:              16,
				featureReportSize:    17,
				firmwareOffset:       5,
				keyStateOffset:       1,
				translateKeyIndex:    translateRightToLeft,
				readKeys:             readKeysForButtonsOnlyInput,
				imagePageSize:        7819,
				imagePageHeaderSize:  16,
				imagePageHeader:      rev1ImagePageHeader,
				flipImage:            flipHorizontally,
				toImageFormat:        toBMP,
				getFirmwareCommand:   c_REV1_FIRMWARE,
				resetCommand:         c_REV1_RESET,
				setBrightnessCommand: c_REV1_BRIGHTNESS,
			}
		case d.VendorID == VID_ELGATO && (d.ProductID == PID_STREAMDECK_MINI || d.ProductID == PID_STREAMDECK_MINI_MK2):
			dev = Device{
				ID:                   d.Path,
				Serial:               d.Serial,
				Columns:              3,
				Rows:                 2,
				Keys:                 6,
				Pixels:               80,
				DPI:                  138,
				Padding:              16,
				featureReportSize:    17,
				firmwareOffset:       5,
				keyStateOffset:       1,
				translateKeyIndex:    identity,
				readKeys:             readKeysForButtonsOnlyInput,
				imagePageSize:        1024,
				imagePageHeaderSize:  16,
				imagePageHeader:      miniImagePageHeader,
				flipImage:            rotateCounterclockwise,
				toImageFormat:        toBMP,
				getFirmwareCommand:   c_REV1_FIRMWARE,
				resetCommand:         c_REV1_RESET,
				setBrightnessCommand: c_REV1_BRIGHTNESS,
			}
		case d.VendorID == VID_ELGATO && (d.ProductID == PID_STREAMDECK_V2 || d.ProductID == PID_STREAMDECK_MK2):
			dev = Device{
				ID:                   d.Path,
				Serial:               d.Serial,
				Columns:              5,
				Rows:                 3,
				Keys:                 15,
				Pixels:               72,
				DPI:                  124,
				Padding:              16,
				featureReportSize:    32,
				firmwareOffset:       6,
				keyStateOffset:       4,
				translateKeyIndex:    identity,
				readKeys:             readKeysForButtonsOnlyInput,
				imagePageSize:        1024,
				imagePageHeaderSize:  8,
				imagePageHeader:      rev2ImagePageHeader,
				flipImage:            flipHorizontallyAndVertically,
				toImageFormat:        toJPEG,
				getFirmwareCommand:   c_REV2_FIRMWARE,
				resetCommand:         c_REV2_RESET,
				setBrightnessCommand: c_REV2_BRIGHTNESS,
			}
		case d.VendorID == VID_ELGATO && d.ProductID == PID_STREAMDECK_XL:
			dev = Device{
				ID:                   d.Path,
				Serial:               d.Serial,
				Columns:              8,
				Rows:                 4,
				Keys:                 32,
				Pixels:               96,
				DPI:                  166,
				Padding:              16,
				featureReportSize:    32,
				firmwareOffset:       6,
				keyStateOffset:       4,
				translateKeyIndex:    identity,
				readKeys:             readKeysForButtonsOnlyInput,
				imagePageSize:        1024,
				imagePageHeaderSize:  8,
				imagePageHeader:      rev2ImagePageHeader,
				flipImage:            flipHorizontallyAndVertically,
				toImageFormat:        toJPEG,
				getFirmwareCommand:   c_REV2_FIRMWARE,
				resetCommand:         c_REV2_RESET,
				setBrightnessCommand: c_REV2_BRIGHTNESS,
			}
		case d.VendorID == VID_ELGATO && d.ProductID == PID_STREAMDECK_PLUS:
			dev = Device{
				ID:                   d.Path,
				Serial:               d.Serial,
				Columns:              4,
				Rows:                 2,
				Keys:                 30,
				Pixels:               120,
				DPI:                  180,
				Padding:              16,
				ScreenWidth:          800,
				ScreenHeight:         100,
				ScreenVerticalDPI:    181, //14mm and 100px
				ScreenHorizontalDPI:  188, //108mm and 800px
				ScreenSegmentsAmount: 4,
				Knobs:                4,
				featureReportSize:    32,
				firmwareOffset:       6,
				keyStateOffset:       4,
				translateKeyIndex:    identity,
				readKeys:             readKeysForMultipleInputTypes,
				imagePageSize:        1024,
				imagePageHeaderSize:  8,
				imagePageHeader:      rev2ImagePageHeader,
				toImageFormat:        toJPEG,
				screenPageSize:       1024,
				screenPageHeaderSize: 16,
				screenPageHeader:     touchScreenImagePageHeader,
				getFirmwareCommand:   c_REV2_FIRMWARE,
				resetCommand:         c_REV2_RESET,
				setBrightnessCommand: c_REV2_BRIGHTNESS,
			}
		}

		if dev.ID != "" {
			dev.keyState = make([]byte, dev.Keys)
			dev.info = d
			dd = append(dd, dev)
		}
	}

	return dd, nil
}

// Open the device for input/output. This must be called before trying to
// communicate with the device.
func (d *Device) Open() error {
	var err error
	d.device, err = d.info.Open()
	d.lastActionTime = time.Now()
	d.sleepMutex = &sync.RWMutex{}
	return err
}

// Close the connection with the device.
func (d *Device) Close() error {
	d.cancelSleepTimer()
	return d.device.Close()
}

// FirmwareVersion returns the firmware version of the device.
func (d Device) FirmwareVersion() (string, error) {
	result, err := d.getFeatureReport(d.getFirmwareCommand)
	if err != nil {
		return "", err
	}
	return string(result[d.firmwareOffset:]), nil
}

// Resets the Stream Deck, clears all button images and shows the standby image.
func (d Device) Reset() error {
	return d.sendFeatureReport(d.resetCommand)
}

// Clears the Stream Deck, setting a black image on all buttons.
func (d Device) Clear() error {
	img := image.NewRGBA(image.Rect(0, 0, int(d.Pixels), int(d.Pixels)))
	draw.Draw(img, img.Bounds(), image.NewUniform(color.RGBA{0, 0, 0, 255}), image.Point{}, draw.Src)
	for i := uint8(0); i <= d.Columns*d.Rows; i++ {
		err := d.SetImage(i, img)
		if err != nil {
			fmt.Println(err)
			return err
		}
	}

	return nil
}

// ReadKeys returns a channel, which it will use to emit key presses/releases.
func (d *Device) ReadKeys() (chan Key, error) {
	return d.readKeys(d)
}

func readKeysForButtonsOnlyInput(d *Device) (chan Key, error) {
	kch := make(chan Key)
	keyBuffer := make([]byte, d.keyStateOffset+len(d.keyState))
	go func() {
		for {
			copy(d.keyState, keyBuffer[d.keyStateOffset:])

			if _, err := d.device.Read(keyBuffer); err != nil {
				close(kch)
				return
			}

			if d.isAwakened() {
				resetKeysStates(d, keyBuffer)
				// Dont trigger a key event, because the key awoke the device
				continue
			}

			d.updateLastActionTimeToNow()

			d.sendButtonKeyEventsToChannel(keyBuffer, kch)
		}
	}()

	return kch, nil
}

func readKeysForMultipleInputTypes(device *Device) (chan Key, error) {
	kch := make(chan Key)
	inputBuffer := make([]byte, 13)
	go func() {
		for {
			if _, err := device.device.Read(inputBuffer); err != nil {
				close(kch)
				return
			}

			if device.isAwakened() {
				resetKeysStates(device, inputBuffer)
				// Dont trigger a key event, because the key awoke the device
				continue
			}

			device.updateLastActionTimeToNow()

			inputType := inputBuffer[INPUT_POSITION_TYPE_ID]

			if inputType == INPUT_TYPE_ID_BUTTON {
				device.sendButtonKeyEventsToChannel(inputBuffer, kch)
			} else if inputType == INPUT_TYPE_ID_KNOB {
				device.sendKnobEventsToChannel(inputBuffer, kch)
			} else if inputType == INPUT_TYPE_ID_TOUCH {
				device.sendTouchEventsToChannel(inputBuffer, kch)
			}
		}
	}()

	return kch, nil
}

func (d *Device) sendTouchEventsToChannel(inputBuffer []byte, kch chan Key) {
	touchUsage := inputBuffer[INPUT_POSITION_TOUCH_USAGE_ID]

	x := binary.LittleEndian.Uint16(inputBuffer[INPUT_POSITION_TOUCH_X_ID:])

	segmentWidth := d.ScreenSegmentWidth()
	segment := uint8(math.Floor(float64(x) / float64(segmentWidth)))

	var keyIndex uint8

	if touchUsage == INPUT_TOUCH_USAGE_SHORT {
		keyIndex = d.Columns*d.Rows + 3*d.Knobs + segment
	} else if touchUsage == INPUT_TOUCH_USAGE_LONG {
		keyIndex = d.Columns*d.Rows + 3*d.Knobs + d.ScreenSegmentsAmount + segment
	} else if touchUsage == INPUT_TOUCH_USAGE_SWIPE {
		x2 := binary.LittleEndian.Uint16(inputBuffer[INPUT_POSITION_TOUCH_X2_ID:])
		startSegment := uint8(math.Floor(float64(x) / 40.0))
		stopSegment := uint8(math.Floor(float64(x2) / 40.0))

		if startSegment < stopSegment { //left to right
			keyIndex = d.Columns*d.Rows + 3*d.Knobs + 2*d.ScreenSegmentsAmount
		} else if startSegment > stopSegment { //right to left
			keyIndex = d.Columns*d.Rows + 3*d.Knobs + 2*d.ScreenSegmentsAmount + 1
		} else {
			return
		}
	}
	kch <- Key{
		Index:    keyIndex,
		Pressed:  true,
		Holdable: false,
	}
}

func (d *Device) sendKnobEventsToChannel(inputBuffer []byte, kch chan Key) {
	knobUsage := inputBuffer[INPUT_POSITION_KNOB_USAGE_ID]

	for i := INPUT_KNOB_STATE_OFFSET; i < INPUT_KNOB_STATE_OFFSET+d.Knobs; i++ {
		keyValue := inputBuffer[i]

		if knobUsage == INPUT_KNOB_USAGE_PRESS {
			keyIndex := i - INPUT_KNOB_STATE_OFFSET + d.Columns*d.Rows

			if keyValue != d.keyState[keyIndex] {
				d.keyState[keyIndex] = keyValue

				kch <- Key{
					Index:    keyIndex,
					Pressed:  keyValue == 1,
					Holdable: true,
				}
			}
		} else if knobUsage == INPUT_KNOB_USAGE_DIAL && inputBuffer[i] > 0 {
			var keyIndex uint8

			if int(keyValue)-128 > 0 { //left turn
				keyIndex = i - INPUT_KNOB_STATE_OFFSET + d.Columns*d.Rows + d.Knobs
			} else { //right turn
				keyIndex = i - INPUT_KNOB_STATE_OFFSET + d.Columns*d.Rows + 2*d.Knobs
			}

			kch <- Key{
				Index:    keyIndex,
				Pressed:  true,
				Holdable: false,
			}
		}
	}
}

func (d *Device) sendButtonKeyEventsToChannel(inputBuffer []byte, kch chan Key) {
	for i := d.keyStateOffset; i < len(inputBuffer); i++ {
		keyIndex := uint8(i - d.keyStateOffset)
		if inputBuffer[i] != d.keyState[keyIndex] {
			d.keyState[keyIndex] = inputBuffer[i]
			kch <- Key{
				Index:    keyIndex,
				Pressed:  inputBuffer[i] == 1,
				Holdable: true,
			}
		}
	}
}

func (d *Device) updateLastActionTimeToNow() {
	d.sleepMutex.Lock()
	d.lastActionTime = time.Now()
	d.sleepMutex.Unlock()
}

// ScreenSegmentWidth returns the width of a screen segment. Returns 0 if there are no segments.
func (d *Device) ScreenSegmentWidth() uint {
	if d.ScreenSegmentsAmount == 0 {
		return 0
	}
	return d.ScreenWidth / uint(d.ScreenSegmentsAmount)
}

// ScreenSegmentHeight returns the width of a screen segment. Returns 0 if there are no segments.
func (d *Device) ScreenSegmentHeight() uint {
	if d.ScreenSegmentsAmount == 0 {
		return 0
	}
	return d.ScreenHeight
}

// Sleep puts the device asleep, waiting for a key event to wake it up.
func (d *Device) Sleep() error {
	d.sleepMutex.Lock()
	defer d.sleepMutex.Unlock()

	d.preSleepBrightness = d.brightness

	if err := d.Fade(d.brightness, 0, d.fadeDuration); err != nil {
		return err
	}

	d.asleep = true
	return d.SetBrightness(0)
}

// Wake wakes the device from sleep.
func (d *Device) Wake() error {
	d.sleepMutex.Lock()
	defer d.sleepMutex.Unlock()

	d.asleep = false
	if err := d.Fade(0, d.preSleepBrightness, d.fadeDuration); err != nil {
		return err
	}

	d.lastActionTime = time.Now()
	return d.SetBrightness(d.preSleepBrightness)
}

// Asleep returns true if the device is asleep.
func (d Device) Asleep() bool {
	return d.asleep
}

func (d *Device) cancelSleepTimer() {
	if d.sleepCancel == nil {
		return
	}

	d.sleepCancel()
	d.sleepCancel = nil
}

// SetSleepFadeDuration sets the duration of the fading animation when the
// device is put to sleep or wakes up.
func (d *Device) SetSleepFadeDuration(t time.Duration) {
	d.fadeDuration = t
}

// SetSleepTimeout sets the time after which the device will sleep if no key
// events are received.
func (d *Device) SetSleepTimeout(t time.Duration) {
	d.cancelSleepTimer()
	if t == 0 {
		return
	}

	var ctx context.Context
	ctx, d.sleepCancel = context.WithCancel(context.Background())

	go func() {
		for {
			select {
			case <-time.After(time.Second):
				d.sleepMutex.RLock()
				since := time.Since(d.lastActionTime)
				d.sleepMutex.RUnlock()

				if !d.asleep && since >= t {
					_ = d.Sleep()
				}

			case <-ctx.Done():
				return
			}
		}
	}()
}

// Fade fades the brightness in or out.
func (d *Device) Fade(start uint8, end uint8, duration time.Duration) error {
	step := (float64(end) - float64(start)) / float64(duration/fadeDelay)
	if step == math.Inf(1) || step == math.Inf(-1) {
		return nil
	}

	for current := float64(start); ; current += step {
		if !((start < end && int8(current) < int8(end)) ||
			(start > end && int8(current) > int8(end))) {
			break
		}
		if err := d.SetBrightness(uint8(current)); err != nil {
			return err
		}

		time.Sleep(fadeDelay)
	}
	return nil
}

// SetBrightness sets the background lighting brightness from 0 to 100 percent.
func (d *Device) SetBrightness(percent uint8) error {
	if percent > 100 {
		percent = 100
	}

	d.brightness = percent
	if d.asleep && percent > 0 {
		// if the device is asleep, remember the brightness, but don't set it
		d.sleepMutex.Lock()
		d.preSleepBrightness = percent
		d.sleepMutex.Unlock()
		return nil
	}

	report := make([]byte, len(d.setBrightnessCommand)+1)
	copy(report, d.setBrightnessCommand)
	report[len(report)-1] = percent

	return d.sendFeatureReport(report)
}

// SetImage sets the image of a button on the Stream Deck. The provided image
// needs to be in the correct resolution for the device. The index starts with
// 0 being the top-left button.
func (d Device) SetImage(index uint8, img image.Image) error {
	if img.Bounds().Dy() != int(d.Pixels) ||
		img.Bounds().Dx() != int(d.Pixels) {
		return fmt.Errorf("supplied image has wrong dimensions, expected %[1]dx%[1]d pixels", d.Pixels)
	}

	imageBytes, err := d.transformImage(img)
	if err != nil {
		return fmt.Errorf("cannot convert image data: %v", err)
	}
	imageData := imageData{
		image:    imageBytes,
		pageSize: d.imagePageSize - d.imagePageHeaderSize,
	}

	data := make([]byte, d.imagePageSize)

	var page int
	var lastPage bool
	for !lastPage {
		var payload []byte
		payload, lastPage = imageData.Page(page)
		header := d.imagePageHeader(page, d.translateKeyIndex(index, d.Columns), len(payload), lastPage)

		copy(data, header)
		copy(data[len(header):], payload)

		_, err := d.device.Write(data)
		if err != nil {
			return fmt.Errorf("cannot write image page %d of %d (%d image bytes) %d bytes: %v",
				page, imageData.PageCount(), imageData.Length(), len(data), err)
		}

		page++
	}

	return nil
}

// SetTouchScreenSegmentImage sets the image of a segment of the Stream Deck Plus touch screen. The provided image
// needs to be in the correct resolution for the device. The index starts with
// 0 to Device.ScreenSegmentsAmount-1.
func (d Device) SetTouchScreenSegmentImage(segmentIndex uint8, img image.Image) error {
	position := image.Point{
		X: int(uint(segmentIndex) * d.ScreenSegmentWidth()),
		Y: 0,
	}

	return d.SetTouchScreenImage(position, d.ScreenSegmentWidth(), d.ScreenSegmentHeight(), img)
}

// SetTouchScreenImage sets the image of the Stream Deck Plus touch screen at the given point. The provided image
// needs to be in the correct resolution for the device.
func (d Device) SetTouchScreenImage(position image.Point, width uint, height uint, img image.Image) error {
	imageBytes, err := d.transformImage(img)

	if err != nil {
		return fmt.Errorf("cannot convert image data: %v", err)
	}

	imageData := imageData{
		image:    imageBytes,
		pageSize: d.screenPageSize - d.screenPageHeaderSize,
	}

	data := make([]byte, d.screenPageSize)

	var page int
	var lastPage bool
	for !lastPage {
		var payload []byte
		payload, lastPage = imageData.Page(page)
		header := d.screenPageHeader(page, position, width, height, len(payload), lastPage)

		copy(data, header)
		copy(data[len(header):], payload)

		_, err := d.device.Write(data)
		if err != nil {
			return fmt.Errorf("cannot write image page %d of %d (%d image bytes) %d bytes: %v",
				page, imageData.PageCount(), imageData.Length(), len(data), err)
		}

		page++
	}

	return nil
}

// getFeatureReport from the device without worries about the correct payload
// size.
func (d Device) getFeatureReport(payload []byte) ([]byte, error) {
	b := make([]byte, d.featureReportSize)
	copy(b, payload)
	_, err := d.device.GetFeatureReport(b)
	if err != nil {
		return nil, err
	}
	return b, nil
}

// sendFeatureReport to the device without worries about the correct payload
// size.
func (d Device) sendFeatureReport(payload []byte) error {
	b := make([]byte, d.featureReportSize)
	copy(b, payload)
	_, err := d.device.SendFeatureReport(b)
	return err
}

// translateRightToLeft translates the given key index from right-to-left to
// left-to-right, based on the given number of columns.
func translateRightToLeft(index, columns uint8) uint8 {
	keyCol := index % columns
	return (index - keyCol) + (columns - 1) - keyCol
}

// identity returns the given key index as it is.
func identity(index, _ uint8) uint8 {
	return index
}

// toRGBA converts an image.Image to an image.RGBA.
func toRGBA(img image.Image) *image.RGBA {
	switch img := img.(type) {
	case *image.RGBA:
		return img
	}
	out := image.NewRGBA(img.Bounds())
	draw.Copy(out, image.Pt(0, 0), img, img.Bounds(), draw.Src, nil)
	return out
}

// transformImage transforms the image for sending it to the device.
func (d *Device) transformImage(img image.Image) ([]byte, error) {
	if d.flipImage != nil {
		img = d.flipImage(img)
	}

	return d.toImageFormat(img)
}

func (d *Device) isAwakened() bool {
	if d.asleep {
		_ = d.Wake()
		return true
	}

	return false
}

func resetKeysStates(device *Device, inputBuffer []byte) {
	// reset state so no spurious key events get triggered
	for i := device.keyStateOffset; i < len(inputBuffer); i++ {
		inputBuffer[i] = 0
	}
}

// flipHorizontally returns the given image horizontally flipped.
func flipHorizontally(img image.Image) image.Image {
	flipped := image.NewRGBA(img.Bounds())
	draw.Copy(flipped, image.Point{}, img, img.Bounds(), draw.Src, nil)
	for y := 0; y < flipped.Bounds().Dy(); y++ {
		for x := 0; x < flipped.Bounds().Dx()/2; x++ {
			xx := flipped.Bounds().Max.X - x - 1
			c := flipped.RGBAAt(x, y)
			flipped.SetRGBA(x, y, flipped.RGBAAt(xx, y))
			flipped.SetRGBA(xx, y, c)
		}
	}
	return flipped
}

// flipHorizontallyAndVertically returns the given image horizontally and
// vertically flipped.
func flipHorizontallyAndVertically(img image.Image) image.Image {
	flipped := image.NewRGBA(img.Bounds())
	draw.Copy(flipped, image.Point{}, img, img.Bounds(), draw.Src, nil)
	for y := 0; y < flipped.Bounds().Dy()/2; y++ {
		yy := flipped.Bounds().Max.Y - y - 1
		for x := 0; x < flipped.Bounds().Dx(); x++ {
			xx := flipped.Bounds().Max.X - x - 1
			c := flipped.RGBAAt(x, y)
			flipped.SetRGBA(x, y, flipped.RGBAAt(xx, yy))
			flipped.SetRGBA(xx, yy, c)
		}
	}
	return flipped
}

// rotateCounterclockwise returns the given image rotated counterclockwise.
func rotateCounterclockwise(img image.Image) image.Image {
	flipped := image.NewRGBA(img.Bounds())
	draw.Copy(flipped, image.Point{}, img, img.Bounds(), draw.Src, nil)
	for y := 0; y < flipped.Bounds().Dy(); y++ {
		for x := y + 1; x < flipped.Bounds().Dx(); x++ {
			c := flipped.RGBAAt(x, y)
			flipped.SetRGBA(x, y, flipped.RGBAAt(y, x))
			flipped.SetRGBA(y, x, c)
		}
	}
	for y := 0; y < flipped.Bounds().Dy()/2; y++ {
		yy := flipped.Bounds().Max.Y - y - 1
		for x := 0; x < flipped.Bounds().Dx(); x++ {
			c := flipped.RGBAAt(x, y)
			flipped.SetRGBA(x, y, flipped.RGBAAt(x, yy))
			flipped.SetRGBA(x, yy, c)
		}
	}
	return flipped
}

// toBMP returns the raw bytes of the given image in BMP format.
func toBMP(img image.Image) ([]byte, error) {
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
		for x := 0; x < rgba.Bounds().Dx(); x++ {
			c := rgba.RGBAAt(x, y)
			buffer[i] = c.B
			buffer[i+1] = c.G
			buffer[i+2] = c.R
			i += 3
		}
	}
	return buffer, nil
}

// toJPEG returns the raw bytes of the given image in JPEG format.
func toJPEG(img image.Image) ([]byte, error) {
	buffer := bytes.NewBuffer([]byte{})
	opts := jpeg.Options{
		Quality: 100,
	}
	err := jpeg.Encode(buffer, img, &opts)
	if err != nil {
		return nil, err
	}
	return buffer.Bytes(), err
}

// rev1ImagePageHeader returns the image page header sequence used by the
// Stream Deck v1.
func rev1ImagePageHeader(pageIndex int, keyIndex uint8, payloadLength int, lastPage bool) []byte {
	var lastPageByte byte
	if lastPage {
		lastPageByte = 1
	}
	return []byte{
		0x02, 0x01,
		byte(pageIndex + 1), 0x00,
		lastPageByte,
		keyIndex + 1,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	}
}

// miniImagePageHeader returns the image page header sequence used by the
// Stream Deck Mini.
func miniImagePageHeader(pageIndex int, keyIndex uint8, payloadLength int, lastPage bool) []byte {
	var lastPageByte byte
	if lastPage {
		lastPageByte = 1
	}
	return []byte{
		0x02, 0x01,
		byte(pageIndex), 0x00,
		lastPageByte,
		keyIndex + 1,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	}
}

// rev2ImagePageHeader returns the image page header sequence used by Stream
// Deck XL and Stream Deck v2.
func rev2ImagePageHeader(pageIndex int, keyIndex uint8, payloadLength int, lastPage bool) []byte {
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

// touchScreenImagePageHeader returns the image page header sequence used by Stream
// Deck Plus for the touch screen.
func touchScreenImagePageHeader(page int, position image.Point, width uint, height uint, payloadLength int, lastPage bool) []byte {
	var lastPageByte byte
	if lastPage {
		lastPageByte = 1
	}

	return []byte{
		0x02,                     // 0 Elgato secret flag value #1
		0x0c,                     // 1 Elgato secret flag value #2
		byte(position.X),         // 2 x low byte
		byte(position.X >> 8),    // 3 x high byte
		byte(position.Y),         // 4 y low byte
		byte(position.Y >> 8),    // 5 y high byte
		byte(width),              // 6 width low byte
		byte(width >> 8),         // 7 width high byte
		byte(height),             // 8 height low byte
		byte(height >> 8),        // 9 height high byte
		lastPageByte,             // 10 last page
		byte(page),               // 12 page low byte
		byte(page >> 8),          // 11 page high byte
		byte(payloadLength),      // 14 payload length high byte
		byte(payloadLength >> 8), // 13 payload length high byte
		0x00,                     // 15 padding
	}
}

// imageData allows to access raw image data in a byte array through pages of a
// given size.
type imageData struct {
	image    []byte
	pageSize int
}

// Page returns the page with the given index and an indication if this is the
// last page.
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

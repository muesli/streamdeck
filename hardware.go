package streamdeck

import (
	"github.com/karalabe/hid"
)

const (
	CLASSIC_KEY_OFFSET   = 1
	CLASSIC_PAYLOAD_SIZE = 17
	XL_KEY_OFFSET        = 4
	XL_PAYLOAD_SIZE      = 32
)

type hardware interface {
	FirmwareVersion() (string, error)
	Reset() error
	SetBrightness(percent uint8) error
	ReadKeyState(state []byte) error
}

type classicHardware struct {
	device    *hid.Device
	keyBuffer []byte
}

func newClassicHardware(device *hid.Device, keyCount uint8) hardware {
	return &classicHardware{
		device:    device,
		keyBuffer: make([]byte, CLASSIC_KEY_OFFSET+keyCount),
	}
}

func (h *classicHardware) FirmwareVersion() (string, error) {
	result, err := getFeatureReport(h.device, CLASSIC_PAYLOAD_SIZE, 0x04)
	if err != nil {
		return "", err
	}
	return string(result[5:]), nil
}

func (h *classicHardware) Reset() error {
	return sendFeatureReport(h.device, CLASSIC_PAYLOAD_SIZE, 0x0B, 0x63)
}

func (h *classicHardware) SetBrightness(percent uint8) error {
	if percent > 100 {
		percent = 100
	}
	return sendFeatureReport(h.device, CLASSIC_PAYLOAD_SIZE, 0x05, 0x55, 0xaa, 0xd1, 0x01, percent)
}

func (h *classicHardware) ReadKeyState(state []byte) error {
	_, err := h.device.Read(h.keyBuffer)
	if err != nil {
		return err
	}
	copy(state, h.keyBuffer[CLASSIC_KEY_OFFSET:])
	return nil
}

type xlHardware struct {
	device    *hid.Device
	keyBuffer []byte
}

func newXLHardware(device *hid.Device, keyCount uint8) hardware {
	return &xlHardware{
		device:    device,
		keyBuffer: make([]byte, XL_KEY_OFFSET+keyCount),
	}
}
func (h *xlHardware) FirmwareVersion() (string, error) {
	result, err := getFeatureReport(h.device, XL_PAYLOAD_SIZE, 0x05)
	if err != nil {
		return "", err
	}
	return string(result[6:]), nil
}

func (h *xlHardware) Reset() error {
	return sendFeatureReport(h.device, 0x03, 0x02)
}

func (h *xlHardware) SetBrightness(percent uint8) error {
	if percent > 100 {
		percent = 100
	}
	return sendFeatureReport(h.device, XL_PAYLOAD_SIZE, 0x03, 0x08, percent)
}

func (h *xlHardware) ReadKeyState(state []byte) error {
	_, err := h.device.Read(h.keyBuffer)
	if err != nil {
		return err
	}
	copy(state, h.keyBuffer[XL_KEY_OFFSET:])
	return nil
}

func getFeatureReport(device *hid.Device, size int, payload ...byte) ([]byte, error) {
	b := make([]byte, size)
	copy(b, payload)
	_, err := device.GetFeatureReport(b)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func sendFeatureReport(device *hid.Device, size int, payload ...byte) error {
	b := make([]byte, size)
	copy(b, payload)
	_, err := device.SendFeatureReport(b)
	return err
}

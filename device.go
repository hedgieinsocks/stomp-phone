package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
)

type FootSwitch struct {
	deviceFile *os.File
	buttons    []Button
	writeMu    sync.Mutex
}

func newFootSwitch(devicePath string, buttons []Button) (*FootSwitch, error) {
	preparedButtons := append([]Button(nil), buttons...)
	if err := prepareButtonMessages(preparedButtons); err != nil {
		return nil, err
	}

	deviceFile, err := os.OpenFile(devicePath, os.O_WRONLY, 0)
	if err != nil {
		return nil, fmt.Errorf("open MIDI device: %w", err)
	}

	return &FootSwitch{
		deviceFile: deviceFile,
		buttons:    preparedButtons,
	}, nil
}

func (fs *FootSwitch) Close() error {
	if err := fs.deviceFile.Close(); err != nil {
		return fmt.Errorf("close MIDI device: %w", err)
	}
	return nil
}

func (fs *FootSwitch) TriggerButton(index int, enabled bool) error {
	if index < 0 || index >= len(fs.buttons) {
		return fmt.Errorf("button index %d out of range", index)
	}

	button := &fs.buttons[index]
	var message []byte
	switch button.Type {
	case ButtonTypeClick, ButtonTypeRadio:
		message = button.messageOnBytes
	case ButtonTypeToggle:
		if enabled {
			message = button.messageOnBytes
		} else {
			message = button.messageOffBytes
		}
	default:
		return fmt.Errorf("invalid button type %q", button.Type)
	}

	return fs.writeMessage(message)
}

func (fs *FootSwitch) writeMessage(message []byte) error {
	fs.writeMu.Lock()
	defer fs.writeMu.Unlock()

	written, err := fs.deviceFile.Write(message)
	if err != nil {
		return fmt.Errorf("write MIDI message: %w", err)
	}
	if written != len(message) {
		return fmt.Errorf("write MIDI message: incomplete write (%d of %d bytes)", written, len(message))
	}
	return nil
}

func prepareButtonMessages(buttons []Button) error {
	for i := range buttons {
		button := &buttons[i]

		messageOnBytes, err := parseHexMessage(button.MessageOn)
		if err != nil {
			return fmt.Errorf("buttons[%d]: messageOn: %w", i, err)
		}
		button.messageOnBytes = messageOnBytes

		if button.MessageOff != "" {
			messageOffBytes, err := parseHexMessage(button.MessageOff)
			if err != nil {
				return fmt.Errorf("buttons[%d]: messageOff: %w", i, err)
			}
			button.messageOffBytes = messageOffBytes
		}
	}
	return nil
}

func parseHexMessage(message string) ([]byte, error) {
	fields := strings.Fields(message)
	if len(fields) == 0 {
		return nil, fmt.Errorf("must contain at least one hexadecimal byte")
	}

	messageBytes := make([]byte, len(fields))
	for i, field := range fields {
		value, err := strconv.ParseUint(field, 16, 8)
		if err != nil {
			return nil, fmt.Errorf("invalid MIDI byte %q", field)
		}
		messageBytes[i] = byte(value)
	}
	return messageBytes, nil
}

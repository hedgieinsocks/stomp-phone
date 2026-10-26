package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"go.yaml.in/yaml/v3"
)

const (
	ButtonTypeRadio  = "radio"
	ButtonTypeToggle = "toggle"
	ButtonTypeClick  = "click"

	defaultPort       = 8080
	defaultButtonSize = 100
	defaultColorOn    = "green"
	defaultColorOff   = "red"
)

type Config struct {
	Port    int      `yaml:"port"`
	Device  string   `yaml:"device"`
	Buttons []Button `yaml:"buttons"`
}

type Button struct {
	Type       string `yaml:"type"`
	Caption    string `yaml:"caption"`
	MessageOn  string `yaml:"messageOn"`
	MessageOff string `yaml:"messageOff"`
	Size       int    `yaml:"size"`
	ColorOn    string `yaml:"colorOn"`
	ColorOff   string `yaml:"colorOff"`

	messageOnBytes  []byte `yaml:"-"`
	messageOffBytes []byte `yaml:"-"`
}

func loadConfig(configPath string) (*Config, error) {
	file, err := os.Open(configPath)
	if err != nil {
		return nil, fmt.Errorf("open configuration file: %w", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			log.Printf("close configuration file: %v", err)
		}
	}()

	decoder := yaml.NewDecoder(file)
	decoder.KnownFields(true)

	var config Config
	if err := decoder.Decode(&config); err != nil {
		return nil, fmt.Errorf("decode configuration: %w", err)
	}

	var trailingDocument any
	if err := decoder.Decode(&trailingDocument); err == nil {
		return nil, fmt.Errorf("decode configuration: multiple YAML documents are not supported")
	} else if err != io.EOF {
		return nil, fmt.Errorf("decode trailing configuration data: %w", err)
	}

	if err := normalizeAndValidateConfig(&config); err != nil {
		return nil, fmt.Errorf("validate configuration: %w", err)
	}
	return &config, nil
}

func normalizeAndValidateConfig(config *Config) error {
	if config.Port == 0 {
		config.Port = defaultPort
	}
	if config.Port < 1 || config.Port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535")
	}

	config.Device = strings.TrimSpace(config.Device)
	if config.Device == "" {
		return fmt.Errorf("device is required")
	}

	if len(config.Buttons) < 1 || len(config.Buttons) > 3 {
		return fmt.Errorf("buttons must have 1-3 entries")
	}

	radioButtonCount := 0
	for i := range config.Buttons {
		button := &config.Buttons[i]
		button.Type = strings.ToLower(strings.TrimSpace(button.Type))
		button.Caption = strings.TrimSpace(button.Caption)
		button.MessageOn = strings.TrimSpace(button.MessageOn)
		button.MessageOff = strings.TrimSpace(button.MessageOff)
		button.ColorOn = strings.TrimSpace(button.ColorOn)
		button.ColorOff = strings.TrimSpace(button.ColorOff)

		switch button.Type {
		case ButtonTypeRadio:
			radioButtonCount++
		case ButtonTypeToggle, ButtonTypeClick:
		default:
			return fmt.Errorf("buttons[%d]: invalid type %q", i, button.Type)
		}

		if button.MessageOn == "" {
			return fmt.Errorf("buttons[%d]: messageOn is required", i)
		}
		if button.Type == ButtonTypeToggle && button.MessageOff == "" {
			return fmt.Errorf("buttons[%d]: messageOff is required for toggle type", i)
		}

		if len(button.Caption) > 12 {
			return fmt.Errorf("buttons[%d]: caption must contain 12 characters max", i)
		}
		if button.Size == 0 {
			button.Size = defaultButtonSize
		}
		if button.Size < 10 || button.Size > 200 {
			return fmt.Errorf("buttons[%d]: size must be between 10 and 200", i)
		}
		if button.ColorOn == "" {
			button.ColorOn = defaultColorOn
		}
		if button.ColorOff == "" {
			button.ColorOff = defaultColorOff
		}
	}

	if radioButtonCount == 1 {
		return fmt.Errorf("radio button setup requires at least two radio buttons")
	}
	return nil
}

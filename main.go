package main

import (
	"flag"
	"fmt"
	"log"
	"os"
)

const version = "0.2.5"

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	configPath, showVersion := parseFlags()
	if showVersion {
		fmt.Println(version)
		return nil
	}

	config, err := loadConfig(configPath)
	if err != nil {
		return fmt.Errorf("load configuration: %w", err)
	}

	footSwitch, err := newFootSwitch(config.Device, config.Buttons)
	if err != nil {
		return fmt.Errorf("initialize foot switch: %w", err)
	}
	defer func() {
		if err := footSwitch.Close(); err != nil {
			log.Printf("close MIDI device: %v", err)
		}
	}()

	server := newServer(config.Port, footSwitch)
	return server.serve()
}

func parseFlags() (string, bool) {
	parser := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	parser.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [-f CONFIG] [-v] [-h]\n\n", os.Args[0])
		fmt.Fprintln(os.Stderr, "Use your smartphone as a MIDI footswitch")
		fmt.Fprintln(os.Stderr, "\nOptions:")
		parser.PrintDefaults()
	}

	configPath := parser.String("f", "config.yaml", "configuration file path")
	showVersion := parser.Bool("v", false, "print version and exit")
	_ = parser.Parse(os.Args[1:])

	return *configPath, *showVersion
}

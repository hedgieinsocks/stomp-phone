package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"gopkg.in/yaml.v3"
)

const version = "0.2.2"

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

	messageOn  []byte `yaml:"-"`
	messageOff []byte `yaml:"-"`
}

const (
	ButtonTypeRadio  = "radio"
	ButtonTypeToggle = "toggle"
	ButtonTypeClick  = "click"
)

func validateConfig(c *Config) error {
	if c.Port == 0 {
		c.Port = 8080
	}

	if c.Device == "" {
		return fmt.Errorf("device is required")
	}

	if len(c.Buttons) < 1 || len(c.Buttons) > 3 {
		return fmt.Errorf("buttons: must have 1-3 entries")
	}

	for i := range c.Buttons {
		b := &c.Buttons[i]

		switch b.Type {
		case ButtonTypeRadio, ButtonTypeToggle, ButtonTypeClick:
		default:
			return fmt.Errorf("buttons[%d]: invalid type %q", i, b.Type)
		}

		if b.MessageOn == "" {
			return fmt.Errorf("buttons[%d]: messageOn is required", i)
		}

		if b.Type == ButtonTypeToggle && b.MessageOff == "" {
			return fmt.Errorf("buttons[%d]: messageOff is required for toggle type", i)
		}

		if b.Size == 0 {
			b.Size = 100
		}
	}

	return nil
}

type buttonRequest struct {
	Index   int  `json:"index"`
	Enabled bool `json:"enabled"`
}

type FootSwitch struct {
	device  *os.File
	buttons []Button
}

func (f *FootSwitch) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		f.handleGet(w)

	case http.MethodPost:
		f.handlePost(w, r)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (f *FootSwitch) handleGet(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/html")

	page, err := buildPage(f.buttons)
	if err != nil {
		log.Println(err)
		http.Error(w, "failed to build page", http.StatusInternalServerError)
		return
	}

	if _, err := fmt.Fprint(w, page); err != nil {
		log.Println(err)
	}
}

func (f *FootSwitch) handlePost(w http.ResponseWriter, r *http.Request) {
	var request buttonRequest

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	if request.Index < 0 || request.Index >= len(f.buttons) {
		http.Error(w, "invalid button index", http.StatusBadRequest)
		return
	}

	button := &f.buttons[request.Index]

	var message []byte

	switch button.Type {
	case ButtonTypeClick, ButtonTypeRadio:
		message = button.messageOn

	case ButtonTypeToggle:
		if request.Enabled {
			message = button.messageOn
		} else {
			message = button.messageOff
		}

	default:
		http.Error(w, "invalid button type", http.StatusBadRequest)
		return
	}

	if _, err := f.device.Write(message); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func loadConfig(filename string) (Config, error) {
	file, err := os.Open(filename)
	if err != nil {
		return Config{}, err
	}
	defer func() {
		if err := file.Close(); err != nil {
			log.Println(err)
		}
	}()

	var config Config

	decoder := yaml.NewDecoder(file)
	decoder.KnownFields(true)

	if err := decoder.Decode(&config); err != nil {
		return Config{}, fmt.Errorf("parse config: %w", err)
	}

	if err := validateConfig(&config); err != nil {
		return Config{}, fmt.Errorf("validate config: %w", err)
	}

	return config, nil
}

func parseHex(s string) ([]byte, error) {
	fields := strings.Fields(s)
	result := make([]byte, len(fields))

	for i, field := range fields {
		value, err := strconv.ParseUint(field, 16, 8)
		if err != nil {
			return nil, fmt.Errorf("invalid MIDI byte %q", field)
		}

		result[i] = byte(value)
	}

	return result, nil
}

func localIP() (string, error) {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "", err
	}
	defer func() {
		if err := conn.Close(); err != nil {
			log.Println(err)
		}
	}()

	return conn.LocalAddr().(*net.UDPAddr).IP.String(), nil
}

func buildPage(buttons []Button) (string, error) {
	var html strings.Builder

	if err := pageTemplate.Execute(&html, buttons); err != nil {
		return "", err
	}

	return html.String(), nil
}

var pageTemplate = template.Must(template.New("page").Parse(`
<!doctype html>
<html>
<head>
  <meta name="viewport"
        content="width=device-width, initial-scale=1,
                 maximum-scale=1, user-scalable=no">
  <style>
    html, body {
      width: 100%;
      height: 100%;
      margin: 0;
      overflow: hidden;
      background: #000;
    }

    body {
      display: flex;
      flex-direction: column-reverse;
      box-sizing: border-box;
      border-top: 2px solid yellow;
    }

    .button {
      flex: 1;
      position: relative;
      display: flex;
      align-items: center;
      justify-content: center;
      touch-action: none;
      border: 2px solid yellow;
      border-top: none;
    }

    .button::after {
      content: "";
      width: var(--size);
      height: var(--size);
      border: 2px solid yellow;
      border-radius: 50%;
      background: red;
    }

    .button.pressed::after {
      opacity: 0.5;
    }

    .button.enabled::after {
      background: green;
    }

    .caption {
      position: absolute;
      right: calc(50% + var(--size) / 2 + 10px);
      writing-mode: vertical-rl;
      transform: rotate(180deg);
      color: white;
      font-size: 20px;
    }
  </style>
</head>
<body>
{{ $activeRadio := false }}
{{ range $i, $button := .}}
  <div
    class="button{{ if and (not $activeRadio) (eq $button.Type "radio") }} enabled{{ end }}"
    data-type="{{ $button.Type }}"
    data-index="{{ $i }}"
    style="--size: {{ $button.Size }}px"
  >
    {{ if $button.Caption }}<span class="caption">{{ $button.Caption }}</span>{{ end }}
  </div>
  {{ if eq $button.Type "radio" }}
    {{ $activeRadio = true }}
  {{ end }}
{{ end }}

<script>
  let selectedRadioButton = document.querySelector('.button.enabled[data-type="radio"]');

  document.querySelectorAll('.button').forEach((button, index) => {
    button.addEventListener('touchstart', e => {
      if (e.touches.length > 1 || button === selectedRadioButton) return;
      button.classList.toggle('enabled');
      button.classList.toggle('pressed');
    });

    button.addEventListener('touchend', e => {
      if (e.touches.length > 0 || button === selectedRadioButton) return;
      button.classList.toggle('pressed');

      if (button.dataset.type === 'click') {
        button.classList.toggle('enabled');
      }

      if (button.dataset.type === 'radio') {
        selectedRadioButton.classList.remove('enabled');
        selectedRadioButton = button;
      }

      fetch(location.href, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json'
        },
        body: JSON.stringify({
          index: Number(button.dataset.index),
          enabled: button.classList.contains('enabled'),
        })
      });
    });
  });
</script>
</body>
</html>
`))

func main() {
	parser := flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	parser.Usage = func() {
		fmt.Fprintf(
			os.Stderr,
			"Usage: %s [-f CONFIG] [-v] [-h]\n\n",
			os.Args[0],
		)
		fmt.Fprintln(os.Stderr, "Use your smartphone as a MIDI footswitch")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "Options:")
		parser.PrintDefaults()
	}

	configFlag := parser.String(
		"f",
		"config.yaml",
		"config file path",
	)

	versionFlag := parser.Bool(
		"v",
		false,
		"print version and exit",
	)

	_ = parser.Parse(os.Args[1:])

	if *versionFlag {
		fmt.Println(version)
		return
	}

	config, err := loadConfig(*configFlag)
	if err != nil {
		log.Fatalf("error: %v", err)
	}

	dev, err := os.OpenFile(config.Device, os.O_WRONLY, 0)
	if err != nil {
		log.Fatalf("error: %v", err)
	}
	defer func() {
		if err := dev.Close(); err != nil {
			log.Println(err)
		}
	}()

	for i := range config.Buttons {
		button := &config.Buttons[i]

		button.messageOn, err = parseHex(button.MessageOn)
		if err != nil {
			log.Fatalf("error: %v", err)
		}

		if button.MessageOff != "" {
			button.messageOff, err = parseHex(button.MessageOff)
			if err != nil {
				log.Fatalf("error: %v", err)
			}
		}
	}

	radioCount := 0

	for _, button := range config.Buttons {
		if button.Type == ButtonTypeRadio {
			radioCount++
		}
	}

	if radioCount == 1 {
		log.Fatalf("error: radio type button setup requires at least two instances")
	}

	footSwitch := &FootSwitch{
		device:  dev,
		buttons: config.Buttons,
	}

	server := &http.Server{
		Addr:    fmt.Sprintf("0.0.0.0:%d", config.Port),
		Handler: footSwitch,
	}

	ip, err := localIP()
	if err != nil {
		log.Fatalf("error: %v", err)
	}

	fmt.Printf(
		"Connect your smartphone to the same Wi-Fi network and open http://%s:%d\n",
		ip,
		config.Port,
	)

	go func() {
		if err := server.ListenAndServe(); err != nil &&
			err != http.ErrServerClosed {
			log.Fatalf("error: %v", err)
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig

	_ = server.Close()
}

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

type Server struct {
	port       int
	footSwitch *FootSwitch
}

type buttonRequest struct {
	Index   int  `json:"index"`
	Enabled bool `json:"enabled"`
}

func newServer(port int, footSwitch *FootSwitch) *Server {
	return &Server{
		port:       port,
		footSwitch: footSwitch,
	}
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleGetPage(w, r)
	case http.MethodPost:
		s.handlePostButton(w, r)
	default:
		w.Header().Set("Allow", "GET, POST")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleGetPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	page, err := renderPage(s.footSwitch.buttons)
	if err != nil {
		log.Printf("render page: %v", err)
		http.Error(w, "failed to render page", http.StatusInternalServerError)
		return
	}
	if _, err := fmt.Fprint(w, page); err != nil {
		log.Printf("write page response: %v", err)
	}
}

func (s *Server) handlePostButton(w http.ResponseWriter, r *http.Request) {
	const maxRequestBodyBytes = 1 << 20
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	var request buttonRequest
	if err := decoder.Decode(&request); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	var trailingValue any
	if err := decoder.Decode(&trailingValue); err != io.EOF {
		http.Error(w, "request body must contain exactly one JSON value", http.StatusBadRequest)
		return
	}

	if request.Index < 0 || request.Index >= len(s.footSwitch.buttons) {
		http.Error(w, "invalid button index", http.StatusBadRequest)
		return
	}

	if err := s.footSwitch.TriggerButton(request.Index, request.Enabled); err != nil {
		log.Printf("trigger button %d: %v", request.Index, err)
		http.Error(w, "failed to trigger button", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) serve() error {
	localIP, err := discoverLocalIP()
	if err != nil {
		return err
	}

	httpServer := &http.Server{
		Addr:              fmt.Sprintf("0.0.0.0:%d", s.port),
		Handler:           s,
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("Connect your smartphone to the same Wi-Fi network and open http://%s:%d", localIP, s.port)

	serverErrors := make(chan error, 1)
	go func() {
		serverErrors <- httpServer.ListenAndServe()
	}()

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(signals)

	select {
	case err := <-serverErrors:
		if err != http.ErrServerClosed {
			return fmt.Errorf("HTTP server: %w", err)
		}
		return nil
	case <-signals:
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("shut down HTTP server: %w", err)
		}
		return nil
	}
}

func discoverLocalIP() (string, error) {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "", fmt.Errorf("discover local IP: %w", err)
	}
	defer func() {
		if err := conn.Close(); err != nil {
			log.Printf("close IP discovery connection: %v", err)
		}
	}()

	udpAddr, ok := conn.LocalAddr().(*net.UDPAddr)
	if !ok {
		return "", fmt.Errorf("discover local IP: unexpected local address type %T", conn.LocalAddr())
	}
	return udpAddr.IP.String(), nil
}

func renderPage(buttons []Button) (string, error) {
	var html strings.Builder
	if err := pageTemplate.Execute(&html, buttons); err != nil {
		return "", fmt.Errorf("execute page template: %w", err)
	}
	return html.String(), nil
}

var pageTemplate = template.Must(template.New("page").Parse(`
<!doctype html>
<html>
<head>
  <meta name="viewport" content="width=device-width, initial-scale=1, maximum-scale=1, user-scalable=no">
  <style>
    html, body { width: 100%; height: 100%; margin: 0; overflow: hidden; background: #000; }
    body { display: flex; flex-direction: column-reverse; box-sizing: border-box; border-top: 2px solid yellow; }
    .button { flex: 1; position: relative; display: flex; align-items: center; justify-content: center; touch-action: none; border: 2px solid yellow; border-top: none; }
    .button::after { content: ""; width: var(--size); height: var(--size); border: 2px solid yellow; border-radius: 50%; background: var(--color-off); }
    .button.pressed::after { opacity: 0.5; }
    .button.enabled::after { background: var(--color-on); }
    .caption { position: absolute; right: calc(50% + var(--size) / 2 + 10px); writing-mode: vertical-rl; transform: rotate(180deg); color: white; font-size: 20px; }
  </style>
</head>
<body>
{{ $activeRadio := false }}
{{ range $index, $button := . }}
  <div class="button{{ if and (not $activeRadio) (eq $button.Type "radio") }} enabled{{ end }}"
       data-type="{{ $button.Type }}"
       data-index="{{ $index }}"
       style="--size: {{ $button.Size }}px; --color-on: {{ $button.ColorOn }}; --color-off: {{ $button.ColorOff }}">
    {{ if $button.Caption }}<span class="caption">{{ $button.Caption }}</span>{{ end }}
  </div>
  {{ if eq $button.Type "radio" }}{{ $activeRadio = true }}{{ end }}
{{ end }}
<script>
  let selectedRadioElement = document.querySelector('.button.enabled[data-type="radio"]');
  let activePointerId = null;

  async function sendButtonMessage(buttonElement, isEnabled) {
    try {
      const response = await fetch(location.href, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ index: Number(buttonElement.dataset.index), enabled: isEnabled })
      });
      if (!response.ok) {
        throw new Error('HTTP ' + response.status);
      }
    } catch (error) {
      console.error('Failed to send MIDI message:', error);
    }
  }

  document.querySelectorAll('.button').forEach(buttonElement => {
    buttonElement.addEventListener('pointerdown', event => {
      if (activePointerId !== null || !event.isPrimary) return;
      if (buttonElement.dataset.type === 'radio' && buttonElement === selectedRadioElement) return;
      activePointerId = event.pointerId;
      buttonElement.setPointerCapture(event.pointerId);
      buttonElement.classList.add('pressed');
      if (buttonElement.dataset.type === 'toggle') buttonElement.classList.toggle('enabled');
    });

    buttonElement.addEventListener('pointerup', event => {
      if (event.pointerId !== activePointerId) return;
      activePointerId = null;
      buttonElement.classList.remove('pressed');
      if (buttonElement.dataset.type === 'radio') {
        if (selectedRadioElement) selectedRadioElement.classList.remove('enabled');
        buttonElement.classList.add('enabled');
        selectedRadioElement = buttonElement;
        sendButtonMessage(buttonElement, true);
      } else if (buttonElement.dataset.type === 'toggle') {
        sendButtonMessage(buttonElement, buttonElement.classList.contains('enabled'));
      } else if (buttonElement.dataset.type === 'click') {
        sendButtonMessage(buttonElement, false);
      }
    });

    function handlePointerCancel(event) {
      if (event.pointerId !== activePointerId) return;
      activePointerId = null;
      buttonElement.classList.remove('pressed');
      if (buttonElement.dataset.type === 'toggle') buttonElement.classList.toggle('enabled');
    }
    buttonElement.addEventListener('pointercancel', handlePointerCancel);
    buttonElement.addEventListener('lostpointercapture', handlePointerCancel);
  });
</script>
</body>
</html>
`))

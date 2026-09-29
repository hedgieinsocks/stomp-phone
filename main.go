package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
)

const version = "0.1.0"

const page = `<!doctype html>
<html>
<head>
  <meta name="viewport"
        content="width=device-width, initial-scale=1,
                 maximum-scale=1, user-scalable=no">
  <style>
    html, body {
      width: 100%;
      height: 100%;
      overflow: hidden;
      background: #000;
    }
    body {
      margin: 0;
      display: flex;
      align-items: center;
      justify-content: center;
    }
    #indicator {
      width: 120px;
      height: 120px;
      border-radius: 50%;
      background: red;
      border: 3px solid yellow;
    }
    #indicator.enabled {
      background: green;
    }
  </style>
</head>
<body>
  <div id="indicator"></div>
  <script>
    const indicator = document.getElementById('indicator');
    let busy = false;
    document.addEventListener('touchstart', e => {
      e.preventDefault();
      if (busy || e.touches.length > 1) return;
      busy = true;
      indicator.classList.toggle('enabled');
      fetch(location.href, { method: 'POST' });
    }, { passive: false });
    const release = e => { if (e.touches.length === 0) busy = false; };
    document.addEventListener('touchend', release, { passive: true });
    document.addEventListener('touchcancel', release, { passive: true });
  </script>
</body>
</html>
`

type Footswitch struct {
	mu         sync.Mutex
	enabled    bool
	send       func([]byte) error
	enableMsg  []byte
	disableMsg []byte
}

func (f *Footswitch) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, page)

	case http.MethodPost:
		f.mu.Lock()
		newState := !f.enabled

		message := f.disableMsg
		if newState {
			message = f.enableMsg
		}

		if err := f.send(message); err != nil {
			f.mu.Unlock()
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		f.enabled = newState
		f.mu.Unlock()

		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
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
	defer conn.Close()

	return conn.LocalAddr().(*net.UDPAddr).IP.String(), nil
}

func main() {
	parser := flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	parser.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s -m DEVICE [-e HEX] [-d HEX] [-p PORT]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Use your smartphone as a MIDI footswitch.\n\n")
		fmt.Fprintln(os.Stderr, "Options:")
		parser.PrintDefaults()
	}

	HTTPServerPort := parser.Int(
		"p",
		8080,
		"HTTP server port",
	)

	MIDIDevicePath := parser.String(
		"m",
		"/dev/snd/midiC3D1",
		"MIDI device path",
	)

	enableMIDIHexMessage := parser.String(
		"e",
		"C0 01",
		"enable MIDI hex message",
	)

	disableMIDIHexMessage := parser.String(
		"d",
		"C0 00",
		"disable MIDI hex message",
	)

	showVersion := parser.Bool(
		"v",
		false,
		"print version and exit",
	)

	parser.Parse(os.Args[1:])

	if *showVersion {
		fmt.Println(version)
		return
	}

	enableMsg, err := parseHex(*enableMIDIHexMessage)
	if err != nil {
		log.Fatalf("error: %v", err)
	}

	disableMsg, err := parseHex(*disableMIDIHexMessage)
	if err != nil {
		log.Fatalf("error: %v", err)
	}

	dev, err := os.OpenFile(*MIDIDevicePath, os.O_WRONLY, 0)
	if err != nil {
		log.Fatalf("error: %v", err)
	}
	defer dev.Close()

	send := func(msg []byte) error {
		_, err := dev.Write(msg)
		return err
	}

	footswitch := &Footswitch{
		send:       send,
		enableMsg:  enableMsg,
		disableMsg: disableMsg,
	}

	server := &http.Server{
		Addr:    fmt.Sprintf("0.0.0.0:%d", *HTTPServerPort),
		Handler: footswitch,
	}

	ip, err := localIP()
	if err != nil {
		log.Fatalf("error: %v", err)
	}

	fmt.Printf(
		"Connect your smartphone to the same WI-FI network and open http://%s:%d\n",
		ip,
		*HTTPServerPort,
	)

	go func() {
		if err := server.ListenAndServe(); err != nil &&
			err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig

	server.Close()
}

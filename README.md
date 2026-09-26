# stomp-phone

Use your smartphone as a MIDI footswitch

This is s simple python script for Linux machines that serves an HTTP page with a full-screen button that can issue MIDI PC or CC commands when pressed. 

I've created it as a companion app for standalone TONE3000 plugin https://github.com/tone-3000/tone3000-plugin to quickly switch back and forth between clean and overdrive profiles during bedroom play-along sessions. 

My feet thumbs are nowhere as nimble as Angine de Poitrine, but it seems to work well enough even while wearing two pairs or socks. The mileage may vary depending on your phone, but if you don't want to buy an extremely overpriced footswitch, it's worth giving a try.

## 🗃️ Dependencies

* `python` (no extra libs)
* `amidi` (a part of `alsa-utils` package)

## 🚀 Usage

```sh
❯ ./stomp-phone -h
usage: stomp-phone [-h] [-q] [-p PORT] [-D DEVICE] [-d HEX] [-e HEX]

Use your smartphone as a MIDI footswitch

options:
  -h, --help           show this help message and exit
  -q, --quiet          do not print extra messages (default: False)
  -p, --port PORT      HTTP server port (default: 8080)
  -D, --device DEVICE  MIDI device name (default: hw:3,1)
  -d, --disable HEX    hexadecimal bytes for disable action (default: C0 00)
  -e, --enable HEX     hexadecimal bytes for enable action (default: C0 01)
```

1. Download the script.

```sh
❯ curl -sLO https://raw.githubusercontent.com/hedgieinsocks/stomp-phone/refs/heads/main/stomp-phone
❯ chmod u+x stomp-phone
```

2. Enable virtual raw MIDI devices.

```sh
❯ sudo modprobe snd-virmidi

❯ amidi -l
Dir Device    Name
IO  hw:3,0    Virtual Raw MIDI (16 subdevices)
IO  hw:3,1    Virtual Raw MIDI (16 subdevices)
IO  hw:3,2    Virtual Raw MIDI (16 subdevices)
IO  hw:3,3    Virtual Raw MIDI (16 subdevices)
```

3. Ensure the target application is listening to the chosen virtual raw MIDI device port.

4. Launch stomp-phone.

```sh
# send PC 3 on enable and PC 2 on disable via hw:3,3 MIDI port
❯ ./stomp-phone -e 'C0 03' -d 'C0 02' -D hw:3,3

# send CC 1 on enable and CC 2 on disable via hw:3,2 MIDI port
❯ ./stomp-phone -e 'B0 01 7F' -d 'B0 02 7F' -D hw:3,2
```

5. Connect your smartphone to the same WI-FI network and open the URL printed in the previous step ( e.g. http://192.168.0.110:8080)

6. Warm up your foot thumb and jam!

## 📜 License

[MIT](LICENSE)

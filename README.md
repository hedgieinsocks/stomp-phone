# stomp-phone

Use your smartphone as a MIDI footswitch.

![Linux](https://img.shields.io/badge/Linux-%23FCC624.svg?style=for-the-badge&logo=linux&logoColor=black)
![Go](https://img.shields.io/badge/go-%2300ADD8.svg?style=for-the-badge&logo=go&logoColor=white)

A simple app to serve an HTTP page with a full-screen button that issues MIDI commands.

## 🚀 Usage

It was created as a companion app for standalone [TONE3000 plugin](https://github.com/tone-3000/tone3000-plugin) to quickly switch back and forth between clean and overdrive profiles during bedroom jams. But it should work for any other plugin.

```sh
Usage: ./stomp-phone -m DEVICE [-e HEX] [-d HEX] [-p PORT]

Use your smartphone as a MIDI footswitch.

Options:
  -d string
        disable MIDI hex message (default "C0 00")
  -e string
        enable MIDI hex message (default "C0 01")
  -m string
        MIDI device path (default "/dev/snd/midiC3D1")
  -p int
        HTTP server port (default 8080)
  -v    print version and exit
```

1. Download the app and make it executable.

```sh
curl -sLO https://github.com/hedgieinsocks/stomp-phone/releases/download/v0.1.0/stomp-phone-v0.1.0-linux-x64
chmod u+x stomp-phone-v0.1.0-linux-x64
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

❯ ls -1 /dev/snd/midi*
/dev/snd/midiC3D0
/dev/snd/midiC3D1
/dev/snd/midiC3D2
/dev/snd/midiC3D3
```

3. Ensure the target application is listening to the chosen virtual raw MIDI device port.

4. Launch stomp-phone.

```sh
# send PC 3 on enable and PC 2 on disable via hw:3,3 MIDI port
❯ ./stomp-phone -e 'C0 03' -d 'C0 02' -m /dev/snd/midiC3D3

# send CC 1 on enable and CC 2 on disable via hw:3,2 MIDI port
❯ ./stomp-phone -e 'B0 01 7F' -d 'B0 02 7F' -m /dev/snd/midiC3D2
```

5. Connect your smartphone to the same WI-FI network and open the URL printed in the previous step (e.g. http://192.168.0.110:8080)

6. Warm up your 🦶 and jam!

## 📜 License

[MIT](LICENSE)

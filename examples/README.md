```yaml
# HTTP port to bind the server
port: 8080
# MIDI device path
device: /dev/snd/midiC3D1
# Button declaration: min=1, max=3
buttons:
    # Caption to display next to the button
  - caption: SD-1
    # Button mode one of: radio, toggle, click
    # - radio button behaves like a radio switch, so it requires at least two instances, most suited for switching between presets
    # - toggle button behaves like a toggle switch with on/off states, so it requires `messageOff`, most suited for e.g. toggling a pedal in front of an amp
    # - click button is stateless, most suited for selecting next/previous present, etc.
    type: toggle
    # MIDI hex message sent when the button is activated
    # - Program Change (PC) hex MIDI message e.g. `C0 01` (for `PC# 1`)
    # - Control Change (CC) hex MIDI message e.g. `B0 01 7F` (for `CC# 1`)
    messageOn: B0 01 7F
    # MIDI hex message sent when the button is deactivated
    # It is only required and used when `type: toggle`
    messageOff: B0 01 7F
```

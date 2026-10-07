```yaml
# HTTP port to bind the server (optional, default=8080)
port: 8080
# MIDI device path (required)
device: /dev/snd/midiC3D1
# Button declaration (required: min=1, max=3)
buttons:
    # Button mode one of: radio, toggle, click (required)
    # - radio button behaves like a radio switch, so it requires at least two instances, most suited for switching between presets
    # - toggle button behaves like a toggle switch with on/off states, so it requires `messageOff`, most suited for e.g. toggling a pedal in front of an amp
    # - click button is stateless, most suited for selecting next/previous present, etc.
  - type: toggle
    # MIDI hex message sent when the button is activated (required)
    # - Program Change (PC) hex MIDI message e.g. `C0 01` (for `PC# 1`)
    # - Control Change (CC) hex MIDI message e.g. `B0 01 7F` (for `CC# 1`)
    messageOn: B0 01 7F
    # MIDI hex message sent when the button is deactivated (required for toggle)
    messageOff: B0 01 7F
    # Caption to display next to the button (optional)
    caption: SD-1
    # Height & width px override for the button (optional, default=100)
    size: 120
    # Color override for the enabled button (optional, default=green)
    colorOn: purple
    # Color override for the disabled button (optional, default=red)
    colorOff: blue
```

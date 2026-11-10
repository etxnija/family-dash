# Raspberry Pi setup (Pi Zero W)

How to prepare the Raspberry Pi that will host the dashboard and, later, drive the Waveshare 7.5" e-Paper screen. Written
from the first setup session (Nov 2026). Steps marked **not yet verified** have not been run on the real hardware (so far: sections 1-8 verified).

## Hardware

- Raspberry Pi Zero W with the 40-pin GPIO header soldered on (the screen's driver board plugs onto it).
- 32 GB micro-SD card and an SD adapter for the Mac.
- 5 V micro-USB power supply (official guidance for all Zero models: 5 V, 2.5 A, 12.5 W). Use the micro-USB port
  labelled **PWR IN**, and a short, good-quality cable; long or thin cables cause under-voltage problems.
- Waveshare 7.5" e-Paper HAT (800x480).

## 1. Choose the right image

Use **Raspberry Pi OS Lite (32-bit)**. The original Zero W is a 32-bit ARMv6 board; the 64-bit images do not support it.
If Imager does not offer the current 32-bit Lite for the Zero, use **Raspberry Pi OS Legacy (32-bit) Lite**, which the
official download page lists for the Zero W explicitly.

## 2. Flash the card (Raspberry Pi Imager, v2.x)

1. Install Raspberry Pi Imager from https://www.raspberrypi.com/software/ and insert the card.
2. **Device:** Raspberry Pi **Zero** (not Zero 2 W or any other model, or Imager offers 64-bit images).
3. **OS:** Raspberry Pi OS (other) -> Raspberry Pi OS Lite (32-bit).
4. **Storage:** the SD card (Imager erases it).
5. **Customisation**, one screen at a time: hostname (`familydash`), time zone (Europe/Stockholm) and keyboard, username
   and password, Wi-Fi, and enable SSH with password login. Skip Raspberry Pi Connect.
6. Write, wait for verify to finish, then eject.
7. **Check the card before using it:** the `bootfs` volume must contain `kernel.img` and `bcm2708-rpi-zero-w.dtb`. If
   it only has `kernel8.img` and `kernel_2712.img`, it is a 64-bit image and the Pi will not boot.

### Wi-Fi notes

- The Zero W only supports **2.4 GHz** Wi-Fi.
- Conference/hotel/guest Wi-Fi often blocks devices from talking to each other (so `familydash.local` fails) or uses a
  login page the Pi cannot complete. A phone hotspot is more reliable: enable Personal Hotspot, turn on "Maximise
  Compatibility" (forces 2.4 GHz), keep that screen open while the Pi first connects, and join the Mac to the same
  hotspot. Enter the hotspot name and password in Imager.
- Imager takes one Wi-Fi network. Switching the Pi to the home network later means editing its Wi-Fi settings; this is
  not documented here yet.

## 3. First boot

1. Insert the card, connect power to **PWR IN**. No monitor needed.
2. The green LED is the SD-card activity light. Normal boot is irregular flickering for one to a few minutes.
3. Confirm it joined the network: `ping familydash.local` (or find its IP in the router/hotspot).
4. `ssh <username>@familydash.local` (or `ssh <username>@<ip>`).
5. **"Connection refused" on port 22 right after it starts answering pings is normal on first boot.** The Pi is still
   doing one-time setup. Wait a few minutes and retry. `nc -zv <ip> 22` tells you when SSH is up. If it is still
   refused after about 10 minutes, SSH was probably not enabled in Imager; reflash.

## 4. Update the system

```bash
sudo apt update && sudo apt upgrade
sudo reboot
```

This is slow on a Zero W (10+ minutes). If a prompt asks about replacing a config file, keep the current version.
Updates use the network the Pi is on, so over a phone hotspot they use mobile data.

## 5. Enable SPI for the screen

```bash
sudo raspi-config     # interface options -> SPI -> enable
ls /dev/spi*          # expect /dev/spidev0.0 and /dev/spidev0.1 (may need a reboot)
```

Verified on the Zero W (Nov 2026): after enabling SPI, `ls /dev/spi*` listed `/dev/spidev0.0` and `/dev/spidev0.1`.

Waveshare's wiki still names `/boot/config.txt`; newer Raspberry Pi OS releases may use a different path, so verify.

## 6. Attach the screen

Power the Pi off, then plug the screen's driver board onto the 40-pin header (or use Waveshare's 8-pin wiring table if
connecting with wires; keep any extension cable under 20 cm). Panel revisions differ: units sold after Sept 2023 use
Waveshare's `7.5V2` program, older units `7.5V2_old`. Check which one you have before choosing a driver. Do not leave the
panel powered for long periods; refresh it on a schedule.

Verified (Nov 2026): with the driver board plugged onto the Zero W's header and SPI enabled, Waveshare's Python demo
from https://github.com/waveshare/e-Paper (`RaspberryPi_JetsonNano/python/examples/`) drove the screen successfully.
Dependencies were installed with apt: `git python3-pil python3-numpy python3-spidev python3-gpiozero python3-lgpio`.

## 7. Drive the screen from Go

Verified (Nov 2026): `github.com/ditrytus/epd7in5v2` (v0.1.0, pure Go on periph.io, MIT) works on the original Pi
Zero W, even though its author only tested it on a Zero 2 W. Full refresh and partial refresh both work.

### Prerequisites on the Pi

- The user must be in the `spi` and `gpio` groups: `sudo usermod -aG spi,gpio <user>`, then log out and back in.
- HAT switches: Interface = 0 (4-line SPI), Display Config = B.

### Cross-compile on the Mac and copy the binary

The Zero W is ARMv6, so build a 32-bit ARM binary without cgo (the library needs Go 1.25 or newer):

```sh
GOOS=linux GOARCH=arm GOARM=6 CGO_ENABLED=0 go build -o eink-spike .
scp eink-spike <user>@familydash.local:~/
```

Run it on the Pi with `./eink-spike`. No Go toolchain is needed on the Pi.

### What was learned

- **A full refresh keeps the image with no power.** E-paper only uses power while the picture changes. After the program
  exits the text stays on the panel. Always call `Sleep()` before exiting; a panel left powered and idle is damaged
  over time. Do a full refresh at least once every 24 hours, and clear to white before long storage.
- **API shape:** `NewEPD`, `GPIOInit`, `Init` / `InitFast` / `InitPart`, `DisplayImage(img)` for a full frame,
  `DisplayPart(img, rect)` for a partial update, `ClearToWhite`, `Sleep`, `Close`, and `ScreenBounds`
  (`image.Rect(0, 0, 800, 480)`). The panel is 1-bit black and white only; there is no greyscale.
- **Refresh modes:** `Init()` is about 4 s and has the best contrast, `InitFast()` about 1.5 s, and `InitPart()` about
  0.4 s (the library's figures; not yet measured on the Zero W).
- **`DisplayPart` gotchas (found by trial):**
  - The `img` argument must be an image **as large as `rect`, with its origin at (0,0)**. Passing the full 800x480
    frame does not crop the area at `rect`; the panel shows the top-left corner of the frame inside `rect` instead.
  - `rect.Min.X` and `rect.Max.X` must be multiples of 8 (one byte holds 8 horizontal pixels). Snap the box outward.
  - `rect` must lie inside `ScreenBounds`, and `InitPart()` must have been called first.
  - Anything wider than `rect` is clipped at its edge. Size the rectangle from the content.
  - Partial updates build up ghosting. Do a periodic full refresh (`Init()` then `DisplayImage`) to clear it, and
    call `Init()` before returning to full-frame mode.
- **Result of the partial-refresh test:** updating a 384x160 px clock area every 2 s changed the time without any
  full-screen flash, and the static text and border around it stayed untouched.
- **The Python demo is not needed.** Waveshare's Python demo (section 6) and the Go driver both work, so the choice
  between them is a design choice, not a hardware limit.

## 8. Run the dashboard on the panel (one-shot)

Verified (Nov 2026): the real dashboard, rendered by the Go binary running on the Pi Zero W itself, shows correctly on
the panel. The Pi fetches calendars, lunch and weather on its own, so it needs internet access (the phone hotspot works).

Build on the Mac and copy the binary and the config to the Pi:

```sh
GOOS=linux GOARCH=arm GOARM=6 CGO_ENABLED=0 go build -o family-dashboard-pi ./cmd/family-dashboard
scp family-dashboard-pi sources.local.json <user>@familydash.local:~/
```

`sources.local.json` holds private calendar URLs; it ends up in the Pi's home directory, so treat the card accordingly.
On the Pi:

```sh
chmod +x family-dashboard-pi
./family-dashboard-pi -config sources.local.json -eink-panel -cable left
```

The program fetches the data, renders the portrait 480x800 frame, rotates it to the panel's native 800x480, does one
full refresh, puts the panel to sleep and exits. The picture stays on the panel without power.

- `-cable left|right` is the side of the hung portrait frame where the panel's ribbon cable exits. With the ribbon
  cable on the left of the frame, `-cable left` gives an upright picture (verified). If the picture is upside down,
  use `-cable right`.
- `-eink-png <path>` renders the same frame to a PNG instead, on any machine, to check a layout without hardware.
- The log (stderr, JSON) contains `rotate_ms` and `refresh_ms`. Measured on the Zero W (Nov 2026): rotating the
  frame took about 170 ms and the full refresh about 3.8 s; fetching the data over the phone hotspot took about 1 s, so
  the whole run is roughly 6 s from start to exit.

## 9. Wi-Fi networks (home, hotspot, venue)

The Pi OS image uses NetworkManager. It keeps a list of known Wi-Fi networks (profiles), joins whichever one it can see
at boot and, if several are in range, prefers the one with the highest `autoconnect-priority`. Moving between home, a
phone hotspot and a venue's Wi-Fi therefore needs no reflashing: add one profile per network over SSH.

A profile can be saved while its network is out of range (`nmcli connection add` only stores it). Do not use
`nmcli device wifi connect` for this: it tries to join at once and drops the current SSH session.

```sh
read -rs -p "Wi-Fi password: " PW; echo
sudo nmcli connection add type wifi con-name home ssid "<SSID>" \
  wifi-sec.key-mgmt wpa-psk wifi-sec.psk "$PW" \
  connection.autoconnect yes connection.autoconnect-priority 20
unset PW
```

The hidden prompt keeps the password out of the shell history. Useful checks (end with `| cat` to avoid the pager):

```sh
nmcli -f NAME,TYPE,AUTOCONNECT,AUTOCONNECT-PRIORITY connection show | cat
nmcli -f connection.id,802-11-wireless.ssid,802-11-wireless-security.key-mgmt,connection.autoconnect-priority connection show <profile> | cat
```

Notes:

- The `NAME` column of `nmcli connection show` is the profile name, not the SSID. The profile created by the Pi
  Imager is called `netplan-wlan0-<SSID>`; the SSID is a property of the profile (second command above).
- The Pi Zero W only supports 2.4 GHz and WPA2 (not WPA3-only). If the router has separate 2.4 and 5 GHz names, use
  the 2.4 GHz one.
- Verified (Nov 2026): a `home` profile at priority 20 saved next to the Imager's venue profile at priority 0, with
  `wpa-psk` security on both. **Not yet verified:** that the Pi joins the home network by itself after being powered
  on at home.
- To find the Pi on a new network, try `familydash.local` first (mDNS); otherwise look for `familydash` in the
  router's device list.

## Troubleshooting: green LED flash codes (Pi 1/2/3/Zero)

A repeating fixed count of flashes followed by a pause is an error code:

| Flashes | Meaning |
|---|---|
| 3 | `start.elf` not found |
| 4 | `start.elf` not launched / corrupt |
| 7 | `kernel.img` not found, usually the wrong image (e.g. 64-bit) or a failed write |
| 8 | SDRAM not recognised, bad image or hardware |

We hit **7 flashes** with a 64-bit image on the card. Fix: reflash with device = Raspberry Pi Zero and a 32-bit image,
then verify `kernel.img` exists on `bootfs`.

## Open questions

- Go driver vs. a separate Python renderer: the hardware supports both; see ADR-0002 and the e-ink adapter design.
- Refresh strategy: partial refresh of the clock every minute, full refresh of the rest about hourly. Partial-refresh
  timing and ghosting over a longer period are not yet measured.
- Refresh loop: the binary only does a one-shot refresh today (section 8); the minute clock, hourly full refresh and
  partial refresh of the clock rectangle are not implemented yet.
- Deployment workflow: cross-compiling and copying a binary works by hand (sections 7-8); a service/autostart setup
  and a repeatable deploy command are not yet defined.
- Switching networks is documented in section 9, but joining the home network by itself at power-on is not yet
  verified.

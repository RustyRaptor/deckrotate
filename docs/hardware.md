# Hardware notes

Findings from investigating orientation sensing on a Steam Deck OLED (DMI
`Galileo`), SteamOS 3.9, Plasma 6.7 Wayland. Collected 2026-11-03.

## No kernel accelerometer

The kernel exposes only two ambient light sensors through IIO:

| IIO device      | Driver     | Use                 |
|-----------------|------------|---------------------|
| `iio:device0`   | `ltrf216a` | ambient light       |
| `iio:device1`   | `ltrf216a` | ambient light       |

`iio-sensor-proxy` runs but `monitor-sensor` reports `No accelerometer`, so
KWin's built-in auto-rotation (which uses iio-sensor-proxy) can never work.

## The controller IMU

The built-in controller (USB `28DE:1205`) has a BMI-class IMU whose data
arrives in the controller's state report. Three hidraw interfaces belong to
it; the state one is identified by a report descriptor starting with
`06 ff ff` (vendor usage page 0xFFFF). Its `/dev/hidrawN` number changes
between boots, so `imu.Find` looks it up through `/sys/class/hidraw`. The
device node has a `uaccess` ACL, so the logged-in user can read it and send
feature reports without root.

### Report 0x09 layout (64 bytes, little endian)

| Offset | Size | Field                          |
|--------|------|--------------------------------|
| 0      | 2    | report version (`01 00`)       |
| 2      | 1    | type, `0x09` = controller state |
| 3      | 1    | length (`0x40`)                |
| 4      | 4    | packet number                  |
| 8      | 8    | buttons                        |
| 16     | 8    | trackpad X/Y left, right       |
| **24** | 6    | **accel X, Y, Z (int16)**      |
| 30     | 6    | gyro X, Y, Z (int16)           |
| 36     | 8    | orientation quaternion W,X,Y,Z |
| 44     | 4    | raw triggers                   |
| 48     | 8    | sticks                         |
| 56     | 4    | trackpad pressure              |

Reports arrive at about 250 Hz. Accelerometer scale is +/-2 g, so
1 g = 16384 counts. Layout cross-checked against SDL's
`src/joystick/hidapi/steam/controller_structs.h`.

### Steam turns the IMU off

With Steam running in desktop mode the accel and gyro fields read all zeros:
the desktop controller config doesn't use the gyro, so Steam sets the IMU
mode to off. One feature report turns raw data back on:

```
report id 0x00
0x87        ID_SET_SETTINGS_VALUES
0x03        payload length: one {u8 register, u16 value} setting
0x30        SETTING_IMU_MODE
0x18 0x00   SEND_RAW_ACCEL (0x08) | SEND_RAW_GYRO (0x10)
...zero padding to 64 bytes
```

sent with `ioctl(fd, HIDIOCSFEATURE(65), buf)`. It stays on until Steam
re-sends settings (for example after a config switch), so the reader re-sends
it after about half a second of all-zero IMU fields.

### Axes

Measured by holding the Deck still in each position (values are the gravity
reaction, i.e. the axis pointing up reads positive):

| Position                       | accel X | accel Y | accel Z |
|--------------------------------|--------:|--------:|--------:|
| Landscape, leaning back ~22°   |   +0.04 |   +0.93 |   +0.38 |
| Upside down                    |   -0.04 |   -0.99 |   +0.01 |
| Left grip down (right edge up) |   +0.99 |   -0.01 |   +0.03 |
| Right grip down (left edge up) |   -1.01 |   +0.01 |   +0.02 |
| Flat, screen down              |    0.00 |   +0.30 |   -0.95 |

So X points to the Deck's right edge, Y to its top edge, Z out of the screen,
matching SDL's convention.

## The panel

The internal panel is `eDP-1` (kscreen output type 7), natively portrait
800x1280 with its top at the Deck's left edge. kscreen rotations needed to
look upright:

| Deck position                  | deckrotate name     | kscreen rotation | value |
|--------------------------------|---------------------|------------------|------:|
| Landscape                      | `landscape`         | `right`          | 8     |
| Right grip down                | `portrait-cw`       | `none`           | 1     |
| Upside down                    | `landscape-flipped` | `left`           | 2     |
| Left grip down                 | `portrait-ccw`      | `inverted`       | 4     |

`kscreen-doctor output.eDP-1.rotation.<name>` applies it; KWin remaps the
touchscreen automatically.

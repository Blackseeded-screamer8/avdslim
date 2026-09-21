# Feature Reference — `avdslim enable` / `avdslim disable`

Everything avdslim turns off, and the exact name you type to turn it back on.

```bash
avdslim enable <feature|alias|package> [device|avd] [--all]
avdslim disable <host-feature> [avd]
```

Two kinds of targets:

| Kind | Lives in | Needs a running emulator? | Takes effect |
|------|----------|---------------------------|--------------|
| **Guest** features, apps, packages | the device (`pm`, `settings`) | yes | immediately |
| **Host** features | the AVD's `config.ini` | no | on the next cold boot |

`--all` applies a guest target to every running emulator. A device can be given
as an index (`1`), a serial (`emulator-5554`) or an AVD name; host targets take
an AVD name or index, and default to the only running emulator's AVD.

---

## Host features (`enable` **and** `disable`)

Written to `~/.android/avd/<name>.avd/config.ini` as an `avdslim.<feature>`
marker plus the matching `hw.*` keys. `start`, `bake`, `tune-avd` and the
Android Studio shim all read that marker, so the choice survives a re-tune.

| Feature | Aliases | What `enable` does | What it stops doing |
|---------|---------|--------------------|---------------------|
| `audio` | `sound`, `mic`, `microphone`, `speaker` | `hw.audioInput=yes`, `hw.audioOutput=yes` | injecting `-no-audio` |
| `camera` | `cam`, `webcam` | `hw.camera.back=emulated`, `hw.camera.front=emulated` (+ re-enables the guest camera apps when that AVD is running) | injecting `-camera-back none -camera-front none` |
| `dpad` | `d-pad` | `hw.dPad=yes` | — |
| `bootanim` | `boot-anim`, `bootanimation` | marker only | injecting `-no-boot-anim` |

```bash
avdslim enable audio                 # only running AVD, or prompts
avdslim enable camera Pixel_10_Pro
avdslim disable audio Pixel_10_Pro
```

After changing one:

- `avdslim restart <avd>` — stop, purge stale `hardware-qemu.ini`/snapshots, relaunch.
- `avdslim bake <avd>` — re-bake the Golden Snapshot; an old snapshot carries the old hardware config.
- `avdslim install-shim` — once, after upgrading avdslim, so Android Studio launches read the markers.

Other host knobs are flags rather than features: `--ram=`, `--heap=`, `--gpu=`
(`tune-avd`, `start`, `bake`), and `--no-lowram` / `--cold` / `--headless` on
`start`.

---

## Guest features (`enable` only)

Settings and services `avdslim on` changes on the device. Turn them all back at
once with `avdslim off`; keep them untouched during slimming with
`avdslim on --skip=<group>`.

| Feature | Aliases | `enable` restores | `--skip` group |
|---------|---------|-------------------|----------------|
| `bluetooth` | `bt` | `bluetooth_on=1`, `bluetooth_manager enable`, the BT packages | `bluetooth` |
| `animations` | `anim` | window/transition/animator scales → `1` | `animations` |
| `sync` | — | `auto_sync=1` | `sync` |
| `location` | `gps` | `location_mode=3` | `location` |
| `bglimit` | — | clears `background_process_limit` + `activity_manager_constants` | `bglimit` |

A `setup` skip group also exists (`--skip=setup`): it leaves `user_setup_complete` /
`device_provisioned` alone. There is no `enable setup` — those are one-way
"already set up" markers.

`com.google.android.bluetooth` is **boot-critical** and is never disabled.
A guest stuck on a black screen is fixed with `avdslim repair`.

---

## App aliases (`enable` only)

Short names for the bloat packages `avdslim on` disables.

| Alias | Package(s) |
|-------|------------|
| `maps` | `com.google.android.apps.maps` |
| `photos` | `com.google.android.apps.photos` |
| `chrome` | `com.android.chrome` |
| `camera` | `com.android.camera2`, `com.android.cameraextensions`, `com.android.DeviceAsWebcam` |
| `store` / `playstore` / `vending` | `com.android.vending` |
| `youtube` | `com.google.android.youtube`, `com.google.android.apps.youtube.music` |
| `music` | `com.google.android.music`, `com.android.music` |
| `sms` / `messaging` | `com.google.android.apps.messaging`, `com.android.mms` |
| `dialer` / `phone` | `com.google.android.dialer`, `com.android.dialer` |
| `contacts` | `com.google.android.contacts`, `com.android.contacts` |
| `docs` / `drive` | `com.google.android.apps.docs` |
| `printing` / `print` | `com.android.printspooler`, `com.android.bips`, `com.google.android.printservice.recommendation` |
| `search` / `assistant` | `com.google.android.googlequicksearchbox` |
| `wellbeing` | `com.google.android.apps.wellbeing` |
| `clock` | `com.android.deskclock`, `com.google.android.deskclock` |
| `calculator` | `com.android.calculator2`, `com.google.android.calculator` |
| `calendar` | `com.android.calendar`, `com.google.android.calendar` |
| `voiceaccess` | `com.google.android.apps.accessibility.voiceaccess` |
| `markup` | `com.google.android.markup` |
| `safety` | `com.google.android.apps.safetyhub` |
| `multidisplay` | `com.android.emulator.multidisplay` |
| `bluetooth` / `bt` | `com.google.android.bluetooth`, `com.android.bluetoothmidiservice` |

Not listed above? Two fallbacks:

```bash
avdslim enable com.google.android.tts   # any package id (contains a dot)
avdslim enable tts                      # last segment of a known bloat package
```

`avdslim profiles` prints every package avdslim ever disables, grouped by
category. `avdslim on --keep=<package>` preserves one up front instead.

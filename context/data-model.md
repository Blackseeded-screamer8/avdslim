# Data model — avdslim

No database. No schemas, no migrations, no API. All state is files on host or
guest, or live emulator flags.

## Entities

- **AVD config** — `~/.android/avd/<name>.avd/config.ini` (or
  `$ANDROID_AVD_HOME`). Flat `key=value`, parsed into `map[string]string`
  (`config.GetInstalledAvds`). Tuned keys: `hw.ramSize` (default 1024),
  `vm.heapSize` (256), `hw.gpu.mode`/`hw.gpu.enabled`, camera/audio off,
  `fastboot.forceColdBoot=yes`. One-time backup `config.ini.bak` (never
  overwritten once present). Owner: `internal/config/tuner.go`.
- **Golden snapshot** — directory `<avd>.avd/snapshots/avdslim_clean/`.
  Existence = "baked" (`config.HasGoldenSnapshot` = dir check only; no
  validity check — corrupt/partial dirs read as present). Created by `bake`
  (cold boot → slim → `emu avd snapshot save`) or `snapshot`/`stop --snap`
  (live state incl. installed apps/logins). Deleted by `unbake` (`RemoveAll`).
  Consumed by `start`/`launch` and the shim as
  `-snapshot avdslim_clean -no-snapshot-save`.
- **Slim state** — `/data/local/tmp/avdslim_state.json` **on the device**
  (`SlimState{timestamp, disabled_packages, preset}`), written by `Slim`,
  read by `Restore`. Absent (or unparseable) → `Restore` re-enables every
  known package (standard + aggressive). Wiped on factory reset; `IsSlimmed`
  in `list`/`watch`/`doctor` is derived from this file's existence.
- **Bloat lists** — compile-time constants in `internal/bloat/packages.go`:
  `StandardBloatCategories map[string][]string` (~35 pkgs), `AggressiveBloatPackages`
  (5 pkgs: Play Store updater, Chrome, setup wizards). Not user-editable
  except per-invocation `--keep=<pkg>`.
- **Shim marker** — SDK `emulator/` dir holds either stock binary or
  (`emulator.real` binary + `emulator` shell/batch script containing
  `# avdslim emulator shim`). `IsShimInstalled` checks both.

## Ownership boundaries

- `config` owns host files (SDK/AVD paths, `config.ini`, snapshot dirs).
- `adb.Client` owns guest state (`pm`, `settings`, state JSON, `emu` console).
- `host` owns nothing persistent — pure live process probing.
- `main.go` handlers orchestrate but persist nothing themselves.

# Changelog

## Unreleased

### Fixed
- **`start` / `bake` targeted the wrong emulator when another was already running**: the boot wait and slimming ran against whichever device adb picked, so `avdslim start 2` with AVD 1 up re-slimmed AVD 1 and left AVD 2 on a black screen. Both now pass `-port` to the emulator and target that serial.

## v1.0.9 — 2026-09-17

### Added
- **`avdslim repair` Command**: Unsticks an AVD hung on a black screen by re-enabling boot-critical packages directly in `package-restrictions.xml` (via `adb root`) and restarting the framework. Keeps user data.

### Fixed
- **AVD stuck on black screen after stop/start**: slimming no longer disables `com.google.android.bluetooth`. On Android 16+ images `system_server` crash-loops at boot (`FATAL: Conflicting system configuration detected`) when the Bluetooth APK is disabled, so every cold boot after the first slim hung. Bluetooth is still switched off at runtime via `bluetooth_on=0`.

## v1.0.8 — 2026-09-17

### Added
- **`avdslim enable` Command**: Selectively re-enable specific features (`bluetooth`, `animations`, `sync`, `location`, `bglimit`) or app packages (`maps`, `photos`, `chrome`, `camera`, `store`, etc.) on running AVDs without performing a full restore. Supports index, AVD name, serial, and `--all` multi-device targeting.
- **Bluetooth Debloating & `--skip bluetooth`**: Added Bluetooth subsystem (`com.google.android.bluetooth`, `com.android.bluetoothmidiservice`, `bluetooth_on`) debloating, saving ~25 MB RAM. Added `--skip bluetooth` option for developers testing BLE peripherals.
- **Privacy Sandbox Debloating**: Silenced `com.google.android.adservices.api` and `com.google.mainline.adservices` background attribution daemons.
- **Continuous Integration (CI)**: Added GitHub Actions CI workflow to run formatting checks, `go vet`, tests, and binary compilation on push and pull requests.

### Changed
- **Animation Default Inverted**: Animations are now ON by default (Fluid 1.0x) for natural UI navigation. Added `--no-anim` / `--no-animations` flag to explicitly disable animations (0x) for instant UI response.
- **Background Process Capping**: Configured `max_cached_processes=4` alongside `background_process_limit=4` to eliminate hidden cached app RAM bloat on Android 14–37.
- **CPU Thread Tuning**: Set `hw.cpu.ncore=2` in `tune-avd` to curb host thread stack allocation and parallel compilation thread churn.
- **PhotoPicker Protected**: Excluded `com.google.android.photopicker` by default to ensure modern `ActivityResultContracts.PickVisualMedia` dialogs remain functional.

## v1.0.7 — 2026-09-17

### Added
- **OS-Specific GPU Backends**: Automatically selects `-gpu host` on macOS (native Apple Metal acceleration), `-gpu auto` on Linux and Windows (native DRI/Vulkan and Direct3D 11 via ANGLE), and `-gpu swiftshader_indirect` on headless Linux CI runners without display servers.
- **Flexible Device Target Resolution**: Commands (`stop`, `snapshot`, `on`, `off`, `bench`) now accept numeric 1-based index numbers (`1`, `2`), AVD names (`Slim_Pixel_5`), or ADB serials (`emulator-5554`).
- **Physical Keyboard Forwarding**: Automatically sets `hw.keyboard = yes` during `tune-avd` so typing on host keyboards works out-of-the-box inside the emulator.

### Changed
- **Default RAM Raised to 1536 MB**: Raised default guest RAM from 1024 MB to 1536 MB across CLI flags, Android Studio shim, AVD tuner, bake snapshots, and GitHub Actions workflows. Balances lightweight host memory (~1.7 GB) with guest OS `lmkd` headroom, preventing Gboard and dev application kills on Android API 34–37 images.

### Fixed
- Fixed `avdslim stop` to correctly target emulators by index, name, or serial with responsive termination polling.
- Fixed `github-repo-stats` workflow downsampling ZeroDivisionError on new or sparse data branches.

## v1.0.6 — 2026-09-17

### Upgrading
- **Run `avdslim install-shim` again after upgrading** if you use the Android
  Studio shim. Shims installed by 1.0.5 or earlier don't read the new defaults
  file. `avdslim doctor` flags an outdated shim.
- `go install github.com/kdbhalala/avdslim/cmd/avdslim@latest` works again
  (the module path now matches the repository).
- Emulators slimmed by 1.0.5 or earlier have no saved setting values, so
  `avdslim off` still resets them to stock (animations 1x, location on).
  Run `avdslim off` then `avdslim on` once to start saving your own values.

### Added
- `--skip=animations,bglimit,sync,location,setup` on `on`, `watch`, `start`,
  `bake` and `snapshot` to leave those settings unchanged.
- Per-user defaults file (`~/Library/Application Support/avdslim/defaults` on
  macOS, `~/.config/avdslim/defaults` on Linux). The shim reads `--ram` from
  it at every launch.
- `doctor` and `watch` warn when an Android Studio emulator update has
  replaced the shim.
- Release tarballs carry build provenance attestations
  (`gh attestation verify`).
- `TRADEMARKS.md`: naming rules for forks and the list of official channels.

### Changed
- `avdslim off` restores each setting to the value it had before
  `avdslim on`, instead of fixed stock values.
- `start` passes `--aggressive`, `--keep` and `--skip` through to slimming.
- `install-shim` refuses on Windows with an explanation (the script wrapper
  never worked there); `uninstall-shim` still removes an old one.

### Fixed
- `avdslim version` now reports the release tag.

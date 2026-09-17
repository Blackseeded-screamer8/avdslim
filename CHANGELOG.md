# Changelog

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

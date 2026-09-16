# Architecture — avdslim

`avdslim` is a zero-dependency (stdlib-only) Go CLI that cuts Android emulator
host RAM (~8 GB → ~1.5 GB) by launching QEMU with `-lowram` + hardware GPU,
disabling ~35 bloat packages via adb, and restoring a pre-baked `avdslim_clean`
snapshot for ~1.5 s boots.

## Layout

- `cmd/avdslim/main.go` — only entrypoint. Hand-rolled arg parsing
  (`os.Args` switch, no cobra/flag lib). All 15+ commands live here as
  `handle*` functions (~1150 lines, the God-file).
- `internal/adb/client.go` — adb subprocess wrapper (`Exec` = `CombinedOutput`).
  Owns guest-side mutations: `Slim` (`pm disable-user`), `Restore` (`pm enable`),
  system settings, memory trim. Slim state persisted **on the device** at
  `/data/local/tmp/avdslim_state.json`; `Restore` reads it, falls back to
  re-enabling every known package when the file is absent.
- `internal/bloat/packages.go` — source of truth for package lists:
  `StandardBloatCategories` (map, safe) + `AggressiveBloatPackages` (slice,
  incl. Play Store updater + Chrome). Never disable anything not listed here;
  GMS core / WebView / sockets are protected by omission.
- `internal/config/` — host-side SDK/AVD knowledge. `paths.go` locates SDK
  (`ANDROID_HOME`/`ANDROID_SDK_ROOT` → OS-default paths), adb/emulator binaries
  (`LookPath` → SDK fallback → bare name), and per-OS GPU mode
  (`host` everywhere except headless Linux → `swiftshader_indirect`).
  `tuner.go` edits `~/.android/avd/<name>.avd/config.ini` (RAM/heap/GPU,
  no camera/audio), keeps a one-time `config.ini.bak`, and purges
  `hardware-qemu.ini{,.lock}` + `snapshots/` to kill stale 4 GB/lavapipe state.
- `internal/host/process.go` (+ `process_unix.go` / `process_windows.go`) —
  host QEMU PID discovery (`lsof -i :<port>` → `ps` fallback) and memory
  probes (`footprint` on macOS → RSS fallback; `/proc/<pid>/status` on Linux).
  `SetDetached` is the only platform-split code (build tags).
- `internal/shim/shim.go` — renames SDK `emulator` → `emulator.real`, writes a
  shell/batch shim injecting `-memory/-lowram/-no-audio/no-camera` + snapshot
  restore. Honors caller's explicit `-memory/-lowram`, `--no-lowram`,
  `--no-slim` passthroughs. Rollback on write failure.
- `internal/doctor/doctor.go` — read-only audit (toolchain, AVD image type,
  RAM/GPU, running emulators, snapshot presence). Never mutates.

## Dependency direction

`main` → `adb` → (`bloat`, `config`); `main` → `config`, `doctor`, `host`,
`shim`; `shim`/`doctor` → `config`. `bloat` is a leaf (no imports).
`host` and `config` never import `adb` — host process probing and SDK paths
stay independent of device state.

## Invariants

- Stdlib only (`go.mod` has zero requires). Do not add dependencies.
- `go.mod` module path is `github.com/kdbhalala/avdslim`, matching the GitHub
  remote. Keep them in sync or `go install` breaks.
- PID detection must exclude own process: filter `avdslim`, `crashpad`,
  `netsimd` and skip `os.Getpid()` or `measure` reports its own 5 MB RSS.
- `launch`/`bake` must pass `-no-snapshot-load` (cold) vs
  `-snapshot avdslim_clean -no-snapshot-save` (golden) — never save over the
  golden snapshot during dev sessions; `stop --snap` is the only path that
  overwrites it.
- `background_process_limit = 4` (not 2 — protects OAuth/biometrics);
  animations restore to `1.0x` on `off`, never delete the keys except
  `background_process_limit`.
- `TuneAvd` rewrites `config.ini` from a map — key order is not preserved.
  Acceptable; do not "fix" by adding ordering unless a bug requires it.
- Golden SDK rule the code enforces in `doctor`/`tuner`: Google APIs 4 KB
  image good; Google Play (no `adb root`, updater churn) and 16 KB page-size
  (QEMU 4096 MB floor) bad.

# ⚡ AVD-SLIM: Project Handoff & Architecture Guide

## 1. Project Overview & Location
- **Path**: `/Users/krunalbhalala/Documents/Projects/avdslim`
- **Purpose**: Low-overhead CLI tool to optimize Android Virtual Device (AVD) host memory consumption, eliminate CPU churn, disable background daemons, and lock low-memory host AVD configuration.
- **Inspired by**: [simslim](https://github.com/MobAI-App/simslim) (iOS simulator optimizer).
- **Core Technology**: Go 1.22+ (100% Go Standard Library, 0 external third-party dependencies).

---

## 2. Directory Structure
```
/Users/krunalbhalala/Documents/Projects/avdslim
├── cmd/
│   └── avdslim/
│       └── main.go           # CLI entrypoint, argument parsing, commands
├── internal/
│   ├── adb/
│   │   └── client.go         # ADB subprocess wrapper, package disable/enable, settings
│   ├── bloat/
│   │   └── profiles.go       # Curated lists of bloat packages (Standard & Aggressive)
│   ├── config/
│   │   └── tuner.go          # Parses & modifies ~/.android/avd/<name>.avd/config.ini
│   └── host/
│       └── process.go        # QEMU host PID resolution & RSS memory measurement
├── bin/
│   └── avdslim               # Compiled static binary
├── Makefile                  # Build, install, test, and release targets
└── README.md
```

---

## 3. Key Fixes & Optimizations Implemented

### Accurate Host QEMU PID Detection
- **Issue**: Previously, searching `ps -eo pid,command` for `emulator` and the serial port matched `./bin/avdslim measure emulator-5554` itself, returning the 5 MB CLI process RSS instead of the ~1.3–2.5 GB QEMU process.
- **Solution**:
  1. Utilized `lsof -i :<port> -sTCP:LISTEN -t` (e.g. port `5554` for `emulator-5554`) to directly query the OS for the listening QEMU process socket.
  2. Implemented fallback process inspection that explicitly filters out `os.Getpid()`, `avdslim`, `crashpad_handler`, and `netsimd`.
  3. Implemented robust cross-platform host PID detection in `internal/host/process.go`.

### Guest Memory Reclamation
- Disables 27–35+ bloat packages (`com.google.android.googlequicksearchbox`, `com.google.android.as`, `com.google.android.apps.photos`, `com.google.android.youtube`, etc.).
- Sets window, transition, and animator scales to `0.0x`.
- Sets `background_process_limit = 2` and disables auto-sync.
- Executes `am trim-memory --all COMPLETE` and root `drop_caches`.
- **Result**: Guest Free RAM increased from **636 MB to 2,140 MB (2.14 GB)**.

### Permanent Host Configuration Tuning & The 8 GB Memory Root Cause
- **Why was the emulator taking 8 GB of RAM in Activity Monitor?**
  1. **Metric Discrepancy**: macOS Activity Monitor displays `phys_footprint` (`task_vm_info`), which counts dirty pages + compressed memory pages. The CLI's previous RSS metric only counted active uncompressed pages.
  2. **Stale Running Process & GPU Lavapipe**: The emulator was running continuously with `hw.gpu.mode = lavapipe` (CPU software Vulkan rasterizer). Lavapipe allocated ~4 GB of software rendering buffers in host RAM on top of the 4 GB guest RAM, peaking at **8,154 MB (8.15 GB)** in Activity Monitor.
  3. **Android 15/16/17 RAM Enforcement**: For API 34+ and 16 KB Page Size images, Android Emulator's C++ code (`main-common.c`) enforces `minRam = 2560` or `4096MB` and overrides `-memory 1536` with `"Increasing RAM size to 4096MB"`.
  4. **The `-lowram` Override**: Passing the official emulator flag `-lowram` triggers `minRam = 0` (`"Removing any lower bound of RAM size"`) and activates Android OS low-RAM kernel mode, strictly locking guest RAM to **1,520 MB (1.5 GB)**.
  5. **Purging Stale Snapshots & Hardware Cache**: QuickBoot snapshots and `hardware-qemu.ini` cached the old 4GB/lavapipe states across boots. `tune-avd` and `restart` now automatically purge stale runtime `.ini` and `snapshots/` files.
  6. **Result**: Activity Monitor memory footprint plummeted from **8,154 MB (8.15 GB) down to 3,534 MB (3.5 GB)** on Pixel 10 Pro with native Metal hardware GPU acceleration.

---

## 5. Quick Command Reference

```bash
cd /Users/krunalbhalala/Documents/Projects/avdslim

# List running emulators with Activity Monitor footprint & RSS
./bin/avdslim list

# Measure guest and host memory breakdown
./bin/avdslim measure emulator-5554

# Slim down running emulator (standard bloat packages)
./bin/avdslim on emulator-5554

# Slim down aggressively (including Play Store updater & Chrome background sync)
./bin/avdslim on emulator-5554 --aggressive

# Restore emulator back to default factory state
./bin/avdslim off emulator-5554

# Selectively re-enable a feature or app on a running emulator
./bin/avdslim enable bluetooth emulator-5554
./bin/avdslim enable maps 1
./bin/avdslim enable sync --all

# Tune an installed AVD config to permanently lock RAM & GPU settings
./bin/avdslim tune-avd Pixel_10_Pro --ram=1536 --heap=256

# Gracefully restart running emulator with clean cache & low-memory flags
./bin/avdslim restart emulator-5554

# Launch an AVD with low-memory host flags and auto-slim
./bin/avdslim launch Pixel_10_Pro --slim --ram=1536
```


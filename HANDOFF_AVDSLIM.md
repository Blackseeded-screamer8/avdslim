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
├── rust_comparison/          # Complete parallel Rust implementation for benchmarking
│   ├── Cargo.toml
│   └── src/main.rs
├── bench/
│   └── benchmark.py          # Automated head-to-head benchmarking script
├── bin/
│   └── avdslim               # Compiled static binary
├── Makefile                  # Build, install, test, and release targets
└── README.md
```

---

## 3. Go vs. Rust Benchmark Results

| Metric | Go (Winner) | Rust | Notes |
|---|---|---|---|
| **Binary Size** | **2.6 MB** | 0.44 MB | Rust is ~6x smaller; 2.6 MB is negligible for CLI tooling. |
| **Startup Latency** | **2.61 ms** | 2.08 ms | Imperceptible ~0.5ms difference. |
| **Incremental Build Time** | **206 ms** | 3,088 ms | Go compiles ~10x faster. |
| **External Dependencies** | **0 (100% Stdlib)** | 11 Crates | Zero supply-chain risk and zero vendor bloat in Go. |
| **Cross-Compilation** | **Native (1 command)** | Complex (linkers required) | Go cross-compiles to Darwin/Linux/Windows natively. |

**Verdict**: Go was selected as the primary production implementation. The Rust version is preserved in `rust_comparison/` as an archived reference.

---

## 4. Key Fixes & Optimizations Implemented

### Accurate Host QEMU PID Detection
- **Issue**: Previously, searching `ps -eo pid,command` for `emulator` and the serial port matched `./bin/avdslim measure emulator-5554` itself, returning the 5 MB CLI process RSS instead of the ~1.3–2.5 GB QEMU process.
- **Solution**:
  1. Utilized `lsof -i :<port> -sTCP:LISTEN -t` (e.g. port `5554` for `emulator-5554`) to directly query the OS for the listening QEMU process socket.
  2. Implemented fallback process inspection that explicitly filters out `os.Getpid()`, `avdslim`, `crashpad_handler`, and `netsimd`.
  3. Synchronized fix across both Go (`internal/host/process.go`) and Rust (`rust_comparison/src/main.rs`).

### Guest Memory Reclamation
- Disables 27–35+ bloat packages (`com.google.android.googlequicksearchbox`, `com.google.android.as`, `com.google.android.apps.photos`, `com.google.android.youtube`, etc.).
- Sets window, transition, and animator scales to `0.0x`.
- Sets `background_process_limit = 2` and disables auto-sync.
- Executes `am trim-memory --all COMPLETE` and root `drop_caches`.
- **Result**: Guest Free RAM increased from **636 MB to 2,140 MB (2.14 GB)**.

### Permanent Host Configuration Tuning
- Edits `~/.android/avd/Pixel_10_Pro.avd/config.ini`:
  - `hw.ramSize = 1536` (down from 2048/4096 MB).
  - `vm.heapSize = 256`.
  - `hw.camera.back = none` & `hw.camera.front = none`.
  - `hw.audioInput = no` & `hw.audioOutput = no`.
  - `hw.gpu.mode = host` (Apple Silicon Metal hardware acceleration).
  - Automatic backup created at `config.ini.bak`.

---

## 5. Quick Command Reference

```bash
cd /Users/krunalbhalala/Documents/Projects/avdslim

# List running emulators and installed AVD configurations
./bin/avdslim list

# Measure guest and host memory footprint
./bin/avdslim measure emulator-5554

# Slim down running emulator (standard bloat packages)
./bin/avdslim on emulator-5554

# Slim down aggressively (including Play Store updater & Chrome background sync)
./bin/avdslim on emulator-5554 --aggressive

# Restore emulator back to default factory state
./bin/avdslim off emulator-5554

# Tune an installed AVD config to permanently lock RAM & GPU settings
./bin/avdslim tune-avd Pixel_10_Pro --ram=1536 --heap=256

# Launch an AVD with low-memory host flags and auto-slim
./bin/avdslim launch Pixel_10_Pro --slim --ram=1536
```

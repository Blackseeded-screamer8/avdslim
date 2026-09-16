# ⚡ AVD-SLIM

> **Android Emulator RAM & CPU Optimizer**  
> *Inspired by [MobAI-App/simslim](https://github.com/MobAI-App/simslim) for iOS simulators.*

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Report Card](https://goreportcard.com/badge/github.com/kdbhalala/avdslim)](https://goreportcard.com/report/github.com/kdbhalala/avdslim)
[![Release](https://img.shields.io/github/v/release/kdbhalala/avdslim)](https://github.com/kdbhalala/avdslim/releases)

`avdslim` is a lightweight, zero-dependency CLI tool that reduces Android Virtual Device (AVD) host memory consumption from **~8 GB down to ~1.5 GB** and cuts idle CPU overhead to near-zero on Apple Silicon & Linux.

```
┌────────────────────────────────────────────────────────────────────────┐
│  Before:  qemu-system-aarch64  ██████████████████████████  8,518 MB    │
│  After:   qemu-system-aarch64  █████                       1,560 MB    │
│                                                                        │
│  ⚡ Reclaimed: ~6.0 GB host RAM (82% reduction)                        │
└────────────────────────────────────────────────────────────────────────┘
```

---

## 🛡️ Fidelity & Safety Guarantee

The #1 fear with debloating tools is silent breakage. `avdslim` is designed to be **safe by default**:

| Subsystem / Service | Status | Guarantee |
| :--- | :---: | :--- |
| **Firebase Cloud Messaging (FCM)** | ✅ **100% Active** | `GcmService` allowlisted; push notifications work out of the box |
| **Firebase Auth & Google Sign-In** | ✅ **100% Active** | Core `com.google.android.gms` APIs are protected and never disabled |
| **Android System WebView** | ✅ **100% Active** | Chromium engine, JavaScript, and in-app browsers untouched |
| **Flutter / React Native / Native** | ✅ **100% Active** | Hot reload, DevTools, debugging, and JNI/NDK runtimes work 100% |
| **Localhost & Network Sockets** | ✅ **100% Active** | TCP/UDP, Metro bundler (`:8081`), and adb reverse unaffected |
| **Zero-Risk Revert (`restore`)** | ✅ **Instant Undo** | One command (`avdslim restore`) instantly re-enables all stock services |

---

## 🎯 The Problem

When developing Android apps on macOS or Linux, developers often discover `qemu-system-aarch64` consuming **5 GB to 8+ GB of RAM** in Activity Monitor.

### Why Does the Emulator Consume 8 GB?
1. **The Lavapipe Trap**: Android Studio frequently defaults `hw.gpu.mode = auto`, which falls back to Mesa CPU software rasterization (`lavapipe`). This allocates **~4 GB of software rendering buffers** directly in host RAM on top of the guest OS RAM.
2. **16 KB Page Size Images**: On modern ARM64 images (`google_apis_ps16k`), QEMU hardcodes a minimum RAM threshold (`minRam = 4096MB`), silently overriding lower RAM settings.
3. **Android Bloatware**: Over 35 non-essential daemons (Google Assistant, System Intelligence, Maps, Photos, YouTube, telemetry) wake CPU cores and pollute memory.
4. **Stale Snapshots**: Android Studio re-loads cached snapshots (`hardware-qemu.ini`) that preserve heavy 4 GB states across reboots.

---

## 💡 The Solution

Just like `simslim` silences iOS simulators via `launchctl`, `avdslim`:
1. **Passes `-lowram` to QEMU**: Removes the internal 4 GB lower bound and boots the Android kernel in low-RAM mode (`hw.ramSize = 1024M` or `1536M`).
2. **Enforces Metal GPU Acceleration**: Forces `-gpu host` to render natively via Apple Silicon Metal, completely bypassing CPU software rasterizers.
3. **Disables 24+ Bloat Daemons**: Silences non-essential Google background services via `pm disable-user --user 0`.
4. **Eliminates Animation Lag**: Sets window, transition, and animator scales to 0x.
5. **Limits Background Churn**: Caps `background_process_limit = 2` and disables auto-sync.
6. **Drops Caches**: Flushes Linux page caches and compacts memory heaps.

---

## 📊 Memory Footprint Breakdown

| Stage | Activity Monitor (`phys_footprint`) | Active Dirty RAM (`footprint`) | Reclaimed |
| :--- | :--- | :--- | :--- |
| **Default Stock Emulator** (Pixel 10 Pro) | **8,518 MB (8.5 GB)** | ~6,500 MB | Baseline |
| **With `avdslim launch` (`-lowram`, Metal GPU)** | **2,498 MB (2.5 GB)** | **1,560 MB (1.5 GB)** | **~6.0 GB saved (71%)** |
| **Standard 1080p Profile** (Pixel 5) | **2,325 MB (2.3 GB)** | **1,306 MB (1.3 GB)** | **~6.2 GB saved (73%)** |

> **Note on Activity Monitor vs Dirty RAM**:  
> macOS Activity Monitor reports `phys_footprint` from Apple's Mach kernel ledger. This includes ~530 MB of compressed pages from the initial boot spike and Metal GPU display pipeline buffers. The actual dirty physical memory held in RAM is **~1.5 GB** (verified with `footprint -p <pid>`).

---

## 🚀 Installation

### Option 1: One-Line Installer (Fastest)
```bash
curl -fsSL https://raw.githubusercontent.com/kdbhalala/avdslim/main/install.sh | bash
```

### Option 2: Homebrew (macOS / Linux directly from this repo)
```bash
brew tap kdbhalala/avdslim https://github.com/kdbhalala/avdslim.git
brew install avdslim
```

### Option 3: Go Install
```bash
go install github.com/kdbhalala/avdslim/cmd/avdslim@latest
```

### Option 4: Pre-built Binaries
Download pre-compiled binaries from [GitHub Releases](https://github.com/kdbhalala/avdslim/releases):
* macOS Apple Silicon: `avdslim_*_darwin_arm64.tar.gz`
* macOS Intel: `avdslim_*_darwin_amd64.tar.gz`
* Linux: `avdslim_*_linux_amd64.tar.gz` / `avdslim_*_linux_arm64.tar.gz`

### Option 5: Build from Source (100% Stdlib, Zero Dependencies)
```bash
git clone https://github.com/kdbhalala/avdslim.git
cd avdslim
make install
```

---

## 🛠️ Usage & Workflows

### 1. Zero-Friction Watch Mode (`watch`)
Don't want to change your workflow? Run `avdslim watch` in the background. Whenever you launch an emulator from Android Studio or VS Code, `avdslim` detects it and automatically silences bloat as soon as it boots:
```bash
avdslim watch
```
*(Options: pass `--aggressive` or `--keep=<package>`)*.

---

### 2. Instant Undo / Restore (`restore`)
Need to verify a bug with 100% stock Google services? One command immediately re-enables all disabled packages, restores animations to 1.0x, and resets background limits:
```bash
avdslim restore
# Or use alias:
avdslim off
```

---

### 3. Slim an Active Emulator (`on`)
Immediately silences background bloat and trims memory on a running emulator:
```bash
# Standard preset (safe for all apps):
avdslim on

# Aggressive preset (also disables Play Store self-updater):
avdslim on --aggressive

# Keep a specific app (e.g. Google Maps):
avdslim on --keep=com.google.android.apps.maps
```

---

### 4. Deep Memory Breakdown (`measure`)
Inspect host macOS memory (`phys_footprint`, resident RSS) alongside the guest Android `dumpsys meminfo`:
```bash
avdslim measure
# Or specify serial:
avdslim measure emulator-5554
```

---

### 5. Tune Host AVD Configuration (`tune-avd`)
Configures an AVD's `config.ini` for optimal memory consumption and purges stale snapshots:
```bash
avdslim tune-avd Pixel_10_Pro --ram=1536 --heap=256
```
* Sets `hw.ramSize = 1536` (or 1024)
* Sets `hw.gpu.mode = host` (Apple Silicon Metal hardware acceleration)
* Disables camera and audio emulation threads
* Purges stale `hardware-qemu.ini` and snapshots

---

### 6. Restart Emulator with Clean Cache (`restart`)
Gracefully shuts down the emulator, purges stale runtime snapshots, and relaunches with low-memory host flags:
```bash
avdslim restart emulator-5554 --ram=1536
```

---

### 7. Launch Emulator (`launch`)
Starts an AVD with `-lowram -gpu host -no-snapshot-load` and automatically slims upon boot:
```bash
avdslim launch Pixel_10_Pro --slim --ram=1536
```

---

### 8. Environment Doctor (`doctor`)
Audits your Android toolchain, active AVDs, 16K page size overhead, and warns about software GPU fallback:
```bash
avdslim doctor
```

---

### 9. View Bloat Profiles (`profiles`)
Inspects the list of disabled packages categorized by function (Assistant, Telephony, Consumer Bloat, etc.) and guaranteed core services:
```bash
avdslim profiles
```

---

## ⚡ Benchmark: Go vs Rust

Both Go and Rust implementations were benchmarked in `bench/`:

| Metric | Go (Winner) | Rust |
|---|---|---|
| **Startup Latency** | **2.61 ms** | **2.08 ms** |
| **Incremental Build Time** | **206 ms** | **3,088 ms** |
| **External Dependencies** | **0 (100% Stdlib)** | 11 crates |
| **Cross-Compilation** | **Native out-of-the-box** | Requires target toolchains & linkers |

**Verdict**: Go was selected for its zero-dependency standard library and instant cross-compilation.

---

## 📜 License

MIT License. See [LICENSE](LICENSE) for details.

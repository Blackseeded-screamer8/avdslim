# ⚡ AVD-SLIM

> **Android Emulator RAM & CPU Optimizer**  
> *Inspired by [MobAI-App/simslim](https://github.com/MobAI-App/simslim) for iOS simulators.*

`avdslim` is a lightweight, zero-dependency CLI tool that reduces Android Virtual Device (AVD) host memory consumption by **50% to 75%** and cuts idle CPU overhead to near-zero.

---

## 🎯 The Problem

When running an Android Emulator (especially Google APIs images), macOS allocates **3 GB to 5 GB of RAM** to `qemu-system-aarch64`. Inside the guest OS, dozens of background daemons consume CPU and dirty memory pages:
- **Google Assistant / Search** (`com.google.android.googlequicksearchbox`): ~250–400 MB RAM
- **Android System Intelligence** (`com.google.android.as`): ~150–250 MB RAM
- **Consumer bloat** (Photos, YouTube, Maps, Gmail, Music): ~200–300 MB RAM
- **Telephony & accessibility daemons**: ~100–150 MB RAM
- **Unrestricted cached background processes & 60fps window animations**

This causes MacBook fans to spin, freezes IDEs, and exhausts system memory.

---

## 💡 The Solution

Just like `simslim` disables background daemons on iOS simulators via `launchctl`, `avdslim`:
1. **Disables 35+ non-essential background daemons** via `adb shell pm disable-user --user 0`.
2. **Eliminates GPU frame-buffer churn** by disabling window, transition, and animator scales (0x).
3. **Restricts background process limits** (`background_process_limit = 2`).
4. **Disables account auto-sync and location polling**.
5. **Trims heap allocations** (`am trim-memory --all COMPLETE`) and drops kernel pagecaches (`/proc/sys/vm/drop_caches`).
6. **Tunes host AVD `config.ini`** to low-memory defaults (1536M RAM, no camera/audio host threads, Metal hardware acceleration).

> **What stays working?**  
> Core Android OS, WebView, Flutter/React-Native/Native runtimes, network, and Google Play Services core APIs (Firebase Auth, Cloud Messaging / FCM push notifications).

---

## 📊 Go vs Rust Head-to-Head Benchmark

Both Go and Rust implementations were built and benchmarked:

| Metric | Go (Winner) | Rust | Notes |
|---|---|---|---|
| **Binary Size** | **2.6 MB** | **0.44 MB** | Rust is ~6x smaller, but 2.6 MB is negligible. |
| **Startup Latency** | **2.61 ms** | **2.08 ms** | Imperceptible ~0.5ms difference. |
| **Incremental Build Time** | **206 ms** | **3,088 ms** | Go builds ~10x faster. |
| **External Dependencies** | **0 (100% Stdlib)** | 11 crates | Go has zero third-party dependencies. |
| **Cross-Compilation** | **Native (1 command)** | Complex (needs linkers) | Go builds for Mac/Linux/Windows out of the box. |

**Winner**: **Go**. Zero third-party dependencies, instant builds, and native cross-compilation for all platforms without external toolchains make Go the clear winner for developer tooling distribution. *(The Rust implementation is archived in `rust_comparison/` for reference.)*

---

## 🚀 Installation

### Option 1: Build from Source (Go 1.22+)
```bash
git clone https://github.com/krunalbhalala/avdslim.git
cd avdslim
make install
```

### Option 2: Pre-built Binary
Download the binary for your platform from Releases:
* macOS Apple Silicon: `avdslim-darwin-arm64`
* macOS Intel: `avdslim-darwin-amd64`
* Linux: `avdslim-linux-amd64`
* Windows: `avdslim-windows-amd64.exe`

---

## 🛠️ Usage & Commands

### 1. List Emulators & AVDs
List all running emulators with their host RSS memory consumption and slimmed status:
```bash
avdslim list
```

### 2. Measure Memory Footprint
Deep memory profiling showing both host QEMU resident memory and in-guest process breakdown:
```bash
avdslim measure
# Or specify device:
avdslim measure emulator-5554
```

### 3. Slim Down Emulator (`on`)
Disables bloat packages, zeroes animation latency, limits background processes, and trims RAM:
```bash
avdslim on
# Optional aggressive mode (also disables Play Store updater & Chrome background sync):
avdslim on --aggressive
```

### 4. Restore to Stock (`off`)
Restores all disabled packages and default settings anytime:
```bash
avdslim off
```

### 5. Tune Host AVD Config (`tune-avd`)
Directly edits `~/.android/avd/<name>.avd/config.ini` to safe low-memory settings (creates `.bak` first):
```bash
avdslim tune-avd Pixel_8_API_34 --ram=1536 --heap=256
```
* Sets `hw.ramSize = 1536` (instead of 2048/4096 MB)
* Sets `vm.heapSize = 256`
* Disables audio and camera host emulation threads (`hw.camera.back = none`, `hw.audioInput = no`)
* Forces Metal GPU hardware acceleration (`hw.gpu.mode = host`)

### 6. Launch with Low-Memory Host Flags (`launch`)
Spawns the emulator with low-memory host flags and optionally auto-slims once booted:
```bash
avdslim launch Pixel_8_API_34 --slim --ram=1536
```

---

## 📜 License

MIT License. See [LICENSE](LICENSE) for details.

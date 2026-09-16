use serde::{Deserialize, Serialize};
use std::collections::HashMap;
use std::env;
use std::fs;
use std::path::{Path, PathBuf};
use std::process::Command;
use std::thread;
use std::time::Duration;

const VERSION: &str = "1.0.0";
const STATE_FILE_PATH: &str = "/data/local/tmp/avdslim_state.json";

const STANDARD_BLOAT: &[(&str, &[&str])] = &[
    (
        "Assistant & Search (~250-400 MB RAM)",
        &[
            "com.google.android.googlequicksearchbox",
            "com.google.android.as",
            "com.google.android.as.oss",
        ],
    ),
    (
        "Media & Consumer Apps (~150-250 MB RAM)",
        &[
            "com.google.android.apps.photos",
            "com.google.android.youtube",
            "com.google.android.videos",
            "com.google.android.music",
            "com.google.android.apps.youtube.music",
            "com.google.android.apps.maps",
            "com.google.android.gm",
            "com.android.gallery3d",
            "com.android.music",
        ],
    ),
    (
        "Telephony & Messaging Daemons (~100-150 MB RAM)",
        &[
            "com.google.android.apps.messaging",
            "com.google.android.dialer",
            "com.google.android.contacts",
            "com.android.mms",
            "com.android.dialer",
            "com.android.contacts",
        ],
    ),
    (
        "Accessibility & Speech (~50-100 MB RAM)",
        &[
            "com.google.android.tts",
            "com.google.android.marvin.talkback",
        ],
    ),
    (
        "Printing & Background Spoolers (~30-60 MB RAM)",
        &[
            "com.android.printspooler",
            "com.android.bips",
            "com.google.android.printservice.recommendation",
        ],
    ),
    (
        "Telemetry, Feedback & Screensavers (~40-80 MB RAM)",
        &[
            "com.google.android.feedback",
            "com.android.traceur",
            "com.android.dreams.basic",
            "com.android.dreams.phototable",
            "com.android.wallpaper.livepicker",
            "com.google.android.apps.wallpaper",
            "com.android.emergency",
            "com.android.calendar",
            "com.google.android.calendar",
            "com.android.deskclock",
            "com.google.android.deskclock",
            "com.android.calculator2",
            "com.google.android.calculator",
        ],
    ),
];

const AGGRESSIVE_BLOAT: &[&str] = &[
    "com.android.vending",
    "com.android.chrome",
    "com.google.android.partnersetup",
    "com.google.android.setupwizard",
    "com.android.setupwizard",
];

#[derive(Debug, Serialize, Deserialize)]
struct SlimState {
    timestamp: String,
    disabled_packages: Vec<String>,
    preset: String,
}

#[derive(Debug)]
struct RunningEmulator {
    serial: String,
    model: String,
    android_version: String,
    api_level: String,
    host_pid: Option<u32>,
    host_rss_mb: Option<u64>,
    is_slimmed: bool,
}

fn main() {
    let args: Vec<String> = env::args().collect();
    if args.len() < 2 {
        print_usage();
        return;
    }

    let cmd = args[1].to_lowercase();
    let sub_args = &args[2..];

    let adb = find_adb();
    let emulator = find_emulator();

    match cmd.as_str() {
        "help" | "-h" | "--help" => print_usage(),
        "version" | "-v" | "--version" => println!("avdslim version {} (rust)", VERSION),
        "list" => cmd_list(&adb),
        "measure" => cmd_measure(&adb, sub_args),
        "on" | "slim" => cmd_on(&adb, sub_args),
        "off" | "unslim" => cmd_off(&adb, sub_args),
        "tune-avd" | "tune" => cmd_tune_avd(sub_args),
        "launch" => cmd_launch(&adb, &emulator, sub_args),
        _ => {
            eprintln!("❌ Unknown command: {}\n", cmd);
            print_usage();
            std::process::exit(1);
        }
    }
}

fn print_usage() {
    println!(
        r#"════════════════════════════════════════════════════════════════════════
 ⚡ AVD-SLIM — Android Emulator RAM & CPU Optimizer
 (Inspired by simslim for iOS simulators) — v{} [Rust]
════════════════════════════════════════════════════════════════════════

Usage:
  avdslim <command> [arguments]

Commands:
  list                 List running emulators (with host RAM) & saved AVDs
  measure [device]     Deep memory breakdown (host QEMU RSS + guest dumpsys)
  on [device]          Slim down emulator: disable bloat daemons & trim RAM
                       Options: --aggressive (also disables Play Store updater)
  off [device]         Restore disabled packages and default settings
  tune-avd [avd_name]  Tune host AVD config.ini (RAM=1536M, no cameras, host GPU)
                       Options: --ram=<MB> (default: 1536), --heap=<MB> (default: 256)
  launch <avd_name>    Launch AVD with low-memory host flags (-no-audio, etc.)
                       Options: --slim (auto-slim once booted), --ram=<MB>
  version              Print avdslim version

Examples:
  avdslim list
  avdslim measure
  avdslim on
  avdslim on emulator-5554 --aggressive
  avdslim off
  avdslim tune-avd Pixel_8_API_34 --ram=1536
  avdslim launch Pixel_8_API_34 --slim
"#,
        VERSION
    );
}

fn cmd_list(adb: &str) {
    println!("🔎 Checking running Android emulators...");
    let running = get_running_emulators(adb);

    if running.is_empty() {
        println!("ℹ️  No running Android emulators detected via adb.\n");
    } else {
        println!("\n📱 Running Emulators ({}):", running.len());
        for emu in &running {
            let host_ram = match emu.host_rss_mb {
                Some(rss) => format!("{} MB", rss),
                None => "unknown".to_string(),
            };
            let status = if emu.is_slimmed {
                "⚡ SLIMMED"
            } else {
                "🔴 FULL (Stock)"
            };
            println!(
                "  • {} ({}, Android {}, API {})",
                emu.serial, emu.model, emu.android_version, emu.api_level
            );
            println!(
                "    Host PID: {} | Host RAM (RSS): {} | Status: {}",
                emu.host_pid
                    .map(|p| p.to_string())
                    .unwrap_or_else(|| "unknown".to_string()),
                host_ram,
                status
            );
        }
        println!();
    }

    println!("💾 Installed AVD Configurations:");
    let avds = get_installed_avds();
    if avds.is_empty() {
        println!("  (No AVDs found in ~/.android/avd)\n");
    } else {
        for avd in avds {
            let name = avd.get("name").cloned().unwrap_or_default();
            let ram = avd.get("hw.ramSize").cloned().unwrap_or_else(|| "unknown".to_string());
            let heap = avd.get("vm.heapSize").cloned().unwrap_or_else(|| "unknown".to_string());
            let gpu = avd.get("hw.gpu.mode").cloned().unwrap_or_else(|| "unknown".to_string());
            println!("  • {} (RAM: {}MB, Heap: {}MB, GPU: {})", name, ram, heap, gpu);
        }
        println!();
    }
}

fn cmd_measure(adb: &str, args: &[String]) {
    let serial = match resolve_device(adb, args) {
        Some(s) => s,
        None => return,
    };

    println!("📊 Measuring memory footprint for {}...\n", serial);

    // 1. Host Memory
    if let Some(pid) = find_host_pid_for_serial(&serial) {
        let rss = get_host_rss_mb(pid);
        println!("🖥️  HOST (macOS) Footprint:");
        println!("   QEMU / Emulator PID: {}", pid);
        println!("   Host Resident RAM (RSS): {} MB\n", rss);
    }

    // 2. Guest Memory
    let out = exec_cmd(adb, &["-s", &serial, "shell", "dumpsys", "meminfo"]);
    print_guest_meminfo(&out);
}

fn cmd_on(adb: &str, args: &[String]) {
    let aggressive = args.iter().any(|a| a == "--aggressive");
    let filtered_args: Vec<String> = args
        .iter()
        .filter(|a| !a.starts_with("--"))
        .cloned()
        .collect();

    let serial = match resolve_device(adb, &filtered_args) {
        Some(s) => s,
        None => return,
    };

    let preset_name = if aggressive { "Aggressive" } else { "Standard" };
    println!(
        "⚡ Slimming Android Emulator ({}) [preset: {}]...\n",
        serial, preset_name
    );

    let host_pid = find_host_pid_for_serial(&serial);
    let before_rss = host_pid.map(get_host_rss_mb);

    let mut target_packages = Vec::new();
    for (_cat, pkgs) in STANDARD_BLOAT {
        for pkg in *pkgs {
            target_packages.push(*pkg);
        }
    }
    if aggressive {
        for pkg in AGGRESSIVE_BLOAT {
            target_packages.push(*pkg);
        }
    }

    let installed_raw = exec_cmd(adb, &["-s", &serial, "shell", "pm", "list", "packages"]);
    let installed_set: HashMap<String, bool> = installed_raw
        .lines()
        .map(|l| l.trim().replace("package:", ""))
        .filter(|l| !l.is_empty())
        .map(|pkg| (pkg, true))
        .collect();

    let mut disabled_list = Vec::new();
    let mut skipped_count = 0;

    println!("1. Disabling non-essential background daemons:");
    for pkg in &target_packages {
        if !installed_set.contains_key(*pkg) {
            skipped_count += 1;
            continue;
        }

        let res = exec_cmd(
            adb,
            &["-s", &serial, "shell", "pm", "disable-user", "--user", "0", pkg],
        );
        if res.contains("disabled-user") || res.contains("new state") {
            disabled_list.push(pkg.to_string());
            println!("   ✓ Disabled: {}", pkg);
        }
    }
    println!(
        "   -> Successfully disabled {} packages ({} not installed in image).\n",
        disabled_list.len(),
        skipped_count
    );

    // Save state
    let state = SlimState {
        timestamp: "now".to_string(),
        disabled_packages: disabled_list,
        preset: preset_name.to_string(),
    };
    if let Ok(json_str) = serde_json::to_string(&state) {
        exec_cmd(
            adb,
            &[
                "-s",
                &serial,
                "shell",
                "echo",
                &format!("'{}'", json_str),
                ">",
                STATE_FILE_PATH,
            ],
        );
    }

    // 2. Tune Settings
    println!("2. Tuning Android system settings:");
    exec_cmd(adb, &["-s", &serial, "shell", "settings", "put", "global", "window_animation_scale", "0"]);
    exec_cmd(adb, &["-s", &serial, "shell", "settings", "put", "global", "transition_animation_scale", "0"]);
    exec_cmd(adb, &["-s", &serial, "shell", "settings", "put", "global", "animator_duration_scale", "0"]);
    println!("   ✓ Window & transition animations set to 0x (instant UI, no GPU buffer churn)");

    exec_cmd(adb, &["-s", &serial, "shell", "settings", "put", "global", "background_process_limit", "2"]);
    println!("   ✓ Background process limit set to 2");

    exec_cmd(adb, &["-s", &serial, "shell", "settings", "put", "global", "auto_sync", "0"]);
    println!("   ✓ Background account auto-sync disabled");

    exec_cmd(adb, &["-s", &serial, "shell", "settings", "put", "secure", "location_mode", "0"]);
    println!("   ✓ Background location polling disabled\n");

    // 3. Trim Memory
    println!("3. Purging cached processes & trimming memory:");
    exec_cmd(adb, &["-s", &serial, "shell", "am", "kill-all"]);
    exec_cmd(adb, &["-s", &serial, "shell", "am", "trim-memory", "--all", "COMPLETE"]);

    let _ = exec_cmd(adb, &["-s", &serial, "shell", "su", "0", "sync"]);
    let _ = exec_cmd(adb, &["-s", &serial, "shell", "su", "0", "echo 3 > /proc/sys/vm/drop_caches"]);
    println!("   ✓ am kill-all & trim-memory COMPLETE executed.\n");

    thread::sleep(Duration::from_secs(1));

    let after_rss = host_pid.map(get_host_rss_mb);

    println!("══════════════════════════════════════════════════════════════");
    println!("🎉 Slimming complete for {}!", serial);
    if let (Some(b), Some(a)) = (before_rss, after_rss) {
        let diff = if b > a { b - a } else { 0 };
        println!("🖥️  Host RAM (RSS): {}MB -> {}MB (Reclaimed: {}MB)", b, a, diff);
    }
    println!("ℹ️  To restore default stock services anytime:\n   avdslim off {}\n", serial);
}

fn cmd_off(adb: &str, args: &[String]) {
    let serial = match resolve_device(adb, args) {
        Some(s) => s,
        None => return,
    };

    println!("🔄 Restoring default services for {}...\n", serial);

    let mut packages_to_enable = Vec::new();
    let state_raw = exec_cmd(adb, &["-s", &serial, "shell", "cat", STATE_FILE_PATH]);
    if let Ok(state) = serde_json::from_str::<SlimState>(&state_raw) {
        packages_to_enable = state.disabled_packages;
    }

    if packages_to_enable.is_empty() {
        for (_cat, pkgs) in STANDARD_BLOAT {
            for pkg in *pkgs {
                packages_to_enable.push(pkg.to_string());
            }
        }
        for pkg in AGGRESSIVE_BLOAT {
            packages_to_enable.push(pkg.to_string());
        }
    }

    println!("1. Re-enabling {} packages:", packages_to_enable.len());
    let mut restored_count = 0;
    for pkg in &packages_to_enable {
        let res = exec_cmd(adb, &["-s", &serial, "shell", "pm", "enable", pkg]);
        if res.contains("enabled") || res.contains("new state") {
            restored_count += 1;
            println!("   ✓ Enabled: {}", pkg);
        }
    }
    println!("   -> Restored {} packages.\n", restored_count);

    println!("2. Restoring default system settings:");
    exec_cmd(adb, &["-s", &serial, "shell", "settings", "put", "global", "window_animation_scale", "1"]);
    exec_cmd(adb, &["-s", &serial, "shell", "settings", "put", "global", "transition_animation_scale", "1"]);
    exec_cmd(adb, &["-s", &serial, "shell", "settings", "put", "global", "animator_duration_scale", "1"]);
    exec_cmd(adb, &["-s", &serial, "shell", "settings", "delete", "global", "background_process_limit"]);
    exec_cmd(adb, &["-s", &serial, "shell", "settings", "put", "global", "auto_sync", "1"]);
    exec_cmd(adb, &["-s", &serial, "shell", "settings", "put", "secure", "location_mode", "3"]);
    println!("   ✓ Animations reset to 1.0x");
    println!("   ✓ Background process limit restored");
    println!("   ✓ Auto-sync restored\n");

    exec_cmd(adb, &["-s", &serial, "shell", "rm", "-f", STATE_FILE_PATH]);
    println!("✅ Successfully restored {} to stock configuration.\n", serial);
}

fn cmd_tune_avd(args: &[String]) {
    let mut ram_mb = 1536;
    let mut heap_mb = 256;
    let mut target_avd = None;

    for a in args {
        if a.starts_with("--ram=") {
            if let Ok(v) = a["--ram=".len()..].parse::<u32>() {
                ram_mb = v;
            }
        } else if a.starts_with("--heap=") {
            if let Ok(v) = a["--heap=".len()..].parse::<u32>() {
                heap_mb = v;
            }
        } else if !a.starts_with("--") {
            target_avd = Some(a.clone());
        }
    }

    let home = env::var("HOME").unwrap_or_default();
    let avd_base = PathBuf::from(home).join(".android").join("avd");
    if !avd_base.exists() {
        eprintln!("❌ AVD directory ~/.android/avd not found.");
        return;
    }

    let mut config_files = Vec::new();
    if let Ok(entries) = fs::read_dir(&avd_base) {
        for entry in entries.flatten() {
            let path = entry.path();
            if path.is_dir() && path.to_string_lossy().ends_with(".avd") {
                let cfg = path.join("config.ini");
                if cfg.exists() {
                    config_files.push(cfg);
                }
            }
        }
    }

    if config_files.is_empty() {
        println!("ℹ️  No AVD configurations found in ~/.android/avd.");
        return;
    }

    let file_to_tune = match target_avd {
        Some(target) => config_files
            .into_iter()
            .find(|f| {
                f.parent()
                    .and_then(|p| p.file_name())
                    .map(|n| n.to_string_lossy().replace(".avd", "").eq_ignore_ascii_case(&target))
                    .unwrap_or(false)
            }),
        None => {
            if config_files.len() == 1 {
                Some(config_files.remove(0))
            } else {
                println!("Found multiple AVDs. Please specify one:");
                for f in config_files {
                    if let Some(p) = f.parent().and_then(|p| p.file_name()) {
                        println!("  • {}", p.to_string_lossy().replace(".avd", ""));
                    }
                }
                return;
            }
        }
    };

    let cfg_path = match file_to_tune {
        Some(p) => p,
        None => {
            eprintln!("❌ Target AVD not found.");
            return;
        }
    };

    let avd_name = cfg_path
        .parent()
        .and_then(|p| p.file_name())
        .map(|n| n.to_string_lossy().replace(".avd", ""))
        .unwrap_or_else(|| "AVD".to_string());

    println!(
        "⚙️  Tuning config.ini for {:?} (RAM: {}MB, Heap: {}MB)...",
        avd_name, ram_mb, heap_mb
    );

    let content = fs::read_to_string(&cfg_path).unwrap_or_default();
    let backup = cfg_path.with_extension("ini.bak");
    if !backup.exists() {
        let _ = fs::write(&backup, &content);
        println!("   ✓ Created backup: {:?}", backup);
    }

    let mut map: HashMap<String, String> = HashMap::new();
    for line in content.lines() {
        if let Some(idx) = line.find('=') {
            let k = line[..idx].trim().to_string();
            let v = line[idx + 1..].trim().to_string();
            map.insert(k, v);
        }
    }

    map.insert("hw.ramSize".into(), ram_mb.to_string());
    map.insert("vm.heapSize".into(), heap_mb.to_string());
    map.insert("hw.camera.back".into(), "none".into());
    map.insert("hw.camera.front".into(), "none".into());
    map.insert("hw.audioInput".into(), "no".into());
    map.insert("hw.audioOutput".into(), "no".into());
    map.insert("hw.gpu.mode".into(), "host".into());
    map.insert("hw.gpu.enabled".into(), "yes".into());
    map.insert("hw.dPad".into(), "no".into());

    let mut out = String::new();
    for (k, v) in map {
        out.push_str(&format!("{}={}\n", k, v));
    }

    if let Err(e) = fs::write(&cfg_path, out) {
        eprintln!("Error writing config: {}", e);
        return;
    }

    println!("✅ Successfully tuned AVD {:?}!", avd_name);
    println!("   • Host RAM allocated: {} MB (prevents host memory pressure)", ram_mb);
    println!("   • VM Heap: {} MB", heap_mb);
    println!("   • Hardware Audio & Camera: disabled (saves host threads/buffers)");
    println!("   • GPU Mode: host (uses Apple Silicon Metal hardware acceleration)\n");
}

fn cmd_launch(adb: &str, emulator: &str, args: &[String]) {
    if args.is_empty() || args[0].starts_with("--") {
        eprintln!("❌ Please specify an AVD name to launch.");
        eprintln!("   Example: avdslim launch Pixel_8_API_34 --slim");
        return;
    }

    let avd_name = &args[0];
    let do_slim = args.iter().any(|a| a == "--slim");
    let mut ram_mb = 1536;

    for a in &args[1..] {
        if a.starts_with("--ram=") {
            if let Ok(v) = a["--ram=".len()..].parse::<u32>() {
                ram_mb = v;
            }
        }
    }

    let emu_args = [
        "-avd",
        avd_name,
        "-memory",
        &ram_mb.to_string(),
        "-no-audio",
        "-camera-back",
        "none",
        "-camera-front",
        "none",
        "-gpu",
        "host",
        "-no-boot-anim",
    ];

    println!(
        "🚀 Launching emulator {:?} with low-memory host flags:\n   emulator {}\n",
        avd_name,
        emu_args.join(" ")
    );

    match Command::new(emulator).args(&emu_args).spawn() {
        Ok(child) => println!("✓ Emulator process spawned (PID: {}).", child.id()),
        Err(e) => {
            eprintln!("Failed to launch emulator: {}", e);
            return;
        }
    }

    if do_slim {
        println!("⏳ Waiting for emulator to finish booting...");
        let _ = exec_cmd(adb, &["wait-for-device"]);

        let mut booted = false;
        for _ in 0..60 {
            let res = exec_cmd(adb, &["shell", "getprop", "sys.boot_completed"]);
            if res.trim() == "1" {
                booted = true;
                break;
            }
            thread::sleep(Duration::from_secs(2));
        }

        if booted {
            println!("✓ Boot complete! Applying avdslim optimizations...");
            cmd_on(adb, &[]);
        } else {
            println!("⚠️  Boot timed out after 120s. You can run `avdslim on` manually.");
        }
    }
}

fn get_running_emulators(adb: &str) -> Vec<RunningEmulator> {
    let out = exec_cmd(adb, &["devices"]);
    let mut list = Vec::new();

    for line in out.lines().skip(1) {
        let parts: Vec<&str> = line.split_whitespace().collect();
        if parts.len() >= 2 && parts[1] == "device" && parts[0].starts_with("emulator-") {
            let serial = parts[0];
            let model = exec_cmd(adb, &["-s", serial, "shell", "getprop", "ro.product.model"]).trim().to_string();
            let ver = exec_cmd(adb, &["-s", serial, "shell", "getprop", "ro.build.version.release"]).trim().to_string();
            let sdk = exec_cmd(adb, &["-s", serial, "shell", "getprop", "ro.build.version.sdk"]).trim().to_string();

            let state_exists = exec_cmd(adb, &["-s", serial, "shell", "ls", STATE_FILE_PATH]);
            let is_slimmed = state_exists.contains("avdslim_state.json");

            let host_pid = find_host_pid_for_serial(serial);
            let host_rss_mb = host_pid.map(get_host_rss_mb);

            list.push(RunningEmulator {
                serial: serial.to_string(),
                model,
                android_version: ver,
                api_level: sdk,
                host_pid,
                host_rss_mb,
                is_slimmed,
            });
        }
    }
    list
}

fn get_installed_avds() -> Vec<HashMap<String, String>> {
    let home = env::var("HOME").unwrap_or_default();
    let avd_base = PathBuf::from(home).join(".android").join("avd");
    let mut list = Vec::new();

    if let Ok(entries) = fs::read_dir(&avd_base) {
        for entry in entries.flatten() {
            let path = entry.path();
            if path.is_dir() && path.to_string_lossy().ends_with(".avd") {
                let cfg = path.join("config.ini");
                if let Ok(content) = fs::read_to_string(&cfg) {
                    let mut map = HashMap::new();
                    let name = path
                        .file_name()
                        .map(|n| n.to_string_lossy().replace(".avd", ""))
                        .unwrap_or_default();
                    map.insert("name".into(), name);
                    for l in content.lines() {
                        if let Some(idx) = l.find('=') {
                            let k = l[..idx].trim().to_string();
                            let v = l[idx + 1..].trim().to_string();
                            map.insert(k, v);
                        }
                    }
                    list.push(map);
                }
            }
        }
    }
    list
}

fn resolve_device(adb: &str, args: &[String]) -> Option<String> {
    if !args.is_empty() && !args[0].starts_with("--") {
        return Some(args[0].clone());
    }
    let emus = get_running_emulators(adb);
    if emus.is_empty() {
        eprintln!("❌ No running Android emulator found.");
        eprintln!("   Please start an emulator or pass device serial: `avdslim on emulator-5554`");
        return None;
    }
    if emus.len() == 1 {
        return Some(emus[0].serial.clone());
    }
    println!("⚠️  Multiple emulators running. Please specify one:");
    for e in emus {
        println!("  • {} ({})", e.serial, e.model);
    }
    None
}

fn find_host_pid_for_serial(serial: &str) -> Option<u32> {
    let port_str = serial.trim_start_matches("emulator-");

    // 1. Try lsof on console port (e.g. 5554)
    if !port_str.is_empty() {
        let out = exec_cmd("lsof", &["-i", &format!(":{}", port_str), "-sTCP:LISTEN", "-t"]);
        if let Some(first) = out.split_whitespace().next() {
            if let Ok(pid) = first.parse::<u32>() {
                if pid > 0 {
                    return Some(pid);
                }
            }
        }
    }

    // 2. Fallback to process inspection
    let out = exec_cmd("ps", &["-eo", "pid,command"]);
    let my_pid = std::process::id();
    let mut candidates = Vec::new();

    for line in out.lines() {
        if line.contains("avdslim") || line.contains("crashpad") || line.contains("netsimd") {
            continue;
        }

        if line.contains("qemu-system") {
            if let Some(first) = line.split_whitespace().next() {
                if let Ok(pid) = first.parse::<u32>() {
                    if pid != my_pid {
                        if line.contains(&format!("-port {}", port_str)) || line.contains(port_str) {
                            return Some(pid);
                        }
                        candidates.push(pid);
                    }
                }
            }
        }
    }

    candidates.into_iter().next()
}

fn get_host_rss_mb(pid: u32) -> u64 {
    let out = exec_cmd("ps", &["-p", &pid.to_string(), "-o", "rss="]);
    if let Ok(kb) = out.trim().parse::<u64>() {
        return kb / 1024;
    }
    0
}

fn print_guest_meminfo(raw: &str) {
    println!("📱 GUEST (Android OS) Memory Breakdown:");
    for l in raw.lines() {
        if l.contains("Total RAM:")
            || l.contains("Free RAM:")
            || l.contains("Used RAM:")
            || l.contains("Lost RAM:")
        {
            println!("   {}", l.trim());
        }
    }
    println!();

    println!("🔝 Top Memory-Consuming Processes in Guest:");
    let mut found = false;
    let mut count = 0;
    for l in raw.lines() {
        if l.contains("Total PSS by process:") {
            found = true;
            continue;
        }
        if found {
            let t = l.trim();
            if t.is_empty() || t.starts_with("Total PSS by category:") {
                break;
            }
            if t.contains("K:") && count < 10 {
                println!("   {}", t);
                count += 1;
            }
        }
    }
    println!();
}

fn exec_cmd(bin: &str, args: &[&str]) -> String {
    match Command::new(bin).args(args).output() {
        Ok(out) => {
            let mut s = String::from_utf8_lossy(&out.stdout).to_string();
            s.push_str(&String::from_utf8_lossy(&out.stderr));
            s
        }
        Err(_) => String::new(),
    }
}

fn find_adb() -> String {
    let home = env::var("HOME").unwrap_or_default();
    let p = Path::new(&home)
        .join("Library")
        .join("Android")
        .join("sdk")
        .join("platform-tools")
        .join("adb");
    if p.exists() {
        return p.to_string_lossy().to_string();
    }
    "adb".to_string()
}

fn find_emulator() -> String {
    let home = env::var("HOME").unwrap_or_default();
    let p = Path::new(&home)
        .join("Library")
        .join("Android")
        .join("sdk")
        .join("emulator")
        .join("emulator");
    if p.exists() {
        return p.to_string_lossy().to_string();
    }
    "emulator".to_string()
}

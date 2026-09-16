# Runbook — avdslim

## Prerequisites

Android SDK (`ANDROID_HOME`/`ANDROID_SDK_ROOT` or OS-default path) with
`platform-tools/adb` and `emulator/`; at least one AVD. Recommended image:
**Google APIs 4 KB** (not Google Play, not 16 KB `ps16k`) — see README
"Golden SDK Recommendation".

## Run / build / verify

```bash
make build          # → bin/avdslim
make install        # copies to ~/.local/bin or /usr/local/bin
go test ./...       # vacuous: no test files exist
go vet ./...        # run manually; no CI gate
./bin/avdslim doctor        # env audit, safe read-only
./bin/avdslim list          # running emulators + installed AVDs
./bin/avdslim on            # slim running emulator (add --aggressive/--keep=)
./bin/avdslim off           # restore stock services
```

Release: push tag `v*` → `release.yml` builds 4 platform tarballs + checksums
and publishes. Then bump `VERSION` (`Makefile`), `version` const
(`cmd/avdslim/main.go`), `install.sh`, and `Formula/avdslim.rb` — these are
manual, nothing syncs them.

## Deploy surface

`install.sh` (pinned `VERSION`, darwin/linux only — no Windows branch),
Homebrew tap (`brew tap kdbhalala/avdslim`), `go install
github.com/kdbhalala/avdslim/cmd/avdslim@latest` (note: go.mod module path is
`github.com/krunalbhalala/avdslim`), GitHub Action (`action.yml`:
installs binary, `install-shim`, optional `watch` daemon).

## Known failure modes

- `measure`/`list` show ~5 MB or PID 0 → QEMU not found: `lsof` missing or
  serial port mismatch; falls back to `ps` scan excluding
  `avdslim|crashpad|netsimd`. Never remove those filters.
- `start` ignores `--ram` on 16 KB images → QEMU enforces 4096 MB floor;
  switch to a 4 KB Google APIs image.
- `Slim` memory trim silently no-ops on Google Play images → `su 0`
  unavailable (production build locks `adb root`).
- Golden boot loads stale heavy state → `hardware-qemu.ini`/`snapshots/`
  not purged; `tune-avd`/`restart` purge them, `start --cold` bypasses load.
- Shim breaks Studio launches → original binary is at `emulator.real`;
  `avdslim uninstall-shim` renames it back. Shim install needs write access
  to the SDK `emulator/` dir.
- Bare `config.ini` rewrite drops comments/ordering (map round-trip) — cosmetic.

# Testing & QA — avdslim

## Commands (all verified in `Makefile` / `release.yml`)

- `go build ./...` / `make build` → `bin/avdslim` (`-s -w` ldflags, version
  injected via `-X main.version`; `VERSION=1.0.11` in `Makefile` — bump together
  with the `version` var in `cmd/avdslim/main.go` and `install.sh`).
- `go vet ./...` — run before any PR; no lint config exists.
- `go test ./...` / `make test` — unit tests in `internal/adb/client_test.go`:
  checks Slim/Restore exact inverse, target resolution, and selective feature/app
  enabling (`TestEnableTarget`) against a fake bash `adb` (skipped on Windows).
- `make cross` — CGO-free builds for darwin-arm64/amd64, linux-amd64,
  windows-amd64.
- `gofmt -l .` — must be clean; no formatter config in repo.

## CI (`.github/workflows/`)

- `ci.yml` — on push & pull_request to `main`: runs `gofmt` verification,
  `go vet ./...`, `go test -v ./...`, and binary compilation.
- `release.yml` — on tag `v*` only: builds 4 platform tarballs
  (`avdslim_<ver>_<os>_<arch>.tar.gz` + README/LICENSE), writes
  `checksums.txt`, publishes via `softprops/action-gh-release`. Go 1.22.
- `github-repo-stats.yml` — monitoring workflow, unrelated to code quality.

## Standard to hold

- Most logic shells out to `adb`/`emulator`/`lsof`/`ps` and mutates a live
  emulator or `~/.android/avd` — it cannot be unit-tested without a device.
  Keep new pure logic (package-list filtering, arg parsing, ini editing,
  GPU-mode selection) in `internal/` packages so it is testable without adb.
- `doctor` must stay read-only; `Slim`/`Restore` must stay exact inverses.
  Guest settings live in the `tweaks` table (`internal/adb/client.go`); Slim
  records each original value in the state JSON before changing it and
  Restore puts it back, so add new settings to that table, not as raw
  `settings put` calls.
- Manual verification when touching device-mutating code: `avdslim doctor`,
  `avdslim on` / `avdslim off`, `avdslim measure` against a running emulator.
  State this was done (or why it wasn't possible) in the PR/summary.

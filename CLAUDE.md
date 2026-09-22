# AGENTS.md — avdslim

Zero-dependency Go CLI: cuts Android emulator host RAM (~8 GB → ~1.5 GB) via
`-lowram` + hardware GPU, adb bloat disabling, and a pre-baked `avdslim_clean`
snapshot for ~1.5 s boots.

- Stack: Go 1.22, stdlib only (zero `go.mod` requires). Darwin/Linux/Windows.
- Entry: `cmd/avdslim/main.go` (hand-rolled arg parsing, all `handle*` cmds).
- Code: `internal/adb` (guest mutations) · `internal/bloat` (package lists) ·
  `internal/config` (SDK/AVD paths, `config.ini` tuner) ·
  `internal/host` (QEMU PID/memory probing) · `internal/shim` (Studio wrapper) ·
  `internal/doctor` (read-only audit) · `internal/adbtest` (fake adb, tests only).
- State: host `~/.android/avd/<name>.avd/` (`config.ini`, `snapshots/avdslim_clean/`);
  guest `/data/local/tmp/avdslim_state.json`. No DB.
- Verify: `make build`, `go vet ./...`, `gofmt -l .`; `go test ./...`
  (all gated in CI via `.github/workflows/ci.yml`).
  `on`/`off`/`enable`/`doctor` are tested end-to-end against a fake adb
  (`cmd/avdslim/main_test.go`); a live emulator is still the final check.
- Explore: `codegraph explore "<symbol or question>"` (index in `.codegraph/`,
  auto-syncs); `gopls` for definitions/references.
- Module path `github.com/kdbhalala/avdslim` must match the remote, or
  `go install` breaks.

Read first:

- `rules/architecture.md` — components, dependency direction, invariants
- `rules/testing-qa.md` — verify commands, CI, QA standard
- `rules/memory-discipline.md` — when to recall/record
- `context/data-model.md` — entities, file state, ownership
- `context/runbook.md` — run/build/release, failure modes

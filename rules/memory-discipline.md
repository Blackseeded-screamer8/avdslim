# Memory discipline — avdslim

Small single-binary Go CLI; most sessions need no memory calls. Use them only
for durable, cross-session value.

## Recall (`memory_recall`, project=`avdslim`)

- Before changing the bloat lists: recall `bloat packages allowlist` — a
  wrongly added package breaks FCM/Auth/WebView with no error.
- Before touching PID detection or memory measurement: recall
  `host PID footprint` — the self-match bug (`measure` reporting its own RSS)
  has been fixed once; don't reintroduce it.
- On any `config.ini`/snapshot/golden-boot change: recall
  `golden snapshot` + `lavapipe 8GB` for the stale-state purge rationale.
- Keywords carry synonyms: `slim debloat disable`, `pid qemu footprint rss`,
  `snapshot bake golden quickboot`, `shim emulator wrapper`.

## Record (`memory_record`, project=`avdslim`)

Record only: new root-cause findings (measured numbers, not theories),
changes to the protected-vs-bloat boundary, new emulator flags with verified
effect, recurring user-environment failures (e.g. Play vs APIs image issues).

- `category="bugfix"` for root causes with numbers (before/after MB).
- `category="architecture"` for boundary/invariant changes.
- `origin="user-confirmed"` only when the user stated it; otherwise
  `agent-inferred`.

## Pin (`memory_pin`, project=`avdslim`)

Pin only invariants that would silently break users if violated:
FCM/Auth/WebView never-disabled set, `-no-snapshot-save` on golden boots,
PID self-exclusion filter. If it is already in `rules/architecture.md`,
don't duplicate it in a pin.

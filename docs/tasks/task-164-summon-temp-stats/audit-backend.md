# Backend Audit — task-164-summon-temp-stats (Go diff)

- **Scope:** `libs/atlas-packet/model/character_temporary_stat.go` (+18), `libs/atlas-packet/model/character_temporary_stat_test.go` (+158)
- **Guidelines Source:** backend-dev-guidelines skill
- **Date:** 2026-08-07
- **Build:** PASS
- **Vet:** PASS
- **Tests:** PASS (`go test -race ./model/... -count=1` → ok, `github.com/Chronicle20/atlas/libs/atlas-packet/model`)
- **Overall:** PASS

## Build & Test Results

```
$ cd libs/atlas-packet && go build ./...          → clean, no output
$ go vet ./...                                     → clean, no output
$ go test -race ./model/... -count=1               → ok  github.com/Chronicle20/atlas/libs/atlas-packet/model  1.103s
$ tools/goroutine-guard.sh                          → exit 0 (libs/atlas-packet listed among scanned dirs, no violations)
$ grep os.Getenv character_temporary_stat.go        → no match
```

## Domain classification

`libs/atlas-packet/model` is a shared packet-codec library package, not a DDD
service domain (no `processor.go`/`resource.go`/`rest.go`/`entity.go`/JSON:API
transport, no Kafka handlers, no tenant DB). The DOM-* checklist (builder/
ToEntity/Make/Transform/Processor-FieldLogger/RegisterInputHandler/etc.) and
the File Responsibilities table target REST/Kafka domain-service packages and
do not apply to this file's architecture. This is a genuine "not applicable by
kind" case, not a bar lowered for prevalence — the same audit applied to an
actual atlas-* service's `internal/<domain>` package would run the full DOM-*
table. Checks below are the subset that generically apply to any Go change in
this repo, plus the design-constraint checks called out in the audit
instructions.

## Applicable Checks

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-21 | No duplication of atlas-constants types | PASS | `serverOnlyStatNames` (character_temporary_stat.go:633-636) keys off `character.TemporaryStatType`, the existing atlas-constants type (`libs/atlas-constants/character/temporary_stat.go:123-124` already declares `TemporaryStatTypeSummon`/`TemporaryStatTypePuppet`). No new type, enum, or numeric constant was declared; the change adds a `map[character.TemporaryStatType]bool` literal only. |
| DOM-26 | Goroutines spawned via routine.Go | PASS | `grep -nE '^\s*go (func\|[A-Za-z_])'` over both changed files: zero matches. `tools/goroutine-guard.sh` exits 0. No goroutines added at all. |
| DOM-12 (generic, no service handlers here but checked anyway) | No `os.Getenv()` | PASS | `grep os.Getenv character_temporary_stat.go` → no match. |
| Design §3.2/§3.3 constraint | Classification stays in `libs/atlas-packet`, not exposed as a domain `ServerOnly()` method, not pushed into atlas-channel writers | PASS | `serverOnlyStatNames` is an unexported (lowercase) package-level `var` in `libs/atlas-packet/model/character_temporary_stat.go:633`, consulted only inside `AddStat` (same file, line 642). `grep -rn serverOnlyStatNames` across `libs/atlas-packet/` and `services/atlas-channel/` returns matches only from this one file. No exported method, no changes under `services/atlas-channel/`. |
| Design constraint | `buildCharacterTemporaryStatRegistry` untouched | PASS | `git diff 11261fb22..e4d76d1bf -- libs/atlas-packet/model/character_temporary_stat.go` shows the sole hunk at lines 621-648 (the `HasDisease`/`AddStat` region); `buildCharacterTemporaryStatRegistry` (lines 63-262 in the current file) has zero diff lines against it. |
| Design constraint | `AddStat` signature unchanged | PASS | Signature at character_temporary_stat.go:638 (`func (m *CharacterTemporaryStat) AddStat(l logrus.FieldLogger) func(t tenant.Model) func(n string, sourceId int32, amount int32, level byte, expiresAt time.Time)`) is identical pre/post-diff per the hunk context lines; the diff inserts only a guard clause inside the existing innermost closure, no new parameter. |
| Design constraint | Skip scoped to `CharacterTemporaryStat` only, `MonsterTemporaryStat` untouched | PASS | `MonsterTemporaryStat.AddStat` (libs/atlas-packet/model/monster.go:447-471) has no `serverOnly`/skip logic and is not part of the diff (file not in the changed-files list; confirmed via grep — no `serverOnlyStatNames` reference in monster.go). |
| Error-path regression guard | Unknown (genuinely unregistered) stat names still ERROR-log and drop, unaffected by the new skip | PASS | `TestCTSAddStatUnknownNameStillErrors` (character_temporary_stat_test.go:985-1014) asserts exactly one ERROR entry with the original message text for a bogus name, looped over every `pt.Variants` tenant version. |
| Logging discipline | New skip path logs at DEBUG, not ERROR/silently | PASS | character_temporary_stat.go:643 uses `l.Debugf(...)`; pinned by `TestCTSServerOnlyStatsSkippedSilently` (character_temporary_stat_test.go:942-978), which fails the test if any entry is `<= logrus.ErrorLevel` or not exactly `logrus.DebugLevel`. |
| Test coverage / no test theater | New behavior is table/version-driven, not a single hardcoded assertion | PASS | All four new tests (`TestCTSServerOnlyStatsSkippedSilently`, `TestCTSAddStatUnknownNameStillErrors`, `TestCTSMixedBuffServerOnlyByteInvariance`, `TestCTSPureServerOnlyBuffEncodesAsEmpty`) iterate `pt.Variants` (character_temporary_stat_test.go:943, 986, 1033, 1066), covering every supported tenant version including the legacy pre-v61 8-byte-mask class (`legacyGmsMask`, same file line 686 of the source file) per the design's FR-7/FR-7.1 requirement, not just GMS v83. |

## Non-Applicable Checklists

- **File Responsibilities Checklist**: Not applicable — `libs/atlas-packet/model` is a shared codec library, not a domain/support service package; it has no `Processor`, `RestModel`, `requests.go`, or `entity.go` responsibilities to misplace, and none were introduced by this diff.
- **Sub-Domain Checklist (SUB-*)**: Not applicable — no `resource.go`/action-event package involved.
- **External HTTP Client Checklist (EXT-*)**: Not applicable — no `requests.GetRequest`/`PostRequest` calls in the diff.
- **Service Scaffolding Checklist**: Not applicable — no new service or channel writer/handler registered.
- **Security Review (SEC-*)**: Not applicable — not an auth-related service.

## Scope-Creep Check

The diff touches exactly the two files declared in scope. No changes to
`buildCharacterTemporaryStatRegistry`, `MonsterTemporaryStat`, `AddStat`'s
signature, atlas-channel, or any exported API surface — confirmed by
`git diff 11261fb22..e4d76d1bf --stat` (2 files changed, both under
`libs/atlas-packet/model/`) and by line-level inspection of the single hunk
in the non-test file. No scope creep found.

## Summary

### Blocking (must fix)
- None found.

### Non-Blocking (should fix)
- None found.

# Plan Audit — task-110-interaction-packet-fixtures

**Plan Path:** docs/tasks/task-110-interaction-packet-fixtures/plan.md
**Audit Date:** 2026-06-24
**Branch:** task-110-interaction-packet-fixtures
**Base Branch:** main (merge-base 5d9c42ff3)

## Plan Adherence Review

### Executive Summary

All 12 target `interaction`-family serverbound cells are `verified` in
`docs/packets/audits/status.json`. Every promoted cell carries the three coupled
artifacts (a `packet-audit:verify` marker on a passing byte-fixture test, a pinned
evidence YAML with a `verifies:` line, and a Verdict-0 audit report). The export
splices/curation are confined to exactly the intended function keys with **no
unrelated drift** (zero function keys removed). The `libs/atlas-packet` module is
clean (`go build`, `go vet`, `go test -race` all pass) and no `go.mod` was touched.
The approved campaign-wide deviation ("live-IDA verify + curate" instead of plan's
"Class A report-gen only" for Invite) is soundly executed and matches the verified
v95 reference. **Overall: PASS.**

### Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 0 | Baseline matrix/build | DONE | Read-only; no commit, as specified. |
| A1–A3 | Invite v83/v87/jms verified | DONE | markers `operation_invite_test.go:11-13`; evidence `docs/packets/evidence/{gms_v83,gms_v87,jms_v185}/…Invite.yaml` (verifies present); reports Verdict 0; commits `d389db10e` (curate), `cc86f3b2d` (verify). |
| A4 | Invite full-row ✅ gate | DONE | status.json row = all five `verified`. |
| B1 | Splice OnTieRequest into v84 export | DONE | key added `gms_v84.json`; commit `789a56840`; body single `Decode1` response. |
| B2 | TieAnswer v84 verified | DONE | marker `operation_memory_game_tie_answer_test.go:13` (ida=0x664034); evidence + Verdict-0 report; commit `8a3d64aa4`. |
| C0 | Hex-pin merchant put/remove fixtures | DONE | `TestOperationMerchantPutItemBytes` / `…RemoveItemBytes`; commit `57f826135`; both PASS. |
| C1 | Merchant arms v83 | DONE | export splice `dcbc33916`; verify `772ac28dc`; markers ida=0x6fd96c/0x6fdcdf; Verdict 0. |
| C2 | Merchant arms v87 | DONE | export splice `95d235f6e`; verify `4cfd03076`; markers ida=0x740ee6/0x741271; Verdict 0. |
| C3 | Merchant arms jms | DONE | export splice `4c8cd1f0b`; verify `34e138b85`; markers ida=0x762a9e/0x762e26; sub-ops 0x1E/0x23 (jms-specific); Verdict 0. |
| C4 | Merchant arms v84 | DONE | export splice `05ab89877`; verify `8ed4f4fc0`; markers ida=0x719c8a/0x719ffd; Verdict 0. |
| C5 | Merchant full-row ✅ gate | DONE | both ops = all five `verified`. |
| D | Final verification gate + PRD tick | DONE | module clean; matrix --check exit 0 no interaction lines; PRD §10 ticked; commit `c693e5942`. |

**Completion Rate:** 12/12 cells (100%)
**Skipped without approval:** 0
**Partial implementations:** 0

### Approved Deviation (verified sound)

The plan classified Invite v83/v87/jms as "Class A report-gen only" with reports
required to be Verdict 0. The user instead chose the campaign-wide bar
"live-IDA verify + curate." Independent verification confirms this was executed
correctly:

- `CField::SendInviteTradingRoomMsg` body in `gms_v83/gms_v87/gms_jms_185.json` was
  curated from the raw multi-call harvest (multiple `Decode1`/`Decode4` guard-split
  arms + a `COutPacket::Init` Delegate artifact) down to the single
  `Decode4 (targetCharacterId)` body — matching the verified v95 reference and the
  Atlas `operation_invite.go` codec. Reports are Verdict 0.
- The TieAnswer v84 and 8 merchant arms were spliced as curated body-only entries
  (dispatcher sub-op byte excluded, COutPacket-delegate stripped), all Verdict 0.
- v84's Invite entry was **not** disturbed (already verified) — correct scoping.

### Independent Verification Results

1. **All 12 cells verified** — `status.json` shows all four packets `verified` on
   gms_v83/gms_v84/gms_v87/gms_v95/jms_v185. (0 incomplete interaction cells.)
2. **Artifact triples present** — 20 markers across the 4 test files (incl. the
   12 new), 12 new evidence YAMLs each with a `verifies:` line pointing at an
   existing passing test (merchant → `…Bytes`, invite/tie → `…RoundTrip`), 24 new
   audit report files all Verdict 0.
3. **Checks** — `matrix --check` exit 0, zero output; `fnamedoc … --check` exit 0
   (with required CSV/template flags); `operations --check` exit 0 with 1
   pre-existing non-interaction `jms_v185 NoteOperation` note. No interaction lines
   in any check.
4. **Module clean** — `libs/atlas-packet`: `go build ./...` exit 0, `go vet ./...`
   exit 0, `go test -race ./...` exit 0 (67 packages ok, no failures/races).
5. **Export hygiene** — diff confined to `gms_v83/gms_v84/gms_v87/gms_jms_185.json`.
   Per-export key delta: **0 keys removed**; added = only the two `#Merchant` arms
   (+ `OnTieRequest` for v84); only existing key whose body changed =
   `CField::SendInviteTradingRoomMsg` (the documented curation) on v83/v87/jms. No
   ~150-key re-export drift. Spliced merchant/tie bodies match the codecs exactly;
   no COutPacket-delegate artifacts remain.
6. **PRD §10** — all four acceptance boxes ticked with accurate evidence; no
   `go.mod` touched so no `docker buildx bake` required (correctly noted).

### Minor Observations (non-blocking)

- The Invite phase was landed as 2 commits (`d389db10e` curate + `cc86f3b2d`
  verify-all-three) rather than the plan's 3 per-version verify commits. This is a
  direct consequence of the approved deviation (curation is a single cross-version
  edit) and does not affect artifact completeness. Not a gap.
- The PRD §10 box for `matrix --check (and fname-doc/operations --check) exit 0`
  is accurate, but note `fnamedoc`/`operations` are subcommands that require the
  global `-csv-*`/`-template` flags to be passed *before* the subcommand; invoked
  bare they error (exit 3). With correct invocation all exit 0 as claimed.

## Overall Assessment

- **Plan Adherence:** FULL (goal met; deviation approved + soundly executed)
- **Recommendation:** READY_TO_MERGE

## Action Items

None required. The campaign goal (12 interaction cells → verified, each as the
coupled artifact triple, matrix regenerated, module clean) is fully met.

---

# Backend Guidelines Review

- **Reviewer:** backend-guidelines-reviewer
- **Date:** 2026-06-24
- **Scope:** 4 test-only files in `libs/atlas-packet/interaction/serverbound/`
- **Overall:** PASS

## Objective Gate (libs/atlas-packet)

| Gate | Result | Evidence |
|------|--------|----------|
| `go build ./...` | PASS | clean, no output |
| `go vet ./...` | PASS | clean, no output |
| `go test -race ./interaction/...` | PASS | `ok` for interaction, interaction/clientbound, interaction/serverbound |

## Checklist Results

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| House pattern match | New hex-pin tests mirror precedent `TestOperationMerchantBuyBytes` | PASS | `operation_merchant_put_item_test.go:47-57` and `operation_merchant_remove_item_test.go:34-44` use identical shape to `operation_merchant_buy_test.go:40-50`: `testlog.NewNullLogger()`, `pt.CreateContext("GMS", 83, 1)`, `hex.EncodeToString(input.Encode(l, ctx)(nil))`, struct literal field init |
| Hex pin accuracy (put_item) | Pinned bytes match codec encode order | PASS | `operation_merchant_put_item.go:36-41` writes byte/Int16/Short/Short/Int. inventoryType=2→`02`, slot=7→`0700`, quantity=15→`0f00`, set=4→`0400`, price=2000(0x7d0)→`d0070000` = `0207000f000400d0070000`, matches pin `:53` |
| Hex pin accuracy (remove_item) | Pinned bytes match codec encode order | PASS | `operation_merchant_remove_item.go:28` writes Short(index). index=42(0x2a)→`2a00`, matches pin `:40` |
| Test helper pattern | No `*_testhelpers.go`, no test-only constructors | PASS | `find . -name '*_testhelpers.go'` empty; structs built with literal field init at `put_item_test.go:50`, `remove_item_test.go:37` |
| Import grouping / dead imports | stdlib then third-party, no unused | PASS | `put_item_test.go:3-9` and `remove_item_test.go:3-8`: `encoding/hex`+`testing` grouped above `pt` and `testlog`; both new imports used (`hex.EncodeToString`, `testlog.NewNullLogger`). `go vet` clean confirms no dead imports |
| Marker comment format | `// packet-audit:verify packet=… version=… ida=0x…` | PASS | All 12 added markers conform; grep for non-conforming markers in the package returned empty. e.g. `invite_test.go:9-13`, `memory_game_tie_answer_test.go:9-13`, `put_item_test.go:11,43-46`, `remove_item_test.go:11,30-33` |

## DOM/SUB Production Checks

N/A — this change adds no production Go. No `model.go`, `processor.go`,
`resource.go`, `administrator.go`, `provider.go`, GORM entities, JSON:API
models, or Kafka emit paths are touched. `go.mod` is unchanged (DOM-22/DOM-23
deploy checks N/A). The package is a pure packet codec library with no domain
layer, so DOM-01..21, SUB-*, EXT-*, SCAFFOLD-*, and SEC-* do not apply.

## Summary

### Blocking (must fix)
None.

### Non-Blocking (should fix)
None.

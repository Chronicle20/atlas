# Backend Audit — task-092-mob-packet-family

- **Branch:** task-092-mob-packet-family
- **Scope:** `git diff main` Go changes only — `libs/atlas-packet/**` (monster/, monster/carnival/, character/serverbound, character/clientbound + tests) and `services/atlas-channel/atlas.com/channel/` (socket/writer/, socket/handler/, main.go). Doc/registry/JSON/template changes out of scope for DOM checks.
- **Guidelines Source:** backend-dev-guidelines skill + `docs/packets/IMPLEMENTING_A_PACKET.md`
- **Date:** 2026-06-14
- **Build:** PASS (both modules)
- **Tests:** PASS (all packages `ok`)
- **Vet:** PASS (both modules)
- **Overall:** PASS

## Why most DOM checks are N/A

This is a packet-codec batch, not a DDD domain service. The changed packages contain
no `model.go` / processor / administrator / REST / JSON:API / provider / GORM / DB
layer, and packet codecs are not expected to. DOM-01..DOM-20 (builder, ToEntity,
Make, Transform/TransformSlice, processor/administrator, RegisterInputHandler,
JSON:API interfaces, providers, error→HTTP status, etc.) are therefore **N/A by
design** and are not manufactured into FAILs. The applicable checks are the
build/vet/test gates, immutable-model codec shape, Encode/Decode symmetry,
FieldLogger usage, wiring completeness, DOM-21 (atlas-constants reuse), and the
round-trip/golden-byte test pattern.

## Build & Test Results

| Gate | libs/atlas-packet | atlas-channel/.../channel |
|------|-------------------|----------------------------|
| `go build ./...` | PASS (clean) | PASS (clean) |
| `go vet ./...` | PASS (clean) | PASS (clean) |
| `go test ./... -count=1` | PASS (all `ok`, incl. monster/{clientbound,serverbound}, carnival/{clientbound,serverbound}, character/serverbound) | PASS (all `ok`, incl. socket/handler, socket/writer) |

## Applicable Checklist Results

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| G-1..6 | build/vet/test both modules | PASS | see table above |
| SHAPE-1 | No exported mutable struct fields | PASS | `git diff main` grep for `+\t[A-Z]\w+ (uint|int|byte|bool|string|float)` → none |
| SHAPE-2 | No setter methods | PASS | grep `func (..) Set[A-Z]` over new non-test .go → none |
| SHAPE-3 | Private fields + constructor + getters only | PASS | mob_escort_full_path.go:22-36,56-74; mob_time_bomb_end.go:42-61; cover.go NewCover added |
| SHAPE-4 | `Operation()` + `String()` on every codec | PASS | sweep over all new codec .go → none missing |
| FLOG-1 | Encode/Decode take `logrus.FieldLogger` | PASS | grep `+.*\*logrus.Logger` across diff → none |
| FLOG-2 | Handlers take `logrus.FieldLogger` | PASS | handler files use `logrus.FieldLogger` |
| SYM-1 | mob_escort_full_path Encode≡Decode | PASS | clientbound/mob_escort_full_path.go:84-100 vs 106-124 — mode, count, per-wp{x,y,kind,[kind==2:extra]}, tail, hasArrive,[arriveDelay], hasReset; conditionals mirrored |
| SYM-2 | mob_time_bomb_end Encode≡Decode | PASS | serverbound/mob_time_bomb_end.go:66-72 vs 79-85 — mobCrc,[boss:bossX,bossY],localX,localY; `boss` carried off-wire on the model, honored both sides (documented :25-28) |
| SYM-3..6 | mob_skill_delay / monster_carnival_start / mob_attacked_by_mob / catch_monster | PASS | sub-agent + spot reads confirm field-for-field mirror incl. version/data-conditional branches |
| WIRE-1 | Every writer Body const registered in main.go | PASS | 24 writer Body files ↔ 24 `…Writer,` constants added to produceWriters |
| WIRE-2 | Every handler registered in main.go (no silent-drop) | PASS | 12 `*HandleFunc` defined ↔ 12 `handlerMap[...] = ...` registrations; exact 1:1 |
| DOM-21 | No duplication of atlas-constants types | PASS | zero new atlas-constants imports needed; id fields are packet-local wire widths (uint32/int32/byte), matching the pre-existing `Control` codec idiom; DOM-21 explicitly excludes packet-local field widths |
| TEST-1 | Each new codec has a sibling `_test.go` | PASS | no-test scan → zero gaps |
| TEST-2 | Round-trip across `pt.Variants` + golden bytes + verify markers | PASS | 37 `pt.RoundTrip` uses; 173 `// packet-audit:verify` markers added |
| CONV-1 | Documented seams honored (not flagged) | PASS | uncalled clientbound writer Body helpers = D2 seam; decode-and-log serverbound handlers = intentional; empty-payload codecs (mob_crc_key_changed_reply) = correct — all per IMPLEMENTING_A_PACKET.md |

## Summary

### Blocking (must fix)
- None.

### Non-Blocking (style nit only)
- Several codecs use `w.WriteInt(...)` / `r.ReadUint32()` for fields documented as
  uint32 while sibling codecs use explicit `WriteInt32`. Consistent with the lib's
  existing helper set and round-trips correctly; cosmetic, not a guideline violation.

## Score
- **PASS: 26**
- **FAIL: 0**
- **N/A (DOM-01..DOM-20 domain-layer checks, not applicable to packet codecs):** documented above, not counted as defects.

## Overall: PASS

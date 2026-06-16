# Backend Audit — task-100 (WHISPER + SPOUSE_CHAT serverbound ops)

- **Scope:** Go changes on branch, diff 47452b3..46eaf6e
- **Guidelines Source:** backend-dev-guidelines skill
- **Date:** 2026-06-16
- **Build:** PASS (libs/atlas-packet, atlas-channel, tools/packet-audit)
- **Vet:** PASS (changed packages)
- **Tests:** PASS (chat/serverbound, field/serverbound, socket/handler)
- **Overall:** PASS

## Nature of the change

This is a packet-codec + channel socket/message-layer change, not a DDD domain
package. No package under the diff has `model.go` / `entity.go` / `resource.go`,
so the DOM-01..DOM-20 mechanical checks (builder/ToEntity/Transform/JSON:API/
administrator/provider) are not applicable. The relevant guidelines are the
functional/immutable patterns, the processor Interface+Impl pattern, the Kafka
producer pattern, and DOM-21 (no reinvented atlas-constants).

## Findings

### Critical
- None.

### Important
- None.

### Minor
- **No handler-level test for the new spouse-chat handler.**
  `services/atlas-channel/atlas.com/channel/socket/handler/character_spouse_chat.go:14`
  has no `*_test.go`. This is consistent with the sibling it mirrors
  (`character_chat_whisper.go` also has no handler test), so it is not a
  regression, but the emit path (`Processor.SpouseChat` →
  `SpouseChatCommandProvider`) is exercised only indirectly. The packet codec
  itself (`spouse_chat.go`) has full golden + round-trip coverage. Low severity.

## Check results (applicable items)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| Build gate | go build all changed modules | PASS | libs/atlas-packet, atlas-channel, tools/packet-audit all build clean |
| Vet gate | go vet changed packages | PASS | chat/serverbound, field/serverbound clean |
| Test gate | go test changed packages | PASS | chat/serverbound 0.004s, field/serverbound 0.008s, socket/handler 0.007s |
| Immutability | private fields + getters | PASS | `field/serverbound/spouse_chat.go:33-35` CoupleMessage has private `spouseName`/`message`, getters `SpouseName()`/`Message()` at :30,:32 |
| Processor Interface+Impl | SpouseChat on both | PASS | `message/processor.go:21` interface method; `message/processor.go:84` ProcessorImpl method |
| Kafka producer pattern | curried ProviderImpl + SingleMessageProvider | PASS | `message/producer.go:57` SpouseChatCommandProvider returns `model.Provider[[]kafka.Message]` via `producer.SingleMessageProvider`; emit via `producer.ProviderImpl(p.l)(p.ctx)(...)` at processor.go:85 — byte-identical to WhisperChatCommandProvider |
| Topic reuse (DOM-23) | no new topic constant | PASS | reuses `message2.EnvCommandTopicChat` (processor.go:85); no new `COMMAND_TOPIC_*`/`EVENT_TOPIC_*` added |
| DOM-21 reinvented constants | no shadow of atlas-constants | PASS | new types are `SpouseChatBody{SpouseName string}` (kafka.go:69) and `ChatTypeSpouse = "SPOUSE"` (kafka.go:20) — chat-DTO/enum, not item/inventory/world/job id reinventions |
| Version gate correctness | whisper updateTime gate matches sibling | PASS | `chat/serverbound/whisper.go:30` `whisperHasUpdateTime` = `(GMS && Major>=87) \|\| JMS`, identical to `interaction/serverbound/operation_chat.go:33` `chatHasUpdateTime`; replaces prior `>=95` bug |
| Error handling | handler logs, no swallow | PASS | `character_spouse_chat.go:20-22` checks err and logs `WithError`; processor returns the producer error |
| No silent stubs | no TODO/501/panic in new code | PASS | grep of all new files: no stub markers |
| packet-audit tool | candidatesFromFName cases added | PASS | `tools/packet-audit/cmd/run.go` adds `CField::SendLocationWhisper` + `CUIStatusBar::SendCoupleMessage` cases; tool builds |
| Test style | table-driven round-trip + golden fixtures | PASS | `spouse_chat_test.go:129` round-trip over `pt.Variants`; per-version golden tests with `packet-audit:verify` markers |

---

# Plan-Adherence Verification — task-100 (independent audit)

- **Auditor:** plan-adherence reviewer (read-only)
- **Date:** 2026-06-16
- **Diff range:** 47452b3..46eaf6e (2 commits)
- **Verdict:** PASS — all stated requirements genuinely implemented with byte-level / command evidence.

## Verification gates (exit codes confirmed)

| Gate | Result |
|------|--------|
| `go run ./tools/packet-audit matrix --check` | EXIT 0 |
| `go run ./tools/packet-audit fname-doc --check` | EXIT 0 (213 structs w/o report carry no fname) |
| `go run ./tools/packet-audit operations --check` | EXIT 0 (2 pre-existing jms absent-writer notes, unrelated) |
| `libs/atlas-packet` build / vet / test | 0 / 0 / 0 (all pkgs ok) |
| `tools/packet-audit` build / vet / test | 0 / 0 / 0 |
| `services/atlas-channel` build / vet / test | 0 / 0 / 0 (0 FAIL) |

## Requirement-by-requirement

### R1a — WHISPER serverbound → verified: PASS
- Gate fix: `whisper.go:28-30` `whisperHasUpdateTime` = `(GMS && Major>=87) || JMS`; replaces prior `>=95` bug at both Encode (`whisper.go:77`) and Decode (`whisper.go:92`).
- Fixtures prove the gate: v83/v84 goldens have NO updateTime; v87/v95/jms insert 4-byte `0x64 00 00 00` at index 1 (`whisper_test.go`).
- Routed in all 5 templates (`CharacterChatWhisperHandle`): gms_83/84 pre-existing, gms_87 0x7E, gms_95 0x8D, jms 0x7A added — opcodes match registry serverbound rows.
- candidatesFromFName linkage: `run.go:1248` adds `case "CField::SendLocationWhisper"` (the registry PRIMARY fname) → `chat/serverbound/Whisper`.
- Per-packet matrix cell `chat/serverbound/ChatWhisper` = verified across v83/v84/v87/v95/jms (status.json).

### R1b — SPOUSE_CHAT serverbound NEW codec → verified: PASS
- New codec `field/serverbound/spouse_chat.go` `CoupleMessage`: `EncodeStr(spouseName)+EncodeStr(message)`, no mode byte, no updateTime — matches fixtures (two strings only).
- Channel handler `character_spouse_chat.go` decodes + calls `message.Processor.SpouseChat`; Kafka command added (`ChatTypeSpouse`, `SpouseChatBody`, `SpouseChatCommandProvider`, processor method); wired in `main.go:855` via `fieldsb` alias.
- Routed in 4 GMS templates (gms_83 0x79, gms_84 0x7B, gms_87 0x7F, gms_95 0x8E) — match registry; jms correctly NOT routed (version-absent).
- candidatesFromFName: `run.go:1257` `case "CUIStatusBar::SendCoupleMessage"` → `field/serverbound/CoupleMessage`.
- Per-packet cell `field/serverbound/FieldCoupleMessage` = verified v83/v84/v87/v95, n-a jms (status.json).

### R2 — v84 serverbound reshift completion: PASS
- Registry gms_v84.yaml IDA-verified +2 values, all confirmed: MULTI_CHAT 121/0x79, SPOUSE_CHAT 123/0x7B, PLAYER_INTERACTION 125/0x7D, DENY_PARTY_REQUEST 127/0x7F — each `provenance: ida-discovered` with `ida.address` + send-site note.
- No duplicate serverbound opcodes in gms_v84 (python scan). (Pre-existing login/AP dups in v87/v95/jms are unrelated and untouched.)

### R3 — Registry hygiene: PASS
- `CField::OnWhisper` removed from WHISPER **serverbound** fname_alts in all 5 versions (now only `CField::SendChatMsgWhisper`). The remaining `OnWhisper` occurrences are the legitimate **clientbound** WHISPER `fname` rows (correct — OnWhisper is the receive decode).

## Expected-❌ confirmation (not a gap)
- WHISPER serverbound OP-ROW = `incomplete` ("no audit report") across all versions — consistent with the mode-prefix flat-diff limitation. The per-packet `chat/serverbound/ChatWhisper` row carries the ✅. This matches the stated expectation; not a missed requirement.

## Artifacts present
- Byte fixtures carry `packet-audit:verify` markers: ChatWhisper ×5 (incl jms), FieldCoupleMessage ×4 (no jms).
- Evidence YAMLs (`docs/packets/evidence/<ver>/...`) with IDA fn/address/decompile_sha256 + `verifies:` test-fn refs.
- Audit reports (`docs/packets/audits/<ver>/{ChatWhisper,FieldCoupleMessage}.{md,json}`) with ✅ verdicts.

## No silent gaps found
Every cell claimed ✅ is backed by a marker fixture + evidence + audit report; no stubs/TODOs in new code.

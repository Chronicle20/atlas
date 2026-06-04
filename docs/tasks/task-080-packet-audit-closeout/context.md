# Packet-Audit Closeout — Context

Companion to `plan.md`. Key files, decisions, and dependencies an executor needs. All paths relative to the worktree root `<repo-root>/.worktrees/task-080-packet-audit-closeout`.

## Modules & how to verify

| Module | go.mod name | Dir |
|---|---|---|
| atlas-packet (encoders/decoders) | `github.com/Chronicle20/atlas/libs/atlas-packet` | `libs/atlas-packet` |
| packet-audit (analyzer tool) | `github.com/Chronicle20/atlas/tools/packet-audit` | `tools/packet-audit` |
| atlas-maps | `atlas-maps` | `services/atlas-maps/atlas.com/maps` |
| atlas-channel | `atlas-channel` | `services/atlas-channel/atlas.com/channel` |
| atlas-cashshop | `atlas-cashshop` | `services/atlas-cashshop/atlas.com/cashshop` |
| atlas-configurations (templates) | `atlas-configurations` | `services/atlas-configurations` |

Per-module gates: `go test -race ./...`, `go vet ./...`, `go build ./...`. Phase F adds `docker buildx bake atlas-<svc>` (from worktree root) for every service whose `go.mod` changed, `tools/redis-key-guard.sh`, and the nesting `awk` cap check. CLAUDE.md is authoritative.

## Test harness (the oracle)

- `libs/atlas-packet/test/context.go`: `pt.CreateContext(region string, major, minor uint16) context.Context`; `pt.Variants` = `[GMS v28, GMS v83, GMS v87, GMS v95, JMS v185]` (`TenantVariant{Name, Region, MajorVersion, MinorVersion}`).
- `libs/atlas-packet/test/roundtrip.go`: `pt.RoundTrip(t, ctx, encode, decode, options)` — encodes, decodes, asserts zero leftover bytes.
- Exact-bytes idiom (no round-trip can catch a wrong-but-symmetric bug): `got := in.Encode(l, pt.CreateContext("GMS",95,1))(opts)` then `len(got)` + `bytes.Equal(got[a:b], []byte{...})`. Exemplars: `stat/clientbound/changed_test.go::TestStatChangedV95WireWidths`, `ui/clientbound/lock_test.go::TestUiLockWireShape`.
- Logger for encode-only tests: `l, _ := testlog.NewNullLogger()` (`github.com/sirupsen/logrus/hooks/test`).

## Cross-cutting idioms (design §3)

- **Version gate:** compute `t := tenant.MustFromContext(ctx)` in the outer fn, then a local bool (`v95Plus := t.Region()=="GMS" && t.MajorVersion()>=95`); apply symmetrically in Encode/Decode. Accessors: `t.Region() string`, `t.MajorVersion() uint16`, `t.MinorVersion() uint16`. Exemplar: `stat/clientbound/changed.go`, `model/avatar.go`.
- **Region-dispatched body** (the >2-version answer; respects the 2-nested-guard cap): dispatch to `m.encodeJMS(w)` / `m.encodeGMS(t, w)` at the top of the closure; never a 3rd nested `if`. Used by B1.5 (EffectWeather) and B5.1 (cash bodies). The repo nesting `awk` must stay clean.
- **Builder/immutable models:** private fields + getters + Builder; no `*_testhelpers.go` (use the model's own constructor/Builder in tests).

## Bucket → file map (verified current state)

### B1.1 AffectedArea (multi-service — do first)
- Packet (rewrite): `libs/atlas-packet/field/clientbound/affected_area_created.go`. **Current** shape: `mistId, ownerId, originX/Y, ltX/Y, rbX/Y, duration, skillLevel` → wire `int(mistKey), int(owner), 6×int16, int32(dur), int(skillLevel)`. `mistKey(id uuid.UUID) uint32 = id.ID()`. **Target:** `dwId, nType, dwOwnerId, nSkillID, nSLV(byte), phase(int16), rcArea(16-byte abs RECT), tEnd`, + `tStart` gated GMS≥95.
- atlas-maps model `mist/model.go`: **already** has `SourceSkillId()`, `SourceSkillLevel()`, origin, lt/rb offsets, `Contains()`. Builder: `NewBuilder(id, f)` + `SetOwner/SetSource/SetOrigin/SetBounds/SetDisease/SetDuration/SetTickInterval`. **Add** `mistType`/`Type()`/`SetType()`.
- atlas-maps event `kafka/message/mist/kafka.go` `CreatedBody`: **drops** skill id/level, **no** type. Add `SourceSkillId`, `SourceSkillLevel`, `Type`.
- atlas-maps producer `mist/producer.go` `createdEventProvider(t, m)`: copies model→event; add the three fields.
- atlas-channel consumer `kafka/consumer/mist/consumer.go` `handleMistCreated`: **hardcodes `0` for skillLevel** in `NewAffectedAreaCreated(...)`. Has a swappable `affectedAreaCreatedBroadcaster` package var for testing (no REST mock needed). Channel-side `kafka/message/mist/kafka.go` `CreatedBody` must mirror the new fields.

### B1.2 chat Multi
- `libs/atlas-packet/chat/serverbound/multi.go`: `Multi{chatType, recipients[], chatText}`. Decode reads byte chatType first. Add leading `updateTime uint32` gated GMS>83. Handler `socket/handler/character_chat_multi.go` decodes it (picks up new field automatically). Note: clientbound `MultiChat` (`NewMultiChat`) is a *separate* packet — not in scope.

### B1.3/B1.4 quest serverbound
- `libs/atlas-packet/quest/serverbound/`: base `Action{action byte, questId uint16}` (Decode reads action+questId). `ActionStart`/`ActionComplete` are constructed with `NewActionStart(autoStart bool)` / `NewActionComplete(autoStart)` — autoStart gates the x,y read. `ActionComplete` has trailing `selection int32`. `ActionRestoreLostItem{unk1, itemId}` (redesign to `itemIds []uint32` count-prefixed).
- Handler `socket/handler/quest_action.go`: action constants `QuestActionRestoreLostItem=0, Start=1, Complete=2, Forfeit=3, ScriptStart=4, ScriptEnd=5`. Reads base `Action`, fetches quest via `quest2.NewProcessor(...).GetById`, then decodes the sub-packet with `q.AutoStart()`. `RestoreItem(field, charId, questId, itemId)` is the processor call to adapt for a slice.

### B1.5 EffectWeather
- `libs/atlas-packet/field/clientbound/effect_weather.go`: `EffectWeather{active, itemId, message}`. **Current** (GMS) Encode: `WriteBool(!active); WriteInt(itemId); if active WriteAsciiString(message)`. Add JMS branch (no leading bool; itemId first; optional extra when type==51; message when itemId!=0) via region dispatch.

### B2.1 NPC continue-conversation
- Handler `socket/handler/npc_continue_conversation.go`: **wrongly** treats `lastMessageType==2` as text. Structs in `libs/atlas-packet/npc/serverbound/continue_conversation*.go` are already correct (`ContinueConversation{lastMessageType, action}`, `ContinueConversationText{text}`, `ContinueConversationSelection{selection, wide}` — selection auto-detects width by `r.Available()>=4`). Fix routing: 3/14→text, 5/8/9→selection, 0/1/2/13→none.

### B2.2/B2.3 merchant
- Serverbound `libs/atlas-packet/merchant/serverbound/operation.go`: **bare constant only** (`HiredMerchantOperationHandle`). No struct.
- Handler `socket/handler/hired_merchant_operation.go`: **TODO stub** (returns immediately).
- Clientbound `libs/atlas-packet/merchant/clientbound/operation.go`: has OpenShop/ErrorSimple/ShopSearch/ShopRename/RemoteShopWarp/ConfirmManage/FreeFormNotice (shape reference). Add mode-8 `EntrustedShopUnknownChannel{shopId, channelId}` + mode-11 constant.

### B5 cash-shop (JMS)
- Serverbound `libs/atlas-packet/cash/serverbound/shop_operation_*.go`. `shop_operation_buy.go` already branches `GMS>=87` inline (2 levels) — refactor to `encodeGMS`/`encodeJMS` before adding JMS to avoid a 3rd guard. gift/couple/friendship/rebate branch `GMS>=95`. `Operation()` returns `"CashShopOperationHandle"`.
- Channel path (region-agnostic, already wired): `socket/handler/cash_shop_operation.go::CashShopOperationHandleFunc` → `cashshop.NewProcessor(...).RequestPurchase(charId, serialNumber, isPoints, currency, zero)` → `cashshop/producer.go::RequestPurchaseCommandProvider(charId, serialNumber, currency)` (drops isPoints+zero) → Kafka `REQUEST_PURCHASE` → atlas-cashshop `kafka/consumer/cashshop/consumer.go::handleCommandRequestPurchase` → `cashshop/processor.go::PurchaseAndEmit(charId, currency, serialNumber)` → wallet `w.Balance(currency)` / `w.Purchase(currency, price)`. `RequestPurchaseCommandBody{Currency uint32, SerialNumber uint32}` — extend only if a JMS field can't be carried.
- Template `services/atlas-configurations/seed-data/templates/template_jms_185_1.json`: has writer `CashShopOperation` (op 0x164) but **no** `CashShopOperationHandle` serverbound handler entry. Add it + op-byte map (buy=3, gift=0x2E, couple=0x1E, friendship=0x24, rebate=0x1B). Also remap interaction PersonalStore `BuyItem` 0x14/0x1F, `DeliverBlackList` 0x1B.

### B6 login
- Exports: `docs/packets/ida-exports/{gms_v83,gms_v87,gms_v95,gms_jms_185}.json`. Writers/handlers under `libs/atlas-packet/login/`. `_pending.md` lists the addressed + bare-handler backlog.

## packet-audit analyzer (§4.7)

- Layout: `cmd/{root,run,export}.go` + `internal/{atlaspacket,diff,idasrc,report,template,csv}/`.
- Invocation (Task A0): `go run . -csv-clientbound <cb.csv> -csv-serverbound <sb.csv> -atlas-packet libs/atlas-packet -ida-source <ver.json|mcp> -output <dir>`.
- Four false-positive classes & loci:
  - Width equivalence — `internal/diff/diff.go` `primWidth`/`idaWidth` + the `case primWidth(...) != idaWidth(...)` arm. (A1)
  - Qualified names — `cmd/run.go` `locateAtlasFile(root,name,pkg,dir)` **already pkg-aware**; gap is `candidatesFromFName` map completeness (`Spawn`/`Destroy`/`Movement`/`ChannelChange`). (A2)
  - Sub-struct descent — `internal/atlaspacket/registry.go` Pass-2 only registers types with Encode/Write/EncodeEntry/EncodeBytes/EncodeForeign; **no fallback** for types without one. `internal/diff/diff.go` `flattenWithRegistryGuarded` passes unknown `KindRecurse` through → 🔍. Add Pass-3 synthesis + `Opaque` flag. (A3)
  - Early-return — `internal/atlaspacket/analyzer.go` already models it (`blockTerminatesWithReturn`, suffix-taint, `TestEarlyReturnThenTaintsSuffix`); verify coverage, fill only real gaps. (A4)
- Verdicts: `internal/diff/diff.go` `Verdict{Match ✅, Minor ⚠️, Blocker ❌, Deferred 🔍}.Symbol()`. SUMMARY.md is **auto-generated** by `cmd/run.go::writeSummary` (`| [Name](Name.md) | Verdict | AtlasFile |`) — never hand-edit SUMMARY.
- Tests: `.go.txt` fixtures under `internal/atlaspacket/testdata/`; `AnalyzeFile("testdata/x.go.txt","Type","Encode")` → `[]Call`. Diff tests build `[]atlaspacket.Call` + `idasrc.Fields` literals.

## docs / ledger (§4.8)

- `docs/packets/audits/{gms_v83,gms_v87,gms_v95,jms_v185}/` each: `SUMMARY.md` (auto-gen), `_pending.md`, per-packet `.md`/`.json`. Cross-task ledger: `docs/packets/audits/gms_v95/TOTAL.md`. Master deferral ledger: `docs/packets/ida-exports/_pending.md` (~1844 lines, 44 sections).
- Curation target: both `_pending.md` reduced to zero actionable items + an **accepted permanent exclusions** registry (IDA evidence + one-line justification each); TOTAL states "baseline complete — zero open actionable deferrals"; new `STARTING_A_NEW_VERSION_PASS.md` guide.

## Key decisions (from design)

1. **B1.1:** extend the existing `CreatedBody` event, do not invent a new one. Only `Type` is genuinely new (skill id/level already on model). RECT on the wire is **absolute** (origin+offset); model stores relative offsets.
2. **B5:** no new command/topic. Route JMS purchases through the existing `RequestPurchase`→wallet flow. Region-dispatch the bodies; extend `RequestPurchaseCommandBody` only if a JMS field genuinely can't fit.
3. **Analyzer (Q4):** fix the three tractable classes in-tool + generalize descent for self-describing types; register the genuinely-opaque residue. Bar = "named classes clean vs registry," not a perfect decompiler.
4. **No regressions** to closed items: storage `Show`, `MonsterControl`, SETFIELD/WarpToMap gates — untouched + green.

## Sequencing

A (analyzer) → B (B1.1 first, then B1.2–B1.5, B2) → C (B5) → D (B3/B4/B6 spikes) → E (docs) → F (verify + review). Docs last (must reflect final code + analyzer). Phase A de-noises every later re-run.

## Risks (design §8)

- B1.1 mist `Type` source unknown → confined Phase-B spike; contract change identical either way.
- JMS body needs an uncarried field → extend command body minimally, no new topic.
- Sub-struct descent scope creep → hard boundary (self-describing only; opaque → registry).
- IDA spike surfaces a large new bug → fix in-task with test+gate; if clearly out of closeout scope, register as a *new* task, never deferred back into `_pending.md`.
- `docker buildx bake` catches a missing `COPY` that `go build` misses → Phase F bakes every touched service.

# Megaphones (All Tiers) & Maple TV Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement the full player megaphone flow (Megaphone / Super / Item / Triple / Avatar) and Maple TV end-to-end — serverbound decode, saga-driven item consumption, Kafka broadcast fan-out, the missing clientbound writers, a world-scoped TV/avatar broadcast queue in atlas-world, seed-template wiring, and byte-fixture verification per version.

**Architecture:** Two broadcast paths. Stateless tiers (basic/super/item/triple) ride a 2-step saga (`DestroyAsset` → `EmitMegaphone`) whose event every atlas-channel pod consumes and renders via `WorldMessage` mode writers. Serialized tiers (Maple TV, avatar megaphone) ride `DestroyAsset` → `EnqueueWorldBroadcast` into a Redis-CAS queue owned by an atlas-world coordinator (leader-gated 1 s sweep) that emits QUEUED/STARTED/ENDED status events; channel pods render SEND_TV / SET_AVATAR_MEGAPHONE / clear packets from those. See `design.md` for decisions D1–D11.

**Tech Stack:** Go workspace monorepo; `libs/atlas-packet` (+ `atlas-socket` reader/writer), `libs/atlas-saga`, `libs/atlas-redis` (`TenantRegistry` CAS), `libs/atlas-lock` (`LeaderElection`), Kafka via `libs/atlas-kafka`, JSON:API REST via `api2go`, seed templates in atlas-configurations, `tools/packet-audit` + ida-pro-mcp for verification.

## Global Constraints

- **Versions in scope:** gms_83, gms_84, gms_87, gms_95, jms_185. **gms_12 and gms_92 are excluded** (login-only templates — design D9). jms ships **no** `AVATAR_MEGAPHONE_RESULT` (op absent in jms; rejection is silent-return for jms).
- **No hard-coded protocol bytes:** WorldMessage mode bytes resolve from the tenant `operations` table via `WithResolvedCode`/`ResolveCode`; opcodes come from tenant templates. Zero `mode: 0x` literals (dispatcher-lint INV-2).
- **Version pitfalls:** never write `MajorVersion() > 83` — use `>= 87` for structure branches (v84 is v83-structured); GMS≥95 is `t.Region() == "GMS" && t.MajorVersion() >= 95` (the `updateTimeFirst` predicate).
- **Every new `socket.handlers` template entry carries a `validator`** (missing validator = silently dropped handler).
- **Multi-tenancy:** every Kafka message carries tenant headers (`consumer.TenantHeaderParser`); processors use `tenant.MustFromContext(ctx)`; all Redis state goes through `libs/atlas-redis` (redis-key-guard invariant); coordinator keys scoped tenant+world+family.
- **Broadcast consumers use `kafka.LastOffset`** (fire-and-forget, not replayable state).
- **No `// TODO`, stubs, or 501s in landed commits.** The existing TODO at `services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go:108` must be resolved (removed) by this work.
- **Test style:** project Builder pattern / plain constructors; no `*_testhelpers.go` files. Packet tests use `pt.Variants` round-trips plus byte fixtures with `// packet-audit:verify` markers.
- **Verification gates (every changed module/service):** `go test -race ./...`, `go vet ./...`, `go build ./...`, `docker buildx bake atlas-<svc>` (channel, world, saga-orchestrator, configurations), `tools/redis-key-guard.sh`, and `go run ./tools/packet-audit` `dispatcher-lint` / `matrix --check` / `fname-doc --check` / `operations --check` all exit 0.
- **Grounding:** wire shapes in Tasks 1–6 are Cosmic-derived or v83/v95-IDA-derived (design §1.2/§1.3); Tasks 18–20 IDA-verify them per version and are allowed to change the structs. Never claim a version verified without an IDB-backed fixture. v83 opcodes/read orders cited in design §1.2 were verified against the open v83 dump (`MapleStory_dump.exe`) and v95 (`GMS_v95.0_U_DEVM.exe`).
- **Commit prefix convention:** `feat(task-123): …` / `test(task-123): …` / `docs(task-123): …`, committed on branch `task-123-megaphones-maple-tv` inside this worktree.

**Working directory for all commands:** the task worktree root (`.worktrees/task-123-megaphones-maple-tv`). All paths below are worktree-relative.

## Key protocol facts (pin these — they are used across tasks)

Opcodes per version, column order **v83 / v84 / v87 / v95 / jms185** (from `docs/packets/audits/STATUS.md`, lines 88, 142–145, 446–448):

| Op | FName | v83 | v84 | v87 | v95 | jms |
|---|---|---|---|---|---|---|
| SERVERMESSAGE (WorldMessage, exists in templates) | CWvsContext::OnBroadcastMsg | 0x044 | 0x044 | 0x046 | 0x047 | 0x03E |
| AVATAR_MEGAPHONE_RESULT | CWvsContext::OnAvatarMegaphoneRes | 0x06E | 0x071 | 0x071 | 0x072 | — (absent) |
| SET_AVATAR_MEGAPHONE | CWvsContext::OnSetAvatarMegaphone | 0x06F | 0x072 | 0x072 | 0x073 | 0x05A |
| CLEAR_AVATAR_MEGAPHONE | CWvsContext::OnClearAvatarMegaphone | 0x070 | 0x073 | 0x073 | 0x074 | 0x05B |
| SEND_TV | CMapleTVMan::OnSetMessage | 0x155 | 0x15F | 0x16A | 0x195 | 0x17A |
| REMOVE_TV | CMapleTVMan::OnClearMessage | 0x156 | 0x160 | 0x16B | 0x196 | 0x17B |
| ENABLE_TV | CMapleTVMan::OnSendMessageResult | 0x157 | 0x161 | 0x16C | 0x197 | 0x17C |

USE_CASH_ITEM serverbound handler (`CharacterCashItemUseHandle`): already wired in gms_83/84 templates (0x4F); **missing** from v87/v95/jms templates — opcodes v87 `0x052`, v95 `0x055`, jms `0x047` (design §1.4).

Client read orders (IDA-verified v83+v95, design §1.2):

- **SEND_TV** (`CMapleTVMan::OnSetMessage`): `byte flag` (bit 2 = has receiver look; Cosmic writes 3 with partner / 1 without), `byte messageType` (0 normal / 1 star / 2 heart — wire value = `tvType <= 2 ? tvType : tvType - 3`), `AvatarLook sender`, `str senderName`, `str receiverName` (always present, empty when none), `str ×5` message lines, `int totalWaitTime` (seconds), then `AvatarLook receiver` iff `flag & 2`.
- **ENABLE_TV** (`CMapleTVMan::OnSendMessageResult`): `byte hasError`; if nonzero `byte code` (1 = non-GM sent GM message, 2 = "waiting line longer than an hour", 3 = wrong user name). `00` = silent success ack. Sender-feedback only — never broadcast.
- **REMOVE_TV** (`CMapleTVMan::OnClearMessage`): no body.
- **SET_AVATAR_MEGAPHONE** (`CWvsContext::OnSetAvatarMegaphone`): `int itemId`, `str name`, `str ×4` message lines, `int channel`, `byte whisper`, `AvatarLook sender`.
- **CLEAR_AVATAR_MEGAPHONE**: `byte` (guarded early-clear, idempotent; Cosmic sends `1`).
- **AVATAR_MEGAPHONE_RESULT** (v83 `0xa2a3bc`): `byte code` — 83 = "waiting line longer than 15 seconds", 84 = level-10 gate (out of scope), anything else = followed by `str` shown as notice dialog.
- **Avatar megaphone auto-clear is client-side at 10 000 ms**; server ENDED clear is an idempotent redundancy.

Serverbound sub-bodies (Cosmic `UseCashItemHandler.java:290-372`, itemType 507 switch on `(itemId / 1000) % 10`; per-version IDA verification in Tasks 19–20):

- case 1 basic megaphone: `str message`
- case 2 super megaphone: `str message`, `byte whisper`
- case 5 Maple TV (`tvType = itemId % 10`): `if tvType != 1 { if tvType >= 3 { if tvType == 3 { byte }; byte ear } else if tvType != 2 { byte }; if tvType != 4 { str receiverName } }`, then `str ×5` lines, then `int` (trailing updateTime on <95). tvType ≥ 3 = "Megassenger" — ALSO fires a super-megaphone broadcast with the concatenated lines and `ear` as whisper (Cosmic parity).
- case 6 item megaphone: `str message`, `byte whisper`, `byte hasItem`, and iff hasItem: `int invType`, `int slot`
- case 7 triple megaphone: `byte lines (1..3)`, `str ×lines`, `byte whisper`
- classification 539 avatar megaphone: `str ×4` lines, `byte whisper`

Business rules: TV durations 15 s base / 30 s (tvType 4) / 60 s (tvType 5) (Cosmic `MapleTVEffect.java:56-61`); TV wait cap 3600 s (client string "longer than an hour"); avatar megaphone duration 10 s, wait cap 15 s (IDA). No late-joiner replay (D7). Medal is always `""` for now, plumbed through (D10).

Kafka names (D2): `EVENT_TOPIC_MEGAPHONE`, `COMMAND_TOPIC_WORLD_BROADCAST`, `EVENT_TOPIC_WORLD_BROADCAST_STATUS`. Tier values `MEGAPHONE|SUPER|ITEM|TRIPLE`; Scope `CHANNEL|WORLD`; Family `TV|AVATAR`; status types `QUEUED|STARTED|ENDED`.

---

### Task 1: Serverbound sub-body structs (`libs/atlas-packet/cash/serverbound`)

**Files:**
- Create: `libs/atlas-packet/cash/serverbound/item_use_megaphone.go`
- Create: `libs/atlas-packet/cash/serverbound/item_use_super_megaphone.go`
- Create: `libs/atlas-packet/cash/serverbound/item_use_item_megaphone.go`
- Create: `libs/atlas-packet/cash/serverbound/item_use_triple_megaphone.go`
- Create: `libs/atlas-packet/cash/serverbound/item_use_maple_tv.go`
- Create: `libs/atlas-packet/cash/serverbound/item_use_avatar_megaphone.go`
- Test: `libs/atlas-packet/cash/serverbound/item_use_megaphone_test.go` (one `_test.go` per struct file, same basenames)

**Interfaces:**
- Consumes: `request.Reader`/`response.Writer` from `libs/atlas-socket`; the `NewItemUseFieldEffect(updateTimeFirst bool)` pattern (`item_use_field_effect.go`).
- Produces (used by Task 12): `NewItemUseMegaphone(updateTimeFirst bool) *ItemUseMegaphone` (`Message() string`), `NewItemUseSuperMegaphone(updateTimeFirst bool) *ItemUseSuperMegaphone` (`Message() string`, `Whisper() bool`), `NewItemUseItemMegaphone(updateTimeFirst bool) *ItemUseItemMegaphone` (`Message() string`, `Whisper() bool`, `HasItem() bool`, `InvType() int32`, `Slot() int32`), `NewItemUseTripleMegaphone(updateTimeFirst bool) *ItemUseTripleMegaphone` (`Lines() []string`, `Whisper() bool`), `NewItemUseMapleTV(updateTimeFirst bool, tvType byte) *ItemUseMapleTV` (`Ear() bool`, `ReceiverName() string`, `Lines() []string` — always 5), `NewItemUseAvatarMegaphone(updateTimeFirst bool) *ItemUseAvatarMegaphone` (`Lines() []string` — always 4, `Whisper() bool`). All expose `UpdateTime() uint32`.

The `updateTimeFirst` convention (from `ItemUseFieldEffect`): when `false` (GMS < 95, JMS) the sub-body carries a **trailing** `uint32` updateTime; when `true` the top-level `ItemUse` already consumed it. This resolves the `character_cash_item_use.go:108` TODO for every branch. Wire shapes here are Cosmic-derived; Tasks 19–20 verify per version via IDA and may adjust.

- [ ] **Step 1: Write the failing round-trip tests**

One test per struct, `pt.Variants` style (copy the shape of `libs/atlas-packet/cash/serverbound/item_use_field_effect_test.go`). Representative — `item_use_maple_tv_test.go` (the other five follow the identical pattern with their own fields):

```go
package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

func TestItemUseMapleTVRoundTrip(t *testing.T) {
	// tvType drives the conditional prefix; cover every arm.
	cases := []struct {
		name   string
		tvType byte
		ear    bool
		recv   string
	}{
		{"tv0_normal", 0, false, "PartnerA"},   // byte pad + receiver
		{"tv1_star", 1, false, ""},             // no prefix, no receiver
		{"tv2_heart", 2, false, "PartnerB"},    // receiver only
		{"tv3_megassenger", 3, true, "PartnerC"}, // byte + ear + receiver
		{"tv4_star_m", 4, true, ""},            // ear, NO receiver
		{"tv5_heart_m", 5, false, "PartnerD"},  // ear + receiver
	}
	for _, v := range pt.Variants {
		for _, tc := range cases {
			t.Run(v.Name+"/"+tc.name, func(t *testing.T) {
				ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
				updateTimeFirst := v.Region == "GMS" && v.MajorVersion >= 95
				input := NewItemUseMapleTV(updateTimeFirst, tc.tvType)
				input.ear = tc.ear
				input.receiverName = tc.recv
				input.lines = [5]string{"l1", "l2", "l3", "l4", "l5"}
				if !updateTimeFirst {
					input.updateTime = 42
				}
				output := NewItemUseMapleTV(updateTimeFirst, tc.tvType)
				pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
				if output.Ear() != input.Ear() {
					t.Errorf("ear: got %v, want %v", output.Ear(), input.Ear())
				}
				if output.ReceiverName() != input.ReceiverName() {
					t.Errorf("receiverName: got %q, want %q", output.ReceiverName(), input.ReceiverName())
				}
				for i := range input.lines {
					if output.Lines()[i] != input.Lines()[i] {
						t.Errorf("line %d: got %q, want %q", i, output.Lines()[i], input.Lines()[i])
					}
				}
				if output.UpdateTime() != input.UpdateTime() {
					t.Errorf("updateTime: got %v, want %v", output.UpdateTime(), input.UpdateTime())
				}
			})
		}
	}
}
```

Write the equivalent tests for the other five structs: megaphone (message + trailing updateTime), super (message, whisper), item megaphone (cover `hasItem` true and false — when false, invType/slot are absent from the wire), triple (cover lines counts 1, 2, 3), avatar (4 lines + whisper).

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd libs/atlas-packet && go test ./cash/serverbound/ -run 'ItemUseMegaphone|ItemUseSuperMegaphone|ItemUseItemMegaphone|ItemUseTripleMegaphone|ItemUseMapleTV|ItemUseAvatarMegaphone' -v`
Expected: FAIL — `undefined: NewItemUseMapleTV` (and the other five constructors).

- [ ] **Step 3: Implement the six structs**

`item_use_megaphone.go`:

```go
package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

// ItemUseMegaphone is the USE_CASH_ITEM sub-body for the basic Megaphone
// (5071xxx, cash-slot type 12 within classification 507).
// Cosmic-derived (UseCashItemHandler case 1); per-version IDA verification in task-123 phases 19-20.
type ItemUseMegaphone struct {
	message         string
	updateTime      uint32
	updateTimeFirst bool
}

func NewItemUseMegaphone(updateTimeFirst bool) *ItemUseMegaphone {
	return &ItemUseMegaphone{updateTimeFirst: updateTimeFirst}
}

func (m ItemUseMegaphone) Message() string    { return m.message }
func (m ItemUseMegaphone) UpdateTime() uint32 { return m.updateTime }

func (m ItemUseMegaphone) Operation() string { return "ItemUseMegaphone" }

func (m ItemUseMegaphone) String() string {
	return fmt.Sprintf("message [%s] updateTime [%d]", m.message, m.updateTime)
}

func (m ItemUseMegaphone) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteAsciiString(m.message)
		if !m.updateTimeFirst {
			w.WriteInt(m.updateTime)
		}
		return w.Bytes()
	}
}

func (m *ItemUseMegaphone) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.message = r.ReadAsciiString()
		if !m.updateTimeFirst {
			m.updateTime = r.ReadUint32()
		}
	}
}
```

`item_use_super_megaphone.go` — same skeleton with:

```go
type ItemUseSuperMegaphone struct {
	message         string
	whisper         bool
	updateTime      uint32
	updateTimeFirst bool
}
// Encode body: WriteAsciiString(message); WriteBool(whisper); trailing updateTime as above.
// Decode mirrors. Accessors: Message() string, Whisper() bool, UpdateTime() uint32.
// Operation() "ItemUseSuperMegaphone".
```

`item_use_item_megaphone.go`:

```go
type ItemUseItemMegaphone struct {
	message         string
	whisper         bool
	hasItem         bool
	invType         int32
	slot            int32
	updateTime      uint32
	updateTimeFirst bool
}

// Encode:
//   w.WriteAsciiString(m.message)
//   w.WriteBool(m.whisper)
//   w.WriteBool(m.hasItem)
//   if m.hasItem { w.WriteInt32(m.invType); w.WriteInt32(m.slot) }
//   if !m.updateTimeFirst { w.WriteInt(m.updateTime) }
// Decode mirrors (invType/slot read only when hasItem).
// Accessors: Message, Whisper, HasItem, InvType, Slot, UpdateTime.
```

`item_use_triple_megaphone.go`:

```go
type ItemUseTripleMegaphone struct {
	lines           []string
	whisper         bool
	updateTime      uint32
	updateTimeFirst bool
}

// Encode:
//   w.WriteByte(byte(len(m.lines)))
//   for _, ln := range m.lines { w.WriteAsciiString(ln) }
//   w.WriteBool(m.whisper)
//   trailing updateTime as above
// Decode:
//   count := r.ReadByte()   // handler validates 1..3; decode reads what the count says
//   m.lines = make([]string, 0, count)
//   for i := byte(0); i < count; i++ { m.lines = append(m.lines, r.ReadAsciiString()) }
//   m.whisper = r.ReadBool()
//   trailing updateTime as above
// Accessors: Lines() []string, Whisper(), UpdateTime().
```

`item_use_maple_tv.go`:

```go
// ItemUseMapleTV is the USE_CASH_ITEM sub-body for Maple TV items
// (5075xxx / 5074000 on GMS>=95). tvType is derived by the CALLER from the
// item id (itemId % 10) — it is not on the wire; it selects which prefix
// fields exist. Cosmic-derived (UseCashItemHandler case 5).
type ItemUseMapleTV struct {
	tvType          byte
	pad             byte // present only when tvType == 3 (meaning unknown; drained)
	ear             bool
	receiverName    string
	lines           [5]string
	updateTime      uint32
	updateTimeFirst bool
}

func NewItemUseMapleTV(updateTimeFirst bool, tvType byte) *ItemUseMapleTV {
	return &ItemUseMapleTV{updateTimeFirst: updateTimeFirst, tvType: tvType}
}

func (m ItemUseMapleTV) TvType() byte          { return m.tvType }
func (m ItemUseMapleTV) Ear() bool             { return m.ear }
func (m ItemUseMapleTV) ReceiverName() string  { return m.receiverName }
func (m ItemUseMapleTV) Lines() []string       { return m.lines[:] }
func (m ItemUseMapleTV) UpdateTime() uint32    { return m.updateTime }

// Encode (test mirror of Decode):
//   if m.tvType != 1 {
//     if m.tvType >= 3 {
//       if m.tvType == 3 { w.WriteByte(m.pad) }
//       w.WriteBool(m.ear)
//     } else if m.tvType != 2 {
//       w.WriteByte(m.pad)
//     }
//     if m.tvType != 4 { w.WriteAsciiString(m.receiverName) }
//   }
//   for _, ln := range m.lines { w.WriteAsciiString(ln) }
//   if !m.updateTimeFirst { w.WriteInt(m.updateTime) }
// Decode mirrors exactly.
```

`item_use_avatar_megaphone.go`:

```go
type ItemUseAvatarMegaphone struct {
	lines           [4]string
	whisper         bool
	updateTime      uint32
	updateTimeFirst bool
}
// Encode: 4× WriteAsciiString, WriteBool(whisper), trailing updateTime.
// Decode mirrors. Accessors: Lines() []string (4), Whisper(), UpdateTime().
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd libs/atlas-packet && go test -race ./cash/serverbound/ -v`
Expected: PASS (all new and existing tests).

- [ ] **Step 5: Vet and commit**

```bash
cd libs/atlas-packet && go vet ./... && cd ../..
git add libs/atlas-packet/cash/serverbound/
git commit -m "feat(task-123): serverbound USE_CASH_ITEM megaphone/TV/avatar sub-body codecs"
```

---

### Task 2: Saga library — type, actions, payloads, unmarshal (+ channel re-exports)

**Files:**
- Modify: `libs/atlas-saga/model.go` (add `MegaphoneUse` type; `EmitMegaphone`, `EnqueueWorldBroadcast` actions)
- Modify: `libs/atlas-saga/payloads.go` (add `AssetSnapshot`, `AvatarSnapshot`, `EmitMegaphonePayload`, `EnqueueWorldBroadcastPayload`)
- Modify: `libs/atlas-saga/unmarshal.go` (two new cases)
- Test: `libs/atlas-saga/unmarshal_test.go` (extend)
- Modify: `services/atlas-channel/atlas.com/channel/saga/model.go` (re-export the new type/actions/payloads)

**Interfaces:**
- Produces (used by Tasks 8, 10, 12, 13, 14):

```go
// model.go
const MegaphoneUse Type = "megaphone_use"
const (
	EmitMegaphone         Action = "emit_megaphone"
	EnqueueWorldBroadcast Action = "enqueue_world_broadcast"
)
```

```go
// payloads.go — snapshot DTOs shared by saga payloads AND the kafka message
// structs of channel/world/orchestrator (single source of truth; PRD Q6:
// snapshot at decode time, never re-resolved).

// AssetSnapshot captures one inventory asset at decode time (item megaphone).
type AssetSnapshot struct {
	Slot         int16     `json:"slot"`
	TemplateId   uint32    `json:"templateId"`
	Expiration   time.Time `json:"expiration"`
	CashId       int64     `json:"cashId"`
	Quantity     uint32    `json:"quantity"`
	Flag         uint16    `json:"flag"`
	Rechargeable uint64    `json:"rechargeable"`
	// equipment stats (zero for non-equips)
	Strength       uint16 `json:"strength"`
	Dexterity      uint16 `json:"dexterity"`
	Intelligence   uint16 `json:"intelligence"`
	Luck           uint16 `json:"luck"`
	Hp             uint16 `json:"hp"`
	Mp             uint16 `json:"mp"`
	WeaponAttack   uint16 `json:"weaponAttack"`
	MagicAttack    uint16 `json:"magicAttack"`
	WeaponDefense  uint16 `json:"weaponDefense"`
	MagicDefense   uint16 `json:"magicDefense"`
	Accuracy       uint16 `json:"accuracy"`
	Avoidability   uint16 `json:"avoidability"`
	Hands          uint16 `json:"hands"`
	Speed          uint16 `json:"speed"`
	Jump           uint16 `json:"jump"`
	Slots          uint16 `json:"slots"`
	LevelType      byte   `json:"levelType"`
	Level          byte   `json:"level"`
	Experience     uint32 `json:"experience"`
	HammersApplied uint32 `json:"hammersApplied"`
	// pet fields (zero/empty for non-pets)
	PetId     uint32 `json:"petId"`
	PetName   string `json:"petName"`
	PetLevel  byte   `json:"petLevel"`
	Closeness uint16 `json:"closeness"`
	Fullness  byte   `json:"fullness"`
}

// AvatarSnapshot captures a character's look at decode time (avatar megaphone / TV).
// Map keys are negated slot positions exactly as packetmodel.NewAvatar expects.
type AvatarSnapshot struct {
	Gender       byte             `json:"gender"`
	SkinColor    byte             `json:"skinColor"`
	Face         uint32           `json:"face"`
	Hair         uint32           `json:"hair"`
	Equips       map[int16]uint32 `json:"equips"`
	MaskedEquips map[int16]uint32 `json:"maskedEquips"`
	Pets         map[int8]uint32  `json:"pets"`
}

// EmitMegaphonePayload — stateless tiers (MEGAPHONE/SUPER/ITEM/TRIPLE).
type EmitMegaphonePayload struct {
	Tier        string         `json:"tier"`  // MEGAPHONE|SUPER|ITEM|TRIPLE
	Scope       string         `json:"scope"` // CHANNEL|WORLD
	WorldId     world.Id       `json:"worldId"`
	ChannelId   channel.Id     `json:"channelId"` // sender's channel
	CharacterId uint32         `json:"characterId"`
	SenderName  string         `json:"senderName"`
	SenderMedal string         `json:"senderMedal"`
	Messages    []string       `json:"messages"`
	WhispersOn  bool           `json:"whispersOn"`
	Item        *AssetSnapshot `json:"item,omitempty"` // ITEM tier only
}

// EnqueueWorldBroadcastPayload — serialized tiers (TV/AVATAR).
type EnqueueWorldBroadcastPayload struct {
	Family          string          `json:"family"` // TV|AVATAR
	WorldId         world.Id        `json:"worldId"`
	ChannelId       channel.Id      `json:"channelId"`
	CharacterId     uint32          `json:"characterId"`
	SenderName      string          `json:"senderName"`
	SenderMedal     string          `json:"senderMedal"`
	Messages        []string        `json:"messages"` // 5 for TV, 4 for AVATAR
	WhispersOn      bool            `json:"whispersOn"`
	ItemId          uint32          `json:"itemId"`        // AVATAR: used item id (SET packet field)
	TvMessageType   byte            `json:"tvMessageType"` // TV wire value 0/1/2
	DurationSeconds uint32          `json:"durationSeconds"`
	SenderLook      AvatarSnapshot  `json:"senderLook"`
	ReceiverName    string          `json:"receiverName"`
	ReceiverLook    *AvatarSnapshot `json:"receiverLook,omitempty"`
}
```

- [ ] **Step 1: Write the failing unmarshal tests**

Extend `libs/atlas-saga/unmarshal_test.go` following its existing per-action test shape: marshal a `Step[any]` with `Action: EmitMegaphone` and a populated `EmitMegaphonePayload` (include a non-nil `Item`), unmarshal, assert the payload round-trips as the concrete type; same for `EnqueueWorldBroadcast` with a `ReceiverLook`.

- [ ] **Step 2: Run to verify failure**

Run: `cd libs/atlas-saga && go test ./... -run Unmarshal -v`
Expected: FAIL — `undefined: EmitMegaphone`.

- [ ] **Step 3: Implement**

Add the constants to `model.go` (saga type next to `PetEvolution`; actions in a new `// Megaphone / world broadcast actions` block), the payload structs to `payloads.go` (imports for `world`/`channel` already present), and to `unmarshal.go` two cases exactly following the `FieldEffectWeather` case at line 492:

```go
	case EmitMegaphone:
		var payload EmitMegaphonePayload
		if err := json.Unmarshal(raw.Payload, &payload); err != nil {
			return err
		}
		s.Payload = any(payload).(T)
	case EnqueueWorldBroadcast:
		var payload EnqueueWorldBroadcastPayload
		if err := json.Unmarshal(raw.Payload, &payload); err != nil {
			return err
		}
		s.Payload = any(payload).(T)
```

(Match the file's actual assignment idiom — copy the neighboring case body verbatim and swap the type.)

- [ ] **Step 4: Run tests**

Run: `cd libs/atlas-saga && go test -race ./... && go vet ./...`
Expected: PASS.

- [ ] **Step 5: Re-export in atlas-channel's saga package**

In `services/atlas-channel/atlas.com/channel/saga/model.go` add to the existing re-export blocks:

```go
	// type block
	EmitMegaphonePayload         = sharedsaga.EmitMegaphonePayload
	EnqueueWorldBroadcastPayload = sharedsaga.EnqueueWorldBroadcastPayload
	AssetSnapshot                = sharedsaga.AssetSnapshot
	AvatarSnapshot               = sharedsaga.AvatarSnapshot

	// const block
	MegaphoneUse          = sharedsaga.MegaphoneUse
	EmitMegaphone         = sharedsaga.EmitMegaphone
	EnqueueWorldBroadcast = sharedsaga.EnqueueWorldBroadcast
```

Run: `cd services/atlas-channel/atlas.com/channel && go build ./...`
Expected: clean.

- [ ] **Step 6: Commit**

```bash
git add libs/atlas-saga/ services/atlas-channel/atlas.com/channel/saga/model.go
git commit -m "feat(task-123): MegaphoneUse saga type, broadcast actions and snapshot payloads"
```

---

### Task 3: chat/clientbound — discrete `WorldMessageMegaphone`, re-cut `WorldMessageItemMegaphone`

**Files:**
- Modify: `libs/atlas-packet/chat/clientbound/world_message.go`
- Test: `libs/atlas-packet/chat/clientbound/world_message_test.go` (extend/adjust)
- Modify: `services/atlas-channel/atlas.com/channel/socket/writer/world_message.go` (fix the `WorldMessageItemMegaphone` call site)

**Interfaces:**
- Consumes: `model.Asset` from `libs/atlas-packet/model/asset.go` (`Encode(l, ctx)(options) []byte`, `Decode`).
- Produces (used by Tasks 4, 13):
  - `NewWorldMessageMegaphone(mode byte, message string) WorldMessageMegaphone` — discrete MEGAPHONE-arm struct (mode + message), required for dispatcher enrollment (a `#`-entry must not map the shared `WorldMessageSimple`, INV-1/AP-1).
  - `NewWorldMessageItemMegaphone(mode byte, message string, channelId byte, whispersOn bool, item *model.Asset) WorldMessageItemMegaphone` — re-cut body per design D4: `mode, str message, byte channel, byte whisper, byte hasItem, [GW_ItemSlotBase item block]`. **The old `slot int32` body is deleted** (it cannot match the client read; nothing emits it today).

Cosmic reference for the item block (`PacketCreator.itemMegaphone`): `writeBool(item != null)`, then when present `addItemInfo(p, item, true)` (zero-position item block). Note Cosmic also writes a `slotPos byte` variant in some sources — the exact presence/order of a slot byte is IDA-verified in Task 19; ship the Cosmic-cited shape `hasItem + item block` now.

- [ ] **Step 1: Write/adjust failing tests**

In `world_message_test.go` add `TestWorldMessageMegaphoneRoundTrip` (mode + message round-trip over `pt.Variants`) and rewrite the existing item-megaphone test to construct an item block:

```go
func TestWorldMessageItemMegaphoneRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			item := model.NewAsset(true, 5, 4001126, time.Time{}).SetStackableInfo(30, 0, 0)
			input := NewWorldMessageItemMegaphone(8, "selling stuff", 2, true, &item)
			output := WorldMessageItemMegaphone{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Message() != input.Message() || output.ChannelId() != input.ChannelId() || output.WhispersOn() != input.WhispersOn() {
				t.Errorf("scalar fields did not round-trip")
			}
			if !output.HasItem() {
				t.Errorf("hasItem: got false, want true")
			}
			// no-item variant
			input2 := NewWorldMessageItemMegaphone(8, "no item", 2, false, nil)
			output2 := WorldMessageItemMegaphone{}
			pt.RoundTrip(t, ctx, input2.Encode, output2.Decode, nil)
			if output2.HasItem() {
				t.Errorf("hasItem: got true, want false")
			}
		})
	}
}
```

(Import `"time"` and `"github.com/Chronicle20/atlas/libs/atlas-packet/model"`. If `model.Asset` decode requires tenant-versioned context, `pt.CreateContext` provides it.)

- [ ] **Step 2: Run to verify failure**

Run: `cd libs/atlas-packet && go test ./chat/clientbound/ -run 'Megaphone' -v`
Expected: FAIL — `undefined: NewWorldMessageMegaphone` / wrong-signature `NewWorldMessageItemMegaphone`.

- [ ] **Step 3: Implement**

In `world_message.go` add the discrete megaphone struct:

```go
// WorldMessageMegaphone - discrete MEGAPHONE arm (mode 2 in v83): mode, message.
// Discrete (not WorldMessageSimple) so the dispatcher #-entry maps one struct
// to one mode (DISPATCHER_FAMILY.md INV-1).
type WorldMessageMegaphone struct {
	mode    byte
	message string
}

func NewWorldMessageMegaphone(mode byte, message string) WorldMessageMegaphone {
	return WorldMessageMegaphone{mode: mode, message: message}
}

func (m WorldMessageMegaphone) Mode() byte        { return m.mode }
func (m WorldMessageMegaphone) Message() string   { return m.message }
func (m WorldMessageMegaphone) Operation() string { return WorldMessageWriter }
func (m WorldMessageMegaphone) String() string    { return "world message megaphone" }

func (m WorldMessageMegaphone) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteAsciiString(m.message)
		return w.Bytes()
	}
}

func (m *WorldMessageMegaphone) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.message = r.ReadAsciiString()
	}
}
```

Replace the `WorldMessageItemMegaphone` struct body:

```go
// WorldMessageItemMegaphone - mode, message, channel, whispersOn, hasItem, [item block]
// Item block is the shared model.Asset (GW_ItemSlotBase-style) encoder. (design D4)
type WorldMessageItemMegaphone struct {
	mode       byte
	message    string
	channelId  byte
	whispersOn bool
	item       *model.Asset
}

func NewWorldMessageItemMegaphone(mode byte, message string, channelId byte, whispersOn bool, item *model.Asset) WorldMessageItemMegaphone {
	return WorldMessageItemMegaphone{mode: mode, message: message, channelId: channelId, whispersOn: whispersOn, item: item}
}

func (m WorldMessageItemMegaphone) Mode() byte         { return m.mode }
func (m WorldMessageItemMegaphone) Message() string    { return m.message }
func (m WorldMessageItemMegaphone) ChannelId() byte    { return m.channelId }
func (m WorldMessageItemMegaphone) WhispersOn() bool   { return m.whispersOn }
func (m WorldMessageItemMegaphone) HasItem() bool      { return m.item != nil }
func (m WorldMessageItemMegaphone) Item() *model.Asset { return m.item }

func (m WorldMessageItemMegaphone) Operation() string { return WorldMessageWriter }
func (m WorldMessageItemMegaphone) String() string    { return "world message item megaphone" }

func (m WorldMessageItemMegaphone) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteAsciiString(m.message)
		w.WriteByte(m.channelId)
		w.WriteBool(m.whispersOn)
		w.WriteBool(m.item != nil)
		if m.item != nil {
			w.WriteByteArray(m.item.Encode(l, ctx)(options))
		}
		return w.Bytes()
	}
}

func (m *WorldMessageItemMegaphone) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.message = r.ReadAsciiString()
		m.channelId = r.ReadByte()
		m.whispersOn = r.ReadBool()
		if r.ReadBool() {
			item := model.Asset{}
			item.Decode(l, ctx)(r, options)
			m.item = &item
		}
	}
}
```

Add the `model` import (`"github.com/Chronicle20/atlas/libs/atlas-packet/model"`). If `model.Asset.Decode` does not exist (check `libs/atlas-packet/model/asset.go` — Encode exists; Decode exists for round-trip tests per `asset_test.go`), mirror whatever the storage/trade round-trip tests do.

In `services/atlas-channel/atlas.com/channel/socket/writer/world_message.go`: the `case WorldMessageItemMegaphone:` arm of `worldMessageBody` no longer compiles (old `slot int32` arg). **Delete that case and the `slot int32` parameter** from `worldMessageBody` (update all its internal call sites — every existing `worldMessageBody(...)` call passes `0` for slot today). Player item megaphones are emitted via the Task 4 body functions, not this switch; no current consumer emits ITEM_MEGAPHONE through the switch (verify with `grep -rn "WorldMessageItemMegaphone" services/atlas-channel` — only the switch itself references it).

- [ ] **Step 4: Run tests**

Run: `cd libs/atlas-packet && go test -race ./chat/... && go vet ./chat/...`
Run: `cd services/atlas-channel/atlas.com/channel && go build ./... && go test -race ./socket/...`
Expected: PASS / clean.

- [ ] **Step 5: Commit**

```bash
git add libs/atlas-packet/chat/clientbound/ services/atlas-channel/atlas.com/channel/socket/writer/world_message.go
git commit -m "feat(task-123): discrete WorldMessageMegaphone; re-cut item megaphone around model.Asset"
```

---

### Task 4: WorldMessage per-mode body functions (`libs/atlas-packet/chat/world_message_body.go`)

**Files:**
- Create: `libs/atlas-packet/chat/world_message_body.go` (new package `chat` at the lib root, sibling of `clientbound/` — the `libs/atlas-packet/field/field_effect_body.go` model)
- Test: `libs/atlas-packet/chat/world_message_body_test.go`

**Interfaces:**
- Consumes: `atlas_packet.WithResolvedCode(codeProperty, key string, factory func(byte) packet.Encoder)` (`libs/atlas-packet/resolve.go:13`); Task 3 structs.
- Produces (used by Task 13 via channel writer wrappers):

```go
package chat

type WorldMessageMode string

const (
	WorldMessageMegaphone      WorldMessageMode = "MEGAPHONE"
	WorldMessageSuperMegaphone WorldMessageMode = "SUPER_MEGAPHONE"
	WorldMessageItemMegaphone  WorldMessageMode = "ITEM_MEGAPHONE"
	WorldMessageMultiMegaphone WorldMessageMode = "MULTI_MEGAPHONE"
)

func WorldMessageMegaphoneBody(message string) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte
func WorldMessageSuperMegaphoneBody(message string, channelId byte, whispersOn bool) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte
func WorldMessageItemMegaphoneBody(message string, channelId byte, whispersOn bool, item *model.Asset) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte
func WorldMessageMultiMegaphoneBody(messages []string, channelId byte, whispersOn bool) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte
```

Each body function fixes its operations key (INV-3) and passes the resolved mode through (INV-2). Pattern, copied from `field_effect_body.go:25`:

```go
func WorldMessageMegaphoneBody(message string) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", string(WorldMessageMegaphone), func(mode byte) packet.Encoder {
		return clientbound.NewWorldMessageMegaphone(mode, message)
	})
}
```

- [ ] **Step 1: Write the failing test** — for each body func, invoke with an options map containing `"operations": map[string]interface{}{"MEGAPHONE": 2.0, "SUPER_MEGAPHONE": 3.0, "ITEM_MEGAPHONE": 8.0, "MULTI_MEGAPHONE": 10.0}` (match how `ResolveCode` reads template-loaded options — check `libs/atlas-packet/resolve.go` and an existing body-func test, e.g. in `libs/atlas-packet/field/`, for the exact map value type) and assert the first emitted byte equals the table value.

- [ ] **Step 2: Run to verify failure** — `cd libs/atlas-packet && go test ./chat/ -v` → FAIL undefined.

- [ ] **Step 3: Implement** the four body functions exactly per the pattern above (`WorldMessageMultiMegaphoneBody` delegates to `clientbound.NewWorldMessageMultiMegaphone(mode, messages, channelId, whispersOn)`).

- [ ] **Step 4: Run tests** — `cd libs/atlas-packet && go test -race ./chat/... && go vet ./chat/...` → PASS.

- [ ] **Step 5: Commit**

```bash
git add libs/atlas-packet/chat/world_message_body.go libs/atlas-packet/chat/world_message_body_test.go
git commit -m "feat(task-123): WorldMessage per-mode body functions with resolved mode bytes"
```

---

### Task 5: Avatar megaphone clientbound structs

**Files:**
- Create: `libs/atlas-packet/chat/clientbound/avatar_megaphone.go`
- Test: `libs/atlas-packet/chat/clientbound/avatar_megaphone_test.go`

**Interfaces:**
- Consumes: `model.Avatar` (`libs/atlas-packet/model/avatar.go` — has Encode and Decode; construct via `model.NewAvatar(gender, skinColor, face, mega, hair, equips, maskedEquips, pets)`; test-value shape per `libs/atlas-packet/messenger/clientbound/add_test.go` `testAvatar()`).
- Produces (used by Task 14):

```go
const SetAvatarMegaphoneWriter = "SetAvatarMegaphone"
const ClearAvatarMegaphoneWriter = "ClearAvatarMegaphone"
const AvatarMegaphoneResultWriter = "AvatarMegaphoneResult"

// SET: int itemId, str name, str×4 lines, int channel, byte whisper, AvatarLook (design §1.2, IDA v83≡v95)
func NewSetAvatarMegaphone(itemId uint32, name string, lines [4]string, channelId uint32, whispersOn bool, look model.Avatar) SetAvatarMegaphone
// CLEAR: single byte (Cosmic sends 1; client clear is guarded/idempotent)
func NewClearAvatarMegaphone() ClearAvatarMegaphone
// RESULT: byte code; iff code not in {83, 84}: str message (design §1.2)
func NewAvatarMegaphoneResult(code byte, message string) AvatarMegaphoneResult
```

Each struct: `Operation()` returns its writer const, accessors per field, Encode/Decode mirrors. `ClearAvatarMegaphone.Encode` writes `w.WriteByte(1)`. `AvatarMegaphoneResult.Encode`: `w.WriteByte(m.code); if m.code != 83 && m.code != 84 { w.WriteAsciiString(m.message) }` — Decode mirrors.

- [ ] **Step 1: Write failing round-trip tests** (`pt.Variants`; SET test uses a `testAvatar()`-style look with equips; RESULT test covers code 83 (no string) and code 1 (with string); CLEAR asserts a 1-byte payload).
- [ ] **Step 2: Run to verify failure** — `cd libs/atlas-packet && go test ./chat/clientbound/ -run Avatar -v` → FAIL undefined.
- [ ] **Step 3: Implement** the three structs in `avatar_megaphone.go` (package `clientbound`, standard skeleton as in Task 3; SET writes `WriteInt(itemId)`, `WriteAsciiString(name)`, 4× `WriteAsciiString`, `WriteInt(channelId)`, `WriteBool(whispersOn)`, `WriteByteArray(look.Encode(l, ctx)(options))`; Decode mirrors with `look.Decode`).
- [ ] **Step 4: Run tests** — `cd libs/atlas-packet && go test -race ./chat/... && go vet ./chat/...` → PASS.
- [ ] **Step 5: Commit**

```bash
git add libs/atlas-packet/chat/clientbound/avatar_megaphone.go libs/atlas-packet/chat/clientbound/avatar_megaphone_test.go
git commit -m "feat(task-123): avatar megaphone clientbound codecs (set/clear/result)"
```

---

### Task 6: Maple TV clientbound package (`libs/atlas-packet/tv/clientbound`)

**Files:**
- Create: `libs/atlas-packet/tv/clientbound/set_message.go`
- Create: `libs/atlas-packet/tv/clientbound/clear_message.go`
- Create: `libs/atlas-packet/tv/clientbound/send_message_result.go`
- Test: matching `_test.go` per file

**Interfaces:**
- Produces (used by Task 14):

```go
const TvSetMessageWriter = "TvSetMessage"
const TvClearMessageWriter = "TvClearMessage"
const TvSendMessageResultWriter = "TvSendMessageResult"

// flag: bit 2 = receiver look present. Cosmic writes 3 with partner / 1 without;
// constructor computes it — callers never pass raw flags.
func NewTvSetMessage(messageType byte, senderLook model.Avatar, senderName string, receiverName string, lines [5]string, totalWaitSeconds uint32, receiverLook *model.Avatar) TvSetMessage
func NewTvClearMessage() TvClearMessage
func NewTvSendMessageResultSuccess() TvSendMessageResult              // 00
func NewTvSendMessageResultError(code byte) TvSendMessageResult      // 01 <code>; code 2 = queue over an hour
```

`TvSetMessage.Encode` (read order design §1.2, v83≡v95): `flag := byte(1); if m.receiverLook != nil { flag = 3 }`, `WriteByte(flag)`, `WriteByte(messageType)`, sender look bytes, `WriteAsciiString(senderName)`, `WriteAsciiString(receiverName)` (empty when none), 5× `WriteAsciiString`, `WriteInt(totalWaitSeconds)`, receiver look bytes iff present. Decode mirrors (`flag & 2` gates the receiver look). `TvClearMessage` has an empty body. `TvSendMessageResult`: `WriteByte(hasError)`, `if hasError != 0 { WriteByte(code) }`.

- [ ] **Step 1: Write failing round-trip tests** (`pt.Variants`; TvSetMessage with and without receiver look; result success and error(2); clear asserts empty payload).
- [ ] **Step 2: Run to verify failure** — `cd libs/atlas-packet && go test ./tv/... -v` → FAIL (package doesn't exist yet — create the test files alongside; `go test` will fail compile on missing types).
- [ ] **Step 3: Implement** the three structs (package `clientbound` under `tv/`; same skeleton as Task 5; import `model` for Avatar).
- [ ] **Step 4: Run tests** — `cd libs/atlas-packet && go test -race ./tv/... && go vet ./tv/...` → PASS.
- [ ] **Step 5: Commit**

```bash
git add libs/atlas-packet/tv/
git commit -m "feat(task-123): Maple TV clientbound codecs (set/clear/result)"
```

---

### Task 7: atlas-world broadcast domain — model, registry, pure queue transitions

**Files:**
- Create: `services/atlas-world/atlas.com/world/broadcast/model.go`
- Create: `services/atlas-world/atlas.com/world/broadcast/registry.go`
- Test: `services/atlas-world/atlas.com/world/broadcast/model_test.go`

**Interfaces:**
- Consumes: `atlas.TenantRegistry[string, QueueModel]` + `atlas.NewSet` (`libs/atlas-redis`), `sharedsaga.AvatarSnapshot`/`AssetSnapshot` (Task 2). atlas-world `go.mod` gains `github.com/Chronicle20/atlas/libs/atlas-saga` (workspace lib — add the `require` + `go mod tidy`; no Dockerfile change, atlas-saga already has COPY lines).
- Produces (used by Tasks 8–9):

```go
package broadcast

const (
	FamilyTV     = "TV"
	FamilyAvatar = "AVATAR"
)

// Payload is the render payload carried from enqueue to STARTED, verbatim.
type Payload struct {
	ChannelId     byte                       `json:"channelId"`
	SenderName    string                     `json:"senderName"`
	SenderMedal   string                     `json:"senderMedal"`
	Messages      []string                   `json:"messages"`
	WhispersOn    bool                       `json:"whispersOn"`
	ItemId        uint32                     `json:"itemId"`
	TvMessageType byte                       `json:"tvMessageType"`
	SenderLook    sharedsaga.AvatarSnapshot  `json:"senderLook"`
	ReceiverName  string                     `json:"receiverName"`
	ReceiverLook  *sharedsaga.AvatarSnapshot `json:"receiverLook,omitempty"`
}

type Entry struct {
	Id              uuid.UUID `json:"id"`
	CharacterId     uint32    `json:"characterId"`
	Payload         Payload   `json:"payload"`
	DurationSeconds uint32    `json:"durationSeconds"`
	ActivatedAt     time.Time `json:"activatedAt,omitempty"`
	ExpiresAt       time.Time `json:"expiresAt,omitempty"`
}

// QueueModel is the per (tenant, world, family) queue stored as one Redis JSON
// value and mutated only through TenantRegistry.Update (WATCH/CAS).
type QueueModel struct {
	Active  *Entry  `json:"active,omitempty"`
	Pending []Entry `json:"pending"`
}

// Pure transitions (no I/O — unit-testable):
func (q QueueModel) Append(e Entry) QueueModel
func (q QueueModel) ActivateNext(now time.Time) (QueueModel, *Entry) // pops head of Pending into Active, stamps ActivatedAt/ExpiresAt; nil if Pending empty
func (q QueueModel) ClearActive() QueueModel
func (q QueueModel) ActiveExpired(now time.Time) bool // Active != nil && now >= ExpiresAt
// WaitSeconds = remaining active time + sum of pending durations; 0 when idle.
func (q QueueModel) WaitSeconds(now time.Time) uint32
```

- Registry (mirror of `services/atlas-world/atlas.com/world/channel/registry.go:16-46` exactly — TenantRegistry + tenant-tracking Set):

```go
type Registry struct {
	queues  *atlas.TenantRegistry[string, QueueModel]
	tenants *atlas.Set
}

func queueKey(worldId world.Id, family string) string { return fmt.Sprintf("%d:%s", worldId, family) }

func InitRegistry(client *goredis.Client) // namespace "world-broadcast", set "world-broadcast:tenants"
func GetRegistry() *Registry
func (r *Registry) Tenants() []tenant.Model                  // copy channel registry's Tenants()/trackTenant verbatim
func (r *Registry) Get(ctx context.Context, t tenant.Model, worldId world.Id, family string) (QueueModel, error)
func (r *Registry) Upsert(ctx context.Context, t tenant.Model, worldId world.Id, family string, fn func(QueueModel) QueueModel) (QueueModel, error)
```

`Upsert` wraps CAS with create-on-missing: call `r.queues.Update(ctx, t, key, fn)`; on `atlas.ErrNotFound`, `Put` an empty `QueueModel{}` then `Update` again (concurrent empty Puts are idempotent; the CAS Update after still serializes). `Upsert` also calls `trackTenant`.

- [ ] **Step 1: Write failing tests for the pure transitions** — table-driven `model_test.go`: Append preserves order; ActivateNext stamps `ActivatedAt=now`, `ExpiresAt=now+Duration` and empties from Pending; ActivateNext on empty Pending returns nil; ActiveExpired boundary (`now == ExpiresAt` → true); WaitSeconds = remaining active (rounded up) + pending durations; WaitSeconds 0 when idle. Use fixed `time.Date(...)` values.
- [ ] **Step 2: Run to verify failure** — `cd services/atlas-world/atlas.com/world && go test ./broadcast/ -v` → FAIL (package missing).
- [ ] **Step 3: Implement** `model.go` (pure functions — value receivers returning modified copies; `WaitSeconds` uses `uint32(math.Ceil(remaining.Seconds()))` for the active remainder) and `registry.go` (copy the channel registry file structure; swap types/namespace; add `Upsert`).
- [ ] **Step 4: Run tests** — `go test -race ./broadcast/ && go vet ./broadcast/` → PASS. (Registry methods are exercised in Task 8's processor tests via the miniredis/live-redis pattern used by `channel/registry_test.go` — follow whatever harness that file uses.)
- [ ] **Step 5: Commit**

```bash
git add services/atlas-world/atlas.com/world/broadcast/ services/atlas-world/atlas.com/world/go.mod services/atlas-world/atlas.com/world/go.sum
git commit -m "feat(task-123): world broadcast queue model and CAS registry"
```

---

### Task 8: atlas-world broadcast — Kafka messages, processor, sweep emissions

**Files:**
- Create: `services/atlas-world/atlas.com/world/kafka/message/broadcast/kafka.go`
- Create: `services/atlas-world/atlas.com/world/kafka/producer/broadcast/producer.go`
- Create: `services/atlas-world/atlas.com/world/broadcast/processor.go`
- Test: `services/atlas-world/atlas.com/world/broadcast/processor_test.go`

**Interfaces:**
- Consumes: Task 7 registry/model; `producer.ProviderImpl` (world's `kafka/producer/producer.go` wrapper — same as channel/rate producers); `kproducer.CreateKey`, `kproducer.SingleMessageProvider`.
- Produces:

```go
// kafka/message/broadcast/kafka.go
const (
	EnvCommandTopicWorldBroadcast     = "COMMAND_TOPIC_WORLD_BROADCAST"
	EnvEventTopicWorldBroadcastStatus = "EVENT_TOPIC_WORLD_BROADCAST_STATUS"

	StatusTypeQueued  = "QUEUED"
	StatusTypeStarted = "STARTED"
	StatusTypeEnded   = "ENDED"
)

type EnqueueCommand struct {
	Family          string                     `json:"family"`
	WorldId         byte                       `json:"worldId"`
	ChannelId       byte                       `json:"channelId"`
	CharacterId     uint32                     `json:"characterId"`
	SenderName      string                     `json:"senderName"`
	SenderMedal     string                     `json:"senderMedal"`
	Messages        []string                   `json:"messages"`
	WhispersOn      bool                       `json:"whispersOn"`
	ItemId          uint32                     `json:"itemId"`
	TvMessageType   byte                       `json:"tvMessageType"`
	DurationSeconds uint32                     `json:"durationSeconds"`
	SenderLook      sharedsaga.AvatarSnapshot  `json:"senderLook"`
	ReceiverName    string                     `json:"receiverName"`
	ReceiverLook    *sharedsaga.AvatarSnapshot `json:"receiverLook,omitempty"`
}

// StatusEvent: QUEUED carries WaitSeconds; STARTED carries the full render
// payload + TotalWaitSeconds (SEND_TV totalWaitTime); ENDED carries only
// Family/WorldId (+CharacterId of the ended entry).
type StatusEvent struct {
	Type             string                     `json:"type"`
	Family           string                     `json:"family"`
	WorldId          byte                       `json:"worldId"`
	CharacterId      uint32                     `json:"characterId"`
	WaitSeconds      uint32                     `json:"waitSeconds"`
	TotalWaitSeconds uint32                     `json:"totalWaitSeconds"`
	ChannelId        byte                       `json:"channelId"`
	SenderName       string                     `json:"senderName"`
	SenderMedal      string                     `json:"senderMedal"`
	Messages         []string                   `json:"messages"`
	WhispersOn       bool                       `json:"whispersOn"`
	ItemId           uint32                     `json:"itemId"`
	TvMessageType    byte                       `json:"tvMessageType"`
	SenderLook       sharedsaga.AvatarSnapshot  `json:"senderLook"`
	ReceiverName     string                     `json:"receiverName"`
	ReceiverLook     *sharedsaga.AvatarSnapshot `json:"receiverLook,omitempty"`
}
```

```go
// kafka/producer/broadcast/producer.go — providers keyed by worldId
func QueuedStatusEventProvider(worldId world.Id, family string, characterId uint32, waitSeconds uint32) model.Provider[[]kafka.Message]
func StartedStatusEventProvider(worldId world.Id, family string, e broadcast.Entry) model.Provider[[]kafka.Message] // TotalWaitSeconds = e.DurationSeconds
func EndedStatusEventProvider(worldId world.Id, family string, characterId uint32) model.Provider[[]kafka.Message]
// each: key := kproducer.CreateKey(int(worldId)); kproducer.SingleMessageProvider(key, &StatusEvent{...})
```

```go
// broadcast/processor.go
type Processor interface {
	Enqueue(worldId world.Id, family string, e Entry) error
	GetQueue(worldId world.Id, family string) (QueueModel, error)
	SweepTenant() error // one tenant (from ctx); called by the leader task per tenant
}
func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor
```

Behavior (all transitions through `Registry.Upsert` CAS; only the CAS winner emits — design D1):

- `Enqueue`: Upsert append; after CAS, if the resulting queue's `Active` is nil, immediately Upsert `ActivateNext(now)` and emit STARTED for the activated entry, plus emit QUEUED (waitSeconds 0). If something is already active, emit QUEUED with `WaitSeconds` computed from the pre-append queue. Log at info: tenant, world, family, characterId, waitSeconds.
- `SweepTenant`: `GetAllEntries` for the tenant; for each queue where `ActiveExpired(now)`: Upsert(`ClearActive` then `ActivateNext`); emit ENDED for the expired entry; if a next entry was activated, emit STARTED for it. Log transitions at info.

Time comes from `time.Now()` at the call site (passed into the pure functions), so tests inject fixed times through the pure layer and exercise the processor against a test Redis.

- [ ] **Step 1: Write failing processor tests** — follow the harness style of `services/atlas-world/atlas.com/world/channel/registry_test.go` / `processor_test.go` (they already solve Redis-in-tests; reuse the same mechanism — miniredis or env-gated live Redis — do NOT invent a new harness). Cases: enqueue-on-idle activates immediately; enqueue-on-busy appends and computes wait; sweep expires the active and promotes the next; sweep on empty queue is a no-op; CAS conflict retries (two concurrent Enqueues both land). Producer emissions: capture via the same producer-test mechanism the world/channel tests use (if none exists, assert queue state only and leave emissions to build verification — do not fabricate a mock kafka broker).
- [ ] **Step 2: Run to verify failure** — `go test ./broadcast/ -v` → FAIL.
- [ ] **Step 3: Implement** message structs, providers, processor.
- [ ] **Step 4: Run tests** — `cd services/atlas-world/atlas.com/world && go test -race ./... && go vet ./...` → PASS.
- [ ] **Step 5: Commit**

```bash
git add services/atlas-world/atlas.com/world/
git commit -m "feat(task-123): world broadcast processor with CAS queue transitions and status events"
```

---

### Task 9: atlas-world — consumer, REST endpoint, leader-gated sweep, main wiring

**Files:**
- Create: `services/atlas-world/atlas.com/world/kafka/consumer/broadcast/consumer.go`
- Create: `services/atlas-world/atlas.com/world/broadcast/resource.go`
- Create: `services/atlas-world/atlas.com/world/broadcast/rest.go`
- Create: `services/atlas-world/atlas.com/world/broadcast/task.go`
- Modify: `services/atlas-world/atlas.com/world/main.go`
- Modify: `services/atlas-world/atlas.com/world/go.mod` (add `github.com/Chronicle20/atlas/libs/atlas-lock`)
- Test: `services/atlas-world/atlas.com/world/broadcast/rest_test.go`

**Interfaces:**
- Consumes: Task 8 processor/messages; `lock.New` + `LeaderElection.Run` (usage pattern: `services/atlas-summons/atlas.com/summons/main.go:98-121`); `tasks.Register` (`services/atlas-world/atlas.com/world/tasks/task.go`); `rest.RegisterHandler`/`rest.ParseWorldId` (world's `rest` package, as used by `world/resource.go`).
- Produces: REST `GET /api/worlds/{worldId}/broadcast-queues/{family}` returning JSON:API resource:

```go
// broadcast/rest.go
type RestModel struct {
	Id           string `json:"-"`            // family
	Family       string `json:"family"`
	ActiveRemainingSeconds uint32 `json:"activeRemainingSeconds"`
	PendingCount int    `json:"pendingCount"`
	WaitSeconds  uint32 `json:"waitSeconds"`
}
func (r RestModel) GetName() string { return "broadcast-queues" }
func (r RestModel) GetID() string   { return r.Id }
func Transform(family string, q QueueModel, now time.Time) (RestModel, error)
```

- Consumer: copy the `kafka/consumer/channel/consumer.go` skeleton — `InitConsumers` registers `EnvCommandTopicWorldBroadcast` with `SpanHeaderParser, TenantHeaderParser` (NO LastOffset — commands must not be skipped); `InitHandlers` adapts `handleEnqueueCommand(l, ctx, cmd broadcast2.EnqueueCommand)` which validates `Family` ∈ {TV, AVATAR} and calls `broadcast.NewProcessor(l, ctx).Enqueue(world.Id(cmd.WorldId), cmd.Family, entry)` with `Entry{Id: uuid.New(), CharacterId: cmd.CharacterId, DurationSeconds: cmd.DurationSeconds, Payload: broadcast.Payload{…all render fields…}}`.
- Sweep task (`broadcast/task.go`, copy `channel/task.go` shape): `func NewSweep(l logrus.FieldLogger, ctx context.Context, interval time.Duration) *Sweep`; `(*Sweep).Run()` iterates `GetRegistry().Tenants()`, per tenant builds `tctx := tenant.WithContext(sctx, te)` and calls `NewProcessor(t.l, tctx).SweepTenant()`; `SleepTime()` returns the interval (1 s).
- `main.go` wiring:
  1. `broadcast.InitRegistry(rc)` next to `channel.InitRegistry(rc)` (line 63).
  2. `broadcastconsumer.InitConsumers(l)(cmf)(consumerGroupId)` + `InitHandlers` next to the channel consumer registrations (lines 89–92).
  3. `AddRouteInitializer(broadcast.InitResource(GetServer()))` (line 111 block).
  4. Replace the bare task registration block (line 147) with leader-gated startup for the broadcast sweep, copying summons `main.go:98-121` verbatim (lease name `"world-broadcast-sweep"`, env prefix `WORLD_BROADCAST_LEADER_*` with the same defaults/helpers summons uses — port the small env-helper funcs): inside the leader callback run `tasks.Register(l, leaderCtx)(broadcast.NewSweep(l, leaderCtx, time.Second))` then `<-leaderCtx.Done()`. The existing `channel.NewExpiration` registration stays as-is (out of scope).

- [ ] **Step 1: Write failing rest_test** — `Transform` maps an idle queue to `{ActiveRemainingSeconds: 0, PendingCount: 0, WaitSeconds: 0}` and a busy queue (active with 10 s remaining + 2 pending × 15 s) to `{10, 2, 40}`.
- [ ] **Step 2: Run to verify failure** — `go test ./broadcast/ -run Rest -v` → FAIL.
- [ ] **Step 3: Implement** rest.go, resource.go (route `router.PathPrefix("/worlds/{worldId}/broadcast-queues").Subrouter()`, handler parses worldId via `rest.ParseWorldId`, family from `mux.Vars(r)["family"]`, 400 unless TV|AVATAR, `GetQueue` treats `ErrNotFound` as empty queue, marshal via `server.MarshalResponse[RestModel]` — copy the `world/resource.go` handler shape), consumer.go, task.go, main.go wiring, go.mod.
- [ ] **Step 4: Verify** — `cd services/atlas-world/atlas.com/world && go test -race ./... && go vet ./... && go build ./...` → clean. `tools/redis-key-guard.sh` from repo root → clean.
- [ ] **Step 5: Commit**

```bash
git add services/atlas-world/atlas.com/world/ 
git commit -m "feat(task-123): world broadcast coordinator - consumer, REST queue view, leader-gated sweep"
```

---

### Task 10: atlas-saga-orchestrator — action handlers and producers

**Files:**
- Create: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/message/megaphone/kafka.go`
- Create: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/message/broadcast/kafka.go`
- Modify: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/producer.go`
- Modify: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/handler.go`
- Test: extend the orchestrator's existing handler test file for these two actions (locate with `grep -rn "handleFieldEffectWeather\|handleEmitGachaponWin" services/atlas-saga-orchestrator --include='*_test.go'` and follow its harness; if those handlers have no direct tests, match whatever coverage convention the file uses — do not leave the new handlers untested if a harness exists)

**Interfaces:**
- Consumes: `EmitMegaphonePayload` / `EnqueueWorldBroadcastPayload` (Task 2 — the orchestrator's `saga` package re-exports lib types the same way it does `EmitGachaponWinPayload`; add re-exports if its model file enumerates them — check `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/model.go` and mirror the channel-side additions from Task 2 Step 5).
- Produces:

```go
// kafka/message/megaphone/kafka.go
const EnvEventTopicMegaphone = "EVENT_TOPIC_MEGAPHONE"
type BroadcastEvent struct {
	Tier        string                    `json:"tier"`
	Scope       string                    `json:"scope"`
	WorldId     byte                      `json:"worldId"`
	ChannelId   byte                      `json:"channelId"`
	CharacterId uint32                    `json:"characterId"`
	SenderName  string                    `json:"senderName"`
	SenderMedal string                    `json:"senderMedal"`
	Messages    []string                  `json:"messages"`
	WhispersOn  bool                      `json:"whispersOn"`
	Item        *sharedsaga.AssetSnapshot `json:"item,omitempty"`
}
// kafka/message/broadcast/kafka.go — EnvCommandTopicWorldBroadcast + EnqueueCommand,
// byte-for-byte the same JSON shape as Task 8's world-side struct.
```

```go
// producer.go additions (copy GachaponRewardWonEventProvider at :148)
func MegaphoneBroadcastEventProvider(payload EmitMegaphonePayload) model.Provider[[]kafka.Message]
// key := kproducer.CreateKey(int(payload.WorldId))
func WorldBroadcastEnqueueCommandProvider(payload EnqueueWorldBroadcastPayload) model.Provider[[]kafka.Message]
// key := kproducer.CreateKey(int(payload.WorldId))  // single-partition ordering per world (D1)
```

Handlers (both fire-and-forget, `handleFieldEffectWeather` model at `handler.go:2855-2880`):

```go
func (h *HandlerImpl) handleEmitMegaphone(s Saga, st Step[any]) error {
	payload, ok := st.Payload().(EmitMegaphonePayload)
	if !ok {
		return errors.New("invalid payload")
	}
	h.l.WithFields(logrus.Fields{
		"transaction_id": s.TransactionId().String(),
		"character_id":   payload.CharacterId,
		"tier":           payload.Tier,
		"world_id":       payload.WorldId,
	}).Info("Emitting megaphone broadcast event.")
	err := producer.ProviderImpl(h.l)(h.ctx)(megaphone2.EnvEventTopicMegaphone)(MegaphoneBroadcastEventProvider(payload))
	if err != nil {
		h.logActionError(s, st, err, "Unable to emit megaphone broadcast event.")
		return err
	}
	_ = NewProcessor(h.l, h.ctx).StepCompleted(s.TransactionId(), true)
	return nil
}
```

`handleEnqueueWorldBroadcast` is identical in shape, producing `WorldBroadcastEnqueueCommandProvider(payload)` to `broadcast2.EnvCommandTopicWorldBroadcast`, logging `family`/`duration_seconds`.

Dispatch registration in the action switch (after `case FieldEffectWeather:` at `handler.go:861`):

```go
	case EmitMegaphone:
		return h.handleEmitMegaphone, true
	case EnqueueWorldBroadcast:
		return h.handleEnqueueWorldBroadcast, true
```

Also register the handler funcs in the `Handler` interface listing at the top of handler.go (the `handleEmitGachaponWin`/`handleFieldEffectWeather` entries at lines 143/151 show the shape). Check for a saga-type validation table (`grep -rn "FieldEffectUse" services/atlas-saga-orchestrator --include='*.go' | grep -iv test`) — if saga types are enumerated anywhere (validation, metrics), add `MegaphoneUse` there too.

- [ ] **Step 1: Write failing tests** (if the harness exists per above): payload-type mismatch returns error; happy path marks step completed.
- [ ] **Step 2: Run to verify failure.**
- [ ] **Step 3: Implement** message packages, providers, handlers, dispatch entries, re-exports.
- [ ] **Step 4: Verify** — `cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator && go test -race ./... && go vet ./... && go build ./...` → clean.
- [ ] **Step 5: Commit**

```bash
git add services/atlas-saga-orchestrator/
git commit -m "feat(task-123): orchestrator EmitMegaphone and EnqueueWorldBroadcast actions"
```

---

### Task 11: atlas-channel — Kafka message packages and world-broadcast REST client

**Files:**
- Create: `services/atlas-channel/atlas.com/channel/kafka/message/megaphone/kafka.go` (identical `BroadcastEvent` + `EnvEventTopicMegaphone` as Task 10's megaphone message)
- Create: `services/atlas-channel/atlas.com/channel/kafka/message/worldbroadcast/kafka.go` (identical `StatusEvent` + `EnvEventTopicWorldBroadcastStatus` + `StatusTypeQueued/Started/Ended` as Task 8's status event)
- Create: `services/atlas-channel/atlas.com/channel/worldbroadcast/requests.go`
- Create: `services/atlas-channel/atlas.com/channel/worldbroadcast/rest.go`
- Create: `services/atlas-channel/atlas.com/channel/worldbroadcast/processor.go`
- Test: `services/atlas-channel/atlas.com/channel/worldbroadcast/rest_test.go`

**Interfaces:**
- Consumes: `requests.RootUrl("WORLDS")` (`atlas-channel/world/requests.go:16` precedent), `requests.GetRequest[RestModel]`.
- Produces (used by Task 12):

```go
package worldbroadcast

const (
	FamilyTV     = "TV"
	FamilyAvatar = "AVATAR"
)

// rest.go
type RestModel struct {
	Id                     string `json:"-"`
	Family                 string `json:"family"`
	ActiveRemainingSeconds uint32 `json:"activeRemainingSeconds"`
	PendingCount           int    `json:"pendingCount"`
	WaitSeconds            uint32 `json:"waitSeconds"`
}
func (r RestModel) GetName() string { return "broadcast-queues" }
func (r *RestModel) SetID(id string) error { r.Id = id; return nil }

// requests.go
const (
	Resource = "worlds/%d/broadcast-queues/%s"
)
func requestQueue(worldId world.Id, family string) requests.Request[RestModel]

// processor.go
type Processor interface {
	GetWaitSeconds(worldId world.Id, family string) (uint32, error)
}
func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor
// GetWaitSeconds resolves requestQueue(...)(l, ctx) and returns WaitSeconds.
// Any transport/decode error is returned to the caller (handler rejects
// conservatively on error — design §6 "never consume-then-drop").
```

- [ ] **Step 1: Write failing rest_test** — JSON:API unmarshal shape test: decode a canned `{"data":{"type":"broadcast-queues","id":"TV","attributes":{...}}}` document through the same helper other channel REST models test with (find one: `grep -rln "SetID" services/atlas-channel/atlas.com/channel/world/` and mirror its test if present; otherwise assert `GetName()`/`SetID` behavior directly).
- [ ] **Step 2: Run to verify failure.**
- [ ] **Step 3: Implement** all five files (message packages are copies of the Task 8/10 shapes — same JSON tags, same const strings).
- [ ] **Step 4: Verify** — `cd services/atlas-channel/atlas.com/channel && go test -race ./worldbroadcast/... && go build ./...` → clean.
- [ ] **Step 5: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/kafka/message/megaphone/ services/atlas-channel/atlas.com/channel/kafka/message/worldbroadcast/ services/atlas-channel/atlas.com/channel/worldbroadcast/
git commit -m "feat(task-123): channel megaphone/world-broadcast message types and queue REST client"
```

---

### Task 12: atlas-channel — USE_CASH_ITEM megaphone handler branches

**Files:**
- Create: `services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use_megaphone.go`
- Modify: `services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go`
- Create: `services/atlas-channel/atlas.com/channel/socket/model/snapshot.go` (snapshot converters)
- Test: `services/atlas-channel/atlas.com/channel/socket/model/snapshot_test.go`

**Interfaces:**
- Consumes: Task 1 decode structs, Task 2 saga payloads (via channel `saga` re-exports), Task 11 `worldbroadcast.NewProcessor(...).GetWaitSeconds`, `character2.Processor.GetItemInSlot/GetById/GetByName` (`character/processor.go:65,208,223`), `socketmodel.NewFromCharacter` (`socket/model/avatar.go:10`), Task 5/6 rejection packets, `session.Announce`.
- Produces:

```go
// socket/model/snapshot.go
package model

// NewAssetSnapshot flattens a channel asset.Model into the saga DTO (decode-time snapshot, PRD Q6).
func NewAssetSnapshot(a asset.Model) sharedsaga.AssetSnapshot
// NewAssetFromSnapshot rebuilds the packet item block from a snapshot (consumer side, Task 13).
func NewAssetFromSnapshot(s sharedsaga.AssetSnapshot) packetmodel.Asset
// NewAvatarSnapshot flattens a character look (same field walk as NewFromCharacter at avatar.go:10-39,
// but into int16-keyed maps).
func NewAvatarSnapshot(c character.Model) sharedsaga.AvatarSnapshot
// NewAvatarFromSnapshot rebuilds packetmodel.Avatar (mega flag per call site; consumer side, Task 14).
func NewAvatarFromSnapshot(s sharedsaga.AvatarSnapshot, mega bool) packetmodel.Avatar
```

`NewAssetSnapshot` copies every accessor of `asset.Model` used by `socket/model/asset.go:13-39` (`NewAsset`) into the DTO; `NewAssetFromSnapshot` replays them through `packetmodel.NewAsset(true, s.Slot, s.TemplateId, s.Expiration)` + `SetEquipmentStats/SetEquipmentMeta/SetCashId/SetStackableInfo/SetPetInfo` guarded by the same `IsEquipment/IsCash/...` conditions (`packetmodel.Asset` exposes `InventoryType()`-derived predicates; for the snapshot use `inventory.TypeFromItemId(item.Id(s.TemplateId))` the way `asset.go:99-101` does). `NewAvatarSnapshot`/`NewAvatarFromSnapshot` convert between `map[int16]uint32` and `map[slot.Position]uint32` with plain loops.

Handler — in `character_cash_item_use.go`, immediately before the fall-through warn (old line 110, after the FieldEffect branch), insert the classification-keyed dispatch (design §1.1: classification FIRST — type 12 collides with teleport rock, 42 with pet evolution):

```go
		category := item.GetClassification(itemId)
		if category == item.ClassificationMegaphones {
			handleMegaphoneUse(l, ctx, wp)(s, r, readerOptions, t, itemId, source, updateTimeFirst)
			return
		}
		if category == item.ClassificationAvatarMegaphone {
			handleAvatarMegaphoneUse(l, ctx, wp)(s, r, readerOptions, t, itemId, source, updateTimeFirst)
			return
		}
```

and change the func signature's ignored writer producer to a named one: `func CharacterCashItemUseHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) ...`. Delete the `// TODO for v83 there is a trailing updateTime.` comment (resolved by the `updateTimeFirst` convention in the sub-bodies).

`character_cash_item_use_megaphone.go` — full new file:

```go
package handler

import (
	character2 "atlas-channel/character"
	"atlas-channel/saga"
	"atlas-channel/session"
	socketmodel "atlas-channel/socket/model"
	"atlas-channel/socket/writer"
	"atlas-channel/worldbroadcast"
	"context"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory/slot"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	cashsb "github.com/Chronicle20/atlas/libs/atlas-packet/cash/serverbound"
	chatpkt "github.com/Chronicle20/atlas/libs/atlas-packet/chat/clientbound"
	tvpkt "github.com/Chronicle20/atlas/libs/atlas-packet/tv/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

const (
	tierMegaphone = "MEGAPHONE"
	tierSuper     = "SUPER"
	tierItem      = "ITEM"
	tierTriple    = "TRIPLE"

	scopeChannel = "CHANNEL"
	scopeWorld   = "WORLD"

	tvWaitCapSeconds     = uint32(3600) // client string: "the waiting line is longer than an hour"
	avatarWaitCapSeconds = uint32(15)   // client string SP_3972 (design §1.2)
	avatarDurationSecs   = uint32(10)   // client auto-clear constant, IDA v83+v95
)

// tvDurationSeconds — Cosmic MapleTVEffect.java:56-61 (server policy values, design D8).
func tvDurationSeconds(tvType byte) uint32 {
	switch tvType {
	case 4:
		return 30
	case 5:
		return 60
	default:
		return 15
	}
}

// tvMessageType — wire value: Cosmic PacketCreator.sendTV call site (type <= 2 ? type : type - 3).
func tvMessageType(tvType byte) byte {
	if tvType <= 2 {
		return tvType
	}
	return tvType - 3
}

func handleMegaphoneUse(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}, t tenant.Model, itemId item.Id, source slot.Position, updateTimeFirst bool) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}, t tenant.Model, itemId item.Id, source slot.Position, updateTimeFirst bool) {
		c, err := character2.NewProcessor(l, ctx).GetById()(s.CharacterId())
		if err != nil {
			l.WithError(err).Warnf("Character [%d] not found for megaphone use.", s.CharacterId())
			return
		}
		f := s.Field()

		// classification 507 sub-family, Cosmic UseCashItemHandler: (itemId / 1000) % 10
		switch (uint32(itemId) / 1000) % 10 {
		case 1: // basic megaphone — channel scope
			sp := cashsb.NewItemUseMegaphone(updateTimeFirst)
			sp.Decode(l, ctx)(r, readerOptions)
			createMegaphoneSaga(l, ctx)(s, itemId, saga.EmitMegaphonePayload{
				Tier: tierMegaphone, Scope: scopeChannel,
				WorldId: f.WorldId(), ChannelId: f.ChannelId(), CharacterId: s.CharacterId(),
				SenderName: c.Name(), SenderMedal: "",
				Messages: []string{sp.Message()},
			})
		case 2: // super megaphone — world scope
			sp := cashsb.NewItemUseSuperMegaphone(updateTimeFirst)
			sp.Decode(l, ctx)(r, readerOptions)
			createMegaphoneSaga(l, ctx)(s, itemId, saga.EmitMegaphonePayload{
				Tier: tierSuper, Scope: scopeWorld,
				WorldId: f.WorldId(), ChannelId: f.ChannelId(), CharacterId: s.CharacterId(),
				SenderName: c.Name(), SenderMedal: "",
				Messages: []string{sp.Message()}, WhispersOn: sp.Whisper(),
			})
		case 4: // 5074000 Skull Megaphone — TV family ONLY on GMS>=95 (classifier: type 0 → no send path on <95, design §1.1)
			if t.Region() == "GMS" && t.MajorVersion() >= 95 {
				handleMapleTVUse(l, ctx, wp)(s, r, readerOptions, itemId, c, updateTimeFirst)
			} else {
				l.Warnf("Character [%d] used megaphone item [%d] with no send path on this version.", s.CharacterId(), itemId)
			}
		case 5: // Maple TV / messenger group (5075xxx)
			handleMapleTVUse(l, ctx, wp)(s, r, readerOptions, itemId, c, updateTimeFirst)
		case 6: // item megaphone
			sp := cashsb.NewItemUseItemMegaphone(updateTimeFirst)
			sp.Decode(l, ctx)(r, readerOptions)
			payload := saga.EmitMegaphonePayload{
				Tier: tierItem, Scope: scopeWorld,
				WorldId: f.WorldId(), ChannelId: f.ChannelId(), CharacterId: s.CharacterId(),
				SenderName: c.Name(), SenderMedal: "",
				Messages: []string{sp.Message()}, WhispersOn: sp.Whisper(),
			}
			if sp.HasItem() {
				ref, err := character2.NewProcessor(l, ctx).GetItemInSlot(s.CharacterId(), inventory.Type(sp.InvType()), int16(sp.Slot()))()
				if err != nil {
					// FR-1.4: empty/mismatched referenced slot rejects the use — no consume, no broadcast.
					l.WithError(err).Warnf("Character [%d] item megaphone referenced empty slot [%d/%d].", s.CharacterId(), sp.InvType(), sp.Slot())
					return
				}
				snap := socketmodel.NewAssetSnapshot(ref)
				payload.Item = &snap
			}
			createMegaphoneSaga(l, ctx)(s, itemId, payload)
		case 7: // triple megaphone
			sp := cashsb.NewItemUseTripleMegaphone(updateTimeFirst)
			sp.Decode(l, ctx)(r, readerOptions)
			if len(sp.Lines()) < 1 || len(sp.Lines()) > 3 {
				l.Warnf("Character [%d] triple megaphone with invalid line count [%d].", s.CharacterId(), len(sp.Lines()))
				return
			}
			createMegaphoneSaga(l, ctx)(s, itemId, saga.EmitMegaphonePayload{
				Tier: tierTriple, Scope: scopeWorld,
				WorldId: f.WorldId(), ChannelId: f.ChannelId(), CharacterId: s.CharacterId(),
				SenderName: c.Name(), SenderMedal: "",
				Messages: sp.Lines(), WhispersOn: sp.Whisper(),
			})
		default:
			// 5070000 Cheap / 5073000 Heart have no client send path (classifier type 0);
			// type-8 (507x8xxx) has no item in v83 WZ (design D11).
			l.Warnf("Character [%d] used unsupported megaphone item [%d].", s.CharacterId(), itemId)
		}
	}
}
```

The saga builder (same file), the `FieldEffectUse` pattern from `character_cash_item_use.go:65-104`:

```go
func createMegaphoneSaga(l logrus.FieldLogger, ctx context.Context) func(s session.Model, itemId item.Id, payload saga.EmitMegaphonePayload) {
	return func(s session.Model, itemId item.Id, payload saga.EmitMegaphonePayload) {
		now := time.Now()
		steps := []saga.Step{
			{
				StepId: "consume_megaphone_item",
				Status: saga.Pending,
				Action: saga.DestroyAsset,
				Payload: saga.DestroyAssetPayload{
					CharacterId: s.CharacterId(),
					TemplateId:  uint32(itemId),
					Quantity:    1,
					RemoveAll:   false,
				},
				CreatedAt: now,
				UpdatedAt: now,
			},
			{
				StepId:    "emit_megaphone_broadcast",
				Status:    saga.Pending,
				Action:    saga.EmitMegaphone,
				Payload:   payload,
				CreatedAt: now,
				UpdatedAt: now,
			},
		}
		_ = saga.NewProcessor(l, ctx).Create(saga.Saga{
			TransactionId: uuid.New(),
			SagaType:      saga.MegaphoneUse,
			InitiatedBy:   "CASH_ITEM_USE",
			Steps:         steps,
		})
	}
}
```

TV branch (same file). Note the megassenger double-broadcast (Cosmic parity — three-step saga):

```go
func handleMapleTVUse(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}, itemId item.Id, c character2.Model, updateTimeFirst bool) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}, itemId item.Id, c character2.Model, updateTimeFirst bool) {
		tvType := byte(uint32(itemId) % 10)
		sp := cashsb.NewItemUseMapleTV(updateTimeFirst, tvType)
		sp.Decode(l, ctx)(r, readerOptions)
		f := s.Field()

		// Wait-cap guard BEFORE consuming (design D3). REST failure rejects conservatively.
		wait, err := worldbroadcast.NewProcessor(l, ctx).GetWaitSeconds(f.WorldId(), worldbroadcast.FamilyTV)
		if err != nil || wait > tvWaitCapSeconds {
			if err != nil {
				l.WithError(err).Warnf("Unable to check TV queue for world [%d]; rejecting without consuming.", f.WorldId())
			}
			_ = session.Announce(l)(ctx)(wp)(tvpkt.TvSendMessageResultWriter)(tvpkt.NewTvSendMessageResultError(2).Encode)(s)
			return
		}

		// Partner lookup by name (design §5: absent/mismatch → self-message).
		var receiverName string
		var receiverLook *saga.AvatarSnapshot
		if sp.ReceiverName() != "" {
			if partner, perr := character2.NewProcessor(l, ctx).GetByName(sp.ReceiverName()); perr == nil {
				snap := socketmodel.NewAvatarSnapshot(partner)
				receiverName = partner.Name()
				receiverLook = &snap
			} else {
				l.Debugf("TV partner [%s] not found; broadcasting without partner.", sp.ReceiverName())
			}
		}

		lines := sp.Lines()
		enqueue := saga.EnqueueWorldBroadcastPayload{
			Family:  worldbroadcast.FamilyTV,
			WorldId: f.WorldId(), ChannelId: f.ChannelId(), CharacterId: s.CharacterId(),
			SenderName: c.Name(), SenderMedal: "",
			Messages:        lines,
			TvMessageType:   tvMessageType(tvType),
			DurationSeconds: tvDurationSeconds(tvType),
			SenderLook:      socketmodel.NewAvatarSnapshot(c),
			ReceiverName:    receiverName,
			ReceiverLook:    receiverLook,
		}

		now := time.Now()
		steps := []saga.Step{
			{StepId: "consume_tv_item", Status: saga.Pending, Action: saga.DestroyAsset,
				Payload:   saga.DestroyAssetPayload{CharacterId: s.CharacterId(), TemplateId: uint32(itemId), Quantity: 1, RemoveAll: false},
				CreatedAt: now, UpdatedAt: now},
			{StepId: "enqueue_tv_broadcast", Status: saga.Pending, Action: saga.EnqueueWorldBroadcast,
				Payload: enqueue, CreatedAt: now, UpdatedAt: now},
		}
		if tvType >= 3 {
			// Megassenger tiers also fire a super megaphone with the concatenated
			// lines and ear-as-whisper (Cosmic UseCashItemHandler case 5 parity).
			combined := ""
			for _, ln := range lines {
				if ln != "" {
					if combined != "" {
						combined += " "
					}
					combined += ln
				}
			}
			steps = append(steps, saga.Step{
				StepId: "emit_megassenger_super", Status: saga.Pending, Action: saga.EmitMegaphone,
				Payload: saga.EmitMegaphonePayload{
					Tier: tierSuper, Scope: scopeWorld,
					WorldId: f.WorldId(), ChannelId: f.ChannelId(), CharacterId: s.CharacterId(),
					SenderName: c.Name(), SenderMedal: "",
					Messages: []string{combined}, WhispersOn: sp.Ear(),
				},
				CreatedAt: now, UpdatedAt: now,
			})
		}
		_ = saga.NewProcessor(l, ctx).Create(saga.Saga{
			TransactionId: uuid.New(), SagaType: saga.MegaphoneUse,
			InitiatedBy: "CASH_ITEM_USE", Steps: steps,
		})
	}
}
```

Avatar branch (same file). jms has no AVATAR_MEGAPHONE_RESULT op — the rejection announce is skipped for JMS (silent return, design D9):

```go
func handleAvatarMegaphoneUse(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}, t tenant.Model, itemId item.Id, source slot.Position, updateTimeFirst bool) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}, t tenant.Model, itemId item.Id, source slot.Position, updateTimeFirst bool) {
		sp := cashsb.NewItemUseAvatarMegaphone(updateTimeFirst)
		sp.Decode(l, ctx)(r, readerOptions)

		c, err := character2.NewProcessor(l, ctx).GetById()(s.CharacterId())
		if err != nil {
			l.WithError(err).Warnf("Character [%d] not found for avatar megaphone use.", s.CharacterId())
			return
		}
		f := s.Field()

		reject := func() {
			if t.Region() == "JMS" {
				return // no AVATAR_MEGAPHONE_RESULT op in jms (STATUS.md line 143)
			}
			_ = session.Announce(l)(ctx)(wp)(chatpkt.AvatarMegaphoneResultWriter)(chatpkt.NewAvatarMegaphoneResult(83, "").Encode)(s)
		}

		wait, err := worldbroadcast.NewProcessor(l, ctx).GetWaitSeconds(f.WorldId(), worldbroadcast.FamilyAvatar)
		if err != nil || wait > avatarWaitCapSeconds {
			if err != nil {
				l.WithError(err).Warnf("Unable to check avatar queue for world [%d]; rejecting without consuming.", f.WorldId())
			}
			reject()
			return
		}

		now := time.Now()
		_ = saga.NewProcessor(l, ctx).Create(saga.Saga{
			TransactionId: uuid.New(), SagaType: saga.MegaphoneUse, InitiatedBy: "CASH_ITEM_USE",
			Steps: []saga.Step{
				{StepId: "consume_avatar_megaphone", Status: saga.Pending, Action: saga.DestroyAsset,
					Payload:   saga.DestroyAssetPayload{CharacterId: s.CharacterId(), TemplateId: uint32(itemId), Quantity: 1, RemoveAll: false},
					CreatedAt: now, UpdatedAt: now},
				{StepId: "enqueue_avatar_broadcast", Status: saga.Pending, Action: saga.EnqueueWorldBroadcast,
					Payload: saga.EnqueueWorldBroadcastPayload{
						Family:  worldbroadcast.FamilyAvatar,
						WorldId: f.WorldId(), ChannelId: f.ChannelId(), CharacterId: s.CharacterId(),
						SenderName: c.Name(), SenderMedal: "",
						Messages: sp.Lines(), WhispersOn: sp.Whisper(),
						ItemId:   uint32(itemId), DurationSeconds: avatarDurationSecs,
						SenderLook: socketmodel.NewAvatarSnapshot(c),
					},
					CreatedAt: now, UpdatedAt: now},
			},
		})
	}
}
```

Adjustment notes for the implementer: (a) if `character2.Model`'s equipment/pets require decorators for `NewAvatarSnapshot`, fetch with `p := character2.NewProcessor(l, ctx); c, err := p.GetById(p.InventoryDecorator)(...)` — check how the messenger consumer fetches before calling `socketmodel.NewFromCharacter` and match it; (b) `inventory.Type(sp.InvType())` — confirm `inventory.Type`'s underlying type in `libs/atlas-constants/inventory` and cast accordingly; (c) decode errors from malformed packets surface as zero values — the slot-mismatch guard (FR-1.3) is the existing top-of-handler check plus the item-megaphone slot resolve.

- [ ] **Step 1: Write failing snapshot tests** — `snapshot_test.go`: `NewAssetSnapshot`→`NewAssetFromSnapshot` preserves every field for an equip-shaped, a stackable-shaped, and a pet-shaped `asset.Model` (build with the asset domain's Builder); `NewAvatarSnapshot`→`NewAvatarFromSnapshot` preserves gender/skin/face/hair/equips/masked/pets for a character built with the character test Builder.
- [ ] **Step 2: Run to verify failure** — `go test ./socket/model/ -run Snapshot -v` → FAIL.
- [ ] **Step 3: Implement** snapshot.go, then the handler branches and dispatch as above.
- [ ] **Step 4: Verify** — `cd services/atlas-channel/atlas.com/channel && go test -race ./... && go vet ./... && go build ./...` → clean.
- [ ] **Step 5: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/socket/
git commit -m "feat(task-123): USE_CASH_ITEM megaphone/TV/avatar handler branches with consume sagas"
```

---

### Task 13: atlas-channel — megaphone broadcast consumer

**Files:**
- Create: `services/atlas-channel/atlas.com/channel/kafka/consumer/megaphone/consumer.go`
- Modify: `services/atlas-channel/atlas.com/channel/socket/writer/world_message.go` (decorating wrapper funcs)

**Interfaces:**
- Consumes: Task 11 `megaphone.BroadcastEvent`; Task 4 body funcs; `decorateMegaphoneMessage` (`world_message.go:55`); gachapon consumer skeleton (`kafka/consumer/gachapon/consumer.go` — copy `InitConsumers`/`InitHandlers` including `kafka.LastOffset` + `TenantHeaderParser`).
- Produces: writer wrappers in `world_message.go`:

```go
func WorldMessageMegaphoneBody(medal string, characterName string, message string) packet.Encode {
	return chat.WorldMessageMegaphoneBody(decorateMegaphoneMessage(medal, characterName, message))
}

func WorldMessageSuperMegaphoneBody(medal string, characterName string, message string, channelId channel.Id, whispersOn bool) packet.Encode {
	return chat.WorldMessageSuperMegaphoneBody(decorateMegaphoneMessage(medal, characterName, message), byte(channelId), whispersOn)
}

func WorldMessageItemMegaphoneBody(medal string, characterName string, message string, channelId channel.Id, whispersOn bool, item *packetmodel.Asset) packet.Encode {
	return chat.WorldMessageItemMegaphoneBody(decorateMegaphoneMessage(medal, characterName, message), byte(channelId), whispersOn, item)
}

func WorldMessageMultiMegaphoneBody(medal string, characterName string, messages []string, channelId channel.Id, whispersOn bool) packet.Encode {
	decorated := make([]string, 0, len(messages))
	for _, m := range messages {
		decorated = append(decorated, decorateMegaphoneMessage(medal, characterName, m))
	}
	return chat.WorldMessageMultiMegaphoneBody(decorated, byte(channelId), whispersOn)
}
```

(import `chat "github.com/Chronicle20/atlas/libs/atlas-packet/chat"` and `packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"`; note `chat.…Body` returns the same shape as `packet.Encode` — assign directly, matching how `worldMessageBody` returns).

Consumer handler:

```go
func handleBroadcast(sc server.Model, wp writer.Producer) message.Handler[megaphone2.BroadcastEvent] {
	return func(l logrus.FieldLogger, ctx context.Context, e megaphone2.BroadcastEvent) {
		t := tenant.MustFromContext(ctx)
		// CHANNEL scope (basic megaphone): only the sender's channel renders.
		if e.Scope == scopeChannel {
			if !sc.Is(t, world.Id(e.WorldId), channel.Id(e.ChannelId)) {
				return
			}
		} else if !sc.IsWorld(t, world.Id(e.WorldId)) {
			return
		}

		var body packet.Encode
		switch e.Tier {
		case tierMegaphone:
			body = writer.WorldMessageMegaphoneBody(e.SenderMedal, e.SenderName, e.Messages[0])
		case tierSuper:
			body = writer.WorldMessageSuperMegaphoneBody(e.SenderMedal, e.SenderName, e.Messages[0], channel.Id(e.ChannelId), e.WhispersOn)
		case tierItem:
			var item *packetmodel.Asset
			if e.Item != nil {
				rebuilt := socketmodel.NewAssetFromSnapshot(*e.Item)
				item = &rebuilt
			}
			body = writer.WorldMessageItemMegaphoneBody(e.SenderMedal, e.SenderName, e.Messages[0], channel.Id(e.ChannelId), e.WhispersOn, item)
		case tierTriple:
			body = writer.WorldMessageMultiMegaphoneBody(e.SenderMedal, e.SenderName, e.Messages, channel.Id(e.ChannelId), e.WhispersOn)
		default:
			l.Warnf("Unhandled megaphone tier [%s].", e.Tier)
			return
		}

		l.WithFields(logrus.Fields{
			"character_id": e.CharacterId, "tier": e.Tier, "world_id": e.WorldId, "channel_id": e.ChannelId,
		}).Infof("Broadcasting megaphone message.")

		sessions, err := session.NewProcessor(l, ctx).AllInChannelProvider(sc.WorldId(), sc.ChannelId())
		if err != nil {
			l.WithError(err).Error("Unable to get sessions for megaphone broadcast.")
			return
		}
		announceOp := session.Announce(l)(ctx)(wp)(chatpkt.WorldMessageWriter)(body)
		for _, sess := range sessions {
			if err := announceOp(sess); err != nil {
				l.WithError(err).Warnf("Unable to send megaphone broadcast to session.")
			}
		}
	}
}
```

(`tierMegaphone` etc. — define the same const strings locally in the consumer package or export them from a shared spot; keep the values identical to Task 12's. Guard `e.Messages` non-empty before indexing — warn + return on empty.)

- [ ] **Step 1: Implement** the consumer package (`InitConsumers` name `"megaphone_broadcast"`, topic `megaphone2.EnvEventTopicMegaphone`, `kafka.LastOffset`) and writer wrappers.
- [ ] **Step 2: Verify** — `cd services/atlas-channel/atlas.com/channel && go build ./... && go vet ./...` → clean (consumer wiring is exercised at Task 15 build + live acceptance; there is no unit harness for channel consumers — match the repo: gachapon's consumer has no test file).
- [ ] **Step 3: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/kafka/consumer/megaphone/ services/atlas-channel/atlas.com/channel/socket/writer/world_message.go
git commit -m "feat(task-123): megaphone broadcast consumer rendering WorldMessage mode arms"
```

---

### Task 14: atlas-channel — world-broadcast status consumer (TV / avatar rendering)

**Files:**
- Create: `services/atlas-channel/atlas.com/channel/kafka/consumer/worldbroadcast/consumer.go`

**Interfaces:**
- Consumes: Task 11 `worldbroadcast.StatusEvent`; Task 5/6 packet structs; `socketmodel.NewAvatarFromSnapshot` (Task 12); `session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())` (single-session ack — precedent `kafka/consumer/buddylist/consumer.go:87`).
- Produces: handler behavior —

```go
func handleStatus(sc server.Model, wp writer.Producer) message.Handler[wb2.StatusEvent] {
	return func(l logrus.FieldLogger, ctx context.Context, e wb2.StatusEvent) {
		t := tenant.MustFromContext(ctx)
		if !sc.IsWorld(t, world.Id(e.WorldId)) {
			return
		}
		switch e.Type {
		case wb2.StatusTypeQueued:
			// success ack to the sender only (TV: ENABLE_TV 00; avatar: none — client shows nothing on queue)
			if e.Family == worldbroadcast.FamilyTV { // channel worldbroadcast pkg consts (Task 11)
				_ = session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(e.CharacterId,
					session.Announce(l)(ctx)(wp)(tvpkt.TvSendMessageResultWriter)(tvpkt.NewTvSendMessageResultSuccess().Encode))
			}
		case wb2.StatusTypeStarted:
			announceStarted(l, ctx, sc, wp, e)
		case wb2.StatusTypeEnded:
			announceEnded(l, ctx, sc, wp, e)
		default:
			l.Warnf("Unhandled world broadcast status [%s].", e.Type)
		}
	}
}
```

`announceStarted` builds the packet once and announces to all sessions in the pod's channel (`AllInChannelProvider`, gachapon loop):

- Family TV: `tvpkt.NewTvSetMessage(e.TvMessageType, socketmodel.NewAvatarFromSnapshot(e.SenderLook, false), decorateName(e.SenderMedal, e.SenderName), e.ReceiverName, [5]string{…from e.Messages, padded…}, e.TotalWaitSeconds, receiverLookOrNil)` — sender name decoration via the exported wrapper added in Task 13 (add `func DecorateNameForMessage(medal, name string) string { return decorateNameForMessage(medal, name) }` to `world_message.go` if needed, or export the helper — pick ONE and use it in both consumers).
- Family AVATAR: `chatpkt.NewSetAvatarMegaphone(e.ItemId, decoratedName, [4]string{…}, uint32(e.ChannelId), e.WhispersOn, socketmodel.NewAvatarFromSnapshot(e.SenderLook, true))`.

`announceEnded`: Family TV → `tvpkt.NewTvClearMessage()`; Family AVATAR → `chatpkt.NewClearAvatarMegaphone()` (idempotent client-side; belt-and-braces on the 10 s auto-clear, design D6). Announce to all sessions in channel.

Structured logs at info on STARTED/ENDED with tenant/world/family/characterId (NFR observability).

- [ ] **Step 1: Implement** the consumer (`InitConsumers` name `"world_broadcast_status"`, topic `wb2.EnvEventTopicWorldBroadcastStatus`, `kafka.LastOffset`, tenant headers).
- [ ] **Step 2: Verify** — `go build ./... && go vet ./...` → clean.
- [ ] **Step 3: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/kafka/consumer/worldbroadcast/ services/atlas-channel/atlas.com/channel/socket/writer/world_message.go
git commit -m "feat(task-123): world-broadcast status consumer rendering TV and avatar megaphone packets"
```

---

### Task 15: atlas-channel main wiring + deploy topic env vars

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/main.go`
- Modify: `deploy/k8s/base/env-configmap.yaml`
- Modify: `deploy/k8s/overlays/pr/kustomization.yaml`
- Modify: `deploy/k8s/overlays/main/kustomization.yaml`

**Interfaces:**
- Consumes: Tasks 13–14 consumer packages; Tasks 5–6 writer name consts.

Steps:

- [ ] **Step 1: Register consumers** — in `main.go` add `megaphoneConsumer.InitConsumers(l)(cmf)(consumerGroupId)` and `worldbroadcastConsumer.InitConsumers(l)(cmf)(consumerGroupId)` to the block at lines 172–211, and the matching `InitHandlers` calls in the per-server handler block (find where `gachapon.InitHandlers` is invoked — same spot, same curried args `(sc)(wp)(rf)`).
- [ ] **Step 2: Register writers** — append to `produceWriters()` (`main.go:608`):

```go
		chatCB.SetAvatarMegaphoneWriter,
		chatCB.ClearAvatarMegaphoneWriter,
		chatCB.AvatarMegaphoneResultWriter,
		tvCB.TvSetMessageWriter,
		tvCB.TvClearMessageWriter,
		tvCB.TvSendMessageResultWriter,
```

(new import `tvCB "github.com/Chronicle20/atlas/libs/atlas-packet/tv/clientbound"`).

- [ ] **Step 3: Topic env vars** — add three entries to each of the three deploy files, alphabetically placed, exactly mirroring the `EVENT_TOPIC_GACHAPON_REWARD_WON` pattern (`env-configmap.yaml:110`, `overlays/pr/kustomization.yaml:201`, `overlays/main/kustomization.yaml:144`):
  - `COMMAND_TOPIC_WORLD_BROADCAST`
  - `EVENT_TOPIC_MEGAPHONE`
  - `EVENT_TOPIC_WORLD_BROADCAST_STATUS`
- [ ] **Step 4: Verify** — `cd services/atlas-channel/atlas.com/channel && go build ./... && go vet ./... && go test -race ./...`; then `kubectl kustomize deploy/k8s/overlays/pr > /dev/null && kubectl kustomize deploy/k8s/overlays/main > /dev/null` → all clean.
- [ ] **Step 5: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/main.go deploy/k8s/
git commit -m "feat(task-123): wire megaphone/world-broadcast consumers, writers and topic env vars"
```

---

### Task 16: Seed templates — writers, USE_CASH_ITEM handler entries (5 versions)

**Files:**
- Modify: `services/atlas-configurations/seed-data/templates/template_gms_83_1.json`
- Modify: `services/atlas-configurations/seed-data/templates/template_gms_84_1.json`
- Modify: `services/atlas-configurations/seed-data/templates/template_gms_87_1.json`
- Modify: `services/atlas-configurations/seed-data/templates/template_gms_95_1.json`
- Modify: `services/atlas-configurations/seed-data/templates/template_jms_185_1.json`

(gms_12 and gms_92 untouched — design D9.)

**Interfaces:**
- Consumes: writer name consts from Tasks 5–6 (`SetAvatarMegaphone`, `ClearAvatarMegaphone`, `AvatarMegaphoneResult`, `TvSetMessage`, `TvClearMessage`, `TvSendMessageResult`); handler name `CharacterCashItemUseHandle`; opcode table from "Key protocol facts".

- [ ] **Step 1: Add the writer entries** to each template's `socket.writers` array (entry shape per the existing `WorldMessage` entry — `{"opCode": "0x…", "writer": "<Name>"}`, no options), with per-version opcodes from the pin table:

  - gms_83: `0x6F SetAvatarMegaphone`, `0x70 ClearAvatarMegaphone`, `0x6E AvatarMegaphoneResult`, `0x155 TvSetMessage`, `0x156 TvClearMessage`, `0x157 TvSendMessageResult`
  - gms_84: `0x72 / 0x73 / 0x71 / 0x15F / 0x160 / 0x161`
  - gms_87: `0x72 / 0x73 / 0x71 / 0x16A / 0x16B / 0x16C`
  - gms_95: `0x73 / 0x74 / 0x72 / 0x195 / 0x196 / 0x197`
  - jms_185: `0x5A SetAvatarMegaphone`, `0x5B ClearAvatarMegaphone`, **no AvatarMegaphoneResult**, `0x17A / 0x17B / 0x17C` TV trio

  Match each file's existing opCode hex formatting (e.g. `"0x6F"` vs `"0x06F"` — inspect neighbors and conform).

- [ ] **Step 2: Add the USE_CASH_ITEM handler entry** where missing (v87/v95/jms; gms_83/84 already have it at `0x4F`):

```json
{ "opCode": "0x52", "validator": "LoggedInValidator", "handler": "CharacterCashItemUseHandle" }
```

with opCode `0x52` (v87), `0x55` (v95), `0x47` (jms). Every entry has a validator (FR-6.2).

- [ ] **Step 3: Run the template symbol check** — `tools/template-symbol-check.sh` (verifies template writer/handler names resolve against the code). Then `cd services/atlas-configurations && go test -race ./... && go build ./...` (seed tests parse templates).
- [ ] **Step 4: Commit**

```bash
git add services/atlas-configurations/seed-data/templates/
git commit -m "feat(task-123): seed avatar-megaphone and Maple TV writers + USE_CASH_ITEM handlers (v83-v95, jms)"
```

*(WorldMessage `operations` table corrections, if the Task 18 IDA pass finds per-version drift, are committed in Task 18 — not here.)*

---

### Task 17: Live-tenant rollout runbook

**Files:**
- Create: `docs/tasks/task-123-megaphones-maple-tv/rollout.md`

Seed templates apply only at tenant creation; existing tenants must be PATCHed and atlas-channel restarted (projection does not hot-reload handlers/writers) — FR-6.3.

- [ ] **Step 1: Write `rollout.md`** containing, concretely:
  1. The full list of writer/handler/operations deltas per version (copy the Task 16 + Task 18 tables — the runbook is version-keyed).
  2. The PATCH procedure for each live tenant's socket configuration via the atlas-configurations REST API (cite the exact endpoint and JSON:API envelope used by existing tenant-config PATCH flows — locate with `grep -rn "configurations" services/atlas-configurations/atlas.com/configurations/*/resource.go` and reference the UI's known envelope requirement: bare bodies 400).
  3. `kubectl rollout restart deployment/atlas-channel` (and note new pods must go Ready before testing).
  4. Deploy-order note: atlas-world and atlas-saga-orchestrator must be deployed **before or with** atlas-channel (channel REST-checks the world queue and creates sagas with new actions); topic env vars (Task 15) must exist in the environment before any pod restart.
  5. Pitfall callouts: new-opcode-silently-dropped (memory: patch live config + restart), missing-validator-silently-dropped-handler.
- [ ] **Step 2: Commit**

```bash
git add docs/tasks/task-123-megaphones-maple-tv/rollout.md
git commit -m "docs(task-123): live-tenant rollout runbook"
```

---

### Task 18: WorldMessage dispatcher family enrollment (IDA: all versions)

> **Environment prerequisite:** the IDA host must have v83 (dump), v84, v87, v95 and jms IDBs open. As of design time only v95 + the v83 dump were loaded — (re)open the others and `select_instance(port)` per version; always `list_instances` and match the binary NAME first. This prerequisite gates Tasks 18–20.

**Files:**
- Create: `docs/packets/dispatchers/worldmessage.yaml`
- Modify: `tools/packet-audit/cmd/run.go` (`candidatesFromFName` `#`-entries)
- Modify: `services/atlas-configurations/seed-data/templates/template_gms_{84,87,95}_1.json`, `template_jms_185_1.json` (operations tables — only if IDA finds drift from the v83-derived copies)
- Modify: fname docs as `fname-doc --check` requires

**Interfaces:**
- Consumes: `docs/packets/DISPATCHER_FAMILY.md` (the canonical recipe — follow it to the letter), `docs/packets/dispatchers/field_effect.yaml` (yaml shape), Task 3/4 structs and body functions.
- Produces: `operations --check` enforcement of the five templates' WorldMessage tables.

- [ ] **Step 1: Derive per-version mode tables from IDA.** For each of gms_v83, gms_v84, gms_v87, gms_v95, jms_v185: decompile `CWvsContext::OnBroadcastMsg` (locate via `func_query` with `name_regex`) and record the switch case → mode mapping for ALL modes present in that version. The v83 seeded table (NOTICE 0 … UNKNOWN_8 18, `template_gms_83_1.json` WorldMessage options) is IDA-verified already (design §1.4); the other four are unverified v83-derived copies — this step confirms or corrects them (design risk 2; the modes may shift per version like the opcode table — bug memory `operations mode tables missing/wrong` family).
- [ ] **Step 2: Author `docs/packets/dispatchers/worldmessage.yaml`** in the `field_effect.yaml` format:

```yaml
# WorldMessage — CWvsContext::OnBroadcastMsg per-version mode table.
# Derived from each version's OnBroadcastMsg switch (IDA), task-123.
writer: WorldMessage
fname: CWvsContext::OnBroadcastMsg
op: SERVERMESSAGE
direction: clientbound
operations:
  - { key: NOTICE,          modes: { gms_v83: 0, gms_v84: <ida>, gms_v87: <ida>, gms_v95: <ida>, jms_v185: <ida> } }
  # … one row per key present in the version; omit a version key where the
  # mode does not exist in that client (e.g. UNKNOWN_8 pre-v95) — follow
  # whatever absent-mode convention `operations --check` accepts (check its
  # handling in tools/packet-audit before authoring).
```

Fill every `<ida>` from Step 1 — no copied values.

- [ ] **Step 3: Correct template operations tables** for any version where Step 1 contradicts the seeded copy; run `go run ./tools/packet-audit operations --check` until exit 0.
- [ ] **Step 4: Add `#`-entries** in `candidatesFromFName` (model: the `CField::OnFieldEffect#…` block at `tools/packet-audit/cmd/run.go:2051-2072`), citing decompile lines:

```go
	case "CWvsContext::OnBroadcastMsg#Megaphone":
		return []candidate{{name: "WorldMessageMegaphone", pkg: "chat", dir: csvpkg.DirClientbound}}
	case "CWvsContext::OnBroadcastMsg#SuperMegaphone":
		return []candidate{{name: "WorldMessageSuperMegaphone", pkg: "chat", dir: csvpkg.DirClientbound}}
	case "CWvsContext::OnBroadcastMsg#ItemMegaphone":
		return []candidate{{name: "WorldMessageItemMegaphone", pkg: "chat", dir: csvpkg.DirClientbound}}
	case "CWvsContext::OnBroadcastMsg#MultiMegaphone":
		return []candidate{{name: "WorldMessageMultiMegaphone", pkg: "chat", dir: csvpkg.DirClientbound}}
```

Only these four arms enroll (design D5 — the op row stays 🧩/❌ until every arm verifies; the other 15 modes are NOT enrolled in this task and their structs stay outside the family lint surface). Do NOT add worldmessage to `families.yaml` (FIELD_EFFECT model) or to the lint baseline.

- [ ] **Step 5: dispatcher-lint** — `go run ./tools/packet-audit dispatcher-lint` → exit 0. Expected friction: INV-5 requires every enrolled struct constructed by a body function (Task 4 provides all four); INV-2/INV-3 must hold for `world_message_body.go`.
- [ ] **Step 6: Commit**

```bash
git add docs/packets/dispatchers/worldmessage.yaml tools/packet-audit/ services/atlas-configurations/seed-data/templates/ docs/packets/
git commit -m "feat(task-123): enroll WorldMessage as dispatcher family with IDA-derived per-version mode tables"
```

---

### Task 19: Packet verification — gms_v83 and gms_v95 (open IDBs)

Follow `docs/packets/audits/VERIFYING_A_PACKET.md` **exactly** for every cell — this task summarizes scope, not procedure. Dispatch pattern: one `packet-verifier`-style pass per packet × version, batched per IDB. Serverbound cells additionally need marker+evidence+REPORT via the root `-ida-source` (memory: packet-audit serverbound verification; export splicing is surgical, never overwrite).

**Scope (× gms_v83, gms_v95):**

1. **Serverbound** `USE_CASH_ITEM` sub-bodies (Task 1 structs) against `CWvsContext::SendConsumeCashItemUseRequest` + `CItemSpeakerDlg::_SendConsumeCashItemUseRequest` (item megaphone) — confirm each Cosmic-derived shape per version; **resolve the trailing-updateTime question for v83 definitively** and adjust `updateTimeFirst` handling if the client contradicts it.
2. **Clientbound WorldMessage arms** (4): `WorldMessageMegaphone`, `WorldMessageSuperMegaphone`, `WorldMessageItemMegaphone` (incl. the item-block read — confirm presence/absence of a slotPos byte and exact `GW_ItemSlotBase` framing), `WorldMessageMultiMegaphone` (**reconcile Cosmic's trailing `channel×10 + ear + 1` against the current struct** — design risk 4; fix the struct if IDA says so).
3. **Avatar family** (3 structs) against `OnSetAvatarMegaphone`/`OnClearAvatarMegaphone`/`OnAvatarMegaphoneRes` (v83 addresses in design §1.2).
4. **TV family** (3 structs) against `OnSetMessage`/`OnClearMessage`/`OnSendMessageResult` (v95 `0x60f870/0x60f5f0`, v83 `0x6371c1/0x6373a0`).

For each verified cell: byte-fixture test with `// packet-audit:verify packet=<pkg>/<dir>/<Struct> version=<ver> ida=<addr>` marker (shape: `libs/atlas-packet/chat/clientbound/general_test.go:9-11`), evidence record pinned, audit report, matrix regenerated.

- [ ] **Step 1:** Verify all serverbound cells for v83, then v95. If a struct's wire shape changes, update Task 1 code + tests in the same commit as the evidence.
- [ ] **Step 2:** Verify the 4 WorldMessage arms for v83, then v95 (per-mode synthetic `#` export entries per DISPATCHER_FAMILY.md §5).
- [ ] **Step 3:** Verify avatar + TV families for v83, then v95.
- [ ] **Step 4:** `go run ./tools/packet-audit matrix --output docs/packets/audits` (regenerate); `matrix --check`, `fname-doc --check`, `dispatcher-lint` → exit 0. `cd libs/atlas-packet && go test -race ./...` → PASS.
- [ ] **Step 5:** Commit per batch (fixtures + evidence + report together, per the verifier convention).

```bash
git add libs/atlas-packet/ docs/packets/
git commit -m "test(task-123): byte-fixture verification for megaphone/TV/avatar packets (gms_v83, gms_v95)"
```

---

### Task 20: Packet verification — gms_v84, gms_v87, jms_v185 + final matrix

> Prerequisite: open the v84, v87 and jms IDBs on the IDA host (Task 18 note). If an IDB genuinely cannot be produced, STOP and escalate (stop-and-ask case) — do not substitute another version's read order as "verified."

**Scope:** the same cell list as Task 19 for gms_v84 (v83 structure + v84 opcode table — structure rules from the task-083/085 findings), gms_v87, jms_v185 (jms: no `AvatarMegaphoneResult` cell — op absent → ⬜).

- [ ] **Step 1:** Verify serverbound sub-bodies per version (v84 expected ≡ v83 structurally — still fixture it, no "assumed" cells).
- [ ] **Step 2:** Verify the 4 WorldMessage arms + avatar + TV families per version.
- [ ] **Step 3:** Regenerate `docs/packets/audits/STATUS.md`; confirm the USE_CASH_ITEM serverbound row and all new clientbound rows reflect actual per-version state; all four `packet-audit` checks exit 0.
- [ ] **Step 4:** Commit.

```bash
git add libs/atlas-packet/ docs/packets/
git commit -m "test(task-123): byte-fixture verification for megaphone/TV/avatar packets (gms_v84, gms_v87, jms_v185)"
```

---

### Task 21: Final verification gates, service docs, acceptance

- [ ] **Step 1: Full local gates** (from the worktree root):

```bash
(cd libs/atlas-packet && go test -race ./... && go vet ./...)
(cd libs/atlas-saga && go test -race ./... && go vet ./...)
(cd services/atlas-channel/atlas.com/channel && go test -race ./... && go vet ./... && go build ./...)
(cd services/atlas-world/atlas.com/world && go test -race ./... && go vet ./... && go build ./...)
(cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator && go test -race ./... && go vet ./... && go build ./...)
(cd services/atlas-configurations && go test -race ./... && go vet ./... && go build ./...)
tools/redis-key-guard.sh
go run ./tools/packet-audit dispatcher-lint
go run ./tools/packet-audit matrix --check
go run ./tools/packet-audit fname-doc --check
go run ./tools/packet-audit operations --check
```

All must exit 0.

- [ ] **Step 2: Docker bakes** (mandatory — `go build` will not catch Dockerfile COPY gaps):

```bash
docker buildx bake atlas-channel
docker buildx bake atlas-world
docker buildx bake atlas-saga-orchestrator
docker buildx bake atlas-configurations
```

- [ ] **Step 3: Service docs** — update `services/atlas-channel/docs/` and `services/atlas-world/docs/` if their capability lists enumerate broadcasts/domains (check; if the docs don't enumerate at this granularity, skip with a note). FR-8.2.
- [ ] **Step 4: TODO sweep** — `grep -rn "TODO" services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go libs/atlas-packet/cash libs/atlas-packet/tv libs/atlas-packet/chat services/atlas-world/atlas.com/world/broadcast` → no hits (the `:108` TODO is gone).
- [ ] **Step 5: Live acceptance on a v83 tenant** (PRD §10) after deploy via the rollout runbook: each tier broadcasts correctly, item consumed exactly once; TV queueing across ≥2 channels; item-megaphone empty-slot rejection; over-cap rejection packets. Record observations in the task folder.
- [ ] **Step 6: Commit any doc updates; run code review** (`superpowers:requesting-code-review`) before opening the PR.

---

## Task dependency order

```
1 (sb codecs)         ─┐
2 (saga lib)          ─┼─ independent starts
3 (worldmessage cuts) ─┤  (3 → 4)
5 (avatar cb)         ─┤
6 (tv cb)             ─┘
7 → 8 → 9 (atlas-world coordinator; needs 2)
10 (orchestrator; needs 2)
11 (channel messages/client; needs 2)
12 (handler; needs 1, 2, 5, 6, 11)
13 (megaphone consumer; needs 3, 4, 11, 12-snapshots)
14 (status consumer; needs 5, 6, 11, 12-snapshots)
15 (wiring/deploy; needs 13, 14)
16 (templates; needs 5, 6)
17 (runbook; needs 15, 16)
18 (dispatcher enrollment; needs 3, 4; IDA)
19 (verify v83/v95; needs 18; IDA)
20 (verify v84/v87/jms; needs 19; IDA)
21 (final gates; needs all)
```

## Self-review notes (spec coverage)

- FR-1 decode branches → Tasks 1, 12 (1.3 reject-without-consume: top-of-handler guard + item-slot resolve; 1.4 snapshot → Task 12 `NewAssetSnapshot`).
- FR-2 saga + fan-out → Tasks 2, 10, 12, 13 (2.3 reach rules: `Scope` + consumer filters; 2.4 medal decoration: Task 13 wrappers).
- FR-3 WorldMessage arms → Tasks 3, 4, 18, 19–20 (3.3 dispatcher policy: Task 18).
- FR-4 avatar family → Tasks 5, 12, 14 (4.2 look provider: `NewFromCharacter` reuse via snapshots; 4.3 clear timing: resolved client-side, ENDED redundancy in Task 14).
- FR-5 Maple TV → Tasks 6, 7–9, 12, 14 (5.3 queue owner: atlas-world coordinator; 5.5 late joiners: none, D7).
- FR-6 config wiring → Tasks 15 (env), 16 (templates, validators), 17 (runbook), 18 (operations).
- FR-7 version scope → global constraints + Tasks 16, 19, 20 (gms_92/gms_12 excluded per design D9 — PRD deviation already flagged in design §9).
- FR-8 matrix + docs → Tasks 19–21.
- Design D3 wait-cap-before-consume → Task 12 REST guards; error table §6 → Task 12 (conservative reject) + Task 8 (CAS transitions).
- Megassenger super-mega side-broadcast: not in design's flows but Cosmic-verified behavior of tvType≥3 items (UseCashItemHandler case 5) — included in Task 12 and flagged here as a deliberate Cosmic-parity addition for reviewer attention.




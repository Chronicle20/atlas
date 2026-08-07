# Portable Chair Recovery Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement STATUS row 562 (`STATE_CHANGE_BY_PORTABLE_CHAIR_REQUEST`, an empty one-shot notification) across all five versions, and make portable-chair recovery server-authoritative by routing `HEAL_OVER_TIME` ticks through atlas-chairs, which validates sit state, sources amounts from item data, and rate-limits.

**Architecture:** Per `design.md` (IDA-verified): row 562 carries no data — it gets an empty-body codec, a thin logging handler, byte fixtures, evidence records, and seed-template routing. The real recovery amounts ride the already-implemented `HEAL_OVER_TIME` packet; atlas-channel's heal handler stops applying stats directly and emits a new `RECOVERY` command on `COMMAND_TOPIC_CHAIR`; atlas-chairs validates (sit state → item data → rate limit) and emits `CHANGE_HP`/`CHANGE_MP` on `COMMAND_TOPIC_CHARACTER`. atlas-data gains `recoveryMP` parsing. The gms_95 seed template's validator-less handler entries (which are silently dropped by `BuildHandlerMap`) get fixed.

**Tech Stack:** Go microservices (atlas-data, atlas-chairs, atlas-channel), libs/atlas-packet codecs, Kafka commands, Redis TenantRegistry, packet-audit tooling, IDA-MCP.

## Global Constraints

- Verification gates (CLAUDE.md): `go test -race ./...`, `go vet ./...`, `go build ./...` clean in every changed module; `docker buildx bake atlas-<svc>` for every touched service; `tools/redis-key-guard.sh` clean from repo root (no `GOWORK=off` prefix).
- No `// TODO`, stubbed handlers, or 501s in landed commits.
- Test setup uses the project's Builder/direct-construction patterns — no `*_testhelpers.go` files.
- Never write literal home/absolute paths into committed files.
- All registry/Kafka state tenant-scoped via `tenant.MustFromContext`.
- Packet ground truth is IDA only (design.md §2 already did this work — do not re-derive from other servers or memory).
- The IDA export files (`docs/packets/ida-exports/*.json`) are NON-idempotent — never regenerate/overwrite a committed export; surgically splice single entries (VERIFYING_A_PACKET.md §10).
- Client-wire values: the new rate-limit constant is server-internal policy, NOT a client-interpreted wire byte, so DOM-25 does not apply (design §7).
- Working directory for all commands: the task worktree root (`.worktrees/task-141-portable-chair-recovery`), except where a module dir is given.

## Scope note (deviation from design, surfaced during planning)

Design §2.4 flags ONE validator-less entry in `template_gms_95_1.json` (`0x64` `CharacterHealOverTimeHandle`). Source inspection during planning found **35** validator-less handler entries in that template (including `0x14 CharacterLoggedInHandle` — login itself would be dead on a freshly-seeded v95 tenant). The design's rationale ("missing validator = silently dropped handler"; §5.5 "Every entry carries an explicit validator") applies identically to all 35, and the fix is mechanical, so Task 4 fixes all of them: `CharacterLoggedInHandle`/`StartErrorHandle`/`PongHandle` → `NoOpValidator`, everything else → `LoggedInValidator` (the convention verified in template_gms_83_1.json and template_jms_185_1.json). gms_83/84/87/jms templates have zero missing validators (verified). The live-tenant runbook covers auditing live v95 config for the same gap.

## File Structure

| File | Change |
|---|---|
| `services/atlas-data/atlas.com/data/setup/rest.go` | + `RecoveryMP` field |
| `services/atlas-data/atlas.com/data/setup/reader.go` | + `recoveryMP` parse |
| `services/atlas-data/atlas.com/data/setup/reader_test.go`, `resource_test.go` | + coverage |
| `libs/atlas-packet/character/serverbound/state_change_by_portable_chair.go` | NEW empty-body codec |
| `libs/atlas-packet/character/serverbound/state_change_by_portable_chair_test.go` | NEW fixtures ×5 |
| `services/atlas-channel/atlas.com/channel/socket/handler/character_state_change_by_portable_chair.go` | NEW logging handler |
| `services/atlas-channel/atlas.com/channel/main.go` | + handler registration |
| `services/atlas-configurations/seed-data/templates/template_{gms_83,gms_84,gms_87,gms_95,jms_185}_1.json` | + 562 route ×5; v95 validator backfill |
| `tools/packet-audit/cmd/run.go` | + `candidatesFromFName` case |
| `docs/packets/ida-exports/{gms_v83,gms_v84,gms_v87,gms_v95,gms_jms_185}.json` | + spliced `CWvsContext::TryRecovery` entry |
| `docs/packets/audits/<version>/StateChangeByPortableChair.{json,md}` ×5 | NEW reports |
| `docs/packets/evidence/<version>/character.serverbound.StateChangeByPortableChair.yaml` ×5 | NEW evidence |
| `docs/packets/audits/STATUS.md`, `status.json` | regenerated |
| `services/atlas-chairs/atlas.com/chairs/chair/model.go` | + recovery timestamps |
| `services/atlas-chairs/atlas.com/chairs/chair/registry_test.go` | + marshal round-trip |
| `services/atlas-chairs/atlas.com/chairs/data/setup/{model,rest,requests,processor}.go` | NEW REST client |
| `services/atlas-chairs/atlas.com/chairs/kafka/message/chair/kafka.go` | + `RECOVERY` command |
| `services/atlas-chairs/atlas.com/chairs/kafka/message/character/kafka.go` | + character command contract |
| `services/atlas-chairs/atlas.com/chairs/chair/producer.go` | + CHANGE_HP/MP providers |
| `services/atlas-chairs/atlas.com/chairs/chair/processor.go`, `processor_test.go` | + Recover orchestration + tests |
| `services/atlas-chairs/atlas.com/chairs/kafka/consumer/chair/consumer.go` | + RECOVERY arm |
| `services/atlas-channel/atlas.com/channel/kafka/message/chair/kafka.go` | + `RECOVERY` command (channel copy) |
| `services/atlas-channel/atlas.com/channel/chair/{processor,producer}.go`, `producer_test.go` | + Recover emission |
| `services/atlas-channel/atlas.com/channel/socket/handler/character_heal_over_time.go` | reroute to chair.Recover |

Module roots (for `go test`/`go vet`/`go build`): `libs/atlas-packet`, `services/atlas-data/atlas.com/data`, `services/atlas-chairs/atlas.com/chairs`, `services/atlas-channel/atlas.com/channel`, `tools/packet-audit`.

---

### Task 1: atlas-data — parse and expose `recoveryMP`

**Files:**
- Modify: `services/atlas-data/atlas.com/data/setup/rest.go`
- Modify: `services/atlas-data/atlas.com/data/setup/reader.go`
- Test: `services/atlas-data/atlas.com/data/setup/reader_test.go`
- Test: `services/atlas-data/atlas.com/data/setup/resource_test.go`

**Interfaces:**
- Consumes: existing `RestModel`/`Read` in the `setup` package.
- Produces: `RestModel.RecoveryMP uint32` serialized as JSON:API attribute `recoveryMP` on `GET /data/setups` and `/data/setups/{id}`. Task 7's atlas-chairs client consumes this attribute name.

- [ ] **Step 1: Write the failing reader tests**

In `reader_test.go`, inside `const testXML`, add one line to the `03010000` `info` imgdir (making it the both-stats chair, 03010136-style), directly after the existing `recoveryHP` line:

```xml
      <int name="recoveryHP" value="50"/>
      <int name="recoveryMP" value="60"/>
```

Append a fourth entry (the "neither" chair) before the closing `</imgdir>` of `0301.img`:

```xml
  <imgdir name="03010900">
    <imgdir name="info">
      <int name="price" value="300"/>
      <int name="slotMax" value="1"/>
    </imgdir>
  </imgdir>
```

In `TestReader`, change the count assertion:

```go
	if len(res) != 4 {
		t.Fatalf("Expected 4 setup items, got %d", len(res))
	}
```

Add assertions (after the existing `RecoveryHP` checks for each setup):

```go
	// setup1 (03010000): both stats
	if setup1.RecoveryMP != 60 {
		t.Fatalf("Expected recoveryMP 60, got %d", setup1.RecoveryMP)
	}
	// setup2 (03010001): HP-only chair
	if setup2.RecoveryMP != 0 {
		t.Fatalf("Expected recoveryMP 0, got %d", setup2.RecoveryMP)
	}
	// setup4 (03010900): neither stat
	setup4 := res[3]
	if setup4.Id != 3010900 {
		t.Fatalf("Expected ID 3010900, got %d", setup4.Id)
	}
	if setup4.RecoveryHP != 0 {
		t.Fatalf("Expected recoveryHP 0, got %d", setup4.RecoveryHP)
	}
	if setup4.RecoveryMP != 0 {
		t.Fatalf("Expected recoveryMP 0, got %d", setup4.RecoveryMP)
	}
```

In `resource_test.go`: in `setupTestSetupData`, add `RecoveryMP: 60,` to the `Id: 3010000` entry; in the `GetSetup` assertions next to the existing `recoveryHP` check, add:

```go
		assert.Equal(t, float64(60), attributes["recoveryMP"])
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd services/atlas-data/atlas.com/data && go test ./setup/...`
Expected: FAIL — `setup1.RecoveryMP` / `RecoveryMP` undefined (compile error).

- [ ] **Step 3: Implement**

`rest.go` — add the field directly after `RecoveryHP`:

```go
	RecoveryHP     uint32 `json:"recoveryHP"`
	RecoveryMP     uint32 `json:"recoveryMP"`
```

`reader.go` — add directly after the `RecoveryHP` parse line:

```go
			m.RecoveryMP = uint32(i.GetIntegerWithDefault("recoveryMP", 0))
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd services/atlas-data/atlas.com/data && go test ./setup/... && go vet ./... && go build ./...`
Expected: PASS, vet/build clean.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-data/atlas.com/data/setup/
git commit -m "feat(data): parse and expose setup recoveryMP (task-141 FR-3)"
```

---

### Task 2: libs/atlas-packet — `StateChangeByPortableChair` empty-body codec + fixtures

**Files:**
- Create: `libs/atlas-packet/character/serverbound/state_change_by_portable_chair.go`
- Create: `libs/atlas-packet/character/serverbound/state_change_by_portable_chair_test.go`

**Interfaces:**
- Produces: `const CharacterStateChangeByPortableChairHandle = "CharacterStateChangeByPortableChairHandle"` and `type StateChangeByPortableChair struct{}` with `Operation() string`, `String() string`, `Encode(l, ctx) func(options map[string]interface{}) []byte`, `Decode(l, ctx) func(r *request.Reader, options map[string]interface{})`. Tasks 3 (handler) and 5 (audit linkage) consume the const/struct names exactly as written here.
- No version branching: the body is empty in all five versions (design §2.1).

- [ ] **Step 1: Write the failing fixture test**

`state_change_by_portable_chair_test.go` (markers use the per-version `CWvsContext::TryRecovery` addresses from design §2.1 — these must match the export entries Task 5 splices):

```go
package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=character/serverbound/StateChangeByPortableChair version=gms_v83 ida=0xa02e34
// packet-audit:verify packet=character/serverbound/StateChangeByPortableChair version=gms_v84 ida=0xa4d05a
// packet-audit:verify packet=character/serverbound/StateChangeByPortableChair version=gms_v87 ida=0xa97e50
// packet-audit:verify packet=character/serverbound/StateChangeByPortableChair version=gms_v95 ida=0x9d4020
// packet-audit:verify packet=character/serverbound/StateChangeByPortableChair version=jms_v185 ida=0xae6f5a
func TestStateChangeByPortableChairRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := StateChangeByPortableChair{}

			// The body is empty in every version: encode must emit zero bytes.
			b := pt.Encode(t, ctx, input.Encode, nil)
			if len(b) != 0 {
				t.Errorf("body: got %d bytes, want 0", len(b))
			}

			// Decode must consume nothing (RoundTrip asserts 0 unconsumed).
			output := StateChangeByPortableChair{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd libs/atlas-packet && go test ./character/serverbound/ -run TestStateChangeByPortableChairRoundTrip -v`
Expected: FAIL — `undefined: StateChangeByPortableChair` (compile error).

- [ ] **Step 3: Implement the codec**

`state_change_by_portable_chair.go`:

```go
package serverbound

import (
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const CharacterStateChangeByPortableChairHandle = "CharacterStateChangeByPortableChairHandle"

// StateChangeByPortableChair - CWvsContext::TryRecovery
// (STATE_CHANGE_BY_PORTABLE_CHAIR_REQUEST, STATUS row 562).
//
// The body is EMPTY in every supported version: the client constructs
// COutPacket(ctor, opcode) and calls CClientSocket::SendPacket with zero
// Encode calls in between (IDA-verified, task-141 design §2.1):
//
//	gms_v83  CWvsContext::TryRecovery @ 0xa02e34, send site 0xa032ad, opcode 0x4A
//	gms_v84  sub_A4D05A               @ 0xa4d05a (structurally identical), opcode 0x4A
//	gms_v87  CWvsContext::TryRecovery @ 0xa97e50, opcode 0x4D
//	gms_v95  CWvsContext::TryRecovery @ 0x9d4020, opcode 0x50
//	jms_v185 CWvsContext::TryRecovery @ 0xae6f5a, opcode 0x42
//
// Send gate (identical semantics in all five): CanSendExclRequest(500, 0)
// passes, an active portable chair id is set, time since sitting >= 20000 ms,
// and a per-sit latch is unset — so the packet fires AT MOST ONCE PER SIT,
// and only for portable chairs whose item data has no `spec` node. No
// clientbound response exists; the client latches locally. Chair recovery
// amounts do NOT ride this packet — they ride HEAL_OVER_TIME (row 577).
type StateChangeByPortableChair struct {
}

func (m StateChangeByPortableChair) Operation() string {
	return CharacterStateChangeByPortableChairHandle
}

func (m StateChangeByPortableChair) String() string {
	return "state change by portable chair (empty body)"
}

func (m StateChangeByPortableChair) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		return w.Bytes()
	}
}

func (m *StateChangeByPortableChair) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd libs/atlas-packet && go test -race ./character/serverbound/ && go vet ./... && go build ./...`
Expected: PASS. (`version_bounds_test.go` tests only specific codecs by hand — no enumeration entry needed.)

- [ ] **Step 5: Commit**

```bash
git add libs/atlas-packet/character/serverbound/state_change_by_portable_chair.go libs/atlas-packet/character/serverbound/state_change_by_portable_chair_test.go
git commit -m "feat(packet): StateChangeByPortableChair empty-body serverbound codec (task-141 row 562)"
```

---

### Task 3: atlas-channel — 562 logging handler + registration

**Files:**
- Create: `services/atlas-channel/atlas.com/channel/socket/handler/character_state_change_by_portable_chair.go`
- Modify: `services/atlas-channel/atlas.com/channel/main.go` (handler map, next to the `CharacterChairPortableHandle` line at ~line 853)

**Interfaces:**
- Consumes: `charsb.CharacterStateChangeByPortableChairHandle`, `charsb.StateChangeByPortableChair` from Task 2.
- Produces: `handler.CharacterStateChangeByPortableChairHandleFunc` registered under the handle constant. The seed-template entries in Task 4 reference the handle string `CharacterStateChangeByPortableChairHandle`.
- Per design §4: decode + debug log only — no Kafka emission, no validation (the packet grants nothing; this is its faithful, complete implementation, not a stub).

- [ ] **Step 1: Write the handler**

`character_state_change_by_portable_chair.go`:

```go
package handler

import (
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	character2 "github.com/Chronicle20/atlas/libs/atlas-packet/character/serverbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

func CharacterStateChangeByPortableChairHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := character2.StateChangeByPortableChair{}
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s] for character [%d].", p.Operation(), p.String(), s.CharacterId())
	}
}
```

- [ ] **Step 2: Register in main.go**

Directly after the line `handlerMap[charsb.CharacterChairPortableHandle] = handler.CharacterChairPortableHandleFunc` add:

```go
	handlerMap[charsb.CharacterStateChangeByPortableChairHandle] = handler.CharacterStateChangeByPortableChairHandleFunc
```

- [ ] **Step 3: Build**

Run: `cd services/atlas-channel/atlas.com/channel && go build ./... && go vet ./...`
Expected: clean. (Behavioral coverage: the empty-body decode is exercised by Task 2's round-trip fixtures; the handler adds only a log line.)

- [ ] **Step 4: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/socket/handler/character_state_change_by_portable_chair.go services/atlas-channel/atlas.com/channel/main.go
git commit -m "feat(channel): handle STATE_CHANGE_BY_PORTABLE_CHAIR_REQUEST as one-shot notification (task-141)"
```

---

### Task 4: Seed templates — route 562 in all five versions + v95 validator backfill

**Files:**
- Modify: `services/atlas-configurations/seed-data/templates/template_gms_83_1.json`
- Modify: `services/atlas-configurations/seed-data/templates/template_gms_84_1.json`
- Modify: `services/atlas-configurations/seed-data/templates/template_gms_87_1.json`
- Modify: `services/atlas-configurations/seed-data/templates/template_gms_95_1.json`
- Modify: `services/atlas-configurations/seed-data/templates/template_jms_185_1.json`

**Interfaces:**
- Consumes: handler string `CharacterStateChangeByPortableChairHandle` (Task 2/3).
- Produces: routed opcodes Task 5's matrix promotion requires (a serverbound cell must be routed in the version's template). Opcodes per design §2.1/STATUS row 562: v83 `0x4A`, v84 `0x4A`, v87 `0x4D`, v95 `0x50`, jms `0x42`.

- [ ] **Step 1: Add the 562 handler entry to each template**

In each template's `socket.handlers` array, insert directly after the `CharacterHealOverTimeHandle` entry (JSON formatting matches neighbors — 6-space indent for keys):

template_gms_83_1.json:
```json
      {
        "opCode": "0x4A",
        "validator": "LoggedInValidator",
        "handler": "CharacterStateChangeByPortableChairHandle"
      },
```
template_gms_84_1.json: same entry, `"opCode": "0x4A"`.
template_gms_87_1.json: same entry, `"opCode": "0x4D"`.
template_gms_95_1.json: same entry, `"opCode": "0x50"`.
template_jms_185_1.json: same entry, `"opCode": "0x42"`.

Before inserting, confirm no existing entry already claims that opCode in the same template (`grep '"0x4A"' services/atlas-configurations/seed-data/templates/template_gms_83_1.json` etc. — a duplicate opCode is a STOP-and-report, not a silent overwrite).

- [ ] **Step 2: Backfill the 35 missing v95 validators**

Run this script (it inserts a `validator` line into each handler entry that lacks one, preserving the file's existing formatting; only `template_gms_95_1.json` has gaps):

```bash
python3 - <<'EOF'
import re

path = 'services/atlas-configurations/seed-data/templates/template_gms_95_1.json'
noop = {'CharacterLoggedInHandle', 'StartErrorHandle', 'PongHandle'}
with open(path) as f:
    lines = f.readlines()

out = []
for i, line in enumerate(lines):
    out.append(line)
    m = re.match(r'^(\s*)"opCode": "(0x[0-9A-Fa-f]+)",\s*$', line)
    if m and i + 1 < len(lines):
        nxt = lines[i + 1]
        hm = re.match(r'^\s*"handler": "([A-Za-z]+)"', nxt)
        if hm:  # opCode line directly followed by handler line == no validator
            v = 'NoOpValidator' if hm.group(1) in noop else 'LoggedInValidator'
            out.append(f'{m.group(1)}"validator": "{v}",\n')

with open(path, 'w') as f:
    f.writelines(out)
EOF
```

- [ ] **Step 3: Verify**

```bash
python3 - <<'EOF'
import json
for name in ['gms_83_1', 'gms_84_1', 'gms_87_1', 'gms_95_1', 'jms_185_1']:
    d = json.load(open(f'services/atlas-configurations/seed-data/templates/template_{name}.json'))
    hs = d['socket']['handlers']
    missing = [h for h in hs if not h.get('validator')]
    sc = [h for h in hs if h['handler'] == 'CharacterStateChangeByPortableChairHandle']
    assert not missing, (name, missing)
    assert len(sc) == 1 and sc[0]['validator'] == 'LoggedInValidator', (name, sc)
    print(name, 'OK', sc[0]['opCode'])
EOF
```

Expected output: `gms_83_1 OK 0x4A`, `gms_84_1 OK 0x4A`, `gms_87_1 OK 0x4D`, `gms_95_1 OK 0x50`, `jms_185_1 OK 0x42`. Then `git diff --stat` — the v95 file should show ~40 insertions (35 validator lines + the 5-line new entry) and no deletions; the other four ~5 insertions each. Confirm the diff contains ONLY added lines (`git diff services/atlas-configurations/ | grep -c '^-[^-]'` → 0).

- [ ] **Step 4: Commit**

```bash
git add services/atlas-configurations/seed-data/templates/
git commit -m "feat(config): route STATE_CHANGE_BY_PORTABLE_CHAIR in all five seed templates; fix 35 validator-less gms_95 handler entries (task-141)"
```

---

### Task 5: packet-audit artifacts — linkage, export splices, reports, evidence, matrix ✅×5

**Files:**
- Modify: `tools/packet-audit/cmd/run.go` (`candidatesFromFName`)
- Modify: `docs/packets/ida-exports/{gms_v83,gms_v84,gms_v87,gms_v95,gms_jms_185}.json` (surgical splice only)
- Create: `docs/packets/audits/{gms_v83,gms_v84,gms_v87,gms_v95,jms_v185}/StateChangeByPortableChair.{json,md}`
- Create: `docs/packets/evidence/{gms_v83,gms_v84,gms_v87,gms_v95,jms_v185}/character.serverbound.StateChangeByPortableChair.yaml`
- Regenerate: `docs/packets/audits/STATUS.md`, `docs/packets/audits/status.json`

**Interfaces:**
- Consumes: Task 2's codec/marker names and Task 4's template routing (a serverbound cell promotes only when marker + evidence + report agree AND the op is routed).
- Produces: STATUS row 562 ✅ ×5.
- Requires: live IDA-MCP instances (design §2.1 ports — v83: 13342, v84: 13345, v87: 13343, v95: 13341, jms: 13344). If an instance is down, this task blocks; report rather than faking artifacts. Follow `docs/packets/audits/VERIFYING_A_PACKET.md` §§6–10 throughout.

- [ ] **Step 1: Add the fname→codec linkage case**

In `tools/packet-audit/cmd/run.go`, in `candidatesFromFName`, directly after the `case "CWvsContext::SendSitOnPortableChairRequest":` block add:

```go
	case "CWvsContext::TryRecovery":
		// Struct is StateChangeByPortableChair; handler constant =
		// "CharacterStateChangeByPortableChairHandle".
		// Empty body in all five versions: COutPacket(ctor, opcode) ->
		// SendPacket with zero Encode calls (once per sit, >=20 s, portable
		// chair without a spec node). task-141.
		return []candidate{{name: "StateChangeByPortableChair", dir: csvpkg.DirServerbound}}
```

Run: `cd tools/packet-audit && go build ./... && go test ./...`
Expected: clean.

- [ ] **Step 2: Name the v84 sender in its IDB**

The v83/v87/v95/jms IDBs already name `CWvsContext::TryRecovery`; v84's is unnamed (`sub_A4D05A` @ `0xa4d05a`, design §2.1). Via ida-pro MCP: `select_instance(13345)`, confirm the binary is `GMS_v84.1_U_DEVM`, decompile `0xa4d05a` and confirm it structurally matches v83's TryRecovery (empty COutPacket 0x4A send, 20 s gate), then `rename` it to `CWvsContext::TryRecovery` and `idb_save`. Do NOT skip the confirm-decompile step — never rename on address alone.

- [ ] **Step 3: Targeted harvest + surgical splice, one version at a time**

For each version (v83→13342, v84→13345, v87→13343, v95→13341, jms→13344; export keys `gms_v83`, `gms_v84`, `gms_v87`, `gms_v95`, `gms_jms_185`):

```bash
mkdir -p /tmp/task-141
echo 'CWvsContext::TryRecovery' > /tmp/task-141/roster.md
go run ./tools/packet-audit export \
  --version gms_v83 --ida-port 13342 \
  --descent-depth 12 --prior-export "" --pending /tmp/task-141/roster.md \
  --output /tmp/task-141/harvest_gms_v83.json
```

Then splice ONLY the `CWvsContext::TryRecovery` entry into the committed export:

```bash
python3 - <<'EOF'
import json, sys

version = 'gms_v83'  # repeat per version
committed = f'docs/packets/ida-exports/{version}.json'
harvest = f'/tmp/task-141/harvest_{version}.json'

with open(committed) as f:
    dst = json.load(f)
with open(harvest) as f:
    src = json.load(f)

entry = src['functions']['CWvsContext::TryRecovery']
# Strip the COutPacket-delegate harvest artifact if present (VERIFYING_A_PACKET.md §10).
if entry.get('calls'):
    entry['calls'] = [c for c in entry['calls'] if c.get('ref') != 'COutPacket'] or None
entry['direction'] = 'serverbound'
fns = dst['functions']
fns['CWvsContext::TryRecovery'] = entry
dst['functions'] = {k: fns[k] for k in sorted(fns)}  # exports keep sorted keys

with open(committed, 'w') as f:
    json.dump(dst, f, indent=2, ensure_ascii=False)
    f.write('\n')
EOF
```

After EACH splice, verify surgical scope: `git diff --stat docs/packets/ida-exports/<version>.json` must show only the new entry's lines (~10–15 insertions, zero deletions elsewhere). If the diff shows whole-file churn (formatting drift), `git checkout -- <file>` and splice the entry by hand with the Edit tool instead. Confirm the spliced `address` equals the marker address from Task 2 (v83 `0xa02e34`, v84 `0xa4d05a`, v87 `0xa97e50`, v95 `0x9d4020`, jms `0xae6f5a`) — a mismatch means the wrong function was harvested: stop and re-check, do not adjust the marker to match.

- [ ] **Step 4: Generate the five audit reports**

Per version (template names: `template_gms_83_1.json`, `template_gms_84_1.json`, `template_gms_87_1.json`, `template_gms_95_1.json`, `template_jms_185_1.json`; audit dirs: `gms_v83`, `gms_v84`, `gms_v87`, `gms_v95`, `jms_v185`):

```bash
go run ./tools/packet-audit \
  -csv-clientbound "docs/packets/MapleStory Ops - ClientBound.csv" \
  -csv-serverbound "docs/packets/MapleStory Ops - ServerBound.csv" \
  -template services/atlas-configurations/seed-data/templates/template_gms_83_1.json \
  -ida-source docs/packets/ida-exports/gms_v83.json \
  -output /tmp/task-141/rpt
cp /tmp/task-141/rpt/gms_v83/StateChangeByPortableChair.json \
   /tmp/task-141/rpt/gms_v83/StateChangeByPortableChair.md \
   docs/packets/audits/gms_v83/
```

Copy ONLY the StateChangeByPortableChair report — never bulk-copy the temp output over the audits dir. Caution for jms: the export/version key is `gms_jms_185` but the committed audits dir is `jms_v185` (known naming mismatch, see project memory on the packet-audit jms dir) — check the actual subdirectory name the tool wrote under `/tmp/task-141/rpt/` before copying, and copy into `docs/packets/audits/jms_v185/`.

- [ ] **Step 5: Pin evidence ×5 and add `verifies:`**

Per version:

```bash
go run ./tools/packet-audit evidence pin \
  --packet character/serverbound/StateChangeByPortableChair \
  --version gms_v83 \
  --ida "CWvsContext::TryRecovery" --category TIER1-FIXTURE
```

Then open each written `docs/packets/evidence/<version>/character.serverbound.StateChangeByPortableChair.yaml` and append:

```yaml
verifies:
    - libs/atlas-packet/character/serverbound/state_change_by_portable_chair_test.go#TestStateChangeByPortableChairRoundTrip
```

(Match the indentation style of the sibling `character.serverbound.ChairPortable.yaml`.)

- [ ] **Step 6: Regenerate matrix and check promotion**

```bash
go run ./tools/packet-audit matrix
go run ./tools/packet-audit matrix --check
grep -n "STATE_CHANGE_BY_PORTABLE_CHAIR_REQUEST" docs/packets/audits/STATUS.md
```

Expected: STATUS row 562 shows ✅ for all five versions. `matrix --check` bar per VERIFYING_A_PACKET.md §8: zero NEW orphan/dangling/stale/drift lines mentioning StateChangeByPortableChair, and the pre-existing conflict count must not increase. Also run `go run ./tools/packet-audit fname-doc --check` and `go run ./tools/packet-audit operations --check` — no new failures.

- [ ] **Step 7: Commit all audit artifacts together**

```bash
git add tools/packet-audit/cmd/run.go docs/packets/ida-exports/ docs/packets/audits/ docs/packets/evidence/
git commit -m "docs(packets): verify STATE_CHANGE_BY_PORTABLE_CHAIR_REQUEST x5 — fixtures, evidence, reports, matrix (task-141)"
```

---

### Task 6: atlas-chairs — registry model recovery timestamps

**Files:**
- Modify: `services/atlas-chairs/atlas.com/chairs/chair/model.go`
- Test: `services/atlas-chairs/atlas.com/chairs/chair/registry_test.go`

**Interfaces:**
- Produces: `Model.LastHpRecoveryAt() int64`, `Model.LastMpRecoveryAt() int64` (unix-milli, zero = never), `Model.WithHpRecoveryAt(at int64) Model`, `Model.WithMpRecoveryAt(at int64) Model` (value-copy builders). Task 8's `Recover` consumes these. Timestamps ride the existing Redis `TenantRegistry` JSON under the same key, so `Clear` discards them with the registration (FR-4.5 for free).

- [ ] **Step 1: Write the failing marshal round-trip test**

Append to `registry_test.go`:

```go
func TestRegistry_RecoveryTimestampsRoundTrip(t *testing.T) {
	setupTestRegistry(t)
	ctx := testCtx()

	characterId := uint32(22345)
	m := Model{id: 3010000, chairType: "PORTABLE"}.
		WithHpRecoveryAt(1751234567890).
		WithMpRecoveryAt(1751234570000)

	GetRegistry().Set(ctx, characterId, m)

	got, ok := GetRegistry().Get(ctx, characterId)
	if !ok {
		t.Fatal("Expected character to exist in registry after Set")
	}
	if got.LastHpRecoveryAt() != 1751234567890 {
		t.Errorf("lastHpRecoveryAt: got %d, want 1751234567890", got.LastHpRecoveryAt())
	}
	if got.LastMpRecoveryAt() != 1751234570000 {
		t.Errorf("lastMpRecoveryAt: got %d, want 1751234570000", got.LastMpRecoveryAt())
	}
	if got.Id() != 3010000 || got.Type() != "PORTABLE" {
		t.Errorf("existing fields lost in round-trip: id %d type %s", got.Id(), got.Type())
	}

	// A model without timestamps unmarshals to zero (backward-compatible with
	// pre-task-141 registry entries).
	GetRegistry().Set(ctx, characterId+1, Model{id: 1, chairType: "FIXED"})
	got2, _ := GetRegistry().Get(ctx, characterId+1)
	if got2.LastHpRecoveryAt() != 0 || got2.LastMpRecoveryAt() != 0 {
		t.Errorf("expected zero timestamps, got %d/%d", got2.LastHpRecoveryAt(), got2.LastMpRecoveryAt())
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-chairs/atlas.com/chairs && go test ./chair/ -run TestRegistry_RecoveryTimestampsRoundTrip`
Expected: FAIL — `WithHpRecoveryAt` undefined.

- [ ] **Step 3: Implement**

Replace `model.go` with:

```go
package chair

import "encoding/json"

type Model struct {
	id               uint32
	chairType        string
	lastHpRecoveryAt int64 // unix-milli of last honored HP recovery tick; zero = never
	lastMpRecoveryAt int64 // unix-milli of last honored MP recovery tick; zero = never
}

func (m Model) Id() uint32 {
	return m.id
}

func (m Model) Type() string {
	return m.chairType
}

func (m Model) LastHpRecoveryAt() int64 {
	return m.lastHpRecoveryAt
}

func (m Model) LastMpRecoveryAt() int64 {
	return m.lastMpRecoveryAt
}

func (m Model) WithHpRecoveryAt(at int64) Model {
	m.lastHpRecoveryAt = at
	return m
}

func (m Model) WithMpRecoveryAt(at int64) Model {
	m.lastMpRecoveryAt = at
	return m
}

func (m Model) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Id               uint32 `json:"id"`
		ChairType        string `json:"chairType"`
		LastHpRecoveryAt int64  `json:"lastHpRecoveryAt,omitempty"`
		LastMpRecoveryAt int64  `json:"lastMpRecoveryAt,omitempty"`
	}{
		Id:               m.id,
		ChairType:        m.chairType,
		LastHpRecoveryAt: m.lastHpRecoveryAt,
		LastMpRecoveryAt: m.lastMpRecoveryAt,
	})
}

func (m *Model) UnmarshalJSON(data []byte) error {
	t := &struct {
		Id               uint32 `json:"id"`
		ChairType        string `json:"chairType"`
		LastHpRecoveryAt int64  `json:"lastHpRecoveryAt"`
		LastMpRecoveryAt int64  `json:"lastMpRecoveryAt"`
	}{}
	if err := json.Unmarshal(data, t); err != nil {
		return err
	}
	m.id = t.Id
	m.chairType = t.ChairType
	m.lastHpRecoveryAt = t.LastHpRecoveryAt
	m.lastMpRecoveryAt = t.LastMpRecoveryAt
	return nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd services/atlas-chairs/atlas.com/chairs && go test -race ./chair/`
Expected: PASS (all existing tests too).

- [ ] **Step 5: Commit**

```bash
git add services/atlas-chairs/atlas.com/chairs/chair/model.go services/atlas-chairs/atlas.com/chairs/chair/registry_test.go
git commit -m "feat(chairs): recovery tick timestamps on chair registry model (task-141 FR-4.3)"
```

---

### Task 7: atlas-chairs — `data/setup` REST client

**Files:**
- Create: `services/atlas-chairs/atlas.com/chairs/data/setup/model.go`
- Create: `services/atlas-chairs/atlas.com/chairs/data/setup/rest.go`
- Create: `services/atlas-chairs/atlas.com/chairs/data/setup/requests.go`
- Create: `services/atlas-chairs/atlas.com/chairs/data/setup/processor.go`

**Interfaces:**
- Consumes: atlas-data `GET /data/setups/{id}` (JSON:API type `setups`, attributes `recoveryHP`/`recoveryMP` — Task 1).
- Produces: `setup.NewProcessor(l, ctx).GetById(itemId uint32) (Model, error)` with `Model.RecoveryHP() uint32`, `Model.RecoveryMP() uint32`. Task 8 consumes this. Mirrors the existing `data/map` package exactly (same file split, same `requests.RootUrl("DATA")` base).

- [ ] **Step 1: Write the four files**

`model.go`:

```go
package setup

type Model struct {
	recoveryHP uint32
	recoveryMP uint32
}

func (m Model) RecoveryHP() uint32 {
	return m.recoveryHP
}

func (m Model) RecoveryMP() uint32 {
	return m.recoveryMP
}
```

`rest.go`:

```go
package setup

import (
	"strconv"

	"github.com/jtumidanski/api2go/jsonapi"
)

type RestModel struct {
	Id         uint32 `json:"-"`
	RecoveryHP uint32 `json:"recoveryHP"`
	RecoveryMP uint32 `json:"recoveryMP"`
}

func (r RestModel) GetName() string {
	return "setups"
}

func (r RestModel) GetID() string {
	return strconv.Itoa(int(r.Id))
}

func (r *RestModel) SetID(idStr string) error {
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return err
	}
	r.Id = uint32(id)
	return nil
}

func (r *RestModel) SetToOneReferenceID(_ string, _ string) error {
	return nil
}

func (r *RestModel) SetToManyReferenceIDs(_ string, _ []string) error {
	return nil
}

func (r *RestModel) SetReferencedStructs(_ map[string]map[string]jsonapi.Data) error {
	return nil
}

func Extract(rm RestModel) (Model, error) {
	return Model{
		recoveryHP: rm.RecoveryHP,
		recoveryMP: rm.RecoveryMP,
	}, nil
}
```

`requests.go`:

```go
package setup

import (
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	getSetup = "data/setups/%d"
)

func getBaseRequest() string {
	return requests.RootUrl("DATA")
}

func requestSetup(itemId uint32) requests.Request[RestModel] {
	return requests.GetRequest[RestModel](fmt.Sprintf(getBaseRequest()+getSetup, itemId))
}
```

`processor.go`:

```go
package setup

import (
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	GetById(itemId uint32) (Model, error)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
	}
}

func (p *ProcessorImpl) GetById(itemId uint32) (Model, error) {
	return requests.Provider[RestModel, Model](p.l, p.ctx)(requestSetup(itemId), Extract)()
}
```

- [ ] **Step 2: Build**

Run: `cd services/atlas-chairs/atlas.com/chairs && go build ./... && go vet ./...`
Expected: clean. (Behavioral coverage lands in Task 8's processor tests, which exercise this client against an httptest server.)

- [ ] **Step 3: Commit**

```bash
git add services/atlas-chairs/atlas.com/chairs/data/setup/
git commit -m "feat(chairs): data/setup REST client for chair recovery stats (task-141)"
```

---

### Task 8: atlas-chairs — RECOVERY command orchestration

**Files:**
- Modify: `services/atlas-chairs/atlas.com/chairs/kafka/message/chair/kafka.go`
- Modify: `services/atlas-chairs/atlas.com/chairs/kafka/message/character/kafka.go`
- Modify: `services/atlas-chairs/atlas.com/chairs/chair/producer.go`
- Modify: `services/atlas-chairs/atlas.com/chairs/chair/processor.go`
- Modify: `services/atlas-chairs/atlas.com/chairs/kafka/consumer/chair/consumer.go`
- Test: `services/atlas-chairs/atlas.com/chairs/chair/processor_test.go`

**Interfaces:**
- Consumes: Task 6 (`WithHpRecoveryAt`/`WithMpRecoveryAt`, `LastHpRecoveryAt`/`LastMpRecoveryAt`), Task 7 (`setup.NewProcessor(l, ctx).GetById`).
- Produces:
  - Contract: `chair.CommandRecovery = "RECOVERY"`, `chair.RecoveryCommandBody{CharacterId uint32; Hp int16; Mp int16}` on `COMMAND_TOPIC_CHAIR` (Task 9's channel copy must match field-for-field).
  - `Processor.RecoverAndEmit(f field.Model, characterId uint32, claimedHp int16, claimedMp int16) error` and pure `Recover(mb *message.Buffer) func(...) error`.
  - Emissions: `Command[ChangeHPCommandBody]`/`Command[ChangeMPCommandBody]` on `COMMAND_TOPIC_CHARACTER`, envelope `{WorldId, CharacterId, Type, Body{ChannelId, Amount}}` — byte-identical to atlas-channel's existing `character` command shape (what atlas-character already consumes).
- Behavior (design §5.3, §7, §9): not-seated / FIXED / non-recovery-portable → pass-through of claimed values when `!= 0` (preserves today's behavior byte-for-byte, including negative jms clamp corrections); seated portable with a recovery stat → item value applied (claim ignored, warn on mismatch), rate-limited at 4000 ms per stat, data-lookup failure drops the tick (fail-closed), rejects never disconnect.

- [ ] **Step 1: Add the Kafka message contracts**

`kafka/message/chair/kafka.go` — extend the command const block and add the body:

```go
const (
	EnvCommandTopic    = "COMMAND_TOPIC_CHAIR"
	CommandUseChair    = "USE"
	CommandCancelChair = "CANCEL"
	CommandRecovery    = "RECOVERY"

	ChairTypeFixed    = "FIXED"
	ChairTypePortable = "PORTABLE"
)
```

and after `CancelChairCommandBody`:

```go
type RecoveryCommandBody struct {
	CharacterId uint32 `json:"characterId"`
	Hp          int16  `json:"hp"` // client-claimed; trusted only for the natural-regen pass-through
	Mp          int16  `json:"mp"`
}
```

`kafka/message/character/kafka.go` — append the command contract (the file currently holds only status events; the envelope mirrors atlas-channel's `character.Command` exactly):

```go
const (
	EnvCommandTopic = "COMMAND_TOPIC_CHARACTER"
	CommandChangeHP = "CHANGE_HP"
	CommandChangeMP = "CHANGE_MP"
)

type Command[E any] struct {
	WorldId     world.Id `json:"worldId"`
	CharacterId uint32   `json:"characterId"`
	Type        string   `json:"type"`
	Body        E        `json:"body"`
}

type ChangeHPCommandBody struct {
	ChannelId channel.Id `json:"channelId"`
	Amount    int16      `json:"amount"`
}

type ChangeMPCommandBody struct {
	ChannelId channel.Id `json:"channelId"`
	Amount    int16      `json:"amount"`
}
```

(`world`, `channel` are already imported in that file.)

- [ ] **Step 2: Add the CHANGE_HP/MP providers**

Append to `chair/producer.go` (add import `character2 "atlas-chairs/kafka/message/character"`):

```go
func changeHPCommandProvider(field field.Model, characterId uint32, amount int16) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &character2.Command[character2.ChangeHPCommandBody]{
		WorldId:     field.WorldId(),
		CharacterId: characterId,
		Type:        character2.CommandChangeHP,
		Body: character2.ChangeHPCommandBody{
			ChannelId: field.ChannelId(),
			Amount:    amount,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func changeMPCommandProvider(field field.Model, characterId uint32, amount int16) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &character2.Command[character2.ChangeMPCommandBody]{
		WorldId:     field.WorldId(),
		CharacterId: characterId,
		Type:        character2.CommandChangeMP,
		Body: character2.ChangeMPCommandBody{
			ChannelId: field.ChannelId(),
			Amount:    amount,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
```

- [ ] **Step 3: Write the failing processor tests**

Append to `chair/processor_test.go` (same package; uses the existing `setupProcessorTestRegistry`/`testTenant` helpers). Common scaffolding for these tests:

```go
func recoveryTestContext(t *testing.T, recoveryHP uint32, recoveryMP uint32) (context.Context, func() int) {
	t.Helper()
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		id := path.Base(r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"data":{"type":"setups","id":"%s","attributes":{"recoveryHP":%d,"recoveryMP":%d}}}`, id, recoveryHP, recoveryMP)
	}))
	t.Cleanup(srv.Close)
	t.Setenv("DATA_SERVICE_URL", srv.URL+"/")
	return tenant.WithContext(context.Background(), testTenant()), func() int { return calls }
}

func recoveryMessages(t *testing.T, buf *message.Buffer) []character2.Command[json.RawMessage] {
	t.Helper()
	var out []character2.Command[json.RawMessage]
	for _, m := range buf.GetAll()[character2.EnvCommandTopic] {
		var c character2.Command[json.RawMessage]
		if err := json.Unmarshal(m.Value, &c); err != nil {
			t.Fatalf("unmarshal emitted command: %v", err)
		}
		out = append(out, c)
	}
	return out
}

func amountOf(t *testing.T, raw json.RawMessage) int16 {
	t.Helper()
	var b struct {
		Amount int16 `json:"amount"`
	}
	if err := json.Unmarshal(raw, &b); err != nil {
		t.Fatalf("unmarshal body: %v", err)
	}
	return b.Amount
}
```

(imports to add: `encoding/json`, `fmt`, `net/http`, `net/http/httptest`, `path`, `time`, `atlas-chairs/kafka/message`, `character2 "atlas-chairs/kafka/message/character"`, `"github.com/Chronicle20/atlas/libs/atlas-constants/field"`.)

Test cases (each creates `f := field.NewBuilder(0, 1, 100000000).Build()` and `p := NewProcessor(l, tctx).(*ProcessorImpl)`):

```go
func TestRecover_SeatedRecoveryChair_AppliesItemValues(t *testing.T) {
	setupProcessorTestRegistry(t)
	l, _ := test.NewNullLogger()
	tctx, calls := recoveryTestContext(t, 60, 60) // 03010136-style both-stats chair
	f := field.NewBuilder(0, 1, 100000000).Build()
	characterId := uint32(1001)
	GetRegistry().Set(tctx, characterId, Model{id: 3010136, chairType: "PORTABLE"})

	buf := message.NewBuffer()
	p := NewProcessor(l, tctx).(*ProcessorImpl)
	if err := p.Recover(buf)(f, characterId, 60, 60); err != nil {
		t.Fatalf("Recover: %v", err)
	}

	msgs := recoveryMessages(t, buf)
	if len(msgs) != 2 {
		t.Fatalf("expected CHANGE_HP+CHANGE_MP, got %d messages", len(msgs))
	}
	if msgs[0].Type != character2.CommandChangeHP || amountOf(t, msgs[0].Body) != 60 {
		t.Errorf("first message: got %s/%d, want CHANGE_HP/60", msgs[0].Type, amountOf(t, msgs[0].Body))
	}
	if msgs[1].Type != character2.CommandChangeMP || amountOf(t, msgs[1].Body) != 60 {
		t.Errorf("second message: got %s/%d, want CHANGE_MP/60", msgs[1].Type, amountOf(t, msgs[1].Body))
	}
	if calls() != 1 {
		t.Errorf("expected 1 setup-data lookup, got %d", calls())
	}
	m, _ := GetRegistry().Get(tctx, characterId)
	if m.LastHpRecoveryAt() == 0 || m.LastMpRecoveryAt() == 0 {
		t.Error("expected recovery timestamps to be recorded")
	}
}

func TestRecover_SeatedRecoveryChair_ItemValueOverridesClaim(t *testing.T) {
	// HP-only chair (recoveryHP=50, recoveryMP=0); claim lies with hp=30000, mp=5.
	setupProcessorTestRegistry(t)
	l, _ := test.NewNullLogger()
	tctx, _ := recoveryTestContext(t, 50, 0)
	f := field.NewBuilder(0, 1, 100000000).Build()
	characterId := uint32(1002)
	GetRegistry().Set(tctx, characterId, Model{id: 3010000, chairType: "PORTABLE"})

	buf := message.NewBuffer()
	p := NewProcessor(l, tctx).(*ProcessorImpl)
	if err := p.Recover(buf)(f, characterId, 30000, 5); err != nil {
		t.Fatalf("Recover: %v", err)
	}

	msgs := recoveryMessages(t, buf)
	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(msgs))
	}
	// HP: item value 50 applied, forged 30000 ignored.
	if msgs[0].Type != character2.CommandChangeHP || amountOf(t, msgs[0].Body) != 50 {
		t.Errorf("HP: got %s/%d, want CHANGE_HP/50", msgs[0].Type, amountOf(t, msgs[0].Body))
	}
	// MP: chair doesn't cover it -> natural pass-through of the claim.
	if msgs[1].Type != character2.CommandChangeMP || amountOf(t, msgs[1].Body) != 5 {
		t.Errorf("MP: got %s/%d, want CHANGE_MP/5", msgs[1].Type, amountOf(t, msgs[1].Body))
	}
}

func TestRecover_RateLimited_Drops(t *testing.T) {
	setupProcessorTestRegistry(t)
	l, _ := test.NewNullLogger()
	tctx, _ := recoveryTestContext(t, 50, 0)
	f := field.NewBuilder(0, 1, 100000000).Build()
	characterId := uint32(1003)
	now := time.Now().UnixMilli()
	GetRegistry().Set(tctx, characterId, Model{id: 3010000, chairType: "PORTABLE"}.WithHpRecoveryAt(now))

	buf := message.NewBuffer()
	p := NewProcessor(l, tctx).(*ProcessorImpl)
	if err := p.Recover(buf)(f, characterId, 50, 0); err != nil {
		t.Fatalf("Recover: %v", err)
	}
	if len(recoveryMessages(t, buf)) != 0 {
		t.Error("expected rate-limited tick to emit nothing")
	}
	m, _ := GetRegistry().Get(tctx, characterId)
	if m.LastHpRecoveryAt() != now {
		t.Error("expected timestamp unchanged on rejected tick")
	}
}

func TestRecover_NotSeated_PassThrough(t *testing.T) {
	setupProcessorTestRegistry(t)
	l, _ := test.NewNullLogger()
	tctx, calls := recoveryTestContext(t, 50, 0)
	f := field.NewBuilder(0, 1, 100000000).Build()

	buf := message.NewBuffer()
	p := NewProcessor(l, tctx).(*ProcessorImpl)
	if err := p.Recover(buf)(f, uint32(1004), 17, -3); err != nil {
		t.Fatalf("Recover: %v", err)
	}
	msgs := recoveryMessages(t, buf)
	if len(msgs) != 2 {
		t.Fatalf("expected 2 pass-through messages, got %d", len(msgs))
	}
	if amountOf(t, msgs[0].Body) != 17 {
		t.Errorf("HP pass-through: got %d, want 17", amountOf(t, msgs[0].Body))
	}
	// Negative claims (jms clamp-to-max corrections) pass through unchanged.
	if amountOf(t, msgs[1].Body) != -3 {
		t.Errorf("MP pass-through: got %d, want -3", amountOf(t, msgs[1].Body))
	}
	if calls() != 0 {
		t.Errorf("not-seated tick must not hit setup data, got %d calls", calls())
	}
}

func TestRecover_FixedSeat_PassThrough(t *testing.T) {
	setupProcessorTestRegistry(t)
	l, _ := test.NewNullLogger()
	tctx, calls := recoveryTestContext(t, 50, 0)
	f := field.NewBuilder(0, 1, 100000000).Build()
	characterId := uint32(1005)
	GetRegistry().Set(tctx, characterId, Model{id: 2, chairType: "FIXED"})

	buf := message.NewBuffer()
	p := NewProcessor(l, tctx).(*ProcessorImpl)
	if err := p.Recover(buf)(f, characterId, 25, 0); err != nil {
		t.Fatalf("Recover: %v", err)
	}
	msgs := recoveryMessages(t, buf)
	if len(msgs) != 1 || amountOf(t, msgs[0].Body) != 25 {
		t.Fatalf("expected single HP pass-through of 25, got %d messages", len(msgs))
	}
	if calls() != 0 {
		t.Errorf("fixed-seat tick must not hit setup data, got %d calls", calls())
	}
}

func TestRecover_NonRecoveryPortable_PassThrough(t *testing.T) {
	setupProcessorTestRegistry(t)
	l, _ := test.NewNullLogger()
	tctx, _ := recoveryTestContext(t, 0, 0) // portable chair with no recovery stats
	f := field.NewBuilder(0, 1, 100000000).Build()
	characterId := uint32(1006)
	GetRegistry().Set(tctx, characterId, Model{id: 3010900, chairType: "PORTABLE"})

	buf := message.NewBuffer()
	p := NewProcessor(l, tctx).(*ProcessorImpl)
	if err := p.Recover(buf)(f, characterId, 17, 3); err != nil {
		t.Fatalf("Recover: %v", err)
	}
	msgs := recoveryMessages(t, buf)
	if len(msgs) != 2 || amountOf(t, msgs[0].Body) != 17 || amountOf(t, msgs[1].Body) != 3 {
		t.Fatalf("expected claimed 17/3 pass-through, got %d messages", len(msgs))
	}
}

func TestRecover_DataLookupFailure_DropsSeatedTick(t *testing.T) {
	setupProcessorTestRegistry(t)
	l, _ := test.NewNullLogger()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)
	t.Setenv("DATA_SERVICE_URL", srv.URL+"/")
	tctx := tenant.WithContext(context.Background(), testTenant())
	f := field.NewBuilder(0, 1, 100000000).Build()
	characterId := uint32(1007)
	GetRegistry().Set(tctx, characterId, Model{id: 3010000, chairType: "PORTABLE"})

	buf := message.NewBuffer()
	p := NewProcessor(l, tctx).(*ProcessorImpl)
	if err := p.Recover(buf)(f, characterId, 50, 0); err != nil {
		t.Fatalf("Recover must swallow the lookup failure, got: %v", err)
	}
	if len(recoveryMessages(t, buf)) != 0 {
		t.Error("expected fail-closed drop (never fall back to claimed value)")
	}
}

func TestRecover_ZeroClaims_EmitNothing(t *testing.T) {
	setupProcessorTestRegistry(t)
	l, _ := test.NewNullLogger()
	tctx, _ := recoveryTestContext(t, 0, 0)
	f := field.NewBuilder(0, 1, 100000000).Build()

	buf := message.NewBuffer()
	p := NewProcessor(l, tctx).(*ProcessorImpl)
	if err := p.Recover(buf)(f, uint32(1008), 0, 0); err != nil {
		t.Fatalf("Recover: %v", err)
	}
	if len(recoveryMessages(t, buf)) != 0 {
		t.Error("zero claims must emit nothing")
	}
}

func TestRecover_ClearResetsTimestamps(t *testing.T) {
	// Standing up removes the registration (and its timestamps) entirely;
	// a subsequent tick takes the not-seated pass-through branch.
	setupProcessorTestRegistry(t)
	l, _ := test.NewNullLogger()
	tctx, _ := recoveryTestContext(t, 50, 0)
	f := field.NewBuilder(0, 1, 100000000).Build()
	characterId := uint32(1009)
	GetRegistry().Set(tctx, characterId, Model{id: 3010000, chairType: "PORTABLE"}.WithHpRecoveryAt(time.Now().UnixMilli()))

	_ = NewProcessor(l, tctx).Clear(f, characterId)

	if _, ok := GetRegistry().Get(tctx, characterId); ok {
		t.Fatal("expected registration (and timestamps) gone after Clear")
	}

	buf := message.NewBuffer()
	p := NewProcessor(l, tctx).(*ProcessorImpl)
	if err := p.Recover(buf)(f, characterId, 17, 0); err != nil {
		t.Fatalf("Recover: %v", err)
	}
	msgs := recoveryMessages(t, buf)
	if len(msgs) != 1 || amountOf(t, msgs[0].Body) != 17 {
		t.Fatal("expected not-seated pass-through after stand-up")
	}
}
```

Note: `Clear` emits a CANCELLED status event via the real producer, which will fail without a broker — but `Clear` ignores... actually `Clear` RETURNS the producer error after removing from the registry. The test above only relies on registry removal having happened; assign `_ =` and don't assert on its error. If the producer path panics in test (no `BOOTSTRAP_SERVERS`), replace the `Clear` call with direct `GetRegistry().Clear(tctx, characterId)` — registry-level clear is what FR-4.5 depends on. Prefer the processor call; fall back only if the environment makes it impossible, and note which was used.

- [ ] **Step 4: Run tests to verify they fail**

Run: `cd services/atlas-chairs/atlas.com/chairs && go test ./chair/ -run TestRecover`
Expected: FAIL — `p.Recover` undefined.

- [ ] **Step 5: Implement `Recover`/`RecoverAndEmit`**

In `chair/processor.go`: extend the interface and add the implementation. New imports: `"time"`, `setup2 "atlas-chairs/data/setup"`, `"atlas-chairs/kafka/message"`, `character2 "atlas-chairs/kafka/message/character"`.

```go
// minRecoveryTickIntervalMillis is the server-side floor between honored
// recovery ticks per stat per character. The client cadence is frame-paced
// (accumulator +30/frame, threshold 10000): ~11.1 s at 30 fps, ~5.6 s at
// 60 fps, ~4.5 s at 75 fps (task-141 design §7). 4000 ms sits comfortably
// below the fastest legitimate cadence while capping spam at ~15 ticks/min.
// Server-internal policy, not a client-wire value (DOM-25 does not apply).
const minRecoveryTickIntervalMillis int64 = 4000

type Processor interface {
	GetById(characterId uint32) (Model, error)
	Set(field field.Model, chairType string, chairId uint32, characterId uint32) error
	Clear(field field.Model, characterId uint32) error
	RecoverAndEmit(field field.Model, characterId uint32, claimedHp int16, claimedMp int16) error
}
```

```go
func (p *ProcessorImpl) RecoverAndEmit(f field.Model, characterId uint32, claimedHp int16, claimedMp int16) error {
	return message.Emit(producer.ProviderImpl(p.l)(p.ctx))(func(buf *message.Buffer) error {
		return p.Recover(buf)(f, characterId, claimedHp, claimedMp)
	})
}

// Recover validates a HEAL_OVER_TIME tick (task-141 design §5.3). Seated on a
// portable chair with recovery stats: the item value is applied (the claim is
// ignored) at most once per minRecoveryTickIntervalMillis per stat. Everything
// else passes the claimed values through unchanged, preserving the pre-task-141
// natural-regen behavior (including negative jms clamp-to-max corrections).
func (p *ProcessorImpl) Recover(mb *message.Buffer) func(f field.Model, characterId uint32, claimedHp int16, claimedMp int16) error {
	return func(f field.Model, characterId uint32, claimedHp int16, claimedMp int16) error {
		m, err := p.GetById(characterId)
		if err != nil || m.Type() != chair2.ChairTypePortable {
			return p.passThrough(mb, f, characterId, claimedHp, claimedMp)
		}

		s, err := setup2.NewProcessor(p.l, p.ctx).GetById(m.Id())
		if err != nil {
			p.l.WithError(err).Warnf("Unable to retrieve setup data for chair [%d]; dropping recovery tick for character [%d].", m.Id(), characterId)
			return nil
		}
		if s.RecoveryHP() == 0 && s.RecoveryMP() == 0 {
			return p.passThrough(mb, f, characterId, claimedHp, claimedMp)
		}

		now := time.Now().UnixMilli()
		updated := m

		if s.RecoveryHP() > 0 {
			if now-updated.LastHpRecoveryAt() < minRecoveryTickIntervalMillis {
				p.l.Debugf("Dropping HP recovery tick for character [%d] on chair [%d]: reason [rate].", characterId, m.Id())
			} else {
				if claimedHp != int16(s.RecoveryHP()) {
					p.l.Warnf("Character [%d] claimed HP recovery [%d] differing from chair [%d] item value [%d]; applying item value.", characterId, claimedHp, m.Id(), s.RecoveryHP())
				}
				if err = mb.Put(character2.EnvCommandTopic, changeHPCommandProvider(f, characterId, int16(s.RecoveryHP()))); err != nil {
					return err
				}
				updated = updated.WithHpRecoveryAt(now)
			}
		} else if claimedHp != 0 {
			if err = mb.Put(character2.EnvCommandTopic, changeHPCommandProvider(f, characterId, claimedHp)); err != nil {
				return err
			}
		}

		if s.RecoveryMP() > 0 {
			if now-updated.LastMpRecoveryAt() < minRecoveryTickIntervalMillis {
				p.l.Debugf("Dropping MP recovery tick for character [%d] on chair [%d]: reason [rate].", characterId, m.Id())
			} else {
				if claimedMp != int16(s.RecoveryMP()) {
					p.l.Warnf("Character [%d] claimed MP recovery [%d] differing from chair [%d] item value [%d]; applying item value.", characterId, claimedMp, m.Id(), s.RecoveryMP())
				}
				if err = mb.Put(character2.EnvCommandTopic, changeMPCommandProvider(f, characterId, int16(s.RecoveryMP()))); err != nil {
					return err
				}
				updated = updated.WithMpRecoveryAt(now)
			}
		} else if claimedMp != 0 {
			if err = mb.Put(character2.EnvCommandTopic, changeMPCommandProvider(f, characterId, claimedMp)); err != nil {
				return err
			}
		}

		if updated != m {
			GetRegistry().Set(p.ctx, characterId, updated)
		}
		return nil
	}
}

func (p *ProcessorImpl) passThrough(mb *message.Buffer, f field.Model, characterId uint32, claimedHp int16, claimedMp int16) error {
	if claimedHp != 0 {
		if err := mb.Put(character2.EnvCommandTopic, changeHPCommandProvider(f, characterId, claimedHp)); err != nil {
			return err
		}
	}
	if claimedMp != 0 {
		if err := mb.Put(character2.EnvCommandTopic, changeMPCommandProvider(f, characterId, claimedMp)); err != nil {
			return err
		}
	}
	return nil
}
```

(Kafka same-key ordering makes the read-modify-write of the registry model single-writer per character — the consumer is the only Recover caller in production.)

- [ ] **Step 6: Add the consumer arm**

In `kafka/consumer/chair/consumer.go`, register a third handler in `InitHandlers`:

```go
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleCommandRecovery))); err != nil {
			return err
		}
```

and add:

```go
func handleCommandRecovery(l logrus.FieldLogger, ctx context.Context, c chair2.Command[chair2.RecoveryCommandBody]) {
	if c.Type != chair2.CommandRecovery {
		return
	}
	f := field.NewBuilder(c.WorldId, c.ChannelId, c.MapId).Build()
	_ = chair.NewProcessor(l, ctx).RecoverAndEmit(f, c.Body.CharacterId, c.Body.Hp, c.Body.Mp)
}
```

- [ ] **Step 7: Run tests to verify they pass**

Run: `cd services/atlas-chairs/atlas.com/chairs && go test -race ./... && go vet ./... && go build ./...`
Expected: PASS/clean.

- [ ] **Step 8: Commit**

```bash
git add services/atlas-chairs/atlas.com/chairs/
git commit -m "feat(chairs): server-authoritative chair recovery via RECOVERY command (task-141 FR-4)"
```

---

### Task 9: atlas-channel — reroute HEAL_OVER_TIME through the RECOVERY command

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/kafka/message/chair/kafka.go`
- Modify: `services/atlas-channel/atlas.com/channel/chair/producer.go`
- Modify: `services/atlas-channel/atlas.com/channel/chair/processor.go`
- Modify: `services/atlas-channel/atlas.com/channel/socket/handler/character_heal_over_time.go`
- Test: `services/atlas-channel/atlas.com/channel/chair/producer_test.go` (new)

**Interfaces:**
- Consumes: `HealOverTime.HP()/MP()` (existing lib decode); session `s.Field()`, `s.CharacterId()`.
- Produces: `chair.Processor.Recover(f field.Model, characterId uint32, hp int16, mp int16) error` emitting `Command[RecoveryCommandBody]{Type: "RECOVERY"}` on `COMMAND_TOPIC_CHAIR` — field-for-field identical to Task 8's chairs-side contract (`characterId`/`hp`/`mp` JSON keys).
- Behavior: the handler stops calling `character.ChangeHP/MP` directly; ticks with both claims zero are dropped at the handler; everything else (including negative values) is forwarded.

- [ ] **Step 1: Add the message contract (channel copy)**

In `kafka/message/chair/kafka.go`, mirror Task 8 exactly: add `CommandRecovery = "RECOVERY"` to the const block and append:

```go
type RecoveryCommandBody struct {
	CharacterId uint32 `json:"characterId"`
	Hp          int16  `json:"hp"` // client-claimed; trusted only for the natural-regen pass-through
	Mp          int16  `json:"mp"`
}
```

- [ ] **Step 2: Write the failing provider test**

`chair/producer_test.go`:

```go
package chair

import (
	"encoding/json"
	"testing"

	chair2 "atlas-channel/kafka/message/chair"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
)

func TestRecoveryCommandProvider(t *testing.T) {
	f := field.NewBuilder(0, 1, 100000000).Build()
	msgs, err := RecoveryCommandProvider(f, 12345, 50, -3)()
	if err != nil {
		t.Fatalf("provider: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}

	var c chair2.Command[chair2.RecoveryCommandBody]
	if err := json.Unmarshal(msgs[0].Value, &c); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if c.Type != chair2.CommandRecovery {
		t.Errorf("type: got %s, want RECOVERY", c.Type)
	}
	if c.WorldId != 0 || c.ChannelId != 1 || c.MapId != 100000000 {
		t.Errorf("field routing: got %d/%d/%d", c.WorldId, c.ChannelId, c.MapId)
	}
	if c.Body.CharacterId != 12345 || c.Body.Hp != 50 || c.Body.Mp != -3 {
		t.Errorf("body: got %+v", c.Body)
	}
}
```

Run: `cd services/atlas-channel/atlas.com/channel && go test ./chair/`
Expected: FAIL — `RecoveryCommandProvider` undefined.

- [ ] **Step 3: Implement provider + processor method**

Append to `chair/producer.go`:

```go
func RecoveryCommandProvider(f field.Model, characterId uint32, hp int16, mp int16) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &chair.Command[chair.RecoveryCommandBody]{
		WorldId:   f.WorldId(),
		ChannelId: f.ChannelId(),
		MapId:     f.MapId(),
		Instance:  f.Instance(),
		Type:      chair.CommandRecovery,
		Body: chair.RecoveryCommandBody{
			CharacterId: characterId,
			Hp:          hp,
			Mp:          mp,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
```

In `chair/processor.go`, add to the `Processor` interface:

```go
	Recover(f field.Model, characterId uint32, hp int16, mp int16) error
```

and the implementation:

```go
func (p *ProcessorImpl) Recover(f field.Model, characterId uint32, hp int16, mp int16) error {
	return producer.ProviderImpl(p.l)(p.ctx)(chair2.EnvCommandTopic)(RecoveryCommandProvider(f, characterId, hp, mp))
}
```

- [ ] **Step 4: Reroute the heal handler**

Replace the body of `socket/handler/character_heal_over_time.go`'s returned closure (imports: drop `atlas-channel/character`, add `atlas-channel/chair`):

```go
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := character2.HealOverTime{}
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())
		if p.HP() == 0 && p.MP() == 0 {
			return
		}
		_ = chair.NewProcessor(l, ctx).Recover(s.Field(), s.CharacterId(), p.HP(), p.MP())
	}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `cd services/atlas-channel/atlas.com/channel && go test -race ./... && go vet ./... && go build ./...`
Expected: PASS/clean.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/
git commit -m "feat(channel): route HEAL_OVER_TIME through chair RECOVERY command (task-141 FR-5.1)"
```

---

### Task 10: Final verification gates

**Files:** none (verification only). Run every command; paste failures verbatim rather than summarizing them away.

- [ ] **Step 1: Per-module test/vet/build sweep**

```bash
for m in libs/atlas-packet services/atlas-data/atlas.com/data services/atlas-chairs/atlas.com/chairs services/atlas-channel/atlas.com/channel tools/packet-audit; do
  (cd "$m" && go test -race ./... && go vet ./... && go build ./...) || echo "FAILED: $m"
done
```

Expected: no `FAILED:` lines.

- [ ] **Step 2: Docker bake for every touched service**

From the worktree root (mandatory per CLAUDE.md — `go build` will not catch a missing Dockerfile `COPY libs/...` line):

```bash
docker buildx bake atlas-data atlas-chairs atlas-channel
```

Expected: all three build clean. (No new shared lib was added, so no Dockerfile/go.work edits are expected — if bake fails on a missing COPY, fix the Dockerfile and re-run.)

- [ ] **Step 3: Redis key guard**

From the repo root, no `GOWORK` prefix:

```bash
tools/redis-key-guard.sh
```

Expected: clean (the registry change stays inside `atlas-redis` `TenantRegistry` types).

- [ ] **Step 4: Packet-audit checks**

```bash
go run ./tools/packet-audit matrix --check
go run ./tools/packet-audit fname-doc --check
go run ./tools/packet-audit operations --check
grep -n "STATE_CHANGE_BY_PORTABLE_CHAIR_REQUEST" docs/packets/audits/STATUS.md
```

Expected: row 562 ✅ ×5; no new check failures attributable to this task (pre-existing conflict backlog excluded per VERIFYING_A_PACKET.md §8).

- [ ] **Step 5: Commit anything the gates changed, then request code review**

Per CLAUDE.md, run `superpowers:requesting-code-review` before any PR. Findings go to `docs/tasks/task-141-portable-chair-recovery/audit.md`.

---

## Rollout runbook (post-merge, per environment — design §8 order)

These are operational steps, not implementation tasks. Keep the order; step 1 before step 3 avoids the benign transient where MP chairs heal HP only.

1. **atlas-data first.** Deploy. Then backfill `recoveryMP` (FR-3.3): re-ingest the canonical tenant's `Item.wz` with the new reader; re-publish the canonical baseline via the admin UI baselines page (atlas-ui → Baselines; the publish/restore endpoints live under atlas-data `/data/baseline*`, POSTs need the JSON:API envelope) so baseline-bootstrap environments pick it up. Tenants with their own non-canonical setup rows additionally need a per-tenant re-ingest (per-id reads prefer tenant rows over canonical fallback). Verify: `GET /api/data/setups/3010136` returns `recoveryMP: 60` on the canonical tenant. Design §13: verify a baseline restore round-trip on an ephemeral env before trusting the re-published baseline.
2. **atlas-chairs.** Deploy. Safe while channel still applies directly — the RECOVERY consumer receives nothing yet.
3. **atlas-channel.** Deploy (carries the lib bump, both handler changes).
4. **Live tenant configs.** Seed templates apply only at tenant creation. For every existing tenant: `GET /api/configurations/tenants/{tenantId}` (atlas-configurations), add the `CharacterStateChangeByPortableChairHandle` entry at the version's opcode (v83/v84 `0x4A`, v87 `0x4D`, v95 `0x50`, jms `0x42`, each with `"validator": "LoggedInValidator"`) to `socket.handlers`, and for v95 tenants also audit the live config for the same validator-less entries Task 4 fixed in the template (at minimum `0x64 CharacterHealOverTimeHandle`) — then `PATCH /api/configurations/tenants/{tenantId}` (JSON:API envelope: `{"data":{"type":"...","id":"...","attributes":{...}}}`) and restart atlas-channel pods (projection does not hot-reload handlers). Full sweep across all tenants, not a spot-check.
5. **Live acceptance (design §11):** on v83 — The Relaxer (3010000) restores HP at client cadence; 3010136 restores HP+MP; standing up stops it; natural regen (not seated) still works; packet 0x4A logs once per sit at debug with no disconnect. On v95 — regen works again after the validator patch (expected behavior change: formerly-dropped HEAL_OVER_TIME packets now apply — design §13 calls this out as intended).

## Known risks (from design §13)

- Regen availability now depends on atlas-chairs + Kafka; ticks buffer during an outage and apply on recovery. Accepted.
- The natural-path routing change touches every character's regen; the live-acceptance step explicitly re-verifies natural regen on v83.
- Task 5 requires live IDA instances; if one is unreachable, stop and report (never fake evidence/exports).

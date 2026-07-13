# Implementing a packet (a new codec × all applicable versions)

The canonical procedure for adding a *new* packet codec to `libs/atlas-packet`,
wiring it into `atlas-channel`, and routing it in the seed templates so the
coverage-matrix cell (`docs/packets/audits/STATUS.md`) can be promoted.

This is the companion to [`VERIFYING_A_PACKET.md`](audits/VERIFYING_A_PACKET.md):
that doc covers *verifying* a codec that already exists; this one covers
*creating* one. The last step of this recipe hands off to that one — the
verify markers and evidence pins are produced here so the cell promotes in the
same change.

Hard rule (CLAUDE.md "Verification Over Memory"): the field order of an
unimplemented op is **unknown until its client function is decompiled**. Never
transcribe a byte layout from MapleStory knowledge or memory. Step 1 produces
the layout as a concrete artifact; every later step reads from it.

This recipe was distilled from task-092 (the MOB/MONSTER family); the worked
example below (`MOB_CRC_KEY_CHANGED`) is a real landed op from that batch.

---

## Step 0 — Confirm it isn't already implemented

Before writing a codec, **check the op isn't already handled under a different name.**
A coverage-matrix ❌ on a serverbound op frequently means an *unverified* codec, not a
missing one — several ops can share one decoder. Grep the channel handlers and
`libs/atlas-packet/model/` for a decoder of that opcode / fname; if one exists (e.g. the
four attack ops all decode `model.AttackInfo`), the task is **verification**, not
implementation — link the existing codec via a thin per-op wrapper (see
`VERIFYING_A_PACKET.md` §9) instead of shipping a duplicate (which is easy to get subtly
wrong). task-092 nearly shipped a duplicate `TOUCH_MONSTER_ATTACK` codec this way.

## The four steps

1. **Derive** — decompile the client read/write order from the IDB.
2. **Model + codec** — write the immutable `libs/atlas-packet` struct with
   `Encode`/`Decode` that mirror the derived order.
3. **Wire** — register the writer/handler in `atlas-channel` and route the
   per-version opcode in all nine seed templates.
4. **Verify** — round-trip + golden-byte test, `// packet-audit:verify`
   markers, evidence pin, regenerate the matrix. (This step is
   `VERIFYING_A_PACKET.md` applied to the codec you just wrote.)

---

## Step 1 — Derive structure from the IDB

For each applicable version, `select_instance(<port>)` then `decompile` the
registry entry's `fname`, descending into helper reads/writes (the same
address-based descent rule the exporter uses). Record the ordered field list
with widths (`Decode1/2/4/Str/Buffer`) and every per-version delta into
`docs/tasks/<task>/structures/<version>.md#<OP>`, including the export address.

Multi-instance IDA ports are assigned per IDA launch order and **must not be
hardcoded** — enumerate with `list_instances` and `select_instance` the one
whose loaded IDB matches the target version. (task-092 used v83=13337,
v87=13338, v95=13339, jms=13340, v84=13341, but that is launch-order specific.)

For a **serverbound** op the "read order" is what the *client* writes (its
`Encode*`/`COutPacket` build site), which the server then reads back. Send-side
functions decompile with `Encode*` calls, not `Decode*` — trace the COutPacket
build site, not a read helper.

### Guards before you write any Go

- **Registry-fname mislabels.** Some non-universal ops carry a stale `fname`
  in the registry (an opcode-cluster off-by-one — e.g. `MOB_SPEAKING` pointing
  at `OnIncMobChargeCount`). Confirm each `fname` against the IDB *before*
  deriving. If wrong, fix the registry row in the same change
  (`provenance: manual`, with an IDA citation in `note`). A wrong fname yields
  a wrong byte layout that round-trips cleanly and still ships broken.
- **Missing fname.** If a registry row has no `fname` (common for serverbound
  send-sites), derive the send-site from the IDB and populate it
  (`provenance: ida-discovered`, address in `ida.address`).
- **Export-resolvability is a precondition for `evidence pin`.** Before you
  rely on being able to pin (Step 4), confirm the op's `fname` resolves as a
  key in `docs/packets/ida-exports/<version>…json`'s `functions` map. If it is
  absent, `evidence pin` fails and the cell cannot verify until the export is
  re-harvested (task-081 playbook). **Escalate an unresolved fname to the user
  — do not auto-re-export, substitute a fname, or fake the hash.**
- **`v84 ≡ v83`.** v84 packet structure is byte-identical to v83 *below* the
  shifted opcode table (the opcode numbers differ; the payload does not). Do
  not invent v84 deltas the IDB doesn't show. Use `MajorAtLeast(87)`-style
  gates so v84 takes the v83 path — never `> 83`, which is an off-by-one that
  wrongly routes v84 down the v87 branch. (Genuine v84-only deltas exist — e.g.
  `MOB_SKILL_DELAY` is present in v84 but absent in v83 — but only adopt one the
  v84 IDB actually proves.)

---

## Step 2 — Model + codec

Add `libs/atlas-packet/<family>/<dir>/<op>.go`. Package is chosen by the
**owning client class family**, matching the existing convention:

| Owner class | Package |
|---|---|
| `CMob::` / `CMobPool::` | `monster/{clientbound,serverbound}` |
| `CField_MonsterCarnival::` | `monster/carnival/{clientbound,serverbound}` |
| `CWvsContext::` | `character/clientbound` |
| `CUserLocal::` | `character/serverbound` |

> **Tier-1 prefix caveat.** A packet is tier-1 (`docs/packets/evidence/tiers.yaml`)
> by directory *prefix*. Monster Carnival packets go under `monster/carnival/...`
> specifically so the `monster/` tier-1 prefix still matches. A top-level
> `carnival/` package would **not** be tier-1 and could be wrongly promoted by
> a flat-diff verdict. Place a new op so its existing tier prefix is preserved.

Both directions are implemented for every op (the round-trip test drives both).
Follow the immutable-model shape exactly: private fields, a constructor, getters
only (no setters), `Operation()`, `String()`, `Encode`, `Decode`.

### Worked example — clientbound `MOB_CRC_KEY_CHANGED`

`libs/atlas-packet/monster/clientbound/mob_crc_key_changed.go`:

```go
package clientbound

const MobCrcKeyChangedWriter = "MobCrcKeyChanged"

// Byte layout (IDA-verified, identical across all 5 versions — a single Decode4):
//   - crcKey : uint32 (CInPacket::Decode4 → this->m_dwMobCrcKey)
// IDA basis: CMobPool::OnMobCrcKeyChanged — v83 @0x6797be, v87 @0x6b5399, v95 @0x657230.
type MobCrcKeyChanged struct {
	crcKey uint32
}

func NewMobCrcKeyChanged(crcKey uint32) MobCrcKeyChanged { return MobCrcKeyChanged{crcKey: crcKey} }

func (m MobCrcKeyChanged) CrcKey() uint32    { return m.crcKey }
func (m MobCrcKeyChanged) Operation() string { return MobCrcKeyChangedWriter }
func (m MobCrcKeyChanged) String() string    { return fmt.Sprintf("crcKey [%d]", m.crcKey) }

func (m MobCrcKeyChanged) Encode(l logrus.FieldLogger, _ context.Context) func(map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.crcKey)
		return w.Bytes()
	}
}

func (m *MobCrcKeyChanged) Decode(_ logrus.FieldLogger, _ context.Context) func(*request.Reader, map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.crcKey = r.ReadUint32()
	}
}
```

`Decode` is the exact mirror of `Encode`, field for field, in the same order —
that symmetry is what the round-trip test in Step 4 proves.

### Version branching

Per-version structural variants live **inside** `Encode`/`Decode`, branching on
`ctx` (`tenant.MustFromContext(ctx)` → `t.Region()`, `t.MajorAtLeast(n)`), not
as separate types — unless a version diverges enough to warrant its own model
(decide per op; default is a single model with branches). A codec with no
version delta can ignore `ctx` (note the `_ context.Context` above). Reuse
shared sub-structs from `libs/atlas-packet/model/` (`Movement`,
`MonsterTemporaryStat`, …) rather than re-deriving them.

### Empty-payload serverbound op

Many serverbound acks carry the opcode only — zero wire fields. Model it
honestly: an empty struct whose `Encode`/`Decode` touch no bytes. From
`libs/atlas-packet/monster/serverbound/mob_crc_key_changed_reply.go`:

```go
const MobCrcKeyChangedReplyHandle = "MobCrcKeyChangedReply"

// EMPTY payload — the reply carries the opcode only. The send site builds the
// COutPacket and SendPacket()s it with no Encode* calls.
type MobCrcKeyChangedReply struct{}

func (m MobCrcKeyChangedReply) Operation() string { return MobCrcKeyChangedReplyHandle }
func (m MobCrcKeyChangedReply) String() string    { return "" }
// Encode/Decode bodies read/write nothing.
```

---

## Step 3 — Wire into atlas-channel + the seed templates

Three edits per op: register in `main.go`, add the channel-side
writer/handler glue, and route the opcode in all nine templates.

### Clientbound

1. Append the writer const to `produceWriters()` in
   `services/atlas-channel/atlas.com/channel/main.go` (sorted near its
   siblings):

   ```go
   monstercb.MobCrcKeyChangedWriter,
   ```

2. Add a thin `socket/writer/<op>.go` `Body` helper that builds the model and
   returns its `Encode`:

   ```go
   func MobCrcKeyChangedBody(crcKey uint32) packet.Encode {
       return func(l logrus.FieldLogger, ctx context.Context) func(map[string]interface{}) []byte {
           return func(options map[string]interface{}) []byte {
               return monsterpkt.NewMobCrcKeyChanged(crcKey).Encode(l, ctx)(options)
           }
       }
   }
   ```

   > **The "no emitter" seam is intentional, not dead code.** A clientbound
   > writer with no caller is deliberate: the codec + route exist so the
   > feature can be switched on later without another packet-plumbing pass.
   > The `Body` helper's doc comment must say so (see the example above), and
   > the backend-guidelines reviewer must be briefed that an uncalled
   > clientbound writer is a documented seam (design decision D2), not a
   > finding. Do **not** add a producer/emitter stub to satisfy "dead code" —
   > that is explicitly out of scope.

### Serverbound

1. Register the handler func in `produceHandlers()` in `main.go`:

   ```go
   handlerMap[monstersb.MobCrcKeyChangedReplyHandle] = handler.MobCrcKeyChangedReplyHandleFunc
   ```

2. Add `socket/handler/<op>.go` that decodes and logs — **no action**:

   ```go
   func MobCrcKeyChangedReplyHandleFunc(l logrus.FieldLogger, ctx context.Context, _ writer.Producer) func(session.Model, *request.Reader, map[string]interface{}) {
       return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
           p := serverbound.MobCrcKeyChangedReply{}
           p.Decode(l, ctx)(r, readerOptions)
           l.Debugf("[%s] read [%s]", p.Operation(), p.String())
           // behavior: deferred (decode-and-log only)
       }
   }
   ```

   Gameplay behavior (acting on the decoded packet) is a separate later task;
   this batch lands decode-and-log only.

### Route in all nine seed templates

In each of `services/atlas-configurations/seed-data/templates/template_{gms_48,gms_61,gms_72,gms_79,gms_83,gms_84,gms_87,gms_95,jms_185}_1.json`,
insert the route in sorted-opcode position, using the **per-version opcode**
from `docs/packets/registry/<version>.yaml` (read it per file — opcodes shift
between versions; see the v84 opcode-table shift below).

Clientbound goes in `socket.writers[]`:

```json
{ "opCode": "0xF9", "writer": "MobCrcKeyChanged" }
```

Serverbound goes in `socket.handlers[]` and **must carry a `validator`**:

```json
{ "opCode": "0xA4", "validator": "LoggedInValidator", "handler": "MobCrcKeyChangedReply" }
```

> **Validator-mandatory — the `BuildHandlerMap` silent-drop trap.** A handler
> entry whose `validator` is missing or names a validator that isn't
> registered is silently dropped — `BuildHandlerMap` `continue`s past it with
> only a warning, so the handler never registers and the client action
> no-ops. Every `socket.handlers` entry needs a `validator`:
> `LoggedInValidator` by default, `NoOpValidator` only for connection-level
> ops (Pong / StartError). Only those two validators exist; do not introduce a
> new validator type. (task-092 routed 52 serverbound handlers, all
> `LoggedInValidator`.)

> **Opcode tables shift between versions — especially v84.** The v84 client
> inserted opcodes into the clientbound/serverbound tables relative to v83
> (cumulative `+2…+10` above ~0x3D), so a v84 opcode is *not* the v83 opcode
> even though the payload is identical. Always read the opcode from the v84 IDB
> dispatcher, not by copying the v83 row. (See memory
> `bug_v84_opcode_table_shifted_vs_v83`.)

---

## Step 4 — Verify (promote the cell)

This is `VERIFYING_A_PACKET.md` applied to the codec you just wrote. Briefly:

1. **Test** — add `libs/atlas-packet/<family>/<dir>/<op>_test.go`. Round-trip
   across `pt.Variants` with `pt.RoundTrip(t, ctx, input.Encode, input.Decode, nil)`,
   **plus** an explicit golden-byte assertion for the v83 baseline citing the
   decompile line per field. Round-trip proves encode/decode symmetry;
   golden-byte proves byte-exactness vs the client. You need both.

   ```go
   // packet-audit:verify packet=monster/clientbound/MonsterMobCrcKeyChanged version=gms_v83 ida=0x6797be
   // packet-audit:verify packet=monster/clientbound/MonsterMobCrcKeyChanged version=gms_v84 ida=0x690354
   // packet-audit:verify packet=monster/clientbound/MonsterMobCrcKeyChanged version=gms_v87 ida=0x6b5399
   // packet-audit:verify packet=monster/clientbound/MonsterMobCrcKeyChanged version=gms_v95 ida=0x657230
   // packet-audit:verify packet=monster/clientbound/MonsterMobCrcKeyChanged version=jms_v185 ida=0x6f8bcb
   func TestMobCrcKeyChanged(t *testing.T) {
       input := NewMobCrcKeyChanged(0x12345678)
       got := input.Encode(nil, pt.CreateContext("GMS", 83, 1))(nil)
       want := []byte{0x78, 0x56, 0x34, 0x12} // crcKey uint32 LE (Decode4 @0x6797be)
       if !bytes.Equal(got, want) { t.Fatalf(...) }
       for _, v := range pt.Variants {
           t.Run(v.Name, func(t *testing.T) {
               ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
               pt.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
           })
       }
   }
   ```

   One `// packet-audit:verify` marker **per applicable version**, above the
   test func. The `ida=` address is that version's export address for the op's
   fname. A version where the op is genuinely absent gets **no** marker — it is
   recorded `VERSION-ABSENT` with IDB evidence, never a silent skip.

2. **Pin evidence** (tier-1 — every MOB/MONSTER op is tier-1), once per
   applicable version:

   ```
   go run ./tools/packet-audit evidence pin --packet <id> --version <key> \
     --ida "<FName>" --category TIER1-FIXTURE
   ```

   `--ida` is the fname exactly as it keys the export's `functions` map
   (fully-qualified, e.g. `CMobPool::OnMobCrcKeyChanged`), not a hex address —
   the tool resolves the address itself. Then open the generated
   `docs/packets/evidence/<version>/<packet dots>.yaml` and add the `verifies:`
   list manually pointing at `<test file>#<TestName>`.

3. **Regenerate + check:**

   ```
   go run ./tools/packet-audit matrix
   go run ./tools/packet-audit matrix --check
   ```

   Confirm the op's row flipped to ✅ for every applicable version and that the
   run introduces **no new** orphan/dangling/stale/drift lines mentioning your
   packet and does not increase the conflict count. Commit the test, the
   evidence YAMLs, and the regenerated `STATUS.md`/`status.json` together.

---

## After it merges — roll out to live tenants

Seed templates apply only at tenant **creation**; existing tenants do **not**
pick up new opcodes automatically (memory `bug_new_opcodes_not_in_live_tenant_config`).
A merged codec is dormant in production until you:

1. PATCH each live tenant's config — add the new `socket.handlers` (with
   validator) and `socket.writers` entries, using that version's opcodes.
2. **Restart `atlas-channel`** — the handler/writer map is built once at
   startup; the config projection does not hot-reload handlers/writers.
3. Post-deploy checks: `grep "Unable to locate validator"` == 0; no new
   error/fatal logs; the serverbound ops no longer emit "unhandled message op
   0xXX".

A task that adds packets should ship a `deploy-notes.md` with the full
per-version opcode table in PATCH shape (task-092's is the reference).

---

## Cross-references

- [`audits/VERIFYING_A_PACKET.md`](audits/VERIFYING_A_PACKET.md) — verifying an
  existing codec (Step 4 in depth, failure modes, dispatcher-family trap).
- [`evidence/tiers.yaml`](evidence/tiers.yaml) — tier-1 membership; governs
  which cells need an evidence pin and the directory-prefix rule.
- [`registry/README.md`](registry/README.md) — registry schema, provenance
  values (`csv-import` / `ida-discovered` / `manual`), and the v84-seeded-from-v83
  note.
- `docs/packets/audits/STATUS.md` / `status.json` — the coverage matrix the
  whole pipeline grades against.

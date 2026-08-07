# task-191 — Implementation Context

Companion to [`plan.md`](plan.md). Everything an implementer needs that is *not* a plan step:
where the code lives, why the decisions were made, and the traps that have already bitten.

Sources: [`prd.md`](prd.md), [`design.md`](design.md), [`current-state.md`](current-state.md), and
the code as observed on branch `task-191-v92-v95-movement-types` on 2026-08-04.

---

## 1. Where the work happens

| Path | Role |
|---|---|
| `.worktrees/task-191-v92-v95-movement-types` | The worktree. Branch `task-191-v92-v95-movement-types`. **All** edits go here. |
| `libs/atlas-packet/model/movement.go` | The generic movement codec — the consumer of the `types` config. The only Go file changed. |
| `libs/atlas-packet/model/movement_test.go` | The declared **byte oracle** for the move-path blob. |
| `libs/atlas-packet/test/` | `CreateContext(region, major, minor)`, `Encode(...)`, `RoundTrip(...)`, and `Variants` (the tenant matrix every wrapper fixture iterates). |
| `services/atlas-configurations/seed-data/templates/template_gms_*.json` | The 11 tenant socket-config seed templates. Three are edited. |
| `tools/template-opcode-order-guard.sh` | The existing template guard — the model for the new one. |
| `.github/workflows/pr-validation.yml` | Guard CI registration: job (~:313), `needs:` (:713), rollup (~:734). |
| `docs/packets/registry/gms_v95.yaml:2290-2294` | The `MOVE_PLAYER` serverbound row. |

---

## 2. How movement decode actually works

`libs/atlas-packet/model/movement.go`:

- **`Movement.Decode` (:33-64)** — reads `StartX` (int16), `StartY` (int16), then a one-byte element
  count, then loops. Per fragment it reads a one-byte element type and calls `isMovementType(...)`
  six times in an if/else chain (`NORMAL` → `TELEPORT` → `START_FALL_DOWN` → `FLYING_BLOCK` → `JUMP`
  → `STAT_CHANGE`), falling through to the bare `Element` decoder.
- **`movementPathAttrFromOptions` (:284-312)** — indexes `options["types"]` **by the element-type
  byte** and returns `(Name, Type)`. Returns `("NOT_FOUND", "DEFAULT")` and logs at error level when
  `types` is missing, is not an array, is empty, the index is out of range, or the entry is not an
  object.
- **`DEFAULT` matches no branch**, so it takes the `Element` fallback: `BMoveAction` (byte) +
  `TElapse` (int16) = 3 bytes, against a real fragment of 9–15 bytes. **That desync is the bug.**
- **`NormalElement.Decode` (:118-138)** reads `FhFallStart` only when the looked-up `Name` is
  `FALL_DOWN` (:126-128), and `XOffset`/`YOffset` only when `!t.IsRegion("GMS") || t.MajorAtLeast(88)`
  (:132-135). The `Encode` half mirrors both exactly (:209-230). **That mirroring is a hard
  invariant** — a one-sided gate corrupts Atlas's own outbound packets.

Consequence: the `types` table is *tenant configuration*, not Go code, because the wire layout of a
fragment is entirely version-specific. One array per move handler, per client version.

## 3. What is broken today

From [`current-state.md`](current-state.md) — `len(options.types)` per move handler:

| template | Character | Monster | Pet | Summon |
|---|---|---|---|---|
| `gms_12_1` | 9 | 9 | — | 9 |
| `gms_48_1` | 23 | 23 | 23 | **None** ← FR-4 |
| `gms_61_1` … `gms_83_1` | 23 | 23 | 23 | 23 |
| `gms_84_1` | 24 | 24 | 24 | 24 |
| `gms_87_1` | 25 | 25 | 25 | 25 |
| `gms_92_1` | **None** | **None** | **—** | **None** |
| `gms_95_1` | **—** | **None** | **None** | **None** |
| `jms_185_1` | 33 | 33 | 33 | 33 |

Plus two defects the PRD did not anticipate, both found during design and both upstream of the
`types` fix:

- **The v92/v95 header gained `vx`/`vy`.** Four `int16` precede the element count where `movement.go`
  reads two — the fix adds `Movement.StartVx` and `Movement.StartVy` behind a v88+ GMS gate. With them unread, `numElems` is parsed out of the low byte of `vx` and everything after
  is garbage — *regardless* of how correct `types` is. This is why PRD §2's "no `movement.go` change"
  non-goal had to be overridden: shipping `types` alone leaves v92/v95 just as broken, only failing
  silently instead of loudly.
- **v92's summon opcode block is mis-registered.** The template has `SummonMoveHandle 0xC8`;
  derivation says summon move is `0xCC`. Under the v95−3 correspondence for that region of the table,
  `0xC8`/`0xC9` would be v92's `PetItemUseHandle`/`PetItemExcludeHandle` — so `SummonAttackHandle`
  (`0xC9`) and `SummonDamageHandle` (`0xCA`) are suspect too, not just the move entry.

## 4. Decisions and why

### 4.1 The table is derived, never copied (PRD FR-1.5)

The v92/v95 table is **not** a right-extension of v87's — it is renumbered:

| Semantic | v83/v84/v87 index | v92/v95 index |
|---|---|---|
| `STAT_CHANGE` | 10 | **9** |
| `START_FALL_DOWN` | 14 | **11** |
| `FALL_DOWN` | 15 | **12** |
| `FLYING_BLOCK` | 23 (v84+) | **17** |

Had v87's array been copied into v92, index 15 (`FALL_DOWN`/`NORMAL` there) would decode as a 15-byte
`NORMAL` element what the v92 client sends as a 3-byte default element — **worse** than the current
bug, because it is silent. Cross-version continuity is a *check*, not a source.

### 4.2 Only five indices get a name

`Name` is load-bearing in exactly one place: `FALL_DOWN`. Because the table was renumbered, the
v83-era semantic names (`IMPACT`, `ASSAULTER`, `RUSH`, `SIT_DOWN`, `ARAN_ADJUST`, …) cannot be carried
across by index, and the client exposes no name strings — the attribute is a bare `int` throughout.
Inventing one per index would be exactly the fabrication CLAUDE.md's grounding rule forbids. So:
0 `NORMAL`, 9 `STAT_CHANGE`, 11 `START_FALL_DOWN`, 12 `FALL_DOWN`, 17 `FLYING_BLOCK`, everything else
`UNKNOWN` — which is how the existing templates already treat unresolved indices.

*Rejected:* recovering names from a client `MPA_*` enum. The v95 IDB is PDB-backed, but a PDB carries
no enumerator names for a value passed as a plain `int`. Nothing to recover.

### 4.3 The header gate is `IsRegion("GMS") && MajorAtLeast(88)` — a different shape from its neighbour

This is the single most dangerous line in the change. `XOffset`/`YOffset` twelve lines below uses
`!t.IsRegion("GMS") || t.MajorAtLeast(88)` — which **includes** JMS. The header gate must **exclude**
JMS: jms v185 `CMovePath::Encode` @`0x70b6c4` was read directly and writes the two-field header.
Reusing the `XOffset` predicate by reflex breaks JMS movement — a live regression on a
currently-working version. The plan's Task 2 test pins JMS v185 to the 5-byte header for exactly this
reason.

**Boundary 88 vs 92.** Observed: v87 no, v92 yes. v88–v91 have no IDB, so the exact boundary is
unobservable. 88 was chosen for consistency with the adjacent `XOffset` gate (same client rework), and
because `deploy/k8s/base/versions.json` ships no GMS version between 87 and 92 — the two constants are
behaviourally indistinguishable for every tenant Atlas can serve. Recorded in the code comment so a
future v88–v91 bring-up knows the constant was *chosen*, not measured.

### 4.4 A permanent guard, not a one-off (user decision)

FR-5 asks for a mechanical check across all 11 templates. **This exact defect has now shipped twice** —
task-179 fixed v48/61/72/79 and missed v48's Summon handler; the v92/v95 templates were seeded with no
`types` at all. A check that runs once cannot catch the third occurrence. The marginal cost over a
one-off is the CI wiring, a known four-touchpoint pattern in this repo.

The guard's `Type`-allowlist check is the one that catches the *silent* failure mode: a typo'd `Type`
degrades that one index to the 3-byte generic decode with **no log line** — the original bug in
miniature.

### 4.5 The whole v92 summon block is in scope (user decision)

Design §5.3 only proposed correcting `SummonMoveHandle`. The planning-phase template survey showed the
adjacent `SummonAttackHandle`/`SummonDamageHandle` sit at the same suspect offsets, so the user
approved deriving and correcting all three. If a real, Atlas-routed v92 handler legitimately owns
`0xC8`/`0xC9`/`0xCA`, that is a **stop-and-ask**, not a silent overwrite.

### 4.6 The v95 registry row: fname only

`docs/packets/registry/gms_v95.yaml` records serverbound `MOVE_PLAYER` at opcode 44 (`0x2C`) with
`fname: CUserLocal::OnKey`. The PRD suspected the opcode because `0x2C` sits *below* v92's `0x2E`,
breaking the otherwise monotonic drift of this opcode across versions. Derivation says the **opcode is
right** — the serverbound opcode table simply is not monotonic across versions. The `fname` is the
misleading part, and the true sender (`CVecCtrlUser::EndUpdateActive`) is already in that row's
`fname_alts`. So: promote the alt to `fname`, demote `CUserLocal::OnKey` to `fname_alts`, leave the
opcode alone.

---

## 5. Traps

### 5.1 IDA

- **Symbols are stored MANGLED.** `lookup_funcs("CMovePath::Encode")` returns "Not found" — every
  time. Use `func_query {queries:[{name_regex:"Encode@CMovePath"}]}` (regex over the *mangled* name),
  take the returned `addr`, and operate by address. The `name`/`name_contains`/`name_substr` keys are
  silently ignored by `func_query` — they dump the first 50 functions from `0x401000` instead of
  filtering.
- **Two endpoints.** `http://192.168.20.3:8745/mcp` is the session server (every IDB at once,
  `database` param per call) — **use this one**. `:13337/mcp` is the old per-port server and rejects
  `database`. `127.0.0.1` does not work from WSL; use the Windows host IP. Confirmed reachable
  2026-08-04 (HTTP 200 on `initialize`).
- **Session ids rotate.** Always `idb_list` first and match by **filename**. Never hardcode a session
  id or a version→port mapping.
- **Cross-check the address.** Before trusting a decompile, confirm the address `func_query` returned
  equals the one cited in the design. Addresses are per-version and per-IDB.
- **Silent truncation.** A short `func_query`/`func_profile` page is **not** end-of-data — loop until
  empty. `xrefs_to`/`xref_query`/`insn_query` cap around 10 results with rich output. `callees` with a
  large `addrs` batch silently drops entries.
- **Name what you reverse.** Rename the v92 `sub_*` senders to their v95 mangled symbols so cited
  addresses resolve on re-read (CLAUDE.md RE discipline). `dir:"vibe"` in the rename output means it
  took.

### 5.2 Templates

- **`CharacterInventoryMoveHandle` is not a move handler.** Despite the name, it is inventory item
  movement and correctly carries no `types`. Excluded from the invariant and from the guard.
- **`template_gms_92_1.json` has no trailing newline.** All three are LF. Do not normalize.
- **Ordering is enforced.** `tools/template-opcode-order-guard.sh` requires strictly ascending
  `opCode` in both `handlers` and `writers`. New entries go at their sorted position.
- **A handler entry without a `validator` is silently dropped at load time** — no error, no log. It
  looks configured and never fires.
- **Near-identical neighbours.** v95's pet block (`0xC7` … `0xCC`) is six five-line entries that
  differ only in `opCode` and `handler`. Always include the `"handler": "…"` line in an Edit's
  `old_string`.

### 5.3 Go

- **Encode/Decode must stay textually identical** for every version gate in `movement.go`.
- **`Movement.Decode` and `Movement.Encode` do not currently resolve a tenant** — both need
  `t := tenant.MustFromContext(ctx)` added *outside* the returned closure, matching
  `NormalElement.Decode`'s existing shape (:118-120).
- **`test.Variants` already includes GMS v92 (index 11) and GMS v95**, and several wrapper fixtures
  iterate it. Those fixtures assert the move-path blob **equals** `model.Movement.Encode`'s output
  rather than pinning literal bytes, so they should stay green — but run them.
- **The one literal-byte movement fixture** (`monster/clientbound/movement_test.go`
  `TestMonsterMovementBytesV79`) is at v79, below the gate. Unaffected.
- **`goimports` trap:** package `tenant` lives in directory `atlas-tenant`, and goimports can
  duplicate the import. `movement.go` already aliases it
  (`tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"`) — leave the alias.
- **No `go.mod` changes**, so no `docker buildx bake` round is required (CLAUDE.md item 4 applies only
  when a `go.mod` is touched). Verify with `git diff --name-only main | grep go.mod`.

### 5.4 Live reconcile

- **Seed templates apply ONLY at tenant provisioning.** Fixing the template does nothing for an
  existing tenant.
- **PATCH is a FULL REPLACE** of the tenant configuration JSON (`handleUpdateConfigurationTenant` →
  `UpdateById`), and it enqueues a tenant-status event. The body must carry complete attributes —
  but the *content* must be the live config with only the move handlers changed.
- **Never swap in the template's whole `socket` block.** That also applies unrelated opcode
  relocations and rewrites `operations` mode tables (client wire values needing their own IDA
  provenance), and silently reverts tenant-specific customization.
- **The PATCH response is not evidence.** A handler missing its `validator` is accepted at the
  transport layer and dropped at load time. Verify with a fresh `GET`.
- **atlas-channel must be restarted.** Handler/writer maps are built at listener-creation time and the
  configuration projection's `ListenerConfig` diff **excludes** handlers/writers, so a handlers-only
  change does not hot-reload.
- **Benign residual warnings.** "Service declares writer [...] but tenant config has no opcode
  mapping" is a template-completeness gap, not caused by the PATCH.
- Recorded live tenants as of 2026-07-30 (**verify, do not trust**): v92 `db1dbfb3…` (parked),
  v95 `c794c706…`, namespace `atlas-main`, ingress `dev.atlas.home` → LB `192.168.23.230`.

---

## 6. Resolved unknowns (do not re-open)

| Question | Answer | Evidence |
|---|---|---|
| v95 `CharacterMoveHandle` opcode | `0x2C` (44) — the registry value was right | `CVecCtrlUser::EndUpdateActive` @`0x9a0d20`, `COutPacket(44)` @`0x9a0ee3` |
| v92 `PetMovementHandle` opcode | `0xC4` | `sub_9781A0`; both v92 and v95 `EncodeBuffer(petLockerSN, 8)` before `Flush` |
| v92 `SummonMoveHandle` opcode | `0xCC` (not `0xC8`) | `sub_9792D0` @`0x97932d`; both encode `Encode4(dwSummonedID)` before `Flush` |
| Array length | 37 for both v92 and v95 — neither 25 (v87) nor a jms-like 33 | max case `0x24` in both `CMovePath` switches |
| Are the rand-count fields on the wire? | **No.** v92/v95 read `usRandCnt`/`usActualRandCnt` only when `CClientOptMan::GetOpt(2)` is truthy (v95 `0x667a57`); `GetOpt` @`0x4ac700` returns 0 for absent keys and `m_mOpt` is populated only by `DecodeOpt` @`0x4acb30` from a server-sent option list. Atlas writes a zero option count. | design §9 |
| Do v92 and v95 differ? | No — `Encode` bodies both `0x552` bytes, `Decode` bodies both `0x31e`, case-for-case identical | design §3 |

**Coupling to remember:** if Atlas ever sends a non-empty client-option list including type 2,
`movement.go` must gain the two `usRandCnt`/`usActualRandCnt` reads per element.

---

## 7. Out of scope

- Promoting packet coverage-matrix cells (`docs/packets/audits/STATUS.md`). This task fixes
  configuration, not codecs; no byte-fixture campaign. The existing
  `packet-audit:verify … version=gms_v95` markers on the movement fixtures pinned a header layout the
  v95 client does not read — a known "round-trip fixture ≠ client-validated" instance. The correction
  is *recorded* in `movement-types-derivation.md` so a later matrix pass has the evidence.
- Any change to the `XOffset`/`YOffset` gate (`movement.go:132-135`, `:223-226`) — already correct and
  already covers v92/v95.
- Any change under `services/atlas-channel/` — runtime beneficiary only.
- `template_gms_12_1.json` — 9 entries, consistent across its three move handlers (no pet handler),
  correct as-is.
- Live play-testing. The requester play-tests v92 and v95 separately; this task's bar is derivation
  evidence plus static invariants plus the reconcile read-back.

---

## 8. Definition of done

- `movement-types-derivation.md` records, per version and per index, the client function/address, the
  observed field set, and the resulting `Name`/`Type` — with no index guessed or inferred from a
  neighbouring template.
- `go test -race ./...` and `go vet ./...` clean in `libs/atlas-packet`.
- `tools/template-movement-types-guard.sh` demonstrated **failing** on the pre-fix tree (Task 3) and
  **passing** after (Task 6), and wired into CI.
- `tools/template-opcode-order-guard.sh` and `tools/lint.sh --check` exit 0.
- `git diff --stat main` confined to the file set in plan Task 9 Step 5; nothing under
  `services/atlas-channel/` or `libs/atlas-packet/` beyond `model/movement*.go`.
- Each live v92/v95 tenant read back with all four move handlers present, validated, and carrying a
  non-empty `types`; the "not configured for use in movement" error line absent from v92/v95 channel
  logs; `reconcile.md` written with the actual output quoted.
- Code review run and findings addressed **before** the PR is opened.

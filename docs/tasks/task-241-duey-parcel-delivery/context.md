# Duey Parcel Delivery — Implementation Context

Task: `task-241-duey-parcel-delivery`
Companion to: `prd.md`, `design.md`, `plan.md`
Created: 2026-08-19

---

## 1. What this feature is, in one paragraph

Duey is asynchronous player-to-player delivery: a sender addresses one item
stack and/or a meso amount to a character by name, pays a tiered fee, and walks
away; 24 hours later the recipient can receive or discard it from a mailbox
dialog. Atlas has none of it today. The work spans a new service
(`atlas-parcel`), two packet families (`PARCEL` clientbound dispatcher,
`DUEY_ACTION` serverbound), four new saga custody actions with two saga types,
an NPC entry point, a cash-item entry point, two background sweeps, and a
world-transfer eligibility gate.

## 2. The three decisions that shape everything else

**Custody, not destroy/award.** The PRD sketched `DestroyAssetFromSlot` on send
and `AwardAsset` on receive. Those do not compose into a custody transfer —
`AwardAsset` mints a fresh row from a template id, so scrolled stats, item
level/EXP, owner tag, lock/karma flags, expiration, cash serial and ring id are
all lost. The design replaced them with the four-action protocol storage, trade,
cash shop and MTS already use (`libs/atlas-saga/model.go:163-232`). This is why
Tasks 11–15 exist at all, and it is why the NFR-2 rollback test (Task 14, the
`send fails at accept` case) is the single most valuable test in the plan: a
re-awarded item loses its stats and a released/re-accepted one does not.

**A new service, not a fold into `atlas-merchant`.** The entity shape is nearly
identical to `frederick`, which is tempting, but `atlas-merchant` is the
hired-merchant service, its status topic is merchant-scoped, and gate 10
(`merchant_open`) already routes there. Gate 12 needs a dependency that can be
down and fail closed with its own distinct reason — a narrow service is a narrow
blast radius. The cost is the full `docs/adding-a-new-service.md` checklist,
which is Task 5 and the largest single block of non-feature work.

**The fee is computed in floating point, deliberately contradicting NFR-8.**
NFR-8 asked for integer arithmetic (`m * 18 / 1000`). The client computes the
fee itself in IEEE-754 double and shows it to the player in the confirm dialog
*before* the packet is sent (v72 `sub_6590A1` @0x6590A1, v83 `sub_6EEDFE`
@0x6EEDFE). An integer-derived charge against a double-derived quote charges the
player a number they were not shown. Task 16 pins the exact truncated value at
all twelve tier boundaries.

## 3. Key files by area

| Area | Files |
|---|---|
| New service | `services/atlas-parcel/atlas.com/parcel/**` (Tasks 1–4, 15, 23, 24) |
| Service registration | `.github/config/services.json`, `docker-bake.hcl`, `go.work`, `deploy/k8s/**`, `deploy/shared/routes.conf`, `tools/db-bootstrap.sh` (Task 5) |
| Packet record | `docs/packets/dispatchers/parcel.yaml`, `duey_action.yaml`, `docs/packets/registry/gms_v72.yaml`, `gms_v79.yaml` (Tasks 6, 28) |
| Codecs | `libs/atlas-packet/parcel/**`, `tools/packet-audit/cmd/run.go` (Tasks 7–10) |
| Saga contract | `libs/atlas-saga/{model,payloads,unmarshal}.go` (Tasks 11, 19) |
| Orchestrator | `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/{saga,parcel,kafka}/**` (Tasks 12–14, 19) |
| Channel wire | `services/atlas-channel/atlas.com/channel/{parcel,socket/handler,kafka/consumer/parcel}/**` (Tasks 16–18, 21, 22, 25) |
| NPC entry | `services/atlas-npc-conversations/.../operation_executor.go`, `deploy/seed/*/*/npc-conversations/npc/npc-9010009.json` (Task 20) |
| World transfer | `services/atlas-character/.../pending_change/{processor_eligibility,requests}.go`, nine seed templates (Tasks 26, 27) |

## 4. Patterns each area copies

- **atlas-parcel structure** → `services/atlas-merchant/atlas.com/merchant/frederick`
  (entity with `TenantId` + jsonb snapshot, `Migration`, `task.go` /
  `notification_task.go` `Run()`+`SleepTime()` pair)
- **atlas-parcel service shell** → `services/atlas-mts/atlas.com/mts/main.go`
  (bootstrap, migrations, consumer registration, periodic task, route
  initializers)
- **Custody protocol** → `atlas-mts`: `libs/atlas-saga/payloads.go:844-895`,
  `saga/processor.go:1943-2130` (expansions), `saga/handler.go:2427-2510`,
  `saga/compensator.go:2340-2430`, `kafka/message/mts/custody/kafka.go`,
  `services/atlas-mts/.../kafka/consumer/custody/consumer.go`
- **Dispatcher family** → `libs/atlas-packet/note/clientbound/operation.go` +
  `operation_body.go` (discrete struct, `WithResolvedCode` with a fixed key),
  `docs/packets/dispatchers/note_operation.yaml`
- **Serverbound mode dispatcher** → `libs/atlas-packet/storage/serverbound/operation.go`
  and `services/atlas-channel/.../socket/handler/storage_operation.go`
- **Pre-flight-then-saga handler** → `services/atlas-channel/.../socket/handler/note_send.go`
- **Cash-item classification branch** → `character_cash_item_use_remote_merchant.go`
- **Notification chain** → `frederick/notification_task.go` →
  `services/atlas-channel/.../kafka/consumer/merchant/consumer.go:454`
- **Eligibility gate** → `processor_eligibility.go`'s gate 11 `checkMtsHolding`
  plus `requests.go:184` `mtsHoldingOpen`

## 5. Dependency order between tasks

```
1 → 2 → 3 → 4 ─┬─────────────────────────────→ 26 → 27
               │
        5 (registration; needs 1's main.go)
               │
6 → 7 → 8 → 9 → 10 ──────────────────────────→ 28
               │
11 → 12 → 13 → 14                              (14 needs 13's processor)
      └──→ 15 (needs 11's payload shapes + 3)
               │
16 → 17 → 18   (17 needs 10, 8, 9, 11, 16)
19 → 20        (20 needs 19's action)
19 → 21        (21 needs 7's OPEN body + 17's REST client)
19 → 22
23, 24 (need 2, 3) → 25 (needs 24 + 9)
```

Tasks 6 and 11 have no predecessors and can start in parallel with Task 1.
Task 28 is last: the matrix grades committed fixtures, so every fixture must
already exist.

## 6. Tasks deliberately left large, and why

- **Task 5 (service registration, ~13 files).** This is
  `docs/adding-a-new-service.md` end to end. Splitting it is worse than leaving
  it large: the doc exists precisely because atlas-mts (task-121) was registered
  in some lists and not others, and every miss was invisible until runtime
  (`:latest` images the bump workflow silently skips, `configMapGenerator`
  `behavior: replace` dropping unlisted keys, topic env vars falling back to
  unsuffixed names with only a warn). A half-registered service passes CI. The
  task ends on `tools/service-registration-guard.sh` exiting 0, which is the
  machine check that makes the size safe.
- **Task 20 (eight identical NPC seed files).** The same mechanical file per
  version — Step 5a's explicit batching case.
- **Task 27 (nine template edits).** Kept as one task on the design's own
  instruction (RISK-5): a missed file is silent, so the acceptance check greps
  all nine at once.
- **Tasks 12 and 14 both touch `saga/processor.go` / `handler.go` /
  `compensator.go`.** They are split by *what they add* (expansion vs
  handling/compensation) rather than by file, because a reviewer can
  meaningfully reject the expansion while approving the compensation. Expect the
  second of the pair to rebase cleanly; if it conflicts, the conflict is in
  adjacent added blocks, not in shared logic.

### `plan-lint` F4 warnings — which are real

`tools/plan-lint.sh` exits 0 (no errors) but reports 18 F4 warnings. It counts
every backticked path under `### Files`, including the "Patterns to copy" and
"Read-only references" bullets, so most counts are inflated:

| Task | Reported | Files the implementer edits | Verdict |
|---|---|---|---|
| 5 | 7 | ~13 (multi-path bullets) | genuinely large — see above |
| 15 | 10, 2 services | 7 (all in atlas-parcel) | the extra 3 and the second "service" are read-only atlas-mts references |
| 17 | 10, 2 services | 6 (all in atlas-channel) | ditto — the atlas-character reference is read-only |
| 20 | 10 | 10 (2 Go + 8 identical seed files) | batched, allowed by Step 5a |
| 21 | 7 | 5 | the extra 2 are read-only references |
| 23, 24 | 2 services | 1 (atlas-parcel) | the second "service" is a read-only frederick/mts reference |
| 27 | 9 | 9 (one mechanical edit each) | deliberate — see above |

Only Tasks 5, 20 and 27 are actually over the guideline, and each is justified
above. The one F5 warning (`checkParcelPending`) is the new gate-12 method Task
26 introduces; it is declared in that task's Interfaces block and correctly
resolves nowhere in the repo yet.

## 7. Open items the implementer must resolve, not assume

These are genuine unknowns the design deferred to derivation, each with the
address to derive from. None may be guessed.

- **The 234-byte `PARCEL` block's +29..233 layout** (Task 7 Step 1).
  Five offsets are pinned (+0 id, +4 name[13], +17 mesos, +21 FILETIME,
  +234 optional item); the remainder is derived from `PARCEL::Decode` @0x4E4345
  and `CTabReceive::SetParcel` @0x6EF69C. Unestablished sub-fields become
  explicit zero padding of the derived width, documented as such.
- **Every version column other than gms_v83** (Task 6 Step 3). The design's arm
  table is v83-derived; each other column re-derives. JMS is expected to shift
  (`storage_operation.yaml`'s jms column is the precedent for recording a shift).
- **The v79 `DUEY_ACTION` opcode** (Task 6 Step 2). v72 is closed at `0x040`
  (`CTabReceive::ReceiveParcel` @0x65AF41 builds `COutPacket(64)`;
  `CTabSend::SendParcel` @0x65D940 the same with mode 2). v79 is derived the
  same way. If the v79 build genuinely lacks it, that reduces the span and must
  be recorded in writing here, per the PRD's acceptance criterion.
- **RISK-4: the 30-day client guard's polarity** (Task 23 Step 1). v72's
  `ReceiveParcel` refuses a receive outside a 30-day window
  (`(parcelTime - now) / 864000000000 < 30`). If the client's window is the
  binding one, server expiry moves from 30 days to 29 so the server always
  retires a parcel before the client refuses it. **Record the finding and the
  decision in this file under a "RISK-4 resolution" heading when made.**
- **JMS coverage of classification 533** (Task 22 Step 4). `dueyCouponEnabled`'s
  JMS arm is derived from the JMS build's `get_cashslot_item_type` handling of
  533, the same way `remoteMerchantEnabled` derives its GMS span. Gate JMS off
  with a documented reason rather than guessing it on.

## 8. Things that are settled and must not be relitigated

- **Recipient resolution needs no new atlas-character endpoint.** The channel
  calls `character.NewProcessor(l, ctx).ByNameProvider(name)()` — tenant-scoped
  and name-filtered but not world-filtered — and filters to `s.WorldId()`
  itself. The model already exposes `WorldId()` and `AccountId()`
  (`services/atlas-channel/atlas.com/channel/character/model.go:241,269`), which
  is what makes the same-account check possible without a second lookup. This
  closed OQ-1 against the PRD's expectation.
- **Discard, expiry and return are NOT sagas.** All three stay inside
  atlas-parcel: one UPDATE plus, for expiry, one INSERT, against rows the
  service owns exclusively. The asset never re-enters an inventory, so there is
  no cross-service atomicity to protect and no compensation to define. Routing
  them through the orchestrator would add a distributed failure mode to a purely
  local one.
- **No GM restriction** (OQ-6). Cosmic gates Duey behind
  `MINIMUM_GM_LEVEL_TO_USE_DUEY`; no PRD goal requires it and no client arm
  expresses it.
- **No per-day meso accumulator** (OQ-5). Mode 0x11 is a *level* gate: level ≤ 15
  may send at most 1,000,000 meso per transaction (`sub_6F3875` @0x6F3875). The
  string calls it a per-day limit; the client enforces it per transaction only,
  and there is no wire support or client display for an accumulator.
- **No notification tier ladder** (contra frederick). A parcel has one arrival
  event and one 30-day death, so a single nullable `LastNotified` with one
  meaning — "the player has been told about this parcel once" — serves FR-24.
- **The channel always sends `ALARM_NAMED` (0x19), never `PARCEL_ARRIVED`
  (0x18).** Choosing 0x18 needs the channel to know the session has an open
  parcel dialog, which it does not track. A player with the dialog open sees a
  toast rather than a live row — documented, low severity (design §7.1).
- **Never disconnect on a malformed request** (NFR-5). Cosmic disconnects and
  autobans on the packet-edit cases; Atlas rejects with the matching result arm
  and logs at warn. Every rejection test in Tasks 17 and 18 asserts the session
  was not closed.

## 9. Two manual, out-of-repo steps

Neither is the implementer's job; both go in the PR description as operator
follow-ups (`docs/adding-a-new-service.md` §6.1 and §6b):

1. Create the `atlas-parcel-main` database on postgres.home before merge — main
   has no wave-0 create job and the pods crash-loop on SQLSTATE 3D000 until it
   exists.
2. After the first image push, flip the `atlas-parcel/atlas-parcel` GHCR package
   to Public, then delete the stuck pod. Every atlas package is public and the
   cluster pulls anonymously; a new package is created private, so the pod sits
   in `ImagePullBackOff` against a 401 while CI reports a clean build.

## 10. Definition of done

- Flagless `tools/verify.sh` exits 0 (only the flagless invocation counts).
- `packet-audit matrix --check`, `dispatcher-lint`, `fname-doc --check` and
  `operations --check` all exit 0, and `PARCEL` is absent from
  `dispatcher-lint-baseline.yaml`.
- `backend-guidelines-reviewer`, `plan-adherence-reviewer` and
  `packet-completeness-critic` all clear before the PR.

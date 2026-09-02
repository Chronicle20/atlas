# bug: Writ of Solomon (2370000) does nothing and gives the player no feedback

Task: task-277-stored-exp-items
Branch: `task-277-stored-exp-items` @ `0aeb0f33a`
Environment: `atlas-pr-1567` (PR #1567), all services on image tag `pr-1567-0aeb0f3`
Tenant: `dbf2f7ba-2533-4885-8cf2-2934697efdd3` — GMS, `ms.version` 83.1
Character: id 2, name `Atlas2`, level 1, `gachaponExperience` 0

## Reproduced

Yes — three times in the live PR environment, 2026-09-02 18:34:25 / 18:34:32 / 18:34:42 UTC.

`@award me item 2370000` → double-click the Writ in the USE compartment.

## Observed

The full serverbound path works. The failure is at the data read.

1. atlas-channel decodes the op and dispatches:
   `[CharacterItemUseSolomonHandle] read [updateTime [264813569], source [1], itemId [2370000]]`
   then `Character [2] using item [2370000] from slot [1]. quantity [1]`.
2. The compartment reserves the item (`RESERVED`, txn `4cc62cdf-…`).
3. atlas-consumables GETs `/api/data/consumables/2370000` and receives a `spec`
   object with **27 keys and no `exp`**, and `maxLevel: 0`.
4. `consumeSolomon` rejects:
   `Character [2] consumed Writ of Solomon [2370000] but its spec/exp is absent or
   non-positive; the tenant's Item.wz likely predates the spec/exp parse.`
5. `ConsumeError` cancels the reservation (`RESERVATION_CANCELLED`) and emits
   `{"characterId":2,"type":"ERROR","body":{"error":"CONSUME_FAILED"}}`.
6. atlas-channel receives that event and takes `actionUnstick` — an empty
   `StatChanged` and nothing else
   (`services/atlas-channel/atlas.com/channel/kafka/consumer/consumable/consumer.go:104-113,169-176`).

Net effect for the player: the item survives, no EXP is banked, and no message,
effect, or dialog is shown. Exactly the symptom reported.

## Expected

Using a Writ of Solomon banks its `spec/exp` (100000 for 02370000) into
`gachapon_experience`; if it is rejected, the player is told why.

## Root cause

**Two independent causes.**

### Cause 1 (why nothing happens): the stored atlas-data document is stale

atlas-data does not parse WZ per request. `document.DbStorage.Add`
(`services/atlas-data/atlas.com/data/document/db_storage.go:120-160`) persists the
marshalled `consumable.RestModel` as a JSON blob in the `documents` table at
ingest time, and `Storage.ByIdProvider` serves that blob verbatim thereafter
(registry cache → DB row → canonical-tenant fallback). Ingestion runs only when
an operator uploads/processes a WZ dataset — never at request time and never at
pod start.

The row for `2370000` in this environment was written by a **pre-task-277**
binary, so it carries neither the `spec/exp` key nor a non-zero `maxLevel`.
Commit `d321b2e92` ("feat(data): expose consumable info/maxLevel and spec/exp")
added `reader.go:152` `m.Spec[SpecTypeExperience] = …`, but a reader change is a
no-op for a tenant that is already ingested.

Verified at the atlas-data source, not just through the consumables client:

```
$ kubectl exec -n atlas-pr-1567 atlas-data-… -- wget -qO- \
    --header="TENANT_ID: dbf2f7ba-…" --header="REGION: GMS" \
    --header="MAJOR_VERSION: 83" --header="MINOR_VERSION: 1" \
    http://localhost:8080/api/data/consumables/2370000
…"maxLevel":0,…"spec":{"acc":0,"curse":0,"darkness":0,"eva":0,"expBuff":0,"hp":0,
"hpR":0,"ignoreContinent":0,"inc":0,"jump":0,"mad":0,"mdd":0,"morph":0,"moveTo":0,
"mp":0,"mpR":0,"onlyPickup":0,"pad":0,"pdd":0,"poison":0,"randomMoveInFieldSet":0,
"returnMapQR":0,"seal":0,"speed":0,"thaw":0,"time":0,"weakness":0}…
```

27 spec keys, all zero, `exp` absent — the spec block ran, so the binary is
simply an older reader's output preserved in the DB. The WZ source does carry
the values (`Item.wz/Consume/0237.img.xml`, node `02370000`):

```xml
<imgdir name="info"> <int name="price" value="1"/> <int name="maxLevel" value="50"/> </imgdir>
<imgdir name="spec"> <int name="exp" value="100000"/> </imgdir>
```

This is not a code defect on the branch. It is the exact "ingest-order caveat"
the task predicted (`prd.md:241-243`, `design.md:342-344,453-454`).

**Correction (post-review).** This section originally claimed no re-ingest entry
existed in `docs/TODO.md`. That was wrong: `## task-277 follow-up: tenant Item.wz
re-ingest for Writ of Solomon spec/exp` was already on the branch at
`docs/TODO.md:681`, added by ancestor commit `49d3a05ca`. The controller's grep
ran against the MAIN repo's `docs/TODO.md` rather than the worktree's and so
missed it. The genuine gap was narrower: the PRD acceptance box at `prd.md:331`
was still unchecked. `1e33a1a78` expanded the existing entry and ticked the box.
Do not cite the original claim elsewhere.

### Cause 2 (why there is no feedback): CONSUME_FAILED is silent by design

`consumeErrorType` maps every non-pet, non-inventory-full, non-Vega failure to
`ErrorTypeConsumeFailed`, and `consumableErrorAction` routes that to
`actionUnstick` — release the exclusive-request lock, send no message. So all
three Writ rejection paths (`no spec/exp`, `level > maxLevel`, `stored balance
non-zero`) are indistinguishable from a no-op at the client. `design.md:453-454`
acknowledged this as "the safe failure … a visible 'nothing happens'".

Note the level gate would not have fired here — `Atlas2` is level 1 and the
item's real `maxLevel` is 50 — but once the data is re-ingested it becomes the
common rejection, and it will be equally silent.

## Fix

### 1. Record the re-ingest follow-up (the PRD's own unchecked acceptance item)

- `docs/TODO.md` — add a `## task-277 follow-up: consumable WZ re-ingest for
  Writ of Solomon spec/exp` section. Mirror the task-219 morph-coupon entry at
  `docs/TODO.md:641` in shape and wording: state that documents ingested before
  `d321b2e92` carry neither `spec/exp` nor `info/maxLevel`, that every Writ of
  Solomon is therefore rejected (item preserved, no EXP banked) until Item.wz
  `Consume` is re-ingested, and give an unchecked action item to re-ingest for
  every provisioned tenant plus the canonical `GMS/83/1` dataset.
- `docs/tasks/task-277-stored-exp-items/prd.md:331` — tick the acceptance box
  once that entry exists.

### 2. Player-facing feedback per rejection reason (RULED IN by the user)

The user ruled that each rejection gets its own distinct message rather than the
silent unstick. Three reasons, three error types, three strings.

Use the **existing, already-wired** notice mechanism — do not add a new codec.
The precedent is the Water of Life failure notice:
`services/atlas-channel/atlas.com/channel/socket/handler/water_of_life.go:31-33`
declares the message text as an exported const and
`services/atlas-channel/atlas.com/channel/kafka/consumer/pet/consumer.go:561`
announces it via `charcb.CharacterStatusMessageWriter` +
`charpkt.CharacterStatusMessageOperationSystemMessageBody(text)`
(`libs/atlas-packet/character/status_message_body.go`, SYSTEM_MESSAGE operation).
Follow that shape exactly.

Files:

- `services/atlas-consumables/atlas.com/consumables/consumable/solomon.go` —
  replace the three `errors.New(...)` literals with package-level sentinel
  errors so `consumeErrorType` can classify them:
  `ErrSolomonNoExperience` (spec/exp absent or <= 0),
  `ErrSolomonLevelExceeded` (character level > `ci.MaxLevel()`),
  `ErrSolomonBalanceNotEmpty` (`c.GachaponExperience() != 0`).
  Keep the existing ordering contract and the Warn logs untouched — every check
  must still reject BEFORE `ConsumeItem`, so a rejected Writ is never destroyed.
- `services/atlas-consumables/atlas.com/consumables/kafka/message/consumable/kafka.go`
  (near `ErrorTypeConsumeFailed` / `ErrorTypePotionLocked`, lines 128-133) — add
  the three producer-side error-type constants.
- `services/atlas-consumables/atlas.com/consumables/consumable/processor.go:509-513`
  (`consumeErrorType`) — add three `errors.Is` arms ahead of the
  `ErrorTypeConsumeFailed` fallthrough.
- `services/atlas-channel/atlas.com/channel/kafka/message/consumable/kafka.go:96-103`
  — declare the same three constants consumer-side. **The literals must match
  the producer byte-for-byte**; this seam has no shared constant (see
  `status.md` "Residual risk"), so pin the strings in a test on both sides.
- `services/atlas-channel/atlas.com/channel/kafka/consumer/consumable/consumer.go`
  — add the three cases to `consumableErrorAction` (lines 116-131) and a new
  action arm alongside `actionInventoryFull` (lines 152-158) that announces the
  system message and then calls `unstick`. Declare the three message strings as
  exported consts following `WaterOfLifeFailedMessage`'s comment style.
- Tests: extend
  `services/atlas-channel/atlas.com/channel/kafka/consumer/consumable/consumer_test.go:25`
  (the table already asserts `"CONSUME_FAILED" -> actionUnstick`) with a row per
  new type; extend
  `services/atlas-consumables/atlas.com/consumables/consumable/solomon_test.go`
  to assert each rejection path emits its own error type AND still leaves the
  item unconsumed.

Do not change the `CONSUME_FAILED` -> `actionUnstick` default; it remains the
catch-all for unrelated failures.

### 3. Operational, NOT a code change — the user is handling this

Re-ingest Item.wz `Consume` against the task-277 build. atlas-data exposes
`PATCH /data/wz` (upload) and `POST /data/process` (start an ingest Job over the
already-stored dataset); `DbStorage.Add` upserts on
`(tenant_id, type, document_id)`, so a re-run rewrites the stale rows in place.
The user declined having this triggered for them. Until it runs, the
`ErrSolomonNoExperience` path is what a live test will hit — which is exactly
why it now needs its own message.

## Not yet answered

- **Message wording.** The three strings are new player-facing copy with no WZ
  or client-string source; the implementer should propose them in the
  `WaterOfLifeFailedMessage` register ("The Writ of Solomon had no effect. It has
  been returned to you." etc.) and flag them for the user's approval rather than
  treat them as verified game text.
- Whether the served document came from the per-tenant row or the canonical
  (`GMS/83/1`) fallback — not inspected. It determines which scope needs
  re-ingesting. `Storage.ByIdProvider` tries tenant first, then canonical.
- Whether any OTHER consumable family on this branch depends on a WZ field added
  after the environment's last ingest. Not swept; only `spec/exp` and
  `info/maxLevel` were confirmed missing.

## Resolution

- `1e33a1a78` — fix(consumables,channel): Writ of Solomon rejections get their
  own message. Implements `## Fix` 1 and 2: the `docs/TODO.md` re-ingest
  follow-up (line 681), `prd.md:331` ticked, three sentinel errors in
  `solomon.go`, three error-type constants on both sides of the Kafka seam,
  three arms in `consumeErrorType` / `consumableErrorAction`, and a new action
  that announces a SYSTEM_MESSAGE before unsticking. Module-local
  `go build ./... && go test ./...` green in both atlas-consumables and
  atlas-channel.
- `993b6af7b` — controller fix: the level-gate copy was inverted. `consumeSolomon`
  rejects when the character's level EXCEEDS `maxLevel` (02370000 caps at 50),
  but the string read "You are not experienced enough…", i.e. the opposite rule.
  Now "Your level is too high to use the Writ of Solomon."

**Message copy still needs the user's approval** — all three strings are new
player-facing text with no WZ or client-string source:
- `SolomonNoExperienceMessage` = "The Writ of Solomon has no effect."
- `SolomonLevelExceededMessage` = "Your level is too high to use the Writ of Solomon."
- `SolomonBalanceNotEmptyMessage` = "You already have stored EXP banked. Use it before using another Writ of Solomon."

**Live re-test: NOT done, and cannot be done until the re-ingest runs.** The user
took the re-ingest themselves. Until Item.wz `Consume` is re-ingested against
this build, `spec/exp` stays absent and every Writ still rejects — it now says so
instead of doing nothing, but banking EXP is unverified end-to-end.

Repo-wide gate and cross-seam review over `0aeb0f33a..993b6af7b`: dispatched,
verdicts recorded in `agent-ledger.tsv`.

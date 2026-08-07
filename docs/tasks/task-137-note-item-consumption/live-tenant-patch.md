# Live-Tenant PATCH: Note Operations/Errors Config (task-137)

Task: task-137-note-item-consumption
Design: `docs/tasks/task-137-note-item-consumption/design.md` (§1.3, §5.1, §5.3)

## 1. Why existing tenants need this

Seed templates under `services/atlas-configurations/seed-data/templates/`
(`template_gms_{48,61,72,79,83,84,87,95}_1.json`, `template_jms_185_1.json`)
apply **only at tenant creation** — a tenant provisioned before this change
already has a socket configuration persisted in atlas-tenants, and that
persisted config is what atlas-channel actually loads at runtime, not the
seed template. Updating the seed template alone does nothing for tenants
that already exist (known bug pattern — config drift between seed templates
and live tenant configuration). Every existing tenant on one of these nine
versions needs its socket configuration PATCHed directly.

The delta being deployed:

1. Clientbound `NoteOperation` (MEMO_RESULT) writer — `options.operations`
   mode table, version-correct (including the **shifted** v48/v61 table).
2. Clientbound `NoteOperation` writer — `options.errors` table gains
   `NO_NOTE_ITEM: 3` (all nine versions).
3. Serverbound `NoteOperationHandle` — `options.operations` inbound dispatch
   table, version-correct, and every entry carries
   `"validator": "LoggedInValidator"` (a validator-less handler entry is
   silently dropped by `BuildHandlerMap`).

## 2. Per-version deltas (reproduce verbatim in the PATCH body)

### 2.1 Shifted versions: gms_v48, gms_v61

**`NoteOperation` writer `options`:**

```json
{
  "operations": {
    "SHOW": 2,
    "SEND_SUCCESS": 3,
    "SEND_ERROR": 4
  },
  "errors": {
    "RECEIVER_ONLINE": 0,
    "RECEIVER_UNKNOWN": 1,
    "RECEIVER_INBOX_FULL": 2,
    "NO_NOTE_ITEM": 3
  }
}
```

> **Critical: `SEND_ERROR` is `4` on gms_v48/gms_v61, NOT `5`.** These two
> versions use a compressed/shifted `OnMemoResult` mode table client-side
> (`mode = Decode1()-2`, design §1.3). A standard `SEND_ERROR: 5` PATCHed
> onto a v48/v61 tenant sends a mode byte the client's `OnMemoResult` switch
> does not handle — the excl-request lock set by
> `SendConsumeCashItemUseRequest` is never cleared via the MEMO_RESULT path,
> and the client stays wedged (further item-use/transfer requests refused)
> until the inventory-operation unlock rides through on a different packet,
> if at all. Do not "normalize" this value to 5.

**`NoteOperationHandle` handler `options`:**

```json
{
  "operations": {
    "SEND": 0,
    "DISCARD": 1
  }
}
```

(`REQUEST: 2` may be included harmlessly if it simplifies the PATCH body —
these client versions never emit it, so an unused key is inert. Both
gms_v48 and gms_v61 templates in this repo already carry `REQUEST: 2` in
their handler table for this reason.)

`validator` must be `"LoggedInValidator"` on the handler entry.

### 2.2 Standard versions: gms_v72, gms_v79, gms_v83, gms_v84, gms_v87, gms_v95, jms_v185

**`NoteOperation` writer `options`:**

```json
{
  "operations": {
    "SHOW": 3,
    "SEND_SUCCESS": 4,
    "SEND_ERROR": 5,
    "REFRESH": 7
  },
  "errors": {
    "RECEIVER_ONLINE": 0,
    "RECEIVER_UNKNOWN": 1,
    "RECEIVER_INBOX_FULL": 2,
    "NO_NOTE_ITEM": 3
  }
}
```

**`NoteOperationHandle` handler `options`:**

```json
{
  "operations": {
    "SEND": 0,
    "DISCARD": 1,
    "REQUEST": 2
  }
}
```

`validator` must be `"LoggedInValidator"` on the handler entry.

### 2.3 Error sub-code semantics (uniform across all nine versions)

`SEND_ERROR` sub-codes: `0` = "the other character is online now, use
whisper" (`RECEIVER_ONLINE`); `1` = "check the receiving character's name"
(`RECEIVER_UNKNOWN`); `2` = "the receiver's inbox is full"
(`RECEIVER_INBOX_FULL`); `3` = `NO_NOTE_ITEM` — there is no dedicated
"missing Note item" dialog in any client version, so `3` is an
IDA-verified-safe out-of-range sub-code: the client's `OnMemoResult`
SEND_ERROR arm clears the excl-request lock unconditionally before decoding
the sub-code, then silently no-ops on an unrecognized value (no dialog,
lock released). This is intentional, not a gap.

## 3. How to apply the PATCH

For each existing tenant running one of the nine versions:

1. **Identify the tenant's version and current socket configuration:**

   ```
   GET /tenants/{tenantId}/configurations/socket
   ```

   (via the atlas-tenants API, or the equivalent panel in the atlas-ui
   configuration editor).

2. **Locate the two entries in the returned document** by role, not by
   array position (the templates are not always kept in ascending-opCode
   order in this branch's history — locate by field, e.g. `jq 'select
   (.writer=="NoteOperation")'` / `select(.handler=="NoteOperationHandle")`
   against the fetched JSON, or the equivalent search in the UI editor).

3. **Edit `options.operations` and `options.errors`** on the `NoteOperation`
   writer entry, and `options.operations` + `validator` on the
   `NoteOperationHandle` handler entry, to the version-correct values in
   §2.1/§2.2 above. Preserve every other key in the document untouched —
   this is a config PATCH, not a template replace.

4. **PATCH the edited document back:**

   ```
   PATCH /tenants/{tenantId}/configurations/socket
   ```

   with the full edited socket configuration body (atlas-tenants
   configuration PATCH replaces the resource; partial-key PATCH is not
   supported for this resource type — send the complete document you
   fetched in step 1 with only the note-related keys changed).

5. Repeat per tenant. If several tenants share the same version and were
   provisioned from the same (stale) seed snapshot, the same edited JSON
   fragment can be reused across their PATCH bodies — confirm each
   tenant's current document first, since a tenant may have accumulated
   its own independent drift.

## 4. Rollout restart (mandatory)

Handler/writer projections used by atlas-channel are loaded once and do
not hot-reload on a tenant configuration change. After PATCHing every
affected tenant:

```
kubectl rollout restart deployment atlas-channel
```

(substitute the per-environment equivalent — e.g. namespace-qualified
deployment name in a non-default namespace). Confirm the rollout completes
(`kubectl rollout status deployment atlas-channel`) before considering the
change live. Channels that are not restarted keep serving the old
in-memory config and will continue to exhibit the pre-fix behavior
(including the v48/v61 wedge) even though the persisted tenant
configuration is now correct.

## 5. Verification checklist (per patched tenant)

Run these against a real character session on the patched tenant, after
the `atlas-channel` restart has completed:

- [ ] **Happy path — item consumed.** Log in a character holding at least
      one Note cash item (item id `5090000`). Compose and send a note to
      an offline receiver. Confirm:
  - The Note item (`5090000`) is consumed (inventory count decrements by
    exactly one; no free sends).
  - The sender receives `SEND_SUCCESS` feedback (the MEMO_RESULT mode byte
    for `SEND_SUCCESS` is version-specific per §2 — `3` on gms_v48/gms_v61,
    `4` on gms_v72 and newer); confirm the sender's client shows the normal
    "note sent" success feedback, not an error dialog.
  - The receiver, on next login, sees the note delivered.

- [ ] **Missing-item rejection (server-side gate).** Via packet injection,
      send a `NOTE_ACTION` `SEND` (memo-operation opcode, not the
      USE_CASH_ITEM arm) for a character that does **not** hold a Note
      item. Confirm:
  - The server does not create a note or consume any item.
  - A warning is logged server-side (atlas-channel logs) identifying the
    rejected send and the missing-item condition.
  - The client receives a `SEND_ERROR` with sub-code `NO_NOTE_ITEM` (`3`)
    and silently unlocks — no error dialog appears (this is the expected,
    IDA-verified client behavior for an out-of-range sub-code), and the
    client accepts further input (confirm by immediately issuing another
    action, e.g. opening the inventory, and observing it is not blocked).

- [ ] **v48/v61 shifted-mode regression guard.** On a gms_v48 or gms_v61
      tenant specifically, repeat the missing-item rejection test above and
      confirm the client actually unlocks. This is the regression this
      task exists to prevent: if the tenant's `NoteOperation` writer
      `options.operations.SEND_ERROR` were left at the standard value `5`
      instead of the shifted value `4`, the client's `OnMemoResult` switch
      would not recognize mode `5` at all, the excl-request lock would
      never clear via this path, and the client would appear to freeze on
      further item-use/transfer actions after the rejected send. Confirm
      mode `4` is what's live (re-fetch `GET
      /tenants/{tenantId}/configurations/socket` and check
      `options.operations.SEND_ERROR == 4` for the `NoteOperation` writer)
      before/alongside the live client test.

- [ ] **Discard path unaffected.** Confirm `NOTE_ACTION` `DISCARD` (mode 1)
      still works normally post-patch — this arm was already load-bearing
      pre-task and must not regress.

# Live-Tenant PATCH: Sue/Claim Report Ops Config (task-145, Task 18)

Task: task-145-player-reports
Design: `docs/tasks/task-145-player-reports/design.md`
Findings: `docs/tasks/task-145-player-reports/packet-findings.md`
Scope amendment: `docs/tasks/task-145-player-reports/scope-amendment.md`

## 0. HIGHEST PRIORITY — pre-existing gms_v92 template bugs this branch also fixed

Tasks 26–28 (the gms_v92 column bring-up, done as part of this branch's scope
amendment) found and fixed two **pre-existing, unrelated-to-sue/claim** wiring
bugs in `template_gms_92_1.json`. Both are already broken in production today
on the live `GMS v92` tenant (`atlas-tenants` id
`db1dbfb3-4345-4731-9223-c40b0c7f6457`, confirmed via `GET /api/tenants`
against the live pod — see `scope-amendment.md` "Live tenant" for the full
citation) and stay broken until this tenant is PATCHed, exactly like the
sue/claim delta below. They are listed here, ahead of §1, because they are
**higher urgency** than the new report feature: they are silent
misdecodes of existing, already-shipped player-facing functionality, not a
missing new feature.

1. **Pet/summon misdecode (`atlas-channel`) — fix first.** Pre-fix,
   `template_gms_92_1.json` routed handlers `0xC8`/`0xC9` and writers
   `0xC3`/`0xC6`/`0xC7` as `Summon*`; the real v92 client uses those opcodes
   for `Pet*` (item-use / item-exclude / activated / movement / chat), with
   the genuine summon family living at `0xCB`–`0xD0`. **Unpatched, today**: a
   v92 player using pet auto-HP/MP-potion or the pet item-exclude-list
   feature has their request server-side misdecoded as a
   `SummonMoveHandle`/`SummonAttackHandle` packet (wrong struct layout read
   from the same bytes) — the server either processes garbage as a
   summon-move/attack command or errors out decoding malformed fields.
   Separately, any genuine summon-family clientbound packet the server emits
   goes out on the pre-fix opcodes, which the real client reads as
   `SHOW_RECOVERY_UPGRADE_COUNT_EFFECT`/`EVOLVE_PET`/nothing — summons never
   render correctly for v92 players today.
2. **Character deletion (`atlas-login`).** Pre-fix, `DeleteCharacterHandle`
   was routed at `0x17`; the real v92 client sends `DELETE_CHAR` at `0x18`
   (confirmed by decompile, `CLogin::SendDeleteCharPacket` @ `0x5cb860`) and
   `0x17` is `CREATE_CHAR_IN_CS` (genuinely unrouted in every version — no
   Atlas handler exists for it). **Unpatched, today**: a v92 player's real
   delete-character request (opcode `0x18`) has no handler at all (silently
   dropped — deletion does nothing), while `0x17` still fires
   `DeleteCharacterHandle` if the client ever sends `CREATE_CHAR_IN_CS`'s
   opcode — a character-creation request misread as a character-deletion
   request. This PATCH is against `atlas-login`'s tenant socket config, not
   `atlas-channel`'s — apply it to the same tenant via that service's
   equivalent configuration resource.
3. **The 59 Class A + 8 Class B newly-wired v92 opcodes** (Task 26–28) —
   chat, skills, reactors, monster carnival, storage, messenger, etc. These
   were previously silent drops (unrouted opcodes are ignored, not
   misdecoded) rather than active misdecodes, so they are lower urgency than
   1–2 but are still real, currently-inert functionality on the live v92
   tenant. Diff `template_gms_92_1.json` between the commit range noted in
   `scope-amendment.md`'s Class A table for the exhaustive opcode list; this
   PATCH does not attempt to enumerate all 67 individually.

These three items are **independent of, and precede, the sue/claim delta in
§2.4 below** — apply them in the same PATCH+restart cycle as §2.4 since they
touch the same tenant, but do not conflate them with the report feature: the
pet/summon and delete-character bugs would need fixing on this tenant even if
task-145 had never existed. This PATCH was not attempted by any agent on this
branch — it is a live-system change, surfaced here for the user's decision,
per `scope-amendment.md`.

## 1. Why existing tenants need this

Seed templates under `services/atlas-configurations/seed-data/templates/`
apply **only at tenant creation** — a tenant provisioned before this change
already has a socket configuration persisted in atlas-tenants, and that
persisted config is what atlas-channel actually loads at runtime, not the
seed template. Updating the seed template alone does nothing for tenants
that already exist. Every existing tenant on one of the eight versions this
task wired needs its socket configuration PATCHed directly.

The delta being deployed, per version:

1. Serverbound `ClaimRequest` handler — new (gms_v72/v79/v83/v84/v87/v92/v95;
   gms_v61 has no claim send-site and gets no handler).
2. Serverbound `SueCharacter` handler — new **only on gms_v92** (every other
   in-scope version already routes it; do not re-add it elsewhere).
3. Clientbound `SueCharacterResult` writer — new, all eight versions
   (gms_v61/v72/v79/v83/v84/v87/v92/v95).
4. Clientbound `ClaimResult` / `ClaimAvailableTime` / `ClaimSvrStatusChanged`
   writers — new, all versions except gms_v61 (claim is version-absent
   there).

Every handler entry below carries `"validator": "LoggedInValidator"` — a
handler entry without a validator is silently dropped at registration time
(no error, it simply never routes).

## 2. Per-version deltas (reproduce verbatim in the PATCH body)

Opcodes are taken from the per-version dispatcher-doc source of truth
(`docs/packets/dispatchers/sue_character_result.yaml`,
`docs/packets/dispatchers/claim_result.yaml`) and the wired seed templates —
do not port an opcode from a neighboring version.

### 2.1 gms_v61 (sue-only — no claim send-site)

**New `socket.writers` entry:**

```json
{
  "opCode": "0x34",
  "writer": "SueCharacterResult",
  "services": ["channel"],
  "options": {
    "operations": {
      "SUCCESS": "0x00",
      "UNABLE_TO_LOCATE": "0x01",
      "DAILY_LIMIT": "0x02",
      "REPORTED_NOTICE": "0x03",
      "GENERIC_FAILURE": "0x04"
    }
  }
}
```

No handler entry and no claim writers — gms_v61 has no `CLAIM_REQUEST`
send-site (verified absence, `packet-findings.md` §7.2); adding claim
writers here would enable a UI the client cannot submit from.

### 2.2 gms_v72 / gms_v79 (claim opcodes 8 lower than gms_v83+)

**New `socket.handlers` entry:**

```json
{ "opCode": "0x69", "validator": "LoggedInValidator", "handler": "ClaimRequest", "services": ["channel"] }
```

(`0x68` on gms_v79 — substitute per version; `SueCharacter` is already
routed on both and must not be re-added or renumbered.)

**New `socket.writers` entries** (identical opcodes on both versions):

```json
{
  "opCode": "0x34",
  "writer": "SueCharacterResult",
  "services": ["channel"],
  "options": {
    "operations": {
      "SUCCESS": "0x00",
      "UNABLE_TO_LOCATE": "0x01",
      "DAILY_LIMIT": "0x02",
      "REPORTED_NOTICE": "0x03",
      "GENERIC_FAILURE": "0x04"
    }
  }
},
{
  "opCode": "0x2A",
  "writer": "ClaimResult",
  "services": ["channel"],
  "options": {
    "operations": {
      "SUCCESS": "0x02",
      "REPORTED_NOTICE": "0x03",
      "TRY_AGAIN": "0x41",
      "RECHECK_NAME": "0x42",
      "NOT_ENOUGH_MESOS": "0x43",
      "CANNOT_CONNECT": "0x44",
      "EXCEEDED": "0x45",
      "TIME_WINDOW": "0x47",
      "FALSE_REPORT_CITED": "0x48"
    }
  }
},
{
  "opCode": "0x2B",
  "writer": "ClaimAvailableTime",
  "services": ["channel"]
},
{
  "opCode": "0x2C",
  "writer": "ClaimSvrStatusChanged",
  "services": ["channel"]
}
```

### 2.3 gms_v83 / gms_v84 / gms_v87 (standard claim block)

**New `socket.handlers` entry** (opcode is per-version — see table):

| version | ClaimRequest opCode |
|---|---|
| gms_v83 | `0x6A` |
| gms_v84 | `0x6A` |
| gms_v87 | `0x6D` |

```json
{ "opCode": "<see table>", "validator": "LoggedInValidator", "handler": "ClaimRequest", "services": ["channel"] }
```

**New `socket.writers` entries** (identical opcodes across all three
versions):

```json
{
  "opCode": "0x37",
  "writer": "SueCharacterResult",
  "services": ["channel"],
  "options": {
    "operations": {
      "SUCCESS": "0x00",
      "UNABLE_TO_LOCATE": "0x01",
      "DAILY_LIMIT": "0x02",
      "REPORTED_NOTICE": "0x03",
      "GENERIC_FAILURE": "0x04"
    }
  }
},
{
  "opCode": "0x2D",
  "writer": "ClaimResult",
  "services": ["channel"],
  "options": {
    "operations": {
      "SUCCESS": "0x02",
      "REPORTED_NOTICE": "0x03",
      "TRY_AGAIN": "0x41",
      "RECHECK_NAME": "0x42",
      "NOT_ENOUGH_MESOS": "0x43",
      "CANNOT_CONNECT": "0x44",
      "EXCEEDED": "0x45",
      "TIME_WINDOW": "0x47",
      "FALSE_REPORT_CITED": "0x48"
    }
  }
},
{
  "opCode": "0x2E",
  "writer": "ClaimAvailableTime",
  "services": ["channel"]
},
{
  "opCode": "0x2F",
  "writer": "ClaimSvrStatusChanged",
  "services": ["channel"]
}
```

### 2.4 gms_v92 — corrected scope (see §5)

**Note the shape here differs from every other version**: gms_v92 needs
**both** new handler entries, because — unlike every other in-scope
version — its `SueCharacter` handler was never routed at all (verified: the
template had zero sue/claim entries prior to this task).

**New `socket.handlers` entries:**

```json
{ "opCode": "0x75", "validator": "LoggedInValidator", "handler": "ClaimRequest", "services": ["channel"] },
{ "opCode": "0x7D", "validator": "LoggedInValidator", "handler": "SueCharacter", "services": ["channel"] }
```

**New `socket.writers` entries:**

```json
{
  "opCode": "0x38",
  "writer": "SueCharacterResult",
  "services": ["channel"],
  "options": {
    "operations": {
      "SUCCESS": "0x00",
      "UNABLE_TO_LOCATE": "0x01",
      "DAILY_LIMIT": "0x02",
      "REPORTED_NOTICE": "0x03",
      "GENERIC_FAILURE": "0x04"
    }
  }
},
{
  "opCode": "0x2E",
  "writer": "ClaimResult",
  "services": ["channel"],
  "options": {
    "operations": {
      "SUCCESS": "0x02",
      "REPORTED_NOTICE": "0x03",
      "TRY_AGAIN": "0x41",
      "RECHECK_NAME": "0x42",
      "NOT_ENOUGH_MESOS": "0x43",
      "CANNOT_CONNECT": "0x44",
      "EXCEEDED": "0x45",
      "TIME_WINDOW": "0x47",
      "FALSE_REPORT_CITED": "0x48"
    }
  }
},
{
  "opCode": "0x2F",
  "writer": "ClaimAvailableTime",
  "services": ["channel"]
},
{
  "opCode": "0x30",
  "writer": "ClaimSvrStatusChanged",
  "services": ["channel"]
}
```

> **`SUE_CHARACTER_RESULT` at `0x38` and `SUE_CHARACTER` at `0x7D` (125) are
> not typos.** Both are genuine gms_v92 anomalies relative to their
> neighbours (v87/v95 both use `0x37` for the result writer; v87's send-site
> is 117, v95's is 126) — confirmed by direct decompile of the gms_v92 IDB
> (`CWvsContext::OnSueCharacterResult` @ `0x9cf950`, a `switch(v2)` with the
> ordinary 0/1/2/3/default arms; the send-site opcode confirmed via `push
> 7Dh` at `0x53b7d0`). Do not "correct" these to match a neighbor.

### 2.5 gms_v95

**New `socket.handlers` entry:**

```json
{ "opCode": "0x76", "validator": "LoggedInValidator", "handler": "ClaimRequest", "services": ["channel"] }
```

**New `socket.writers` entries:**

```json
{
  "opCode": "0x37",
  "writer": "SueCharacterResult",
  "services": ["channel"],
  "options": {
    "operations": {
      "SUCCESS": "0x00",
      "UNABLE_TO_LOCATE": "0x01",
      "DAILY_LIMIT": "0x02",
      "REPORTED_NOTICE": "0x03",
      "GENERIC_FAILURE": "0x04"
    }
  }
},
{
  "opCode": "0x2C",
  "writer": "ClaimResult",
  "services": ["channel"],
  "options": {
    "operations": {
      "SUCCESS": "0x02",
      "REPORTED_NOTICE": "0x03",
      "TRY_AGAIN": "0x41",
      "RECHECK_NAME": "0x42",
      "NOT_ENOUGH_MESOS": "0x43",
      "CANNOT_CONNECT": "0x44",
      "EXCEEDED": "0x45",
      "TIME_WINDOW": "0x47",
      "FALSE_REPORT_CITED": "0x48"
    }
  }
},
{
  "opCode": "0x2D",
  "writer": "ClaimAvailableTime",
  "services": ["channel"]
},
{
  "opCode": "0x2E",
  "writer": "ClaimSvrStatusChanged",
  "services": ["channel"]
}
```

## 3. How to apply the PATCH

The live tenant-configuration API is JSON:API and lives on atlas-tenants
under `/api/configurations/tenants` (distinct from the seed-template
resource at `/api/configurations/templates`) — see
`services/atlas-ui/src/services/api/tenants.service.ts`. **A bare (non
JSON:API) body 400s.** The endpoint replaces the full `attributes` document;
there is no partial/merge-patch mode, so every step below fetches first and
sends the complete edited document back.

For each existing tenant on one of the versions in §2:

1. **Fetch the tenant's current configuration:**

   ```
   GET /api/configurations/tenants/{tenantId}
   ```

2. **Locate `data.attributes.socket.handlers` / `.writers`.** Confirm by
   field (`handler`/`writer` name), not array position — the templates in
   this repo are not guaranteed to stay in the same relative order across
   branch history.

3. **Insert the version-appropriate entries from §2** at their sorted
   `opCode` position within the existing array (matches the seed-template
   convention this repo's `tools/template-opcode-order-guard.sh` enforces;
   the live JSONB document is not itself order-checked by that tool, but
   keeping it sorted avoids confusion during the next manual edit). Leave
   every other key in the document untouched — this is a config PATCH, not
   a template replace.

4. **PATCH the edited document back, full `attributes` body, JSON:API
   envelope:**

   ```
   PATCH /api/configurations/tenants/{tenantId}
   ```

   ```json
   {
     "data": {
       "id": "{tenantId}",
       "type": "tenants",
       "attributes": { "...": "the full edited attributes document from step 1, with the socket array patched" }
     }
   }
   ```

5. Repeat per tenant. Tenants sharing a version and provisioned from the
   same (stale) seed snapshot can reuse the same edited fragment, but
   confirm each tenant's current document first — a tenant may carry its
   own independent drift from prior manual edits.

## 4. Rollout restart (mandatory)

atlas-channel's handler/writer projection is loaded once at startup and does
not hot-reload on a tenant configuration change (known pattern — see
project memory `reference_reconcile_live_tenant_socket_to_template.md`).
After PATCHing every affected tenant:

```
kubectl rollout restart deployment atlas-channel
```

(substitute the per-environment/namespace-qualified deployment name.)
Confirm completion with `kubectl rollout status deployment atlas-channel`
before considering the change live. A channel pod that is not restarted
keeps serving its old in-memory routing table — `ClaimRequest`/`SueCharacter`
packets from a patched tenant's players will still be silently dropped
(unrouted handler) and the new writers will never fire, even though the
persisted tenant configuration is correct.

## 5. New env vars for existing deployments

Two services gained new, **optional** runtime configuration as part of this
feature (both default sanely — no env change is required for the feature to
degrade gracefully, only to reach full fidelity):

| Service | Variable | Default if unset | Purpose |
|---|---|---|---|
| atlas-ban | `CHARACTERS_SERVICE_URL` | falls back to `BASE_SERVICE_URL` | Base URL used to resolve reporter/accused character identity for a submitted report. |
| atlas-ban | `MESSAGES_SERVICE_URL` | falls back to `BASE_SERVICE_URL` | Base URL used to fetch the corroborating chat transcript for a report. |
| atlas-messages | `CHAT_CAPTURE_RETENTION_SECONDS` | `900` | How long captured chat lines are retained before purge. |
| atlas-messages | `CHAT_CAPTURE_MAX_LINES` | `200` | Per-pair cap on retained chat lines. |

`deploy/k8s/base/env-configmap.yaml` already carries the fleet-wide
`BASE_SERVICE_URL` fallback (routed through atlas-ingress) — every existing
deployment already satisfies the reporter/accused-resolution and
transcript-fetch dependency without any change. Set the two `*_SERVICE_URL`
overrides only if atlas-ban should bypass the ingress and call
atlas-character/atlas-messages directly. No existing deployment needs a
config change for `CHAT_CAPTURE_RETENTION_SECONDS`/`CHAT_CAPTURE_MAX_LINES`
unless the 900s/200-line defaults are wrong for the environment.

The two new Kafka topics (`COMMAND_TOPIC_REPORT`, `EVENT_TOPIC_REPORT_STATUS`,
added to `deploy/k8s/base/env-configmap.yaml` and both kustomize overlays in
this task) are picked up automatically by any deployment that re-applies the
base configmap — no manual per-tenant action needed for those.

## 6. Which tenants get NO patch

- **jms_185**: out of scope for this task (`template_jms_185_1.json` was not
  touched — its disposition is an open question, tracked separately, not
  resolved here). jms tenants get no patch and no new routing; the feature
  stays disabled there exactly as before this task.
- **gms_v12 / gms_v48**: excluded per the PRD non-goal — gms_v12 has no
  registry/matrix column at all, and gms_v48's client has both clientbound
  receivers but no send-site for either op (adding writers there would
  answer requests that can never arrive). No patch applies.
- **gms_v92 is NOT excluded** — despite an earlier plan draft saying so.
  The scope was corrected mid-branch (`scope-amendment.md`) once the
  registry/IDA verification the original exclusion cited as missing was
  produced (Tasks 26–30 on this branch). A `GMS v92` tenant is confirmed to
  exist in the live `atlas-main` environment (see `scope-amendment.md`,
  "Live tenant" section) and **does** need the §2.4 patch — do not skip it
  on the assumption the old exclusion still holds.

## 7. Verification checklist (per patched tenant)

Run against a real character session on the patched tenant, after the
`atlas-channel` restart has completed:

- [ ] **Sue happy path.** From a character's chat window, submit a `/sue`
      (or client UI equivalent) against another online character. Confirm
      the sender receives a `SueCharacterResult` chat-log line (not a modal)
      with the success message, and that a report record is created
      server-side (atlas-ban).
- [ ] **Sue result codes.** Trigger `UNABLE_TO_LOCATE` (target offline/not
      found) and `DAILY_LIMIT` (repeat past the daily cap) and confirm the
      client shows the corresponding chat-log line, not the generic-failure
      line — this exercises the mode table, not just the wire path.
- [ ] **Claim availability gating (all versions except gms_v61).** Confirm
      `ClaimSvrStatusChanged` (`connected = 1`) and `ClaimAvailableTime` are
      sent to the client on login/channel-enter; without `connected = 1` the
      claim UI refuses to open client-side (this is client behavior, not a
      server bug — verify the server is actually sending it).
- [ ] **Claim happy path.** Open the claim/report dialog, submit a claim
      against another character. Confirm `ClaimResult` mode `2` (SUCCESS)
      renders the modal with the correct remaining-reports count.
- [ ] **Claim failure codes.** Exercise at least `TRY_AGAIN` (`0x41`) and
      `NOT_ENOUGH_MESOS` (`0x43`) and confirm the correct modal text appears
      — this is the strongest signal the mode table (not just the opcode)
      is correct for the patched version.
- [ ] **gms_v92 specifically — the two-handler case.** Since gms_v92 needed
      both `ClaimRequest` and `SueCharacter` newly routed (§2.4), confirm
      **both** independently: a `/sue` command AND a claim submission both
      reach the server (check atlas-channel logs for the handler firing,
      not just client-side "sent" behavior) — a missing validator on either
      would silently drop it with no client-visible symptom beyond "nothing
      happens."
- [ ] **gms_v61 boundary.** On a gms_v61 tenant, confirm `/sue` still works
      (`SueCharacterResult` at `0x34`) and confirm there is **no** claim UI
      available client-side (expected — no send-site exists on this
      version; this is not a bug to "fix").
- [ ] **Report transcript corroboration (atlas-ban → atlas-messages).**
      Submit a chat-claim (`bChatClaim = 1`) against a character the
      reporter has an active chat log with. Confirm the resulting report
      record in atlas-ban includes a non-null chat transcript fetched via
      atlas-messages (exercises the `MESSAGES_SERVICE_URL`/
      `BASE_SERVICE_URL` fallback path from §5).

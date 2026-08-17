# World transfer: an unreachable license dialog, and a rejection reason that reaches no client string

Reported: 2026-08-17, testing task-227 on `atlas-pr-1370` (tenant
`d606f1cb-ba79-45ca-a989-cf0dc956fee7`, GMS 83.1). The birthday-verification
crash from `bug-world-transfer-client-crash.md` is fixed — the client survives
the credential check and renders the license notice. Two new defects surface
immediately after it.

This document is a Phase 5 diagnosis. `design.md` is untouched; it remains the
design of record for the work already landed.

---

## Symptom 1 — the "rules for character transfer" dialog is dead

The player reaches the license notice, but it does not respond to input, and a
second dialog is visible behind it reading:

> Your Cash Shop storage in this world is tied to your account, not this
> character. Because this is your only remaining character here, it will become
> inaccessible once the transfer completes.

### That second dialog is ours

It is `warnIfStrandingStorage`, emitted from
`services/atlas-channel/atlas.com/channel/socket/handler/cash_shop_check_transfer_world_possible.go:198`
— the FR-4.7 courtesy notice, sent as a `POP_UP` world message immediately
after the ALLOWED result.

### Why it kills the dialog in front of it

Both dialogs are modal, and ours opens *inside* the other's modal loop.

| Step | Evidence (GMS v83, `MapleStory_dump.exe`) |
|---|---|
| ALLOWED result, arm 0, opens the license notice | `CCashShop::OnCheckTransferWorldPossibleResult` `@0x47bd9b` case 0 → `CUITransferWorldLicenseNotice::CUITransferWorldLicenseNotice` → **`CDialog::DoModal(v17)`** |
| `DoModal` blocks; packets keep being pumped inside its nested message loop | `CDialog::DoModal` `@0x4edba1` |
| Our `POP_UP` arrives and is dispatched in that nested loop | `POP_UP` = mode **1** (`template_gms_83_1.json`, WorldMessage `0x44` operations table) |
| Arm 1 is the *unguarded* `CUtilDlg::Notice` path — no `CUIStatusBar` check, unlike arms 0/12/13 | `CWvsContext::OnBroadcastMsg` `@0xa22785` case 1 → `LABEL_185` → `CUtilDlg::Notice` |
| `CUtilDlg::Notice` opens a **second** modal | `CUtilDlg::Notice` `@0x9929dd` → `CDialog::CreateDlg` → **`CDialog::DoModal(v21)`** |

Two nested modal loops. The inner one (our storage notice) owns the input
grab; the license notice is still painted on top but can never receive a click.
The player sees exactly what was reported: a live-looking dialog in front, a
dead one, and ours behind it.

The `warnIfStrandingStorage` doc comment's reasoning for choosing `POP_UP` over
`PINK_TEXT` is correct as far as it goes — `PINK_TEXT` (arm 5) routes through
`CHATLOG_ADD`, which is guarded by `TSingleton<CUIStatusBar>::ms_pInstance` and
so is invisible in the cash shop. What it missed is that `POP_UP`'s replacement
is *modal*, and the ALLOWED arm it is emitted alongside is *also* modal.

### The fix is timing, not suppression

The warning is real and worth keeping — the storage genuinely does strand. It
must simply not be in flight while the license notice owns the screen. Options,
in the order I'd rank them:

1. **Emit at `BUY_WORLD_TRANSFER` time** (`handleBuyWorldTransfer`), on the
   rejection-free path, once the player has already dismissed the license
   notice and picked a destination. This is also the first moment the warning
   is actually actionable.
2. **Emit after the transfer resolves**, on the existing
   `PENDING_CHANGE_RESOLVED` notification path (design §3.9), which already has
   an offline-safe delivery story.
3. **Suppress it entirely.** Cheapest, and loses a genuine warning. Only if
   both above are rejected.

Do **not** simply switch it to `PINK_TEXT` — per the evidence above it would
render nowhere in the cash shop, which is silent data loss dressed up as a fix.

---

## Symptom 2 — transfer to Bera rejected as `in_family`

```
10:07:44.176  GET /api/families/tree/1  ->  200, body is HTML
              error: invalid character '<' looking for beginning of value
              "Unable to check family membership for character [1]."
10:07:44.176  World-transfer eligibility check ... rejected: in_family
10:07:44.175  POST /api/characters/1/pending-changes -> 422
10:07:44.179  Code [in_family] not configured in property [errors].
              Defaulting to 99 which will likely cause a client crash.
```

Three independent defects, stacked. Only the first is about families.

### 2a — `inFamily` cannot distinguish "no family" from "no answer"

`services/atlas-character/atlas.com/character/pending_change/requests.go:113`:

```go
_, err := requestFamilyTree(characterId)(l, ctx)
if err != nil {
    if errors.Is(err, requests.ErrNotFound) {
        return false, nil
    }
    return false, err
}
return true, nil
```

Two bugs in nine lines:

- **Any non-404 error is propagated**, and gate 8 turns that into an
  affirmative `in_family`. Here the body was HTML, JSON decoding failed, and a
  transport-level failure was reported to the player as a family membership.
- **A `200` with an empty tree returns `true`.** Even against a healthy
  atlas-families, a character with no relatives would be reported as being in a
  family. The predicate never looks at the response at all — it only looks at
  whether the call succeeded. It should be `len(members) > 1` (or an explicit
  senior/junior check); a tree containing only yourself is not a family.

**Why the body was HTML — resolved 2026-08-17.** `atlas-families` has never been
deployed anywhere. It is not a namespace-specific miss; it is absent from every
layer of the deploy path:

| Layer | Checked | Result |
|---|---|---|
| Live namespace | `kubectl get deploy -n atlas-pr-1370` | 69 deployments, **no `atlas-families`** |
| Base manifests | `deploy/k8s/base/` | **no `atlas-families.yaml`** (every other service has one) |
| Ingress routes | `deploy/shared/routes.conf` (115 `location` blocks) | **no `/api/families` route** |
| Version pinning | `deploy/k8s/base/versions.json` | **no families entry** |
| Image build | `docker-bake.hcl:56`, `deploy/k8s/overlays/{pr,main}/kustomization.yaml` | present — the image *is* built and tagged |

So the image is built and never scheduled, and the ingress has no route for
`/api/families/...`. The request fell through nginx to the default backend,
which returned HTML — exactly matching the observed
`invalid character '<' looking for beginning of value`. The guild gate answered
JSON in the same trace because `/api/guilds` *is* routed.

`RootUrl("FAMILIES")` (`libs/atlas-rest/requests/url.go:14`) falls back to
`BASE_SERVICE_URL` when `FAMILIES_SERVICE` is unset, which is why the call went
to the shared ingress rather than failing to resolve.

`atlas-families` does exist in the repo and does serve
`GET /families/tree/{characterId}`
(`services/atlas-families/atlas.com/family/family/`); it has been in-tree since
task-118 (`293ff9b4e`). Nothing about gate 8 can work in any environment until
the service is routed and scheduled — **this is a prerequisite, not a
side-issue.**

### 2b — every gate reports an outage as an affirmative block

Gate 8 is not special. All eleven gates in
`services/atlas-character/atlas.com/character/pending_change/processor_eligibility.go`
share one shape:

```go
x, err := p.gates.someCheck(...)
if err != nil {
    p.l.WithError(err).Errorf("Unable to check ...")
    return reject("some_affirmative_reason")   // <- an outage, reported as a fact
}
if x {
    return reject("some_affirmative_reason")
}
```

`world_unknown`, `no_character_slot`, `name_taken`, `banned`,
`is_guild_master`, `in_family`, `trade_open`, `merchant_open`,
`mts_listings_open` — each is returned both when the condition genuinely holds
and when the check could not be performed. Failing closed is the right call and
should stay. Reporting a guess as a finding is not.

**Fix (all 11 gates, per the decision on this task):** add a distinct
`check_unavailable` outcome to the reason taxonomy (design §6). The transfer is
still refused, the log still names the real dependency and error, and the
player-facing message stops asserting something the server does not know. The
gate table is already data-shaped, so this is a uniform change, and
`processor_eligibility_test.go` already drives each gate through a `gateDeps`
seam — add an error-injection case per gate asserting `check_unavailable`
rather than the affirmative reason.

### 2c — the reason never reaches a client string (the widest defect)

`worldTransferRejectionReason` (`cash_shop_operation.go:385`) returns the raw
machine reason, which
`CashShopTransferWorldFailedBody` (`libs/atlas-packet/cash/clientbound/shop_operation_body.go:458`)
feeds straight to `ResolveCode(l, options, "errors", message)`.

The tenant `errors` table is not keyed by our reasons. **Not one** of
`in_family`, `world_same`, `world_full`, `world_unknown`, `no_character_slot`,
`name_taken`, `banned`, `is_guild_master`, `is_gm`, `trade_open`,
`merchant_open`, `mts_listings_open` or `unknown_error` appears in any of the
ten templates. So *every* world-transfer rejection resolves to the 99 sentinel.
`in_family` is simply the first one that happened to be hit.

**Correcting the log line:** `ResolveCode`'s "will likely cause a client crash"
is a generic warning and is **wrong for this arm**. `CCashShop::NoticeFailReason`
`@0x47c17a` has a `default:` case that loads StringPool id 557 and shows a
generic notice. 99 is not in its switch, so it lands on that default. The
player gets a vague-but-harmless message; the client does not die. This is a
correctness/UX bug, not a stability one — which matches the observed behaviour.

#### The `errors` table is closed, and it is version-specific

Two findings that constrain the fix, both verified rather than assumed:

**It is a complete transcription of `NoticeFailReason`.** I enumerated the
switch cases at `@0x47c17a` (`0xA3, 0xA5..0xAD, 0xB0, 0xB2..0xC4, 0xC7, 0xC8,
0xCD, 0xD0, 0xD2..0xE0, 0xE6..0xE9`) and compared them to
`template_gms_83_1.json`'s `errors` values. **53 codes, exact match, identical
gaps** (164, 174, 175, 177, 197, 198, 201–204, 206, 207, 209, 225–229 are absent
from both). The table is therefore already correct and already complete. There
is no unused code to claim — any new key must *alias* one of the 53.

**The numbers differ on every version; only the names are stable:**

| Template | codes | `CANNOT_TRANSFER_TO_SAME_WORLD` | `CANNOT_TRANSFER_OUT` | `PLEASE_TRY_AGAIN` |
|---|---|---|---|---|
| gms_12_1 | *no table at all* | — | — | — |
| gms_48_1 | 113–155 (40) | 153 | 155 | — |
| gms_61_1 | 128–179 (43) | 177 | 179 | — |
| gms_72_1 | 141–199 (49) | 195 | 197 | — |
| gms_79_1 | 155–222 (53) | 209 | 211 | 220 |
| gms_83_1 | 163–233 (53) | 220 | 222 | 231 |
| gms_84_1 | 172–242 (53) | 229 | 231 | 240 |
| gms_87_1 | 178–248 (53) | 235 | 237 | 246 |
| gms_92_1 | 1–69 (51) | 58 | 60 | 69 |
| gms_95_1 | 1–69 (51) | 58 | 60 | 69 |
| jms_185_1 | 178–211 (30) | *absent* | *absent* | *absent* |

`gms_12_1` has no `errors` table and `jms_185_1` has no transfer codes at all,
so on those two versions there is nothing honest to alias to — they must fall
back to the generic arm, and that must be a deliberate, logged decision rather
than a silent 99.

#### A concern with seeding reason keys into the templates

The decision on this task is to add reason keys to all ten templates. I'll flag
one thing and then build it that way.

The `errors` table's **names are already the version-abstraction layer** — that
is precisely why `CANNOT_TRANSFER_TO_SAME_WORLD` means the same thing at 153,
177, 195, 209, 220, 229, 235 and 58. Adding `world_same: 220` to
`template_gms_83_1.json`, `world_same: 58` to `template_gms_95_1.json` and so
on duplicates one aliasing decision across ten files, and across every template
added later. Mapping `world_same -> "CANNOT_TRANSFER_TO_SAME_WORLD"` once in
`worldTransferRejectionReason` and letting the existing table resolve the byte
gets the identical wire result with one edit and no new seed data. If that
reading is accepted, prefer it; the per-template seeding is otherwise fine and
is what's specified below.

Either way, two reasons remain genuinely unmappable and must not be invented:

- **`in_family` has no code in this table.** The family string (StringPool
  5017) is reachable only from `OnCheckTransferWorldPossibleResult` arm 8 and
  from the client's own pre-send gate — never from `NoticeFailReason`.
- **`is_gm` / `is_guild_master`** likewise: those strings
  (`"GM can not transfer worlds."`, `"Guild Master can not transfer worlds."`)
  are `ZXString<char>::Format` literals inside
  `CCashShop::CheckTransferWorldPossible` `@0x4734e5`, a different function
  that runs before the request is sent.

I could not read the rendered text of StringPool ids 4002–4008 (the transfer
block) — the ids are raw integers in `NoticeFailReason` with no named enum
members, and there is no String.wz in this checkout. The existing template
*names* are the prior derivation and are the best available evidence, but
**confirm the actual rendered strings against a live client before shipping the
alias table.** Do not treat the names as verified.

---

## The better fix for 2c, and the dead code that already implements it

`libs/atlas-packet/cash/clientbound/check_transfer_world_possible_result.go`
already contains a fully derived, fully documented, fully tested reason→arm
mapper — `CheckTransferWorldPossibleResultRejectedBody` and
`checkTransferWorldPossibleReasonArms`. It routes `in_family` to arm 8 (the
confirmed 5017 text) and everything else to `UNKNOWN_ERROR`, with an explicit
written argument for why the other arms are *not* claimed.

**It is dead code.** `grep` across the repo finds no caller outside its own file
and tests. Nothing ever emits a rejected CHECK result.

That is the design gap the CHECK handler's own doc comment records: the handler
"validates the credential and the PIC-attempt lockout only, exactly as the
sibling name-change handler does, and answers ALLOWED on a valid credential
with no further gate evaluation," because `CheckTransferEligibility` requires a
`destinationWorldId` that this op does not carry.

The arm evidence resolves this. `OnCheckTransferWorldPossibleResult` `@0x47bd9b`
has arms 0–8 with real distinct StringPool text (4002, 4009, 4003, 4004, 4015,
4010, 5017) — and the gates that *are* destination-independent are exactly the
ones this op could report: `is_gm`, `banned`, `is_guild_master`, `in_family`,
`trade_open`, `merchant_open`, `mts_listings_open`.

So the clean shape is:

- **Split the gate table** in `processor_eligibility.go` into
  destination-independent (gates 6–11) and destination-dependent (gates 1–5),
  and expose the independent half through a destination-free entry point on
  atlas-character.
- **CHECK time** evaluates the independent half and answers via the existing
  `CheckTransferWorldPossibleResultRejectedBody`. `in_family` gets its real
  client string. The player is told *before* the license notice, which is also
  where the client's own gate would have told them.
- **BUY time** keeps the destination-dependent half and the `errors`-table
  path, which is where `world_same` / `world_full` / `no_character_slot` belong
  and where `CANNOT_TRANSFER_*` codes actually exist.

This closes OQ-7, which `design.md` §2 deferred to per-cell derivation and which
was never closed — the reason keys were plumbed as though they were
`errors`-table keys without anyone checking that the table contained them. The
`check_transfer_world_possible_result.go` mapper is that derivation; it was done
correctly and then not wired up.

---

## Suggested fix order

1. **Move the storage warning off the license-notice modal** (symptom 1). One
   file, no derivation needed, unblocks live testing of everything else.
2. **Fix `inFamily`'s two bugs** (2a) — non-404 must not mean "in family", and
   a tree of one is not a family. Confirm the atlas-families deployment first.
3. **Add `check_unavailable` across all 11 gates** (2b), with an error-injection
   test per gate.
4. **Wire the CHECK-time rejection path** and split the gate table, closing
   OQ-7 and giving `in_family` its real string.
5. **Seed the reason keys / alias table for the BUY-time `errors` path** (2c),
   after confirming the StringPool text against a live client, and with an
   explicit logged fallback on `gms_12_1` and `jms_185_1`.

Steps 1–3 need no further client derivation. Steps 4–5 do.

## Rulings (2026-08-17, decided on this task)

All four open decisions above are now settled. These are binding on the fix;
do not re-litigate them.

1. **Storage warning → emit at BUY time.** Move `warnIfStrandingStorage` out of
   the CHECK handler and into `handleBuyWorldTransfer`, on the rejection-free
   path. It keeps failing open (a lookup error is logged and swallowed). The
   CHECK handler stops emitting any `POP_UP`, so the license notice's modal
   loop is never re-entered.
2. **BUY-time reasons → seed reason keys into all ten templates.** The
   alias-in-code alternative is explicitly **not** taken. Each template's
   `errors` table gains reason keys pointing at *that version's own* code for
   the aliased name (see the alias table below).
3. **Families → wire up the deployment.** `deploy/k8s/base/atlas-families.yaml`,
   the `/api/families` route in `deploy/shared/routes.conf` (then regenerate
   `routes.conf.template.generated` with `tools/gen-routes.sh`), the
   `versions.json` entry, and the `kustomization.yaml` resource entry — plus
   `inFamily`'s two logic bugs. Follow
   [`docs/adding-a-new-service.md`](../../adding-a-new-service.md) for the
   manifest shape; model it on a sibling read-only service.
4. **Scope → all five steps** in the suggested fix order.

### The alias table (ruling 2)

Derived from `template_gms_83_1.json` `/socket/writers/211/options/errors`
(53 entries, enumerated in full). Only these names are transfer-relevant:

```
219 CANNOT_TRANSFER_UNDER_LEVEL_TWENTY
220 CANNOT_TRANSFER_TO_SAME_WORLD
221 CANNOT_TRANSFER_TO_NEW_WORLD
222 CANNOT_TRANSFER_OUT
223 CANNOT_TRANSFER_NO_EMPTY_SLOTS
231 PLEASE_TRY_AGAIN
```

| Reason key | Aliases to | Why |
|---|---|---|
| `world_same` | `CANNOT_TRANSFER_TO_SAME_WORLD` | exact match |
| `world_full` | `CANNOT_TRANSFER_TO_NEW_WORLD` | destination-side refusal |
| `world_unknown` | `CANNOT_TRANSFER_TO_NEW_WORLD` | destination-side refusal |
| `no_character_slot` | `CANNOT_TRANSFER_NO_EMPTY_SLOTS` | exact match |
| `check_unavailable` | `PLEASE_TRY_AGAIN` | transient, and honest — the server does not know |
| `unknown_error` | `PLEASE_TRY_AGAIN` | transient |
| `name_taken`, `banned`, `is_guild_master`, `is_gm`, `in_family`, `trade_open`, `merchant_open`, `mts_listings_open` | `CANNOT_TRANSFER_OUT` | source-side refusal; no more specific code exists in the table |

**Resolve the numeric code per template from that template's own table** — the
numbers differ on every version (see the version table above). Never copy a
gms_83_1 number into another template.

Two templates cannot carry the aliases and must fall back explicitly, with a
logged reason rather than a silent 99:

- `template_gms_12_1.json` — has no `errors` table at all.
- `template_jms_185_1.json` — has an `errors` table but none of the
  `CANNOT_TRANSFER_*` names.

**Caveat carried forward:** the alias *names* are prior derivation from the
existing templates, not text confirmed against a live client (StringPool ids
4002–4008 could not be read — no String.wz in this checkout). This was accepted
when scope was set to all five steps. Confirm the rendered strings during live
re-test.

After step 4 lands, `in_family`, `is_gm` and `is_guild_master` should be
answered at CHECK time and should not normally reach the BUY-time `errors`
path; their aliases remain as a correctness backstop.

## Verification notes

- `tools/verify.sh` (flagless) must exit 0 before this branch is called done.
- The gate changes cross the atlas-character → atlas-channel seam: trace the
  new `check_unavailable` reason into `worldTransferRejectionReason` and
  `checkTransferWorldPossibleReasonKey` by hand and assert the NEW contract in
  a test, per CLAUDE.md's cross-service rule.
- `checkTransferWorldPossibleReasonArms` has an exhaustiveness test over the
  taxonomy; adding `check_unavailable` to design §6 will fail it until the arm
  table is updated. That failure is the test doing its job.

## Evidence index

All addresses are GMS v83, `MapleStory_dump.exe.i64`, IDA session `41f09cce`.

| What | Where |
|---|---|
| Cash-shop result dispatcher, arms 71–161 | `CCashShop::OnCashItemResult` `@0x47915f` |
| TRANSFER_WORLD_FAILED arm (161) | `CCashShop::OnCashItemResTransferWorldFailed` `@0x47c072` |
| The `errors` code space + default arm (id 557) | `CCashShop::NoticeFailReason` `@0x47c17a` |
| CHECK result arms 0–8, family text 5017 at arm 8 | `CCashShop::OnCheckTransferWorldPossibleResult` `@0x47bd9b` |
| Client-side pre-send gate (guild master / GM / family) | `CCashShop::CheckTransferWorldPossible` `@0x4734e5` |
| POP_UP is arm 1, unguarded `CUtilDlg::Notice` | `CWvsContext::OnBroadcastMsg` `@0xa22785` |
| `CUtilDlg::Notice` is modal | `@0x9929dd` → `CDialog::DoModal` `@0x4edba1` |

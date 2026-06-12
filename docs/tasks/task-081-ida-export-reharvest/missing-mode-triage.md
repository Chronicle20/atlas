# Missing-Mode Triage (Task 8, Part 2)

**Date:** 2026-06-09. Triage of the case↔mode bijection's `missing-mode` findings
(client dispatch cases with no Atlas `#Mode` writer) across the four IDBs.

## Bijection correctness fix first

The initial raw count was **403** missing-mode. Investigation found it inflated by a
per-address grouping bug: a handler whose `#Mode` entries span multiple addresses (v95
outlines party/guild case bodies into separate functions — `OnPartyResult` alone is at 8
addresses) had its client case-set diffed once *per address*, so every case was emitted
multiple times and a case bound at a sibling address was falsely reported missing. Fixed in
`fix(task-081): bijection groups by base handler across addresses` — accumulate each base
handler's client case-set + bound cases across all its addresses, diff once.

**Corrected: 403 → 251 distinct missing-mode** (v95 alone 267 → 123).

## The 251 are all partial-implementation gaps

| Version | missing | distinct handlers |
|---|---|---|
| gms_v83 | 33 | 5 |
| gms_v87 | 59 | 7 |
| gms_v95 | 123 | 12 |
| gms_jms_185 | 36 | 4 |

**Every handler with missing modes also has ≥1 implemented mode** — Atlas knows the handler
and deliberately implements its core sub-ops, not the long tail. The counts match real
MapleStory mode counts (e.g. v95 `OnGuildResult` ≈42 client modes, Atlas implements 2;
`OnPartyResult` ≈35, implements 3). **0 extra-mode everywhere** — no Atlas writer targets a
non-existent client case.

Recurring handlers (implemented / missing, across versions):

| Handler | nature | impl | missing (max across versions) |
|---|---|---|---|
| `CWvsContext::OnFriendResult` | buddy-list ops | 1–5 | 15 |
| `CWvsContext::OnPartyResult` | party ops | 1–3 | 32 |
| `CWvsContext::OnGuildResult` | guild ops | 2 | 40 |
| `CWvsContext::OnEntrustedShopCheckResult` | hired-merchant check | 4 | 7 |
| `CField::OnFieldEffect` | field effects | 4 | 6 |
| `CLogin::OnViewAllCharResult` | view-all-characters | 2 | 6 |
| `CLogin::OnCheckPasswordResult` | login result codes | 1 | 17 |
| `CTrunkDlg::OnPacket` | storage | 2 | 10 |
| `CWvsContext::OnGivePopularityResult` | fame | 2 | 4 |
| `CWvsContext::OnMemoResult`, `CShopDlg::OnPacket`, `CLogin::SendSelectCharPacket*` | misc | 1–2 | ≤3 |

## Action: bulk-allowlisted as known partial implementation

All 251 are seeded into per-version allowlists with reason
`"partial implementation — sub-op not built (task-081 triage 2026-06-09)"`:

- `docs/packets/audits/gms_v83/_unimplemented.json` — 33
- `docs/packets/audits/gms_v87/_unimplemented.json` — 59
- `docs/packets/audits/gms_v95/_unimplemented.json` — 123
- `docs/packets/audits/jms_v185/_unimplemented.json` — 36 (jms audit-dir naming quirk)

After allowlisting, validate reports **missing-mode 0 / extra-mode 0** on all four, with the
251 counted under `allowlisted`. The allowlist is the documented record of every
intentionally-unimplemented client sub-op (handler + case), so a *new* gap — a regression, or
a mode a future version adds that Atlas should handle — surfaces as un-allowlisted
missing-mode instead of being lost in the noise.

## How to act on a gap later

To implement a currently-allowlisted client mode: add the Atlas `#Mode` writer + a baseline
`#Mode` entry with its dispatch selector, then remove that `{fname, case}` from the version's
`_unimplemented.json`. Validate will move it from `allowlisted` to `verified` (or surface a
real divergence to fix).

## Final validate roll-up (with selectors + allowlists, 2026-06-09)

| Version | verified | divergent | missing | extra | unverifiable | allowlisted |
|---|---|---|---|---|---|---|
| v83 | 76 | 66 | 0 | 0 | 114 | 33 |
| v87 | 79 | 78 | 0 | 0 | 99 | 59 |
| v95 | 122 | 106 | 0 | 0 | 126 | 123 |
| jms | 75 | 61 | 0 | 0 | 95 | 36 |
| **Σ** | **352** | 311 | **0** | **0** | **434** | **251** |

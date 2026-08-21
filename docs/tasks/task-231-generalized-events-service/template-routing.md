# Template Routing Check — `ContiMove` (Task 30, FR-V6 / PRD §15)

PRD §15 asks that the plan's "all nine templates" claim for `ContiMove`
routing be confirmed against the real template catalog rather than trusted.
This file records the verbatim output of that check, run from the worktree
root against `services/atlas-configurations/seed-data/templates/` (per
controller ruling 1 — this is the correct, non-stale path; the
`deploy/seed/shared/all/` note applies only to the events-definitions seed).

## Commands and verbatim output

```
$ ls services/atlas-configurations/seed-data/templates/
template_gms_12_1.json  24.8K
template_gms_48_1.json  125.1K
template_gms_61_1.json  159.1K
template_gms_72_1.json  176.3K
template_gms_79_1.json  194.6K
template_gms_83_1.json  205.7K
template_gms_84_1.json  213.7K
template_gms_87_1.json  204.5K
template_gms_92_1.json  164.1K
template_gms_95_1.json  207.2K
template_jms_185_1.json  197.4K
```

```
$ grep -c '"writer"' services/atlas-configurations/seed-data/templates/*.json
services/atlas-configurations/seed-data/templates/template_gms_12_1.json:44
services/atlas-configurations/seed-data/templates/template_gms_48_1.json:122
services/atlas-configurations/seed-data/templates/template_gms_61_1.json:159
services/atlas-configurations/seed-data/templates/template_gms_72_1.json:170
services/atlas-configurations/seed-data/templates/template_gms_79_1.json:206
services/atlas-configurations/seed-data/templates/template_gms_83_1.json:220
services/atlas-configurations/seed-data/templates/template_gms_84_1.json:221
services/atlas-configurations/seed-data/templates/template_gms_87_1.json:218
services/atlas-configurations/seed-data/templates/template_gms_92_1.json:135
services/atlas-configurations/seed-data/templates/template_gms_95_1.json:226
services/atlas-configurations/seed-data/templates/template_jms_185_1.json:206
```

```
$ grep -l '"ContiMove"' services/atlas-configurations/seed-data/templates/*.json
services/atlas-configurations/seed-data/templates/template_gms_79_1.json
services/atlas-configurations/seed-data/templates/template_gms_83_1.json
services/atlas-configurations/seed-data/templates/template_gms_84_1.json
services/atlas-configurations/seed-data/templates/template_gms_87_1.json
services/atlas-configurations/seed-data/templates/template_gms_95_1.json
services/atlas-configurations/seed-data/templates/template_jms_185_1.json
```

## Observed result vs. the brief's claim

**Confirmed — the brief's numbers match what was actually observed on this
branch.** There are **eleven** template files. **Six** route `ContiMove`:
`gms_79`, `gms_83`, `gms_84`, `gms_87`, `gms_95`, `jms_185`. The remaining
five do not: `gms_12`, `gms_48`, `gms_61`, `gms_72`, `gms_92`.

This was re-run independently for this task (per controller ruling 4) rather
than copied from the brief, and the re-run output above is what is reported —
it happens to agree with the brief's figures exactly.

## Verify-marker cross-check (controller ruling 5)

`libs/atlas-packet/field/clientbound/conti_move_test.go` carries
`packet-audit:verify` markers for exactly these six versions:

```
libs/atlas-packet/field/clientbound/conti_move_test.go:17:// packet-audit:verify packet=field/clientbound/FieldContiMove version=gms_v79 ida=0x5374c1
libs/atlas-packet/field/clientbound/conti_move_test.go:18:// packet-audit:verify packet=field/clientbound/FieldContiMove version=gms_v83 ida=0x54dca3
libs/atlas-packet/field/clientbound/conti_move_test.go:19:// packet-audit:verify packet=field/clientbound/FieldContiMove version=gms_v84 ida=0x55a4e2
libs/atlas-packet/field/clientbound/conti_move_test.go:20:// packet-audit:verify packet=field/clientbound/FieldContiMove version=gms_v87 ida=0x577bbc
libs/atlas-packet/field/clientbound/conti_move_test.go:21:// packet-audit:verify packet=field/clientbound/FieldContiMove version=gms_v95 ida=0x54d680
libs/atlas-packet/field/clientbound/conti_move_test.go:22:// packet-audit:verify packet=field/clientbound/FieldContiMove version=jms_v185 ida=0x58e21b
```

That is exactly the same six versions that already route `ContiMove` in the
seed templates — `gms_79`, `gms_83`, `gms_84`, `gms_87`, `gms_95`,
`jms_185`. No verified `ContiMove` codec cell lacks a route.

## Conclusion / action taken

**No template edit was made.** The five templates without a `ContiMove`
route (`gms_12`, `gms_48`, `gms_61`, `gms_72`, `gms_92`) are partial
bring-ups — `gms_92` alone routes 135 writers against `gms_83`'s 220, and the
same pattern holds for the other four (writer counts 44/122/159/170 vs.
206–226 for the fully-routed versions). None of those five versions has a
verified `ContiMove` codec cell to route against. Adding a route there would
require deriving the opcode's wire behavior per version from the live IDB —
that is `/bringup-version` work with its own playbook
(`docs/packets/audits/STARTING_A_NEW_VERSION_PASS.md`), not this task. This
is a version-bring-up gap, not a regression this task introduces or a
silent-drop bug this task can fix.

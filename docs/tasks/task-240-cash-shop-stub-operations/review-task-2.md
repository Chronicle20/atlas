# Review: Task 2 — Register `CashShopOpen` writer for gms_95

**Range reviewed:** `7fd10bf15..66e9c0f26` (1 commit, `66e9c0f26`)
**Brief:** `.superpowers/sdd/plan/task-2-brief.md`
**Report:** `.superpowers/sdd/plan/task-2-report.md`

## Scope confirmation

`git diff --stat 7fd10bf15..66e9c0f26` shows exactly one file changed:
`services/atlas-configurations/seed-data/templates/template_gms_95_1.json`,
8 insertions, 0 deletions. This matches the brief's Step 2/4 scope exactly
— a single JSON template edit, no Go module touched. Scope confirmed, no
mismatch.

## Checklist

### 1. Is `0x8F` correct and free?

`docs/tasks/task-240-cash-shop-stub-operations/derivation.md:46` states:
`## 1. D1 — the v95 opcode for CStage::OnSetCashShop — RESOLVED: 0x8F (143)`,
and line 54 confirms `143 = 0x8F`, sourced from `CStage::OnSetCashShop` at
`0x71adf0`, dispatched from `case 143` in `CStage::OnPacket` (`0x71b0b0`,
call site `0x71b0cd`). Matches the task description's citation exactly.

Verified programmatically against the template's `socket.writers` array
(post-change):

```
0x8F in writers: 1   -> {'opCode': '0x8F', 'writer': 'CashShopOpen', 'fname': 'CStage::OnSetCashShop', 'services': ['channel']}
0x8F in handlers: 1  -> {'opCode': '0x8F', 'validator': 'LoggedInValidator', 'handler': 'MessengerOperationHandle', 'fname': 'CFadeWnd::SendCloseMessage', 'services': ['channel']}
```

Confirmed: the new entry is the *only* `0x8F` in `writers`; the pre-existing
`0x8F` in `handlers` (`MessengerOperationHandle`) is a different array and
not a collision, exactly as the task description anticipated. PASS.

### 2. Was v92's opcode copied forward?

`template_gms_92_1.json` registers `CashShopOpen` at `0x8E`
(`services/atlas-configurations/seed-data/templates/template_gms_92_1.json:2414-2415`).
The new gms_95 entry uses `0x8F`, not `0x8E` — v92's opcode was not copied
forward. On v95, `0x8E` is already claimed by `SetItc` /
`CStage::OnSetITC` (diff context, immediately preceding the new entry), so
copying v92's value would have been a real collision. PASS.

### 3. Shape and placement

New entry (`git diff` hunk):

```json
{
  "opCode": "0x8F",
  "writer": "CashShopOpen",
  "fname": "CStage::OnSetCashShop",
  "services": [
    "channel"
  ]
}
```

Carries `writer: "CashShopOpen"`, `fname: "CStage::OnSetCashShop"`,
`services: ["channel"]` — matches v92's shape (opCode aside) and the
surrounding gms_95 entries' 6-space/8-space indentation and multi-line
`services` array formatting exactly. Opcode-ordered check on the
surrounding `writers` entries:

```
0x8D SetField
0x8E SetItc
0x8F CashShopOpen   <- new
0x93 BlockedMap
0x94 BlockedServer
```

Correctly sorted between `0x8E` and `0x93`. PASS.

### 4. Blast radius

`git diff --stat` for the commit: `8 insertions(+)`, `0 deletions(-)`, one
file. The diff hunk shows only the new 9-line JSON object (8 added content
lines plus no removed lines) inserted between two existing, unmodified
entries — no reformatting or line-ending churn to the surrounding file.
PASS.

### 5. JSON parses / `CashShopOpen` appears exactly once

Verified independently (not taking the report's word for it):

```
$ python3 -c "import json;json.load(open('.../template_gms_95_1.json'));print('ok')"
ok
$ python3 -c "print(open('.../template_gms_95_1.json').read().count('CashShopOpen'))"
1
```

PASS.

### 6. No other tenant template touched

`git diff --name-only 7fd10bf15..66e9c0f26` returns exactly:

```
services/atlas-configurations/seed-data/templates/template_gms_95_1.json
```

`template_gms_92_1.json` and every other tenant template are untouched.
PASS.

## Global constraints

- Mode bytes/opcodes config-resolved, not hard-coded in Go: satisfied — this
  is a pure config/template change, no Go source touched.
- No invented values: `0x8F` traces to `derivation.md` D1, which in turn
  traces to a named IDA address and disassembly line, not to memory or to
  another version's template (v92's `0x8E` was explicitly rejected).
- Line-ending preservation: diff is purely additive (8 insertions, 0
  deletions), no evidence of CRLF/LF normalization.

## Not evaluable

- Live validation (an actual v95 tenant announcing/opening the cash shop)
  is explicitly out of scope per the brief and task instructions, deferred
  to Task 24 / post-PR testing. Not counted as a finding.

## Verdict

No blocking or non-blocking issues found. The single commit does exactly
what the brief specified: registers `CashShopOpen` at the derivation-sourced
`0x8F` in the `writers` array of `template_gms_95_1.json`, correctly
distinguishing it from the coincidental pre-existing `0x8F` handler entry,
correctly declining to copy v92's stale `0x8E`, matching surrounding style,
opcode-ordered, and confined in blast radius to the 8 added lines.

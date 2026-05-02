# Task 044 — Packet Opcode Name Alignment + Drift Logging

## Goal

Fix three handlers and one writer in `template_gms_83_1.json` that have been silently
unrouted since `fa5a0601f` (2025-12-23) by realigning `atlas-packet` constant string
values to match the template. Add `Warnf` logging to `atlas-opcodes` so future drift
between tenant config and service code surfaces immediately instead of failing silently.

The handler/writer pairs currently broken on v83:

| Direction | Template entry | Code value (current) | Code value (target) |
|---|---|---|---|
| Handler | `CompartmentMergeHandle` (op `0x45`) | `"CompartmentMerge"` | `"CompartmentMergeHandle"` |
| Handler | `CompartmentSortHandle` (op `0x46`) | `"CompartmentSort"` | `"CompartmentSortHandle"` |
| Handler | `CharacterMultiChatHandle` | `"CharacterChatMultiHandle"` | `"CharacterMultiChatHandle"` |
| Writer  | `CharacterMultiChat` | `"CharacterChatMulti"` | `"CharacterMultiChat"` |

User-visible effect: inventory merge (`0x45`), inventory sort (`0x46`), and inbound /
outbound multi-target chat (party / buddy / guild / spouse) are dropped on the floor on
v83. Other templates (`gms_12`, `gms_87`, `gms_92`, `gms_95`, `jms_185`) do not list
these entries, so they're unaffected today.

## Out of scope

- Renaming the Go identifiers (`CompartmentMergeRequestHandle`,
  `CharacterChatMultiHandle`, `MultiChatWriter`) to match their new values. The
  identifiers are referenced from many call sites and renaming them is a larger,
  cosmetic change. Listed as optional follow-up at the end.
- Touching `template_jms_185_1.json`'s orphan `CreateSecurityHandle` entry — separate
  work item; no Go constant exists for it yet.
- Touching the `CompartmentMergeWriter` / `CompartmentSortWriter` values — they already
  match `template_gms_83_1.json`'s `"CompartmentMerge"` / `"CompartmentSort"` writer
  entries. Do not change them.

## Step 1 — Realign the four packet constant values

### 1a. `libs/atlas-packet/inventory/serverbound/compartment_merge.go:12`
```diff
-const CompartmentMergeRequestHandle = "CompartmentMerge"
+const CompartmentMergeRequestHandle = "CompartmentMergeHandle"
```

### 1b. `libs/atlas-packet/inventory/serverbound/compartment_sort.go:12`
```diff
-const CompartmentSortRequestHandle = "CompartmentSort"
+const CompartmentSortRequestHandle = "CompartmentSortHandle"
```

### 1c. `libs/atlas-packet/chat/serverbound/multi.go:13`
```diff
-const CharacterChatMultiHandle = "CharacterChatMultiHandle"
+const CharacterChatMultiHandle = "CharacterMultiChatHandle"
```

### 1d. `libs/atlas-packet/chat/clientbound/multi.go:12`
```diff
-const MultiChatWriter = "CharacterChatMulti"
+const MultiChatWriter = "CharacterMultiChat"
```

No other files need to change in this step — every call site references the constants
by Go identifier (already verified by grep across `libs/` and `services/`), so the
rename of the string value is invisible to dispatch and to the `Operation()` log
strings except as the new value.

### Verification for Step 1

- `cd libs/atlas-packet && go build ./... && go test ./...` (no test pins these
  values; round-trip tests use struct identifiers).
- `cd services/atlas-channel/atlas.com/channel && go build ./... && go test ./...`
- Grep regression check: `grep -rn '"CompartmentMerge"\|"CompartmentSort"\|"CharacterChatMulti"\|"CharacterChatMultiHandle"' --include="*.go" libs/ services/` should return only the
  unchanged writer constants `CompartmentMergeWriter = "CompartmentMerge"` and
  `CompartmentSortWriter = "CompartmentSort"` (those still match the template).

## Step 2 — Warn on handler/writer name drift in `atlas-opcodes`

`libs/atlas-opcodes/producer.go` currently `continue`s without logging when:
- A tenant config handler name has no entry in the service's `handlerMap` (line 50)
- A service-declared `availableWriter` is never matched against the tenant config

Both cases are config drift. Surface them.

### 2a. `libs/atlas-opcodes/producer.go:48-51` — `BuildHandlerMap`

```diff
 		h, ok := handlerMap[hc.Handler]
 		if !ok {
+			l.Warnf("Tenant config references handler [%s] for opcode [%s], but no handler is registered.", hc.Handler, hc.OpCode)
 			continue
 		}
```

This mirrors the existing validator-not-found warning two lines above. Keeps the
silent-skip behavior (handler still isn't dispatched), just makes the drift visible.

### 2b. `libs/atlas-opcodes/producer.go:13-30` — `BuildWriterProducer`

Two complementary signals to add. Pick the inverse-side warning (declared writer
never matched) — that's the actionable case for a service author, and it can't be
noisy because `availableWriters` is hand-curated per service.

```diff
 func BuildWriterProducer(l logrus.FieldLogger, writers []WriterConfig, availableWriters []string, opWriter socket.OpWriter) sw.Producer {
 	rwm := make(map[string]sw.BodyFunc)
 	for _, wc := range writers {
 		op, err := strconv.ParseUint(wc.OpCode, 0, 16)
 		if err != nil {
 			l.WithError(err).Errorf("Unable to configure writer [%s] for opcode [%s].", wc.Writer, wc.OpCode)
 			continue
 		}

 		for _, wn := range availableWriters {
 			if wn == wc.Writer {
 				rwm[wc.Writer] = sw.MessageGetter(opWriter.Write(uint16(op)), wc.Options)
 			}
 		}
 	}
+	for _, wn := range availableWriters {
+		if _, ok := rwm[wn]; !ok {
+			l.Warnf("Service declares writer [%s] but no opcode is configured for it in tenant config.", wn)
+		}
+	}
 	return sw.ProducerGetter(rwm)
 }
```

Do **not** add the symmetric "tenant config declares writer X not in availableWriters"
warning — every channel template lists writers owned by other services (login, etc.),
so that direction would spam at startup.

### Verification for Step 2

- `cd libs/atlas-opcodes && go build ./... && go test ./...` — existing
  `registry_test.go` does not exercise `producer.go`, so no test breakage expected.
- Add a focused unit test `producer_test.go` covering both new warnings using a
  captured `logrus` hook (or a `logrus.New()` with a hook from `sirupsen/logrus/hooks/test`):
  - `TestBuildHandlerMap_WarnsOnUnknownHandler` — provide a `HandlerConfig` with a
    Handler name absent from `handlerMap`; assert one warning entry containing the
    name and opcode.
  - `TestBuildWriterProducer_WarnsOnUnconfiguredAvailableWriter` — pass an
    `availableWriters` value that has no matching `WriterConfig`; assert one warning
    entry naming it.
- Smoke check: launch atlas-channel locally against the v83 template after Step 1.
  Expect zero new warnings (all v83 handlers/writers should now match). Without Step
  1 you would see warnings for the four affected names — that's the point.

## Step 3 — Manual end-to-end smoke

After both code changes, run a v83 client against a local `docker compose` stack and
exercise:

1. **Inventory merge (`0x45`)** — drag two stacks of the same item onto each other in
   the same compartment. Pre-fix: nothing happens. Post-fix: stacks merge, client
   updates.
2. **Inventory sort (`0x46`)** — open inventory, click the auto-sort button. Pre-fix:
   no-op. Post-fix: items reorder.
3. **Multi-target chat** — type `/p hello` from a party member. Pre-fix: party
   chat never lands. Post-fix: other members see the message.

Capture screenshots / log excerpts in `audit.md` if you want a paper trail; otherwise
the PR description is fine.

## Risk

- **Cross-service ripple**: the four constants are imported by atlas-channel only
  (verified via grep). atlas-login does not consume them. Low risk.
- **Log noise**: the new `Warnf` in `BuildHandlerMap` will surface every existing
  silent miss across all services on first deploy. Expected initial set after Step 1:
  empty for v83. If other tenants/templates contain stale handler names, they'll
  appear at startup — that's a benefit, not a regression. Sweep them in a follow-up.
- **Wire compatibility**: changing the *value* of these constants has no effect on
  the wire protocol — the wire opcode is the JSON-configured byte, not these
  strings. Strings only flow through the in-process dispatcher and log messages.

## Optional follow-up (not part of this task)

- Rename Go identifiers to match values for visual consistency:
  - `CharacterChatMultiHandle` → `CharacterMultiChatHandle` in
    `libs/atlas-packet/chat/serverbound/multi.go`
  - `MultiChatWriter` → already a poor name vs new value; consider
    `CharacterMultiChatWriter` or leave alone.
  - Rename usages in `services/atlas-channel/atlas.com/channel/main.go:518` and
    `kafka/consumer/message/consumer.go:129`.
- Add a CI job that diffs `template_*.json` `handler` / `writer` strings against
  `grep -rE '(Handle|Writer)\s*=\s*"' libs/atlas-packet/` and fails on mismatch.
- Investigate `template_jms_185_1.json`'s orphan `CreateSecurityHandle`.

## Definition of done

- All four constant values updated (Step 1).
- Two `Warnf` calls added with focused tests (Step 2).
- `go build ./...` and `go test ./...` clean for `libs/atlas-opcodes`,
  `libs/atlas-packet`, and `services/atlas-channel`.
- Manual smoke in Step 3 verifies the three v83 features now route end-to-end.
- No new warnings emitted on atlas-channel startup against v83 template.

# Task 26 Review — cash-shop name-change / world-transfer credential check

- **Diff:** `aad2a35a7..768966c32` (`3264eebbd` + lint fix `768966c32`)
- **Brief:** `.superpowers/sdd/plan/task-26-brief-cont2.md`
- **Build/test:** `libs/atlas-packet` and `services/atlas-channel/atlas.com/channel` both `go build ./...` and `go test ./...` (scoped to touched packages: `cash/...`, `socket/handler`, `account`) — all PASS.

## Priority checks

### 1. Credential version split (security-critical) — PASS

Both handlers validate the SAME field the decoder populated, driven by the codec's own exported gate:

- `services/atlas-channel/atlas.com/channel/socket/handler/cash_shop_check_name_change_possible.go:111-119` `nameChangeCredentialMatches` — `cashsb.CredentialIsString(ctx)` (name-change gate) selects `p.Spw() == a.PIC()` vs `p.BirthDate() == a.BirthDate()`.
- `services/atlas-channel/atlas.com/channel/socket/handler/cash_shop_check_transfer_world_possible.go:109-117` `transferWorldCredentialMatches` — `cashsb.TransferCredentialIsString(ctx)` (transfer gate, JMS-aware) does the same.
- Codec side: `libs/atlas-packet/cash/serverbound/check_name_change_possible.go:126-131` `Decode` uses the unexported `credentialIsString` (same predicate `CredentialIsString` delegates to); `check_transfer_world_possible.go` analogous.
- Tests assert version isolation explicitly: `cash_shop_check_name_change_possible_test.go:224-247` `TestNameChangePossibleVersionPathsAreIsolated` proves pre-v95 never falls back to PIC and v95 never falls back to BirthDate.

No divergent-gate bug found.

### 2. Version gate not duplicated — PASS

- `libs/atlas-packet/cash/serverbound/check_name_change_possible.go:116-118` exports `CredentialIsString(ctx) bool { return credentialIsString(ctx) }` — thin delegate, single predicate.
- `libs/atlas-packet/cash/serverbound/check_transfer_world_possible.go:145-152` exports `TransferCredentialIsString(ctx) bool { return transferCredentialIsString(ctx) }`, which correctly differs from the name-change gate by including the JMS arm (`Region() == "JMS" || MajorAtLeast(95)` — verified in the pre-existing unexported `transferCredentialIsString`, untouched by this diff).
- The handler layer never re-derives `tenant.MustFromContext(ctx).MajorAtLeast(95)` anywhere — confirmed by grep: only call sites of `MajorAtLeast` in the touched handler files are inside the two exported gate delegates in `libs/atlas-packet`, not in `services/atlas-channel`.
- `cash_shop_check_transfer_world_possible_test.go:34-59` `TestTransferWorldPossibleJmsUsesSessionCharacterIdAndStringCredential` exercises the JMS arm end-to-end (region "JMS", major 185) and confirms the string-credential path is taken and `characterId` correctly falls back to `s.CharacterId()` via the separate `cashsb.TransferBodyHasCharacterId` gate (`check_transfer_world_possible.go:68-70`).

One predicate per op, reused by both decode and validation, as ruling 2 required.

### 3. Fail-closed on unset credential — PASS, with explicit tests

- `nameChangeCredentialMatches` (`cash_shop_check_name_change_possible.go:115-118`) and `transferWorldCredentialMatches` (`cash_shop_check_transfer_world_possible.go:113-116`) both check `if a.BirthDate() == 0 { return false }` before ever comparing to the wire value — never populated from what the client sent, matching ruling 3.
- Explicit tests: `TestNameChangePossibleUnsetStoredBirthDateAlwaysFails` (`cash_shop_check_name_change_possible_test.go:214-222`) and `TestTransferWorldPossibleUnsetStoredBirthDateAlwaysFails` (`cash_shop_check_transfer_world_possible_test.go:76-84`) both assert a `0`/`0` wire-vs-stored comparison never passes, and the name-change test additionally asserts exactly one failed PIC attempt was recorded (line 220-221).

### 4. Credential never reaches a log line — PASS

- `libs/atlas-packet/cash/serverbound/check_name_change_possible.go:99-101` `String()` returns `"characterId [%d], credential [REDACTED]"` — never formats `birthDate` or `spw`. Same for `check_transfer_world_possible.go:102-104`.
- Both handlers log only `p.String()` (`cash_shop_check_name_change_possible.go:70`, `cash_shop_check_transfer_world_possible.go:65`) — no other `%v`/`Debugf`/`Errorf` call in either file references `p.Spw()` or `p.BirthDate()`; confirmed by grep across both files (only the redaction-warning comments and the credential-compare functions touch those accessors).
- Explicit tests capture actual log output and assert non-containment of the real secret value: `TestNameChangePossibleNeverLogsTheCredential` (`cash_shop_check_name_change_possible_test.go:296-311`, both pre-v95 and v95 subtests) and `TestTransferWorldPossibleNeverLogsTheCredential` (`cash_shop_check_transfer_world_possible_test.go:112-119`).

Note: `a.BirthDate()` is passed on to `cashcb.CheckNameChangePossibleResultAllowedBody(...)` / `CheckTransferWorldPossibleResultAllowedBody(...)` (the clientbound wire body, echoing the credential back to the client per the client's own read format) — this is a wire write, not a log line, so it does not violate this requirement, but it is worth naming explicitly since it is the one place the raw birth date value flows anywhere outside the compare functions.

### RecordPicAttempt invoked on both outcomes, both version paths — PASS

- Name-change handler: failure branch calls `checkPossibleRecordPicAttemptFunc(..., false, ...)` (`cash_shop_check_name_change_possible.go:83`) unconditionally on any credential mismatch (both version paths funnel through the single `nameChangeCredentialMatches` boolean); success branch calls it with `true` (line 95) before answering ALLOWED. Same structure in the transfer handler (lines 83 and 96 of `cash_shop_check_transfer_world_possible.go`).
- Because the recording call sits after the single boolean gate rather than being duplicated per version arm, there is no version path that can skip it — verified by reading both functions top to bottom; no early return bypasses the record call on either branch.
- Tests: `TestNameChangePossibleLockoutAndSuccessRecording` (both subtests), `TestTransferWorldPossibleValidatesBeforeAnswering`, `TestTransferWorldPossibleLockoutReusesUnknownError` all assert on `env.picAttempts`.
- `services/atlas-channel/atlas.com/channel/account/processor.go:81-87` `RecordPicAttempt` and `requests.go:27-30` `requestRecordPicAttempt` mirror atlas-login's `services/atlas-login/atlas.com/login/account/processor.go:153-159` / `requests.go` pattern exactly (POST `accounts/{id}/pic-attempts`).
- `limitReached` correctly short-circuits before calling atlas-character in both handlers (there is in fact no atlas-character call in either handler at all — see gap note below, which is a documented, ruled-on scope decision, not a defect).

## Other findings

### Missing: FR-4.7 pink-text storage warning (brief-cont2, "Still required from the original brief") — **finding, not blocking per se but a stated requirement was dropped silently**

`task-26-brief-cont2.md:202-217` lists, under "Still required from the original brief": *"The FR-4.7 pink-text storage warning (Step 4) — advisory; it must not flip the result to unavailable."* It names the pattern to copy (`services/atlas-channel/atlas.com/channel/socket/writer/world_message.go:107` `WorldMessagePinkTextBody`) and a reference handler (`character_cash_item_use_point_reset.go`).

Grepped the full diff (`git diff aad2a35a7..768966c32 | grep -i pink`) and both handler files (`cash_shop_check_transfer_world_possible.go`, `cash_shop_check_name_change_possible.go`) for `pink`/`Pink`/`storage` — **zero matches**. `writer.WorldMessagePinkTextBody` is never called. Unlike the destination-dependent gates and the world-name list, which are explicitly documented in the handler's doc comment as reported gaps (`cash_shop_check_transfer_world_possible.go:32-57`), the pink-text warning has no such acknowledgment anywhere in code or comments — it is simply absent, with nothing telling a future reader it was dropped.

This is a requirement the authoritative brief explicitly re-asserted as still owed, and the report/implementation gives no reasoned deferral for it the way it does for the destination-dependent gates (ruling 5) and the world-name list. Flag for follow-up before calling Task 26 complete.

### EXT-02 gap: no httptest-backed integration test for the new `RecordPicAttempt` REST client

`services/atlas-channel/atlas.com/channel/account/requests.go:27-30` adds `requestRecordPicAttempt`, exercised through `requests.PostRequest[PicAttemptOutputRestModel]`. No test in the diff or the package drives this through an actual HTTP round-trip and JSON unmarshal:

- The two new handler test files swap out `checkPossibleRecordPicAttemptFunc` entirely (`cash_shop_check_name_change_possible_test.go:97-101`), so they never touch `account.ProcessorImpl.RecordPicAttempt` or the real request/unmarshal path.
- `services/atlas-channel/atlas.com/channel/account/processor_drain_test.go:51` has an `httptest.NewServer`, but it covers `AllProvider`'s drain path, not `pic-attempts`.
- No new `_test.go` in this diff adds an `httptest` fixture for `pic-attempts`.

Per the External HTTP Client checklist, "FakeClient mocks alone do NOT satisfy this — they bypass unmarshal," which is exactly what the seam-swap does here. Note: `services/atlas-login/atlas.com/login/account` has the identical gap for its own `RecordPicAttempt` (also no httptest coverage), so this is not a regression relative to the precedent it mirrors — but per this review's standing instruction, prevalence is not compliance, and it is recorded as a genuine (pre-existing-pattern) finding rather than waved through.

## Context items intentionally not re-litigated (per instructions)

- Only two clientbound `Check*` writers registered in `main.go` (`services/atlas-channel/atlas.com/channel/main.go:642-643`) — correct, `CashShopCheckNameChangeWriter` answers a different serverbound op with no emitter here. Confirmed by re-reading `main.go:637-644`; not flagged.
- Neither handler evaluates eligibility gates (is_gm/banned/guild-master/etc.) — per user ruling, enforced at `pending_change/processor.go:205 Create()`. Confirmed both handler doc comments (`cash_shop_check_transfer_world_possible.go:32-51`) document this explicitly as a reported design gap, not a silent omission — this is the correct treatment, in contrast to the pink-text gap above which has no such acknowledgment. Not flagged.

## Template validator check (ruling 6)

Verified `services/atlas-configurations/seed-data/templates/template_{gms_83,gms_84,gms_87,gms_92,gms_95,jms_185}_1.json` `socket.handlers[]`:
- `NAME_TRANSFER` (`CashShopCheckNameChangePossibleHandle`) present with `validator: "LoggedInValidator"` on all five GMS versions (v83/84/87/92/95); correctly absent on jms_v185 per the codec's own documentation that jms has no name-change feature.
- `WORLD_TRANSFER` (`CashShopCheckTransferWorldPossibleHandle`) present with `validator: "LoggedInValidator"` on all five GMS versions AND jms_v185.
- No empty-validator entries found for either handler on any applicable version.

## Summary

**Blocking:**
- FR-4.7 pink-text storage warning is entirely absent from the implementation despite being explicitly re-listed as "still required" in the authoritative brief, with no documented-gap acknowledgment in code (unlike the other, properly-documented scope gaps in the same handler).

**Non-blocking (should fix):**
- EXT-02: no httptest-backed test for the new `account.RecordPicAttempt` REST round-trip (mirrors a pre-existing gap in atlas-login's identical pattern, but is still a gap against the guideline).

**Verified PASS, no defects found:**
- Credential version split (priority 1): handler validates the field the decoder populated on every version path.
- Version gate not duplicated (priority 2): both ops export and reuse their own codec-owned predicate; the JMS-aware world-transfer gate is distinct from the name-change gate and used correctly.
- Fail-closed on unset credential (priority 3): explicit tests cover the `0` case for both ops.
- Credential never reaches a log line (priority 4): `String()` redacts on both codecs; explicit tests assert non-leakage.
- `RecordPicAttempt` invoked on both outcomes and both version paths for both ops.
- Template validator wiring (ruling 6) is complete and non-empty across all applicable versions.

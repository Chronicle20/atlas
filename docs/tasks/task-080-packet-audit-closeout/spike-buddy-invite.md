# Spike: BuddyInvite clientbound packet — inviter jobId+level (task-080 follow-up #39)

## Problem

Atlas's `BuddyInvite` (CWvsContext::OnFriendResult case 9) omitted the inviter's
`jobId` + `level` int4 fields for GMS v87/v95 and JMS185, corrupting the packet on
those clients (the `CreateFriendReg` dialog reads two int4 there, so every byte
after the originator name was misaligned).

## Four-version decompiled read-order (IDA — `CWvsContext::OnFriendResult` case 9)

The `BuddyOperation` dispatcher consumes the mode byte (=9). The remaining body:

| Version | Address    | Read-order                                                                                          | jobId/level |
|---------|------------|-----------------------------------------------------------------------------------------------------|-------------|
| GMS v83 | `0xa3f2e8` | `Decode4 originatorId`, `DecodeStr originatorName`, `GW_Friend(39)`, `Decode1 inShop`                | **absent**  |
| GMS v87 | `0xad7ae5` | `Decode4 originatorId`, `DecodeStr originatorName`, `Decode4 jobId`, `Decode4 level`, `GW_Friend(39)`, `Decode1 inShop` | present |
| GMS v95 | `0xa12630` | same as v87                                                                                          | present     |
| JMS185  | `0xb2a873` | same as v87                                                                                          | present     |

`GW_Friend` is the 39-byte buddy buffer: `int4 friendId` + `char[13] name` +
`byte flag` + `int4 channel` + `char[17] group`. The `inShop` trailing byte is
present in **all** versions. Atlas already wrote `GW_Friend` (via `model.Buddy`)
and the `inShop` byte correctly in every version; only `jobId`/`level` were
missing.

`jobId`/`level` are the **inviter's** (originator's) job id and level — consumed
by the client's `CreateFriendReg` dialog ("X (Job, Lv.Y) wants to add you").
They go BETWEEN the originator name and the `GW_Friend` buffer.

## Gate

```go
hasJobLevel := t.Region() != "GMS" || t.MajorVersion() >= 87
```

- GMS v28 → false (no fields)
- GMS v83 → false (no fields)
- GMS v87 / v95 → true (fields present)
- JMS185 → true (fields present)

Applied symmetrically in both `Encode` and `Decode`. For v28/v83 the fields are
off-wire, so they round-trip back as 0.

## Mistraced JSON exports (accepted exclusion)

THREE of the four `docs/packets/ida-exports/*.json` read-orders for BuddyInvite
are **mistraced** relative to the verified decompiles:

- **v83 export** models the body as a `count + loop` (variable-length friend
  list) — wrong. The real v83 body is a single `GW_Friend(39)` buffer + `inShop`.
- **JMS185 export** omits the `GW_Friend` buffer entirely ("no friend buffer") —
  wrong. JMS185 has the same `GW_Friend(39)` + `inShop` tail as GMS.

The real shape is `GW_Friend(39) + inShop` in **all** versions, with `jobId`/
`level` present only on the ≥87/JMS gate. Because the exports are wrong, the
packet-audit SUMMARYs may still flag BuddyInvite (❌/🔍) — that residual is an
**export truncation/mistrace accepted-exclusion**, NOT an Atlas defect. Atlas's
wire is now IDA-correct per the four decompiles. Do not "fix" Atlas to match a
mistraced export.

## Value-population disposition

**Real values wired** (not 0). The invite consumer
(`services/atlas-channel/.../kafka/consumer/invite/consumer.go`) already fetches
the originator character `rc` via `character.NewProcessor(...).GetById()(OriginatorId)`
and already passes `rc.JobId()`/`rc.Level()` into the party invite path. The
buddy path now threads the same `uint32(rc.JobId())`, `uint32(rc.Level())` into
`BuddyInviteBody(...)`. No extra fetch and no upstream event change were needed.

## Verification

- `go test -race ./buddy/...` (atlas-packet) — pass, incl. per-version byte-shape
  test (v83/v28 = 57 bytes no fields; v87/v95/JMS = 65 bytes with jobId/level int4
  LE right after the name) + round-trip.
- `go vet ./...` (atlas-packet) — clean.
- `go build ./...` / `go vet ./...` (atlas-channel) — clean.

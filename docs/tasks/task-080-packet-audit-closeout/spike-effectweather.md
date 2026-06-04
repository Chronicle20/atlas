# Spike: EffectWeather JMS185 read-order (task-080 B1.5)

## Verified client read-order (JMS185 `sub_5723E6` — the weather handler)

Decompiled JMS185 reads the `FieldEffectWeather` body as:

1. `itemId = Decode4()` — LEADING `itemId`, with **no** active/bool byte first.
2. `if get_consume_cash_item_type(itemId) == 51 { extra = Decode4() }` — an
   optional 4-byte `extra`, read **after** `itemId` and **before** the message.
3. `if itemId != 0 { message = DecodeStr() }` — an optional ASCII string.

So the JMS encode order is:
`itemId(4)`, then `extra(4)` only when the type-51 condition holds, then
`message` only when `itemId != 0`.

## GMS (v28/v83/v87/v95) — unchanged

GMS was already correct and stays byte-identical to before this task:

```
WriteBool(!active); WriteInt(itemId); if active { WriteAsciiString(message) }
```

## Decision

- Region-dispatch the body at the top of `Encode`/`Decode`:
  `t.Region() == "JMS"` → `encodeJMS`/`decodeJMS`, else `encodeGMS`/`decodeGMS`.
  This keeps each body at ≤2 guards (no 3rd nested `if`), per design §3.2.
- `encodeGMS`/`decodeGMS` are the verbatim pre-task GMS bytes (leading `!active`
  bool, then `itemId`, then `message` when `active`).
- `encodeJMS`/`decodeJMS` follow the verified read-order: `itemId`, optional
  `extra`, optional `message` (when `itemId != 0`).
- Added struct fields `extra uint32` + `hasExtra bool`. The `extra`/type-51 path
  is **encode-only / server-driven**: the Go decode side cannot call the
  client's `get_consume_cash_item_type`, so `hasExtra` is carried on the struct.
  Encode emits `extra` iff `hasExtra`; decode reads it back iff `hasExtra` on the
  same struct, so round-trip stays byte-symmetric. `NewFieldEffectWeatherStart`
  produces `hasExtra=false` (normal weather start, no cash-item extra), so the
  optional `extra` is not exercised by round-trip tests and is documented as
  encode-only.

## Verification

- `go test -race ./field/...` — clean (GMS v28/v83/v87/v95 + JMS v185 round-trip).
- `go vet ./...` — clean.
- New `TestEffectWeatherJMSBranch` asserts JMS leading `itemId` at `b[0:4]` and
  GMS leading bool `0x00` at `g[0]` (start packet → `!active == false`).
- GMS bytes confirmed unchanged: `encodeGMS`/`decodeGMS` are the verbatim
  pre-task three-line bodies; the existing `TestFieldEffectWeatherStart` /
  `TestFieldEffectWeatherEnd` round-trips still pass for all GMS variants.

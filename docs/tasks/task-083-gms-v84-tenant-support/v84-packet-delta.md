# v83 → v84 Packet / Opcode / Version-Branch Delta

Source of truth for task-083 (FR-1.4). Every code/template change cites a row here.

## 0. IDB inventory & dispatch-table anchors
| IDB | port | dispatch table (inbound) addr | dispatch table (outbound) addr | naming density |
|---|---|---|---|---|

## 1. Inbound (handler) opcode map  (FR-1.1, FR-1.3)
| logical name | v83 opcode | v84 opcode | classification | evidence (IDB fn/addr or ref version) |
|---|---|---|---|---|

## 2. Outbound (writer) opcode map  (FR-1.1, FR-1.3)
| logical name | v83 opcode | v84 opcode | classification | evidence |
|---|---|---|---|---|

## 3. Packet-structure delta (FR-1.2)
### 3.1 In-scope flows (exhaustive): login handshake, auth, world/channel list, character list, character select / PIC-PIN, enter-channel, map load (spawn/field), movement, chat
### 3.2 Spot-checked elsewhere (what was checked, what was assumed)

## 4. usesPin determination (OQ-1)

## 5. Version-branch audit table (FR-3.1, FR-3.3)
| branch site (file:line) | predicate | v83 result | v84 result | correct for v84? | action | delta evidence |
|---|---|---|---|---|---|---|

## 6. Provisioning runbook (FR-5.1) + restart sequence (OQ-6)

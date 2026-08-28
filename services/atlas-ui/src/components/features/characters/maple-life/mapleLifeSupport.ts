import type { SocketConfig } from "@/types/models/socket";

/**
 * The handler *implementation name* that gates Maple Life support, never an
 * opcode: opcodes differ per version (0x100 / 0x10E / 0x12D / 0x137,
 * `libs/atlas-packet/maplelife/serverbound/check_name.go:13`), but the
 * implementation name is stable across versions that ship the block.
 *
 * Cross-checked against all eleven seed templates under
 * `services/atlas-configurations/seed-data/templates/`: presence of a
 * `socket.handlers` entry with this handler (plus the `MapleLifeResult` /
 * `MapleLifeError` writers) matches presence of the `mapleLife` block
 * exactly. Note `gms_84_1` has neither handler nor block, so a
 * `majorVersion >= 83` rule would be wrong — this predicate, not a version
 * cutoff, is the source of truth.
 */
export const MAPLE_LIFE_HANDLER = "MapleLifeCheckNameHandle";

/**
 * True iff `socket.handlers` contains an entry implemented by
 * `MAPLE_LIFE_HANDLER`. An absent/undefined socket (still loading, or no
 * data) is treated as unsupported.
 */
export function supportsMapleLife(socket: SocketConfig | undefined): boolean {
  return (
    socket?.handlers.some((h) => h.handler === MAPLE_LIFE_HANDLER) ?? false
  );
}

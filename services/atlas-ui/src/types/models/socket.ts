/**
 * The socket configuration shape, shared by Templates and Tenants.
 *
 * This was previously declared inline in BOTH types/models/template.ts and
 * services/api/tenants.service.ts. Both needed the same two new fields, so the
 * shape now lives here and both import it.
 */

/** One serverbound entry: opcode -> validator -> handler implementation. */
export interface SocketHandlerEntry {
  opCode: string;
  validator: string;
  handler: string;
  /**
   * Client-side function name, e.g. "CLogin::SendCheckPasswordPacket".
   * Informational only - it never participates in comparison, validation or
   * ancestry classification (PRD FR-10.4). Optional and omitted when empty.
   */
  fname?: string;
  /** Free-form wire tables the codec reads at runtime. Absent when unset. */
  options?: unknown;
  services?: string[];
}

/** One clientbound entry: opcode -> writer implementation. */
export interface SocketWriterEntry {
  opCode: string;
  writer: string;
  /** See SocketHandlerEntry.fname. */
  fname?: string;
  options?: unknown;
  services?: string[];
}

/**
 * Definitions audited and confirmed absent for this Region/Version. Holding
 * them here rather than in the arrays is what makes "audited, this version does
 * not have this packet" distinguishable from "nobody has looked yet".
 *
 * Entries are IMPLEMENTATION NAMES, never opcodes.
 */
export interface SocketUnsupported {
  handlers: string[];
  writers: string[];
}

export interface SocketConfig {
  handlers: SocketHandlerEntry[];
  writers: SocketWriterEntry[];
  /** Optional for backwards compatibility: absent means both lists are empty. */
  unsupported?: SocketUnsupported;
}

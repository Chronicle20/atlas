/**
 * The only place a stored opcode string is interpreted.
 *
 * Stored values are NEVER rewritten: canonicalization is display-only, because
 * rewriting "0x0B8" to "0xB8" on save would be a data change this task does not
 * make. Two bindings whose parsed values are equal are surfaced as a duplicate,
 * not silently merged.
 */

/** Matches the accepted wire form: 0x/0X followed by 1-4 hex digits. */
const STORED = /^0[xX][0-9A-Fa-f]{1,4}$/;

/** Parses a stored opcode, or returns null if it is not in the wire form. */
export function parseOpcode(raw: string): number | null {
  const s = raw.trim();
  if (!STORED.test(s)) return null;
  const n = Number.parseInt(s.slice(2), 16);
  return Number.isNaN(n) ? null : n;
}

/** Renders the canonical display form: 0x + upper-case hex, at least 2 digits. */
export function formatOpcode(n: number): string {
  return `0x${n.toString(16).toUpperCase().padStart(2, "0")}`;
}

/**
 * FR-4.3. A search query matches an opcode value if it reads as that value
 * under any plausible interpretation:
 *
 *   "0x2A" -> hex only            -> 42
 *   "2A"   -> hex only (has a-f)  -> 42
 *   "42"   -> hex OR decimal      -> 66 or 42
 *
 * The ambiguity is deliberate. A bare "42" is exactly as likely to mean the
 * decimal opcode as the hex one, and a search that silently picked one would
 * hide the other.
 */
export function matchesOpcodeQuery(query: string, value: number): boolean {
  const q = query.trim();
  if (q === "") return false;

  if (/^0[xX][0-9A-Fa-f]{1,4}$/.test(q)) {
    return Number.parseInt(q.slice(2), 16) === value;
  }
  if (/^[0-9A-Fa-f]{1,4}$/.test(q)) {
    if (Number.parseInt(q, 16) === value) return true;
    if (/^[0-9]+$/.test(q) && Number.parseInt(q, 10) === value) return true;
  }
  return false;
}

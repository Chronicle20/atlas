/**
 * `<input type="datetime-local">` yields a local wall-clock string with no
 * zone ("2026-06-01T09:30"). The coupon API takes RFC 3339, so the value is
 * interpreted in the operator's own timezone (which is what `new Date(...)`
 * does for a zoneless datetime string) and re-emitted as UTC.
 */
export function localInputToIso(local: string): string {
  const parsed = new Date(local);
  if (Number.isNaN(parsed.getTime())) return local;
  return parsed.toISOString();
}

/** The inverse, for pre-filling a datetime-local input from a stored value. */
export function isoToLocalInput(iso: string | undefined): string {
  if (!iso) return "";
  const parsed = new Date(iso);
  if (Number.isNaN(parsed.getTime())) return "";
  const offsetMs = parsed.getTimezoneOffset() * 60_000;
  return new Date(parsed.getTime() - offsetMs).toISOString().slice(0, 16);
}

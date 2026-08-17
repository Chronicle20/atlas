/**
 * An account's birth date is stored as an INTEGER in `yyyymmdd` form
 * (atlas-account `account.BirthDate uint32`), not as a date or a timestamp.
 * The client sends the same shape on the wire as the account's second
 * password on pre-v95 name-change and world-transfer requests, which is why
 * the field exists at all.
 *
 * 0 means UNSET. atlas-account treats 0 as "leave unchanged" on PATCH and
 * atlas-channel treats a stored 0 as a failed credential check, so an unset
 * birth date is never the same as a birth date that happens to be zero.
 *
 * These helpers convert between that integer and the `yyyy-mm-dd` string an
 * `<input type="date">` reads and writes. They are deliberately timezone-free:
 * a birth date is a calendar date, so parsing it through `Date` would shift it
 * by a day for operators west of UTC.
 */

/** Formats a stored yyyymmdd integer as the `yyyy-mm-dd` an input expects. */
export function birthDateToInput(birthDate: number | undefined): string {
  if (!birthDate) return "";
  const s = String(birthDate).padStart(8, "0");
  if (s.length !== 8) return "";
  return `${s.slice(0, 4)}-${s.slice(4, 6)}-${s.slice(6, 8)}`;
}

/**
 * Parses a `yyyy-mm-dd` input value into the stored yyyymmdd integer, or
 * `null` when the value is empty or not a real calendar date. Callers treat
 * `null` as "reject the submission" rather than as 0 — 0 would round-trip
 * through atlas-account as "no change", silently doing nothing.
 */
export function inputToBirthDate(value: string): number | null {
  const m = /^(\d{4})-(\d{2})-(\d{2})$/.exec(value.trim());
  if (!m) return null;

  const year = Number(m[1]);
  const month = Number(m[2]);
  const day = Number(m[3]);
  if (year < 1900 || month < 1 || month > 12 || day < 1) return null;
  if (day > daysInMonth(year, month)) return null;

  return year * 10000 + month * 100 + day;
}

/** Renders a stored yyyymmdd integer for display. "Not set" when unset. */
export function formatBirthDate(birthDate: number | undefined): string {
  const input = birthDateToInput(birthDate);
  if (!input) return "Not set";
  return input;
}

const MONTH_LENGTHS = [31, 28, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31] as const;

function daysInMonth(year: number, month: number): number {
  if (month === 2 && isLeapYear(year)) return 29;
  return MONTH_LENGTHS[month - 1] ?? 0;
}

function isLeapYear(year: number): boolean {
  return (year % 4 === 0 && year % 100 !== 0) || year % 400 === 0;
}

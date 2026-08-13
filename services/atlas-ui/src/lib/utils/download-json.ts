/**
 * Hand a JSON payload to the browser's download mechanism.
 *
 * Serialisation happens BEFORE createObjectURL, so a throw (cyclic structure,
 * BigInt) cannot leak an object URL - there is nothing to revoke yet. The
 * anchor teardown and the revoke live in a `finally` so a throw from click()
 * cannot leak either.
 *
 * The trailing newline matches the checked-in seed files under
 * services/atlas-configurations/seed-data/templates/, which all end with one.
 */
export function downloadJson(filename: string, payload: unknown): void {
  const body = `${JSON.stringify(payload, null, 2)}\n`;
  const url = URL.createObjectURL(
    new Blob([body], { type: "application/json" }),
  );
  const anchor = document.createElement("a");
  try {
    anchor.href = url;
    anchor.download = filename;
    document.body.appendChild(anchor);
    anchor.click();
  } finally {
    // remove() rather than removeChild() - a no-op when the append never
    // happened, which keeps this block total.
    anchor.remove();
    URL.revokeObjectURL(url);
  }
}

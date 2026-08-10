/**
 * react-hook-form nests a field array's errors as an array of per-field error
 * objects (`errors.rewards[0].amount.message`), and the exact nesting depends
 * on which branch of `rewardRowSchema`'s discriminated union failed. Rather
 * than enumerate the branches, these helpers walk the error node and surface
 * the first message they find — one line per row is all the editor shows.
 */

/** Depth-first search for the first `message` string under `node`. */
export function firstMessage(node: unknown): string | undefined {
  if (node === null || typeof node !== "object") return undefined;
  const record = node as Record<string, unknown>;
  if (typeof record["message"] === "string" && record["message"] !== "") {
    return record["message"];
  }
  for (const value of Object.values(record)) {
    const found = firstMessage(value);
    if (found) return found;
  }
  return undefined;
}

export interface RewardFieldErrors {
  rowErrors: (string | undefined)[];
  arrayError: string | undefined;
}

/**
 * Split a `rewards` error node into per-row messages and the array-level
 * message (the `min(1)` failure, which RHF may hang off `.root` or the node
 * itself depending on version).
 */
export function splitRewardErrors(
  node: unknown,
  rowCount: number,
): RewardFieldErrors {
  const rowErrors: (string | undefined)[] = [];
  for (let i = 0; i < rowCount; i++) {
    rowErrors.push(
      Array.isArray(node) ? firstMessage((node as unknown[])[i]) : undefined,
    );
  }

  let arrayError: string | undefined;
  if (node !== null && typeof node === "object") {
    const record = node as Record<string, unknown>;
    arrayError =
      (typeof record["message"] === "string" && record["message"] !== ""
        ? record["message"]
        : undefined) ?? firstMessage(record["root"]);
  }

  return { rowErrors, arrayError };
}

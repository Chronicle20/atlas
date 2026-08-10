/**
 * Batched id → name resolution for accounts and characters.
 *
 * Audit views (coupon redemptions, and anything else that records "who did
 * this" as a bare numeric id) need a name per row without mounting a hook per
 * row. Both hooks reuse the existing `accountKeys.detail` / `characterKeys.
 * detail` query keys, so a resolved name is shared with the account/character
 * detail pages rather than fetched twice.
 *
 * `undefined` for an id means "still loading, or the lookup failed" — callers
 * degrade to the numeric id rather than blocking the row.
 */

import { useQueries } from "@tanstack/react-query";
import { accountsService } from "@/services/api/accounts.service";
import { charactersService } from "@/services/api/characters.service";
import { accountKeys } from "@/lib/hooks/api/useAccounts";
import { characterKeys } from "@/lib/hooks/api/useCharacters";
import { useTenant } from "@/context/tenant-context";
import type { Tenant } from "@/types/models/tenant";

/**
 * Distinct, truthy ids — id 0 is "none", never a real account/character.
 * Empty without a tenant, so no query is even constructed until there is one
 * (which also keeps every key identical to the detail pages' keys, both of
 * which fold a non-null tenant in). The hooks below still carry the usual
 * `enabled: !!activeTenant` rather than leaning on this as the only guard.
 */
function lookupIds(ids: number[], tenant: Tenant | null): number[] {
  if (!tenant) return [];
  return Array.from(new Set(ids.filter((id) => id > 0)));
}

function toRecord(
  ids: number[],
  results: { data?: string | undefined }[],
): Record<number, string | undefined> {
  const names: Record<number, string | undefined> = {};
  ids.forEach((id, i) => {
    names[id] = results[i]?.data;
  });
  return names;
}

export function useAccountNames(
  ids: number[],
): Record<number, string | undefined> {
  const { activeTenant } = useTenant();
  const unique = lookupIds(ids, activeTenant);
  const results = useQueries({
    queries: unique.map((id) => ({
      queryKey: accountKeys.detail(activeTenant, String(id)),
      queryFn: async () => {
        const account = await accountsService.getAccountById(String(id));
        return account.attributes.name;
      },
      enabled: !!activeTenant,
      staleTime: 10 * 60 * 1000,
      gcTime: 30 * 60 * 1000,
    })),
  });
  return toRecord(unique, results);
}

export function useCharacterNames(
  ids: number[],
): Record<number, string | undefined> {
  const { activeTenant } = useTenant();
  const unique = lookupIds(ids, activeTenant);
  const results = useQueries({
    queries: unique.map((id) => ({
      queryKey: characterKeys.detail(activeTenant, String(id)),
      queryFn: async () => {
        const character = await charactersService.getById(String(id));
        return character.attributes.name;
      },
      enabled: !!activeTenant,
      staleTime: 5 * 60 * 1000,
      gcTime: 30 * 60 * 1000,
    })),
  });
  return toRecord(unique, results);
}

import {
  useQueries,
  useQuery,
  type UseQueryResult,
} from "@tanstack/react-query";
import { useTenant } from "@/context/tenant-context";
import {
  liveMonstersService,
  type LiveMonsterData,
} from "@/services/api/live-monsters.service";
import {
  fieldsService,
  type FieldCharacterData,
} from "@/services/api/fields.service";
import { charactersService } from "@/services/api/characters.service";
import { characterKeys } from "@/lib/hooks/api/useCharacters";
import type { Character } from "@/types/models/character";

// Runtime read model (live field state) — short-lived cache, no polling
// (FR-39). Deliberately omits tenant from the key (D9): tenant-context.tsx
// clears the whole query client on tenant change. `monsters` is the first of
// several field-runtime concerns landing on this namespace across Tasks
// 18/19/22 — add one exported hook per concern, keyed under this object.
export const fieldRuntimeKeys = {
  monsters: (w: number, c: number, m: number, i: string) =>
    ["fields", w, c, m, i, "monsters"] as const,
  characters: (w: number, c: number, m: number, i: string) =>
    ["fields", w, c, m, i, "characters"] as const,
};

const RUNTIME_STALE_TIME = 5 * 1000;
const RUNTIME_GC_TIME = 60 * 1000;

export function useLiveMonsters(
  worldId: number,
  channelId: number,
  mapId: number,
  instanceId: string,
  enabled = true,
): UseQueryResult<LiveMonsterData[], Error> {
  const { activeTenant } = useTenant();
  return useQuery({
    queryKey: fieldRuntimeKeys.monsters(worldId, channelId, mapId, instanceId),
    queryFn: () =>
      liveMonstersService.getMonsters(worldId, channelId, mapId, instanceId),
    staleTime: RUNTIME_STALE_TIME,
    gcTime: RUNTIME_GC_TIME,
    enabled: enabled && !!activeTenant,
  });
}

// FR-22: the character roster is also the field's liveness signal — an
// empty (resolved) result means the field has been torn down, not an error.
export function useFieldCharacters(
  worldId: number,
  channelId: number,
  mapId: number,
  instanceId: string,
  enabled = true,
): UseQueryResult<FieldCharacterData[], Error> {
  const { activeTenant } = useTenant();
  return useQuery({
    queryKey: fieldRuntimeKeys.characters(
      worldId,
      channelId,
      mapId,
      instanceId,
    ),
    queryFn: () =>
      fieldsService.getFieldCharacters(worldId, channelId, mapId, instanceId),
    staleTime: RUNTIME_STALE_TIME,
    gcTime: RUNTIME_GC_TIME,
    enabled: enabled && !!activeTenant,
  });
}

/**
 * Batched character-detail enrichment for a field's character roster
 * (bug-fields-ui-round2 item 4). The map-image overlay needs `x`/`y` per
 * character at page scope, above the tabs, so it can't reuse the per-row
 * `useCharacter` call inside `FieldCharacterRow`. This hook uses the
 * identical `characterKeys.detail` query key and `charactersService.getById`
 * query fn, so React Query dedupes it against the table rows' own queries —
 * no extra network requests, and the existing Refresh path (which
 * invalidates `characterKeys.details()`) keeps refreshing both.
 */
export function useFieldCharacterDetails(
  characterIds: string[],
): UseQueryResult<Character, Error>[] {
  const { activeTenant } = useTenant();
  return useQueries({
    queries: characterIds.map((characterId) => ({
      queryKey: characterKeys.detail(activeTenant, characterId),
      queryFn: () =>
        charactersService.getById(characterId, { useCache: false }),
      enabled: !!activeTenant,
      gcTime: 5 * 60 * 1000,
    })),
  });
}

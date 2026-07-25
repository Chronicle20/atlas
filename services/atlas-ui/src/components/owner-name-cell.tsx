import type { Tenant } from "@/types/models/tenant";
import { useEffect, useState } from "react";
import { charactersService } from "@/services/api/characters.service";

const characterNameCache = new Map<string, string>();

// Resolves a character id to its display name, caching results across rows.
// Falls back to the id while loading or if the lookup fails. The caller is
// responsible for any surrounding <Link> so the name stays clickable.
export function OwnerNameCell({
  characterId,
  tenant,
}: {
  characterId: string;
  tenant: Tenant | null;
}) {
  const [resolved, setResolved] = useState<Record<string, string>>({});

  useEffect(() => {
    if (!tenant || !characterId || characterNameCache.has(characterId)) return;

    let cancelled = false;
    charactersService
      .getById(characterId)
      .then((character) => {
        const characterName = character.attributes.name;
        characterNameCache.set(characterId, characterName);
        if (!cancelled)
          setResolved((prev) => ({ ...prev, [characterId]: characterName }));
      })
      .catch(() => {
        // Fall back to the id on failure.
        if (!cancelled)
          setResolved((prev) => ({ ...prev, [characterId]: characterId }));
      });
    return () => {
      cancelled = true;
    };
  }, [characterId, tenant]);

  const name =
    characterNameCache.get(characterId) ?? resolved[characterId] ?? characterId;
  return <>{name}</>;
}

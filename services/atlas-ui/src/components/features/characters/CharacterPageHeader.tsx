// services/atlas-ui/src/components/features/characters/CharacterPageHeader.tsx
import { Shield, MapPin } from "lucide-react";
import type { Character } from "@/types/models/character";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { CopyableIdHeader } from "@/components/common/CopyableIdHeader";

interface Props {
  character: Character;
  onChangeGm: () => void;
  onChangeMap: () => void;
}

export function CharacterPageHeader({ character, onChangeGm, onChangeMap }: Props) {
  const gm = character.attributes.gm ?? 0;
  return (
    <CopyableIdHeader
      title={character.attributes.name}
      id={character.id}
      actions={
        <>
          {gm > 0 && <Badge variant="destructive">GM {gm}</Badge>}
          <Button variant="outline" size="sm" onClick={onChangeGm}>
            <Shield className="mr-1 h-4 w-4" />
            {gm > 0 ? "Change GM" : "Promote to GM"}
          </Button>
          <Button variant="outline" size="sm" onClick={onChangeMap}>
            <MapPin className="mr-1 h-4 w-4" />
            Change Map
          </Button>
        </>
      }
    />
  );
}

import { useState } from "react";
import { Sparkles, X } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { useSkillData } from "@/lib/hooks/useSkillData";
import { ItemSearchCombobox } from "./ItemSearchCombobox";
import { ItemRow } from "./ItemRow";

interface StartingKitSectionProps {
  items: number[];
  skills: number[];
  onAddItem: (id: number) => void;
  onRemoveItem: (entryIndex: number) => void;
  onAddSkill: (id: number) => void;
  onRemoveSkill: (entryIndex: number) => void;
}

function SkillRow({ id, onRemove }: { id: number; onRemove: () => void }) {
  const skill = useSkillData(id);
  const [iconFailed, setIconFailed] = useState(false);
  return (
    <div className="flex items-center gap-2 rounded-md border px-2 py-1.5">
      {skill.iconUrl && !iconFailed ? (
        <img
          src={skill.iconUrl}
          alt=""
          width={28}
          height={28}
          loading="lazy"
          onError={() => setIconFailed(true)}
          className="[image-rendering:pixelated]"
        />
      ) : (
        <Sparkles className="size-7 p-1 text-muted-foreground" />
      )}
      <span className="flex-1 truncate text-sm">
        {skill.name ?? "Unknown skill"}
      </span>
      <span className="font-mono text-xs text-muted-foreground">{id}</span>
      <Button
        type="button"
        variant="ghost"
        size="icon"
        aria-label={`Remove skill ${id}`}
        onClick={onRemove}
      >
        <X className="size-4" />
      </Button>
    </div>
  );
}

export function StartingKitSection({
  items,
  skills,
  onAddItem,
  onRemoveItem,
  onAddSkill,
  onRemoveSkill,
}: StartingKitSectionProps) {
  const [skillInput, setSkillInput] = useState("");
  const skillId = /^\d+$/.test(skillInput.trim())
    ? Number(skillInput.trim())
    : undefined;

  return (
    <section className="space-y-4">
      <div className="space-y-2">
        <div className="flex items-center justify-between">
          <div className="flex items-baseline gap-2">
            <h3 className="text-sm font-semibold">Starting items</h3>
            <span className="text-xs text-muted-foreground">
              {items.length} granted
            </span>
          </div>
          <ItemSearchCombobox
            poolKey="items"
            existingIds={items}
            onAdd={onAddItem}
          />
        </div>
        <div className="space-y-1">
          {items.map((id, idx) => (
            <ItemRow
              key={`${id}-${idx}`}
              id={id}
              onRemove={() => onRemoveItem(idx)}
              removeAriaLabel={`Remove ${id}`}
            />
          ))}
        </div>
      </div>
      <div className="space-y-2">
        <div className="flex items-center justify-between">
          <div className="flex items-baseline gap-2">
            <h3 className="text-sm font-semibold">Starting skills</h3>
            <span className="text-xs text-muted-foreground">
              {skills.length} granted
            </span>
          </div>
          <div className="flex items-center gap-1">
            <Input
              aria-label="Skill id"
              inputMode="numeric"
              className="h-8 w-28"
              placeholder="Skill id…"
              value={skillInput}
              onChange={(e) => setSkillInput(e.target.value)}
            />
            <Button
              type="button"
              variant="outline"
              size="sm"
              disabled={skillId === undefined || skills.includes(skillId)}
              onClick={() => {
                if (skillId !== undefined) {
                  onAddSkill(skillId);
                  setSkillInput("");
                }
              }}
            >
              Add skill
            </Button>
          </div>
        </div>
        {skills.length === 0 ? (
          <p className="text-sm text-muted-foreground">
            This class starts with no granted skills.
          </p>
        ) : (
          <div className="space-y-1">
            {skills.map((id, idx) => (
              <SkillRow
                key={`${id}-${idx}`}
                id={id}
                onRemove={() => onRemoveSkill(idx)}
              />
            ))}
          </div>
        )}
      </div>
    </section>
  );
}

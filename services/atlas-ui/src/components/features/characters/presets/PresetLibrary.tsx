import { useMemo, useState } from "react";
import { EmptyState } from "@/components/common/EmptyState";
import { LibraryToolbar } from "./LibraryToolbar";
import { NewPresetCard } from "./NewPresetCard";
import { PresetCard } from "./PresetCard";
import type { WorkingPreset } from "./presetEditorState";

interface PresetLibraryProps {
  presets: WorkingPreset[];
  dirtyKeys: Set<string>;
  canApply: boolean;
  onOpen: (key: string) => void;
  onNew: () => void;
  onDuplicate: (key: string) => void;
  onApply: (key: string) => void;
}

function matchesQuery(preset: WorkingPreset, query: string): boolean {
  if (!query) return true;
  const q = query.toLowerCase();
  const { name, description, tags } = preset.attributes;
  const haystack = [name, description, ...tags].join(" ").toLowerCase();
  return haystack.includes(q);
}

export function PresetLibrary({
  presets,
  dirtyKeys,
  canApply,
  onOpen,
  onNew,
  onDuplicate,
  onApply,
}: PresetLibraryProps) {
  const [query, setQuery] = useState("");
  const [activeTag, setActiveTag] = useState<string | null>(null);

  const tags = useMemo(
    () => Array.from(new Set(presets.flatMap((p) => p.attributes.tags))).sort(),
    [presets],
  );

  const filtered = useMemo(
    () =>
      presets.filter(
        (p) =>
          matchesQuery(p, query) &&
          (activeTag === null || p.attributes.tags.includes(activeTag)),
      ),
    [presets, query, activeTag],
  );

  if (presets.length === 0) {
    return (
      <EmptyState
        title="No character presets"
        description="Create a preset to quickly spawn characters with a known loadout."
        action={{ label: "Add preset", onClick: onNew }}
      />
    );
  }

  return (
    <div className="flex flex-col gap-4">
      <LibraryToolbar
        query={query}
        onQuery={setQuery}
        tags={tags}
        activeTag={activeTag}
        onTag={setActiveTag}
        onNew={onNew}
      />
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
        {filtered.map((preset) => (
          <PresetCard
            key={preset.key}
            preset={preset}
            dirty={dirtyKeys.has(preset.key)}
            onOpen={() => onOpen(preset.key)}
            onDuplicate={() => onDuplicate(preset.key)}
            {...(canApply ? { onApply: () => onApply(preset.key) } : {})}
          />
        ))}
        <NewPresetCard onNew={onNew} />
      </div>
    </div>
  );
}

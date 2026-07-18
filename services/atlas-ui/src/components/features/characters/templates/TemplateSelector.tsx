import { Plus } from "lucide-react";
import type { CharacterTemplate } from "@/types/models/template";
import { cn } from "@/lib/utils";
import { templateLabels } from "./jobNames";

interface TemplateSelectorProps {
  templates: Pick<CharacterTemplate, "jobIndex" | "gender">[];
  selectedIndex: number;
  onSelect: (index: number) => void;
  onAdd: () => void;
}

/**
 * Segmented control (recessed track, flat text segments) — no thumbnails by
 * design: sprites always mean "rendered output", never navigation.
 */
export function TemplateSelector({
  templates,
  selectedIndex,
  onSelect,
  onAdd,
}: TemplateSelectorProps) {
  const labels = templateLabels(templates);
  return (
    <div
      role="tablist"
      aria-label="Character templates"
      className="flex flex-wrap items-center gap-1 rounded-lg bg-muted p-1"
    >
      {labels.map((label, index) => (
        <button
          key={index}
          type="button"
          role="tab"
          aria-selected={index === selectedIndex}
          onClick={() => onSelect(index)}
          className={cn(
            "rounded-md px-3 py-1.5 text-sm transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring",
            index === selectedIndex
              ? "bg-background font-medium shadow-sm"
              : "text-muted-foreground hover:text-foreground",
          )}
        >
          {label}
        </button>
      ))}
      <button
        type="button"
        onClick={onAdd}
        className="flex items-center gap-1 rounded-md px-3 py-1.5 text-sm text-muted-foreground transition-colors hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
      >
        <Plus className="size-4" /> New
      </button>
    </div>
  );
}

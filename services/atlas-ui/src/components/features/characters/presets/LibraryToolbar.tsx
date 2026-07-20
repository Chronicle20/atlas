import { Plus, Search } from "lucide-react";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";

interface LibraryToolbarProps {
  query: string;
  onQuery: (query: string) => void;
  tags: string[];
  activeTag: string | null;
  onTag: (tag: string | null) => void;
  onNew: () => void;
}

export function LibraryToolbar({ query, onQuery, tags, activeTag, onTag, onNew }: LibraryToolbarProps) {
  return (
    <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
      <div className="flex flex-1 flex-wrap items-center gap-3">
        <div className="relative w-full max-w-xs">
          <Search className="pointer-events-none absolute left-2.5 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            type="search"
            aria-label="Search presets"
            placeholder="Search presets..."
            value={query}
            onChange={(e) => onQuery(e.target.value)}
            className="pl-8"
          />
        </div>
        <div className="flex flex-wrap items-center gap-1.5">
          <button
            type="button"
            aria-label="All"
            aria-pressed={activeTag === null}
            onClick={() => onTag(null)}
            className={cn(
              "rounded-full border px-2.5 py-1 text-xs font-semibold transition",
              activeTag === null
                ? "border-primary bg-primary text-primary-foreground"
                : "border-input text-muted-foreground hover:border-primary hover:text-foreground",
            )}
          >
            All
          </button>
          {tags.map((tag) => (
            <button
              key={tag}
              type="button"
              aria-label={tag}
              aria-pressed={activeTag === tag}
              onClick={() => onTag(activeTag === tag ? null : tag)}
              className={cn(
                "rounded-full border px-2.5 py-1 text-xs font-semibold transition",
                activeTag === tag
                  ? "border-primary bg-primary text-primary-foreground"
                  : "border-input text-muted-foreground hover:border-primary hover:text-foreground",
              )}
            >
              {tag}
            </button>
          ))}
        </div>
      </div>
      <Button type="button" aria-label="New preset" onClick={onNew} className="gap-1.5">
        <Plus className="h-4 w-4" />
        New preset
      </Button>
    </div>
  );
}

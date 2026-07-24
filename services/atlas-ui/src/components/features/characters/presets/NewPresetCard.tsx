import { Plus } from "lucide-react";

interface NewPresetCardProps {
  onNew: () => void;
}

/**
 * Dashed-border tile appended after the preset grid. Distinct accessible
 * name from LibraryToolbar's "+ New preset" button (both are on-screen at
 * once) so assistive tech and role-based queries can disambiguate them.
 */
export function NewPresetCard({ onNew }: NewPresetCardProps) {
  return (
    <button
      type="button"
      aria-label="Add preset"
      onClick={onNew}
      className="flex min-h-[220px] flex-col items-center justify-center gap-2 rounded-xl border-2 border-dashed border-muted-foreground/30 text-muted-foreground transition hover:border-primary hover:text-primary focus-visible:outline-hidden focus-visible:ring-1 focus-visible:ring-ring"
    >
      <Plus className="h-6 w-6" />
      <span className="text-sm font-semibold">New preset</span>
    </button>
  );
}

// services/atlas-ui/src/components/features/accounts/EmptySlotTile.tsx
import { cn } from "@/lib/utils";
import { tileFrameClasses } from "./tile-frame";

interface EmptySlotTileProps {
  onClick: () => void;
  disabled?: boolean;
}

export function EmptySlotTile({ onClick, disabled }: EmptySlotTileProps) {
  return (
    <button
      type="button"
      onClick={onClick}
      disabled={disabled}
      aria-label="Add character to slot"
      className={cn(
        tileFrameClasses,
        "flex flex-col items-center justify-center gap-2 hover:bg-accent/50 focus-visible:ring-2 focus-visible:ring-ring disabled:opacity-50 disabled:cursor-not-allowed",
      )}
    >
      <img
        src="/default-character-avatar.svg"
        width={192}
        height={192}
        alt=""
        loading="lazy"
        className="opacity-70"
      />
      <span className="text-sm text-muted-foreground">+ Add character</span>
    </button>
  );
}

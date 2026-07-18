import { useState } from "react";
import { ImageOff, X } from "lucide-react";
import { cn } from "@/lib/utils";

// Crop of the 192×256 stand1 render at resize=2 down to the head region.
// Starting values from prototype.html; tune against real renders at the end
// (Task 17, design D6).
export const THUMB_SIZE = 76;
export const THUMB_OFFSET_X = -74;
export const THUMB_OFFSET_Y = -70;

interface AppearanceThumbProps {
  url: string;
  idLabel: string | number;
  ariaLabel: string;
  selected?: boolean;
  onSelect?: () => void;
  onRemove?: () => void;
  removeAriaLabel?: string;
  marked?: boolean;
}

export function AppearanceThumb({
  url,
  idLabel,
  ariaLabel,
  selected = false,
  onSelect,
  onRemove,
  removeAriaLabel,
  marked = false,
}: AppearanceThumbProps) {
  const [failed, setFailed] = useState(false);

  return (
    <div className="group relative">
      <button
        type="button"
        aria-label={ariaLabel}
        aria-pressed={selected}
        onClick={onSelect}
        disabled={marked}
        className={cn(
          "relative overflow-hidden rounded-md border bg-muted/40 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring",
          selected && "ring-2 ring-primary",
          marked && "cursor-not-allowed opacity-50",
        )}
        style={{ width: THUMB_SIZE, height: THUMB_SIZE }}
      >
        {failed ? (
          <span className="flex h-full w-full items-center justify-center text-muted-foreground">
            <ImageOff className="size-5" />
          </span>
        ) : (
          <img
            src={url}
            alt=""
            width={192}
            height={256}
            loading="lazy"
            onError={() => setFailed(true)}
            className="absolute max-w-none [image-rendering:pixelated]"
            style={{ left: THUMB_OFFSET_X, top: THUMB_OFFSET_Y }}
          />
        )}
        <span className="absolute inset-x-0 bottom-0 bg-background/80 text-center font-mono text-[10px] leading-4">
          {idLabel}
        </span>
      </button>
      {onRemove && (
        <button
          type="button"
          aria-label={removeAriaLabel ?? `Remove ${idLabel}`}
          onClick={onRemove}
          className="absolute -right-1.5 -top-1.5 hidden size-5 items-center justify-center rounded-full border bg-background text-muted-foreground shadow-sm hover:text-destructive group-focus-within:flex group-hover:flex"
        >
          <X className="size-3" />
        </button>
      )}
    </div>
  );
}

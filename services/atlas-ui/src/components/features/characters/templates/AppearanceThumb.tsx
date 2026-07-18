import { useState } from "react";
import { ImageOff, X } from "lucide-react";
import { cn } from "@/lib/utils";

// Head-region crop of the live stand1 render (design D6). The compositor uses a
// fixed canvas (feet at 119/128 = 0.93 of image height; see CharacterRenderer),
// so the head sits in the top ~40% and is horizontally centered — a fixed-pixel
// crop of an aspect-distorted box does NOT work across renders. Instead we scale
// the render by HEIGHT (width auto → no distortion), center it horizontally, and
// nudge it up so the face lands in the window. THUMB_ZOOM sets how large the head
// appears; THUMB_OFFSET_Y trims the transparent padding above the hair.
export const THUMB_SIZE = 76; // px window (square)
export const THUMB_ZOOM = 200; // px displayed render height
export const THUMB_OFFSET_Y = -6; // px vertical nudge (negative = shift render up)

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
            loading="lazy"
            onError={() => setFailed(true)}
            className="pointer-events-none absolute left-1/2 max-w-none -translate-x-1/2 [image-rendering:pixelated]"
            style={{ height: THUMB_ZOOM, top: THUMB_OFFSET_Y }}
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

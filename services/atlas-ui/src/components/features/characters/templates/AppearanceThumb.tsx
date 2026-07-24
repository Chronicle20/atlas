import { useState } from "react";
import { ImageOff, X } from "lucide-react";
import { cn } from "@/lib/utils";

// Head-region crop of the live stand1 render (design D6). The compositor emits a
// FIXED 192×256 canvas (resize=2 of a 96×128 frame): measured against real v84
// renders the body occupies y≈112..239 (feet at 119/128; top ~44% is transparent
// headroom) and the head sits at x≈88, y≈112..172 — stable across gender and
// equipment because the canvas is fixed. So a fixed-pixel crop of the 192×256
// image is correct; we position a 76px window over the head (center x≈88, y≈145).
export const THUMB_SIZE = 76; // px window (square)
export const RENDER_W = 192; // px fixed render width
export const RENDER_H = 256; // px fixed render height
export const THUMB_OFFSET_X = -50; // window shows img x 50..126 (head center ~88)
export const THUMB_OFFSET_Y = -106; // window shows img y 106..182 (head ~112..172)

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
            width={RENDER_W}
            height={RENDER_H}
            loading="lazy"
            onError={() => setFailed(true)}
            className="pointer-events-none absolute max-w-none [image-rendering:pixelated]"
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

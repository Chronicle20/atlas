import { Button } from "@/components/ui/button";
import {
  ChevronLeft,
  ChevronRight,
  ChevronsLeft,
  ChevronsRight,
} from "lucide-react";

export interface PagerProps {
  page: number;
  lastPage: number;
  total: number;
  pageSize: number;
  onPageChange: (page: number) => void;
}

const WINDOW_RADIUS = 2;

function buildWindow(page: number, lastPage: number): number[] {
  const start = Math.max(1, page - WINDOW_RADIUS);
  const end = Math.min(lastPage, page + WINDOW_RADIUS);
  const out: number[] = [];
  for (let i = start; i <= end; i++) out.push(i);
  return out;
}

export function Pager({ page, lastPage, total, pageSize: _pageSize, onPageChange }: PagerProps) {
  const onFirst = page <= 1;
  const onLast = page >= lastPage;
  const window = buildWindow(page, lastPage);

  const status = total === 0
    ? "No results"
    : `Page ${page} of ${lastPage} • ${total} results`;

  return (
    <div className="flex items-center justify-between gap-4 py-2">
      <div className="text-sm text-muted-foreground" aria-live="polite">
        {status}
      </div>
      <div className="flex items-center gap-1" role="navigation" aria-label="Pagination">
        <Button
          variant="outline"
          size="icon"
          aria-label="First page"
          disabled={onFirst}
          onClick={() => onPageChange(1)}
        >
          <ChevronsLeft className="h-4 w-4" />
        </Button>
        <Button
          variant="outline"
          size="icon"
          aria-label="Previous page"
          disabled={onFirst}
          onClick={() => onPageChange(page - 1)}
        >
          <ChevronLeft className="h-4 w-4" />
        </Button>
        {window.map((n) => (
          <Button
            key={n}
            variant={n === page ? "default" : "outline"}
            size="sm"
            aria-current={n === page ? "page" : undefined}
            aria-label={String(n)}
            onClick={() => onPageChange(n)}
          >
            {n}
          </Button>
        ))}
        <Button
          variant="outline"
          size="icon"
          aria-label="Next page"
          disabled={onLast}
          onClick={() => onPageChange(page + 1)}
        >
          <ChevronRight className="h-4 w-4" />
        </Button>
        <Button
          variant="outline"
          size="icon"
          aria-label="Last page"
          disabled={onLast}
          onClick={() => onPageChange(lastPage)}
        >
          <ChevronsRight className="h-4 w-4" />
        </Button>
      </div>
    </div>
  );
}

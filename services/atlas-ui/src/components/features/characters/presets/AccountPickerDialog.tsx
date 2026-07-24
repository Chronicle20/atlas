// services/atlas-ui/src/components/features/characters/presets/AccountPickerDialog.tsx
import { useEffect, useState } from "react";
import { Search } from "lucide-react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Skeleton } from "@/components/ui/skeleton";
import { ErrorDisplay } from "@/components/common/ErrorDisplay";
import { useAccountSearch } from "@/lib/hooks/api/useAccounts";
import type { Tenant } from "@/types/models/tenant";

interface AccountPickerDialogProps {
  tenant: Tenant;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onPick: (accountId: number) => void;
}

const DEBOUNCE_MS = 200;

export function AccountPickerDialog({
  tenant,
  open,
  onOpenChange,
  onPick,
}: AccountPickerDialogProps) {
  const [query, setQuery] = useState("");
  const [debounced, setDebounced] = useState("");

  useEffect(() => {
    const handle = setTimeout(() => setDebounced(query), DEBOUNCE_MS);
    return () => clearTimeout(handle);
  }, [query]);

  // Reset the search on every open — adjusted during render (rather than in
  // an effect) via React's "adjust state when a prop changes" pattern
  // (https://react.dev/learn/you-might-not-need-an-effect#adjusting-some-state-when-a-prop-changes),
  // so this doesn't trip react-hooks/set-state-in-effect.
  const [prevOpen, setPrevOpen] = useState(open);
  if (prevOpen !== open) {
    setPrevOpen(open);
    if (open) {
      setQuery("");
      setDebounced("");
    }
  }

  const { data, isLoading, isError } = useAccountSearch(tenant, debounced);

  const handlePick = (accountId: string) => {
    onPick(Number(accountId));
    onOpenChange(false);
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Select an account</DialogTitle>
        </DialogHeader>
        <div className="relative">
          <Search className="pointer-events-none absolute left-2.5 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            type="search"
            aria-label="Search accounts"
            placeholder="Search accounts..."
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            className="pl-8"
          />
        </div>
        <div className="max-h-72 space-y-1 overflow-y-auto">
          {isLoading ? (
            <div className="space-y-2 py-2">
              <Skeleton className="h-9 w-full" />
              <Skeleton className="h-9 w-full" />
            </div>
          ) : isError ? (
            <ErrorDisplay error="Failed to search accounts." />
          ) : data === undefined ? (
            <p className="py-6 text-center text-sm text-muted-foreground">
              Type to search accounts.
            </p>
          ) : data.length === 0 ? (
            <p className="py-6 text-center text-sm text-muted-foreground">
              No accounts match.
            </p>
          ) : (
            data.map((account) => (
              <button
                key={account.id}
                type="button"
                aria-label={`${account.attributes.name} (#${account.id})`}
                onClick={() => handlePick(account.id)}
                className="flex w-full items-center justify-between rounded-md border px-3 py-2 text-left text-sm transition hover:border-primary hover:bg-accent"
              >
                <span className="font-medium">{account.attributes.name}</span>
                <span className="text-xs text-muted-foreground">
                  #{account.id}
                </span>
              </button>
            ))
          )}
        </div>
      </DialogContent>
    </Dialog>
  );
}

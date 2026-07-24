import { useState } from "react";
import { ChevronsUpDown } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { cn } from "@/lib/utils";
import { PRESET_JOBS, jobLabel } from "./presetJobs";

interface JobComboboxProps {
  value: number;
  onChange: (jobId: number) => void;
}

/**
 * Single job picker: filter the curated job list by name, or type a numeric
 * id to use one the list doesn't cover (the backend is the validator of
 * record). Replaces the old Select + standalone "Advanced job id" pair.
 */
export function JobCombobox({ value, onChange }: JobComboboxProps) {
  const [open, setOpen] = useState(false);
  const [search, setSearch] = useState("");

  const term = search.trim().toLowerCase();
  const rows = term
    ? PRESET_JOBS.filter(
        (j) =>
          j.name.toLowerCase().includes(term) || String(j.id).startsWith(term),
      )
    : PRESET_JOBS;

  const manualId = /^\d+$/.test(search.trim())
    ? Number(search.trim())
    : undefined;

  const pick = (id: number) => {
    onChange(id);
    setOpen(false);
    setSearch("");
  };

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <Button
          type="button"
          variant="outline"
          role="combobox"
          aria-expanded={open}
          aria-label="Class"
          className="w-full justify-between font-normal"
        >
          {jobLabel(value)}
          <ChevronsUpDown className="size-4 opacity-50" />
        </Button>
      </PopoverTrigger>
      <PopoverContent className="w-64 p-2" align="start">
        <Input
          autoFocus
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          placeholder="Search by name or enter an id…"
        />
        <ul
          role="listbox"
          className="mt-2 max-h-64 space-y-0.5 overflow-y-auto"
        >
          {rows.map((j) => (
            <li
              key={j.id}
              role="option"
              aria-selected={j.id === value}
              tabIndex={0}
              onClick={() => pick(j.id)}
              onKeyDown={(e) => {
                if (e.key === "Enter" || e.key === " ") {
                  e.preventDefault();
                  pick(j.id);
                }
              }}
              className={cn(
                "flex cursor-pointer items-center gap-2 rounded px-2 py-1 hover:bg-accent focus-visible:bg-accent",
                j.id === value && "bg-accent/60",
              )}
            >
              <span className="flex-1 truncate text-sm">{j.name}</span>
              <span className="font-mono text-xs text-muted-foreground">
                {j.id}
              </span>
            </li>
          ))}
          {manualId !== undefined &&
            !PRESET_JOBS.some((j) => j.id === manualId) && (
              <li
                role="option"
                aria-selected={false}
                tabIndex={0}
                onClick={() => pick(manualId)}
                onKeyDown={(e) => {
                  if (e.key === "Enter" || e.key === " ") {
                    e.preventDefault();
                    pick(manualId);
                  }
                }}
                className="cursor-pointer rounded px-2 py-1 text-sm hover:bg-accent focus-visible:bg-accent"
              >
                Use id {manualId}
              </li>
            )}
          {rows.length === 0 && manualId === undefined && (
            <li className="px-2 py-1 text-sm text-muted-foreground">
              No matches.
            </li>
          )}
        </ul>
      </PopoverContent>
    </Popover>
  );
}

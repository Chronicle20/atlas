import { useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { Layers, Loader2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { useTenant } from "@/context/tenant-context";
import { jobsService } from "@/services/api/jobs.service";
import { jobSkillsKeys } from "@/lib/hooks/api/useJobSkills";
import { PRESET_JOBS } from "./presetJobs";

interface JobSkillsAddButtonProps {
  onAddMany: (skillIds: number[]) => void;
}

/**
 * Adds every skill in a chosen job family at once (e.g. all Hermit skills).
 * Pick a job by name — or type a numeric id for a job not in the curated
 * list — and its `/jobs/{id}/skills` set is granted (deduped, level 1) by the
 * reducer. Fetches through the React Query cache via `fetchQuery` so a job's
 * skill list is shared with the rest of the app.
 */
export function JobSkillsAddButton({ onAddMany }: JobSkillsAddButtonProps) {
  const { activeTenant } = useTenant();
  const queryClient = useQueryClient();
  const [open, setOpen] = useState(false);
  const [search, setSearch] = useState("");
  const [pending, setPending] = useState<number | null>(null);

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

  const pick = async (jobId: number) => {
    if (!activeTenant || pending !== null) return;
    setPending(jobId);
    try {
      const ids = await queryClient.fetchQuery({
        queryKey: jobSkillsKeys.detail(activeTenant.id, jobId),
        queryFn: () => jobsService.getSkillsByJobId(jobId),
        staleTime: 30 * 60 * 1000,
      });
      onAddMany(ids);
      setOpen(false);
      setSearch("");
    } finally {
      setPending(null);
    }
  };

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <Button type="button" variant="outline" size="sm">
          <Layers className="size-4" /> Job skills
        </Button>
      </PopoverTrigger>
      <PopoverContent className="w-72 p-2" align="end">
        <Input
          autoFocus
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          placeholder="Add all skills for a job…"
        />
        <ul
          role="listbox"
          className="mt-2 max-h-64 space-y-0.5 overflow-y-auto"
        >
          {rows.map((j) => (
            <li
              key={j.id}
              role="option"
              aria-selected={false}
              tabIndex={0}
              onClick={() => void pick(j.id)}
              onKeyDown={(e) => {
                if (e.key === "Enter" || e.key === " ") {
                  e.preventDefault();
                  void pick(j.id);
                }
              }}
              className="flex cursor-pointer items-center gap-2 rounded px-2 py-1 hover:bg-accent focus-visible:bg-accent"
            >
              <span className="flex-1 truncate text-sm">{j.name}</span>
              {pending === j.id ? (
                <Loader2 className="size-3.5 animate-spin text-muted-foreground" />
              ) : (
                <span className="font-mono text-xs text-muted-foreground">
                  {j.id}
                </span>
              )}
            </li>
          ))}
          {manualId !== undefined &&
            !PRESET_JOBS.some((j) => j.id === manualId) && (
              <li
                role="option"
                aria-selected={false}
                tabIndex={0}
                onClick={() => void pick(manualId)}
                onKeyDown={(e) => {
                  if (e.key === "Enter" || e.key === " ") {
                    e.preventDefault();
                    void pick(manualId);
                  }
                }}
                className="flex cursor-pointer items-center gap-2 rounded px-2 py-1 hover:bg-accent focus-visible:bg-accent"
              >
                <span className="flex-1 text-sm">
                  Add all skills for job {manualId}
                </span>
                {pending === manualId && (
                  <Loader2 className="size-3.5 animate-spin text-muted-foreground" />
                )}
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

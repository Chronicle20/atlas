import { useState } from "react";
import { useTenant } from "@/context/tenant-context";
import { useTenantConfiguration } from "@/lib/hooks/api/useTenants";
import { useRankings } from "@/lib/hooks/api/useRankings";
import { LeaderboardRow } from "@/components/features/rankings/LeaderboardRow";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Button } from "@/components/ui/button";

const JOB_CATEGORIES: { label: string; value: number | undefined }[] = [
  { label: "All jobs", value: undefined },
  { label: "Beginner", value: 0 },
  { label: "Warrior", value: 1 },
  { label: "Magician", value: 2 },
  { label: "Bowman", value: 3 },
  { label: "Thief", value: 4 },
  { label: "Pirate", value: 5 },
];

const PAGE_SIZE = 25;

export function RankingsPage() {
  const { activeTenant } = useTenant();
  const tenantConfigQuery = useTenantConfiguration(activeTenant?.id ?? "");
  const worlds = tenantConfigQuery.data?.attributes.worlds ?? [];

  const [worldId, setWorldId] = useState(0);
  const [jobCategory, setJobCategory] = useState<number | undefined>(undefined);
  const [page, setPage] = useState(0); // zero-based

  const view = jobCategory === undefined ? "overall" : "job";
  const filter = { jobCategory, page, pageSize: PAGE_SIZE };
  const query = useRankings(
    activeTenant?.id ?? "",
    worldId,
    filter,
    !!activeTenant,
  );

  const total = query.data?.total ?? 0;
  const lastPage = query.data?.lastPage ?? 1;

  return (
    <div className="space-y-4 p-4">
      <h1 className="text-2xl font-semibold">Rankings</h1>

      <div className="flex flex-wrap gap-4">
        <div className="space-y-1">
          <label className="text-sm font-medium">World</label>
          <Select
            value={String(worldId)}
            onValueChange={(v) => {
              setWorldId(parseInt(v, 10));
              setPage(0);
            }}
          >
            <SelectTrigger className="w-40">
              <SelectValue placeholder="Select a world" />
            </SelectTrigger>
            <SelectContent>
              {worlds.length > 0 ? (
                worlds.map((world, index) => (
                  <SelectItem key={index} value={String(index)}>
                    {world.name || `World ${index}`}
                  </SelectItem>
                ))
              ) : (
                <SelectItem value="0">World 0</SelectItem>
              )}
            </SelectContent>
          </Select>
        </div>

        <div className="space-y-1">
          <label className="text-sm font-medium">Job</label>
          <Select
            value={jobCategory === undefined ? "all" : String(jobCategory)}
            onValueChange={(v) => {
              setJobCategory(v === "all" ? undefined : parseInt(v, 10));
              setPage(0);
            }}
          >
            <SelectTrigger className="w-40">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {JOB_CATEGORIES.map((c) => (
                <SelectItem
                  key={c.label}
                  value={c.value === undefined ? "all" : String(c.value)}
                >
                  {c.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
      </div>

      {query.isError ? (
        <p className="text-destructive">Failed to load rankings.</p>
      ) : query.isLoading ? (
        <p className="text-sm text-muted-foreground">Loading…</p>
      ) : (
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b text-left text-muted-foreground">
              <th className="px-3 py-2">Rank</th>
              <th className="px-3 py-2">Character</th>
              <th className="px-3 py-2">Name</th>
              <th className="px-3 py-2">Level</th>
              <th className="px-3 py-2">Job</th>
              <th className="px-3 py-2">Move</th>
            </tr>
          </thead>
          <tbody>
            {(query.data?.entries ?? []).map((entry) => (
              <LeaderboardRow key={entry.id} entry={entry} view={view} />
            ))}
          </tbody>
        </table>
      )}

      <div className="flex items-center gap-3">
        <Button
          variant="outline"
          size="sm"
          disabled={page <= 0}
          onClick={() => setPage((p) => Math.max(0, p - 1))}
        >
          Previous
        </Button>
        <span className="text-sm text-muted-foreground">
          Page {page + 1} of {lastPage} ({total} ranked)
        </span>
        <Button
          variant="outline"
          size="sm"
          disabled={page + 1 >= lastPage}
          onClick={() => setPage((p) => p + 1)}
        >
          Next
        </Button>
      </div>
    </div>
  );
}

import { useTenant } from "@/context/tenant-context";
import { DataTableWrapper } from "@/components/common/DataTableWrapper";
import { getColumns, hiddenColumns } from "@/pages/quests-columns";
import { useMemo, useState } from "react";
import { useQuests, useQuestCategories, useInvalidateQuests } from "@/lib/hooks/api/useQuests";
import { Toaster } from "sonner";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Label } from "@/components/ui/label";

function QuestPageSkeleton() {
  return (
    <div className="flex flex-col flex-1 space-y-6 p-10 pb-16">
      <div className="items-center justify-between space-y-2">
        <Skeleton className="h-8 w-32" />
      </div>
      <div className="flex gap-4">
        <Skeleton className="h-10 w-48" />
      </div>
      <div className="space-y-2">
        <Skeleton className="h-10 w-full" />
        <Skeleton className="h-10 w-full" />
        <Skeleton className="h-10 w-full" />
        <Skeleton className="h-10 w-full" />
        <Skeleton className="h-10 w-full" />
      </div>
    </div>
  );
}

export function QuestsPage() {
  const { activeTenant } = useTenant();
  const questsQuery = useQuests(activeTenant);
  const categoriesQuery = useQuestCategories(activeTenant);
  const { invalidateAll } = useInvalidateQuests();

  const [selectedCategory, setSelectedCategory] = useState<string>("all");

  const quests = questsQuery.data ?? [];
  const categories = categoriesQuery.data ?? [];
  const loading = questsQuery.isLoading || categoriesQuery.isLoading;
  const error = questsQuery.error?.message ?? categoriesQuery.error?.message ?? null;

  const filteredQuests = useMemo(() => {
    if (selectedCategory === "all") return quests;
    return quests.filter(q => q.attributes.parent === selectedCategory);
  }, [quests, selectedCategory]);

  const columns = useMemo(() => getColumns(), []);

  if (loading) {
    return <QuestPageSkeleton />;
  }

  return (
    <div className="flex flex-col flex-1 space-y-6 p-10 pb-16">
      <div className="items-center justify-between space-y-2">
        <div>
          <h2 className="text-2xl font-bold tracking-tight">Quests</h2>
          <p className="text-muted-foreground">
            Browse and manage quest definitions ({quests.length} total)
          </p>
        </div>
      </div>

      <div className="flex gap-4 items-end">
        <div className="space-y-1">
          <Label htmlFor="category-filter">Category</Label>
          <Select value={selectedCategory} onValueChange={setSelectedCategory}>
            <SelectTrigger id="category-filter" className="w-48">
              <SelectValue placeholder="All Categories" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">All Categories</SelectItem>
              {categories.map((category) => (
                <SelectItem key={category} value={category}>
                  {category}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
      </div>

      <div className="mt-4">
        <DataTableWrapper
          columns={columns}
          data={filteredQuests}
          error={error}
          onRefresh={() => invalidateAll()}
          initialVisibilityState={hiddenColumns}
          emptyState={{
            title: "No quests found",
            description: selectedCategory !== "all"
              ? `No quests found in the "${selectedCategory}" category.`
              : "There are no quest definitions to display.",
          }}
        />
      </div>
      <Toaster richColors />
    </div>
  );
}

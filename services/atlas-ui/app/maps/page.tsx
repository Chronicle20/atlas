"use client"

import { useTenant } from "@/context/tenant-context";
import { Suspense, useCallback, useEffect, useRef, useState } from "react";
import { mapsService, type MapData } from "@/services/api/maps.service";
import { toast } from "sonner";
import { createErrorFromUnknown } from "@/types/api/errors";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { DataTableWrapper } from "@/components/common/DataTableWrapper";
import { columns } from "./columns";
import { Map, Search, Loader2 } from "lucide-react";
import { useSearchParams, useRouter, usePathname } from "next/navigation";

export default function MapsPage() {
  return (
    <Suspense>
      <MapsPageContent />
    </Suspense>
  );
}

function MapsPageContent() {
  const { activeTenant } = useTenant();
  const searchParams = useSearchParams();
  const router = useRouter();
  const pathname = usePathname();
  const initialQuery = searchParams.get("q") ?? "";
  const [searchQuery, setSearchQuery] = useState(initialQuery);
  const [maps, setMaps] = useState<MapData[]>([]);
  const [loading, setLoading] = useState(false);
  const [hasSearched, setHasSearched] = useState(false);
  const autoSearched = useRef(false);

  const handleSearch = useCallback(async () => {
    if (!activeTenant) {
      toast.error("No tenant selected");
      return;
    }

    if (!searchQuery.trim()) {
      toast.error("Please enter a search term");
      return;
    }

    setLoading(true);
    setHasSearched(true);
    router.replace(`${pathname}?q=${encodeURIComponent(searchQuery.trim())}`, { scroll: false });

    try {
      const data = await mapsService.searchMaps(searchQuery.trim(), activeTenant);
      setMaps(data);

      if (data.length === 0) {
        toast.info("No maps found matching your search");
      }
    } catch (err: unknown) {
      const errorInfo = createErrorFromUnknown(err, "Failed to search maps");
      toast.error(errorInfo.message);
    } finally {
      setLoading(false);
    }
  }, [activeTenant, searchQuery, router, pathname]);

  const handleClear = () => {
    setSearchQuery("");
    setMaps([]);
    setHasSearched(false);
    router.replace(pathname, { scroll: false });
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Enter") {
      handleSearch();
    }
  };

  useEffect(() => {
    if (activeTenant && initialQuery && !autoSearched.current) {
      autoSearched.current = true;
      handleSearch();
    }
  }, [activeTenant, initialQuery, handleSearch]);

  return (
    <div className="flex flex-col flex-1 min-h-0 space-y-6 p-10 pb-16">
      <div className="flex items-center gap-2">
        <Map className="h-6 w-6" />
        <h2 className="text-2xl font-bold tracking-tight">Maps</h2>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Search Maps</CardTitle>
          <CardDescription>
            Search for maps by ID, name, or street name. Results are limited to 50 entries.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="flex gap-4 items-end">
            <div className="flex-1">
              <Input
                placeholder="Enter map ID, name, or street name..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                onKeyDown={handleKeyDown}
              />
            </div>
            <Button onClick={handleSearch} disabled={loading}>
              {loading ? (
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              ) : (
                <Search className="mr-2 h-4 w-4" />
              )}
              Search
            </Button>
            <Button variant="outline" onClick={handleClear} disabled={loading}>
              Clear
            </Button>
          </div>
        </CardContent>
      </Card>

      {hasSearched && (
        <div className="flex-1 min-h-0">
          <DataTableWrapper
            columns={columns}
            data={maps}
            error={null}
            onRefresh={handleSearch}
            emptyState={{
              title: "No maps found",
              description: "Try a different search term.",
            }}
          />
        </div>
      )}
    </div>
  );
}

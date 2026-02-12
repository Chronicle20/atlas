"use client"

import { useTenant } from "@/context/tenant-context";
import { useCallback, useState } from "react";
import { itemsService } from "@/services/api/items.service";
import { type ItemSearchResult, getItemTypeBadgeVariant } from "@/types/models/item";
import { toast } from "sonner";
import { createErrorFromUnknown } from "@/types/api/errors";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Package, Search, Loader2 } from "lucide-react";
import Link from "next/link";

export default function ItemsPage() {
  const { activeTenant } = useTenant();
  const [searchQuery, setSearchQuery] = useState("");
  const [items, setItems] = useState<ItemSearchResult[]>([]);
  const [loading, setLoading] = useState(false);
  const [hasSearched, setHasSearched] = useState(false);

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

    try {
      const data = await itemsService.searchItems(searchQuery.trim(), activeTenant);
      setItems(data);

      if (data.length === 0) {
        toast.info("No items found matching your search");
      }
    } catch (err: unknown) {
      const errorInfo = createErrorFromUnknown(err, "Failed to search items");
      toast.error(errorInfo.message);
    } finally {
      setLoading(false);
    }
  }, [activeTenant, searchQuery]);

  const handleClear = () => {
    setSearchQuery("");
    setItems([]);
    setHasSearched(false);
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Enter") {
      handleSearch();
    }
  };

  return (
    <div className="flex flex-col flex-1 min-h-0 space-y-6 p-10 pb-16">
      <div className="flex items-center gap-2">
        <Package className="h-6 w-6" />
        <h2 className="text-2xl font-bold tracking-tight">Items</h2>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Search Items</CardTitle>
          <CardDescription>
            Search for items by ID or name. Results are limited to 50 entries.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="flex gap-4 items-end">
            <div className="flex-1">
              <Input
                placeholder="Enter item ID or name..."
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
        <Card className="flex-1 min-h-0 flex flex-col">
          <CardHeader className="shrink-0">
            <CardTitle>
              Results
              {items.length > 0 && (
                <span className="ml-2 text-muted-foreground font-normal">
                  ({items.length} {items.length === 1 ? "item" : "items"})
                </span>
              )}
            </CardTitle>
          </CardHeader>
          <CardContent className="flex-1 min-h-0 flex flex-col">
            {items.length === 0 ? (
              <div className="text-center py-8 text-muted-foreground">
                No items found matching your search criteria.
              </div>
            ) : (
              <div className="rounded-md border flex-1 min-h-0 overflow-auto">
                <Table>
                  <TableHeader className="sticky top-0 bg-background z-10">
                    <TableRow>
                      <TableHead>Item ID</TableHead>
                      <TableHead>Name</TableHead>
                      <TableHead>Type</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {items.map((item) => (
                      <TableRow key={item.id}>
                        <TableCell>
                          <Link href={`/items/${item.id}`} className="font-mono text-primary hover:underline">
                            {item.id}
                          </Link>
                        </TableCell>
                        <TableCell>
                          <Link href={`/items/${item.id}`} className="font-medium hover:underline">
                            {item.name}
                          </Link>
                        </TableCell>
                        <TableCell>
                          <Badge variant="secondary" className={getItemTypeBadgeVariant(item.type)}>
                            {item.type}
                          </Badge>
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </div>
            )}
          </CardContent>
        </Card>
      )}
    </div>
  );
}

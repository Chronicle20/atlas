"use client"

import { useTenant } from "@/context/tenant-context";
import { Suspense, useCallback, useEffect, useRef, useState } from "react";
import { npcsService } from "@/services/api";
import { type NpcSearchResult, type Commodity } from "@/types/models/npc";
import { tenantHeaders } from "@/lib/headers";
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
import { Users, Search, Loader2, ShoppingBag, MessageCircle } from "lucide-react";
import Link from "next/link";
import { useSearchParams, useRouter, usePathname } from "next/navigation";
import { toast } from "sonner";
import { Toaster } from "@/components/ui/sonner";
import { createErrorFromUnknown } from "@/types/api/errors";
import { NpcImage } from "@/components/features/npc/NpcImage";
import { getAssetIconUrl } from "@/lib/utils/asset-url";
import dynamic from "next/dynamic";

const NpcDialogs = dynamic(() => import("@/components/features/npc/NpcDialogs").then(mod => ({ default: mod.NpcDialogs })), {
  loading: () => null,
  ssr: false,
});

const AdvancedNpcActions = dynamic(() => import("@/components/features/npc/AdvancedNpcActions").then(mod => ({ default: mod.AdvancedNpcActions })), {
  loading: () => null,
  ssr: false,
});

export default function NpcsPage() {
  return (
    <Suspense>
      <NpcsPageContent />
    </Suspense>
  );
}

function NpcsPageContent() {
  const { activeTenant } = useTenant();
  const searchParams = useSearchParams();
  const router = useRouter();
  const pathname = usePathname();
  const initialQuery = searchParams.get("q") ?? "";
  const [searchQuery, setSearchQuery] = useState(initialQuery);
  const [results, setResults] = useState<NpcSearchResult[]>([]);
  const [npcStatus, setNpcStatus] = useState<Map<number, { hasShop: boolean; hasConversation: boolean }>>(new Map());
  const [loading, setLoading] = useState(false);
  const [statusLoading, setStatusLoading] = useState(false);
  const [hasSearched, setHasSearched] = useState(false);
  const autoSearched = useRef(false);

  // Shop management state
  const [isCreateShopDialogOpen, setIsCreateShopDialogOpen] = useState(false);
  const [isDeleteAllShopsDialogOpen, setIsDeleteAllShopsDialogOpen] = useState(false);
  const [isBulkUpdateShopDialogOpen, setIsBulkUpdateShopDialogOpen] = useState(false);
  const [selectedNpcId, setSelectedNpcId] = useState<number | null>(null);
  const [createShopJson, setCreateShopJson] = useState("");
  const [bulkUpdateShopJson, setBulkUpdateShopJson] = useState("");

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
      const data = await npcsService.searchNpcs(searchQuery.trim(), activeTenant);
      setResults(data);

      if (data.length === 0) {
        toast.info("No NPCs found matching your search");
      } else {
        // Lazy load shop/conversation status
        setStatusLoading(true);
        npcsService.getAllNPCs(activeTenant)
          .then((allNpcs) => {
            const statusMap = new Map<number, { hasShop: boolean; hasConversation: boolean }>();
            allNpcs.forEach(npc => {
              statusMap.set(npc.id, { hasShop: npc.hasShop, hasConversation: npc.hasConversation });
            });
            setNpcStatus(statusMap);
          })
          .catch((err) => {
            console.error("Failed to load NPC status:", err);
          })
          .finally(() => setStatusLoading(false));
      }
    } catch (err: unknown) {
      const errorInfo = createErrorFromUnknown(err, "Failed to search NPCs");
      toast.error(errorInfo.message);
    } finally {
      setLoading(false);
    }
  }, [activeTenant, searchQuery, router, pathname]);

  const handleClear = () => {
    setSearchQuery("");
    setResults([]);
    setNpcStatus(new Map());
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

  const handleCreateShop = async () => {
    if (!activeTenant) return;

    try {
      const jsonData = JSON.parse(createShopJson);

      if (!jsonData.data || !jsonData.data.attributes || !jsonData.data.attributes.npcId) {
        toast.error("Invalid JSON format. Missing npcId in data.attributes");
        return;
      }

      const npcId = parseInt(jsonData.data.attributes.npcId);
      if (isNaN(npcId)) {
        toast.error("Please provide a valid NPC ID in the JSON");
        return;
      }

      const rootUrl = process.env.NEXT_PUBLIC_ROOT_API_URL || window.location.origin;
      const response = await fetch(rootUrl + "/api/npcs/" + npcId + "/shop", {
        method: "POST",
        headers: tenantHeaders(activeTenant),
        body: createShopJson
      });

      if (!response.ok) {
        throw new Error("Failed to create shop.");
      }
      await response.json();
      toast.success("Shop created successfully");
      setIsCreateShopDialogOpen(false);
      setCreateShopJson("");
    } catch (err: unknown) {
      toast.error("Failed to create shop: " + (err instanceof Error ? err.message : String(err)));
    }
  };

  const handleDeleteAllShops = async () => {
    if (!activeTenant) return;

    try {
      await npcsService.deleteAllShops(activeTenant);
      toast.success("All shops deleted successfully");
      setIsDeleteAllShopsDialogOpen(false);
    } catch (err: unknown) {
      toast.error("Failed to delete all shops: " + (err instanceof Error ? err.message : String(err)));
    }
  };

  const handleBulkUpdateShop = async () => {
    if (!activeTenant || !selectedNpcId) return;

    try {
      const jsonData = JSON.parse(bulkUpdateShopJson);

      let commoditiesToUpdate: Commodity[] = [];
      if (jsonData.included && jsonData.included.length > 0) {
        commoditiesToUpdate = jsonData.included;
      } else if (jsonData.data.included && jsonData.data.included.length > 0) {
        commoditiesToUpdate = jsonData.data.included;
      }

      const rechargerValue = jsonData.data.attributes?.recharger;

      await npcsService.updateShop(selectedNpcId, commoditiesToUpdate, activeTenant, rechargerValue);
      setIsBulkUpdateShopDialogOpen(false);
      setBulkUpdateShopJson("");
      toast.success("Shop updated successfully");
    } catch (err: unknown) {
      toast.error("Failed to update shop: " + (err instanceof Error ? err.message : String(err)));
    }
  };

  return (
    <div className="flex flex-col flex-1 min-h-0 space-y-6 p-10 pb-16">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <Users className="h-6 w-6" />
          <h2 className="text-2xl font-bold tracking-tight">NPCs</h2>
        </div>
        <AdvancedNpcActions
          onCreateShop={() => setIsCreateShopDialogOpen(true)}
          onDeleteAllShops={() => setIsDeleteAllShopsDialogOpen(true)}
        />
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Search NPCs</CardTitle>
          <CardDescription>
            Search for NPCs by ID or name. Results are limited to 50 entries.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="flex gap-4 items-end">
            <div className="flex-1">
              <Input
                placeholder="Enter NPC ID or name..."
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
              {results.length > 0 && (
                <span className="ml-2 text-muted-foreground font-normal">
                  ({results.length} {results.length === 1 ? "NPC" : "NPCs"})
                </span>
              )}
            </CardTitle>
          </CardHeader>
          <CardContent className="flex-1 min-h-0 flex flex-col">
            {results.length === 0 ? (
              <div className="text-center py-8 text-muted-foreground">
                No NPCs found matching your search criteria.
              </div>
            ) : (
              <div className="rounded-md border flex-1 min-h-0 overflow-auto">
                <Table>
                  <TableHeader className="sticky top-0 bg-background z-10">
                    <TableRow>
                      <TableHead className="w-10">Icon</TableHead>
                      <TableHead>NPC ID</TableHead>
                      <TableHead>Name</TableHead>
                      <TableHead className="w-20">Shop</TableHead>
                      <TableHead className="w-28">Conversation</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {results.map((npc) => {
                      const status = npcStatus.get(npc.id);
                      const iconUrl = activeTenant ? getAssetIconUrl(
                        activeTenant.id,
                        activeTenant.attributes.region,
                        activeTenant.attributes.majorVersion,
                        activeTenant.attributes.minorVersion,
                        'npc',
                        npc.id,
                      ) : undefined;
                      return (
                        <TableRow key={npc.id}>
                          <TableCell>
                            <NpcImage
                              npcId={npc.id}
                              name={npc.name}
                              iconUrl={iconUrl}
                              size={32}
                              lazy={true}
                              showRetryButton={false}
                              maxRetries={1}
                            />
                          </TableCell>
                          <TableCell>
                            <Link href={`/npcs/${npc.id}`} className="font-mono text-primary hover:underline">
                              {npc.id}
                            </Link>
                          </TableCell>
                          <TableCell>
                            <Link href={`/npcs/${npc.id}`} className="font-medium hover:underline">
                              {npc.name}
                            </Link>
                          </TableCell>
                          <TableCell>
                            {statusLoading ? (
                              <Loader2 className="h-3 w-3 animate-spin text-muted-foreground" />
                            ) : status?.hasShop ? (
                              <Link href={`/npcs/${npc.id}/shop`}>
                                <Badge variant="default" className="cursor-pointer">
                                  <ShoppingBag className="h-3 w-3 mr-1" />
                                  Shop
                                </Badge>
                              </Link>
                            ) : status ? (
                              <Badge variant="outline" className="text-muted-foreground">None</Badge>
                            ) : null}
                          </TableCell>
                          <TableCell>
                            {statusLoading ? (
                              <Loader2 className="h-3 w-3 animate-spin text-muted-foreground" />
                            ) : status?.hasConversation ? (
                              <Link href={`/npcs/${npc.id}/conversations`}>
                                <Badge variant="default" className="cursor-pointer">
                                  <MessageCircle className="h-3 w-3 mr-1" />
                                  Chat
                                </Badge>
                              </Link>
                            ) : status ? (
                              <Badge variant="outline" className="text-muted-foreground">None</Badge>
                            ) : null}
                          </TableCell>
                        </TableRow>
                      );
                    })}
                  </TableBody>
                </Table>
              </div>
            )}
          </CardContent>
        </Card>
      )}

      <NpcDialogs
        isCreateShopDialogOpen={isCreateShopDialogOpen}
        setIsCreateShopDialogOpen={setIsCreateShopDialogOpen}
        isDeleteAllShopsDialogOpen={isDeleteAllShopsDialogOpen}
        setIsDeleteAllShopsDialogOpen={setIsDeleteAllShopsDialogOpen}
        isBulkUpdateShopDialogOpen={isBulkUpdateShopDialogOpen}
        setIsBulkUpdateShopDialogOpen={setIsBulkUpdateShopDialogOpen}
        createShopJson={createShopJson}
        setCreateShopJson={setCreateShopJson}
        bulkUpdateShopJson={bulkUpdateShopJson}
        setBulkUpdateShopJson={setBulkUpdateShopJson}
        handleCreateShop={handleCreateShop}
        handleDeleteAllShops={handleDeleteAllShops}
        handleBulkUpdateShop={handleBulkUpdateShop}
      />

      <Toaster richColors />
    </div>
  );
}

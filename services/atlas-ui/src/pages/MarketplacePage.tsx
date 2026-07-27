import { useState } from "react";
import { useTenant } from "@/context/tenant-context";
import { useTenantConfiguration } from "@/lib/hooks/api/useTenants";
import { useMtsListings } from "@/lib/hooks/api/useMtsListings";
import type {
  MtsListing,
  MtsListingAttributes,
} from "@/services/api/mts-listings.service";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Store, Search, Loader2, Tag } from "lucide-react";
import { SealIcon } from "@/components/seal-icon";
import { Link } from "react-router-dom";
import { ItemNameCell } from "@/components/item-name-cell";
import { Pager } from "@/components/common/Pager";
import { FLAG_LOCK } from "@/lib/utils/asset-flags";
import type { Tenant } from "@/types/models/tenant";

const LISTINGS_PAGE_SIZE = 16;
const SALE_TYPE_ANY = "any";

function formatEndsAt(endsAt?: string): string {
  if (!endsAt) return "—";
  const d = new Date(endsAt);
  return Number.isNaN(d.getTime()) ? "—" : d.toLocaleString();
}

export function MarketplacePage() {
  const { activeTenant } = useTenant();
  const tenantConfigQuery = useTenantConfiguration(activeTenant?.id ?? "");
  const worlds = tenantConfigQuery.data?.attributes.worlds ?? [];

  const [worldId, setWorldId] = useState(0);
  const [page, setPage] = useState(1);

  // Pending (form) filter state vs the applied filter that drives the query.
  const [categoryInput, setCategoryInput] = useState("");
  const [subCategoryInput, setSubCategoryInput] = useState("");
  const [saleTypeInput, setSaleTypeInput] = useState(SALE_TYPE_ANY);
  const [sellerNameInput, setSellerNameInput] = useState("");
  const [itemIdInput, setItemIdInput] = useState("");

  const [applied, setApplied] = useState<{
    category: string;
    subCategory: string;
    saleType: string;
    sellerName: string;
    itemId: number;
  }>({
    category: "",
    subCategory: "",
    saleType: "",
    sellerName: "",
    itemId: 0,
  });

  const listingsQuery = useMtsListings(
    activeTenant?.id ?? "",
    worldId,
    {
      category: applied.category || undefined,
      subCategory: applied.subCategory || undefined,
      saleType: applied.saleType || undefined,
      sellerName: applied.sellerName || undefined,
      itemId: applied.itemId || undefined,
      // The endpoint is zero-based (page=0 is the first page); the UI page state
      // is 1-based for display, so convert on the wire.
      page: page - 1,
      pageSize: LISTINGS_PAGE_SIZE,
    },
    !!activeTenant,
  );

  const listings = listingsQuery.data?.listings ?? [];
  const loading = listingsQuery.isFetching;
  // Total and last page come from the response meta — authoritative, not
  // inferred from the returned length.
  const total = listingsQuery.data?.total ?? 0;
  const lastPage = listingsQuery.data?.lastPage ?? 1;

  const applyFilters = () => {
    if (!activeTenant) {
      toast.error("No tenant selected");
      return;
    }
    const itemId = parseInt(itemIdInput, 10);
    setApplied({
      category: categoryInput.trim(),
      subCategory: subCategoryInput.trim(),
      saleType: saleTypeInput === SALE_TYPE_ANY ? "" : saleTypeInput,
      sellerName: sellerNameInput.trim(),
      itemId: !Number.isNaN(itemId) && itemId > 0 ? itemId : 0,
    });
    setPage(1);
  };

  const clearFilters = () => {
    setCategoryInput("");
    setSubCategoryInput("");
    setSaleTypeInput(SALE_TYPE_ANY);
    setSellerNameInput("");
    setItemIdInput("");
    setApplied({
      category: "",
      subCategory: "",
      saleType: "",
      sellerName: "",
      itemId: 0,
    });
    setPage(1);
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Enter") applyFilters();
  };

  return (
    <div className="flex flex-col flex-1 min-h-0 space-y-6 p-10 pb-16">
      <div className="flex items-center gap-2">
        <Store className="h-6 w-6" />
        <h2 className="text-2xl font-bold tracking-tight">Marketplace</h2>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Browse Listings</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <div>
              <label className="text-sm font-medium">World</label>
              <Select
                value={String(worldId)}
                onValueChange={(v) => {
                  setWorldId(parseInt(v, 10));
                  setPage(1);
                }}
              >
                <SelectTrigger>
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
            <div>
              <label className="text-sm font-medium">Sale Type</label>
              <Select value={saleTypeInput} onValueChange={setSaleTypeInput}>
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value={SALE_TYPE_ANY}>Any</SelectItem>
                  <SelectItem value="fixed">Buy Now</SelectItem>
                  <SelectItem value="auction">Auction</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div>
              <label className="text-sm font-medium">Item ID</label>
              <Input
                placeholder="e.g. 1302000"
                value={itemIdInput}
                onChange={(e) => setItemIdInput(e.target.value)}
                onKeyDown={handleKeyDown}
              />
            </div>
            <div>
              <label className="text-sm font-medium">Category</label>
              <Input
                placeholder="Category"
                value={categoryInput}
                onChange={(e) => setCategoryInput(e.target.value)}
                onKeyDown={handleKeyDown}
              />
            </div>
            <div>
              <label className="text-sm font-medium">Sub Category</label>
              <Input
                placeholder="Sub category"
                value={subCategoryInput}
                onChange={(e) => setSubCategoryInput(e.target.value)}
                onKeyDown={handleKeyDown}
              />
            </div>
            <div>
              <label className="text-sm font-medium">Seller Name</label>
              <Input
                placeholder="Seller name"
                value={sellerNameInput}
                onChange={(e) => setSellerNameInput(e.target.value)}
                onKeyDown={handleKeyDown}
              />
            </div>
          </div>
          <div className="flex gap-2 justify-end">
            <Button variant="outline" onClick={clearFilters} disabled={loading}>
              Clear
            </Button>
            <Button onClick={applyFilters} disabled={loading}>
              {loading ? (
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              ) : (
                <Search className="mr-2 h-4 w-4" />
              )}
              Search
            </Button>
          </div>
        </CardContent>
      </Card>

      <Card className="flex-1 min-h-0 flex flex-col">
        <CardHeader className="shrink-0">
          <CardTitle>
            Listings
            {total > 0 && (
              <span className="ml-2 text-muted-foreground font-normal">
                ({total} total, {listings.length} on this page)
              </span>
            )}
          </CardTitle>
        </CardHeader>
        <CardContent className="flex-1 min-h-0 flex flex-col">
          {!activeTenant ? (
            <div className="text-center py-8 text-muted-foreground">
              Select a tenant to browse listings.
            </div>
          ) : listingsQuery.error ? (
            <div className="text-center py-8 text-destructive">
              {listingsQuery.error.message}
            </div>
          ) : loading ? (
            <div className="text-center py-8 text-muted-foreground">
              Loading listings...
            </div>
          ) : listings.length === 0 ? (
            <div className="text-center py-8 text-muted-foreground">
              No listings found matching your search criteria.
            </div>
          ) : (
            <div className="rounded-md border flex-1 min-h-0 overflow-auto">
              <Table>
                <TableHeader className="sticky top-0 bg-background z-10">
                  <TableRow>
                    <TableHead>Item</TableHead>
                    <TableHead>Seller</TableHead>
                    <TableHead>Sale Type</TableHead>
                    <TableHead>State</TableHead>
                    <TableHead>Qty</TableHead>
                    <TableHead>Category</TableHead>
                    <TableHead>List Value</TableHead>
                    <TableHead>Buy Now</TableHead>
                    <TableHead>Current Bid</TableHead>
                    <TableHead>Ends At</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {listings.map((listing) => (
                    <ListingRow key={listing.id} listing={listing} />
                  ))}
                </TableBody>
              </Table>
            </div>
          )}
          {total > 0 && (
            <Pager
              page={page}
              lastPage={lastPage}
              total={total}
              pageSize={LISTINGS_PAGE_SIZE}
              onPageChange={setPage}
            />
          )}
        </CardContent>
      </Card>
    </div>
  );
}

/**
 * Item cell for a Marketplace listing row: the item name/icon (linked to the
 * item detail page) plus inline sealing-lock and item-tag-owner indicators,
 * mirroring the inventory badges (InventoryCard/EquipmentCell) but sized for
 * a table row. `flags`/`owner` come straight off the listing's `MtsListingAttributes`
 * rather than the `Asset` shape the `isSealed`/`isTagged` helpers expect, so the
 * checks are inlined here.
 */
export function ListingItemCell({
  attributes,
  tenant,
}: {
  attributes: MtsListingAttributes;
  tenant: Tenant | null;
}) {
  const tagged = (attributes.owner ?? "").trim() !== "";
  const sealed = (attributes.flags & FLAG_LOCK) !== 0;
  return (
    <div className="flex items-center gap-1.5">
      <Link to={`/items/${attributes.templateId}`}>
        <ItemNameCell itemId={String(attributes.templateId)} tenant={tenant} />
      </Link>
      {tagged && (
        <span className="inline-flex items-center gap-0.5 text-xs text-muted-foreground">
          <Tag
            data-testid="tag-icon"
            className="h-3 w-3 text-amber-500"
            aria-label="Named item"
          />
          {attributes.owner}
        </span>
      )}
      {sealed && (
        <SealIcon tenant={tenant} className="h-3 w-3 text-amber-500" />
      )}
    </div>
  );
}

function ListingRow({ listing }: { listing: MtsListing }) {
  const { activeTenant } = useTenant();
  const a = listing.attributes;
  return (
    <TableRow>
      <TableCell>
        <ListingItemCell attributes={a} tenant={activeTenant} />
      </TableCell>
      <TableCell>{a.sellerName}</TableCell>
      <TableCell>
        <Badge variant="secondary">{a.saleType}</Badge>
      </TableCell>
      <TableCell>
        <Badge variant="outline">{a.state}</Badge>
      </TableCell>
      <TableCell>{a.quantity}</TableCell>
      <TableCell>
        <span className="text-muted-foreground">
          {[a.category, a.subCategory].filter(Boolean).join(" / ") || "—"}
        </span>
      </TableCell>
      <TableCell>
        <span className="font-mono">{a.listValue.toLocaleString()}</span>
      </TableCell>
      <TableCell>
        <span className="font-mono">
          {a.buyNowPrice !== undefined ? a.buyNowPrice.toLocaleString() : "—"}
        </span>
      </TableCell>
      <TableCell>
        <span className="font-mono">
          {a.currentBid > 0 ? a.currentBid.toLocaleString() : "—"}
        </span>
      </TableCell>
      <TableCell>{formatEndsAt(a.endsAt)}</TableCell>
    </TableRow>
  );
}

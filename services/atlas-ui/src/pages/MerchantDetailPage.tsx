
import { useTenant } from "@/context/tenant-context";
import { useQuery } from "@tanstack/react-query";
import { useParams } from "react-router-dom";
import { merchantsService } from "@/services/api/merchants.service";
import type { MerchantListing } from "@/types/models/merchant";
import { useTenantConfiguration } from "@/lib/hooks/api/useTenants";
import { getShopTypeName, getShopTypeBadgeVariant, getShopStateName, getShopStateBadgeVariant } from "@/types/models/merchant";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { PageLoader } from "@/components/common/PageLoader";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Store } from "lucide-react";
import { Link } from "react-router-dom";
import { getAssetIconUrl } from "@/lib/utils/asset-url";
import { MapCell } from "@/components/map-cell";
import { ItemNameCell } from "@/components/item-name-cell";

export function MerchantDetailPage() {
  const { activeTenant } = useTenant();
  const params = useParams();
  const shopId = params.id as string;

  const shopQuery = useQuery({
    queryKey: ["merchants", "shop", activeTenant?.id ?? "no-tenant", shopId],
    queryFn: () => merchantsService.getShopById(shopId),
    enabled: !!activeTenant && !!shopId,
    staleTime: 60 * 1000,
  });

  const listingsQuery = useQuery({
    queryKey: ["merchants", "listings", activeTenant?.id ?? "no-tenant", shopId],
    queryFn: () => merchantsService.getShopListings(shopId),
    enabled: !!activeTenant && !!shopId,
    staleTime: 60 * 1000,
  });

  const tenantConfigQuery = useTenantConfiguration(activeTenant?.id ?? "");

  const shop = shopQuery.data ?? null;
  const listings: MerchantListing[] = listingsQuery.data ?? [];
  const tenantConfig = tenantConfigQuery.data ?? null;
  const loading = shopQuery.isLoading || listingsQuery.isLoading || tenantConfigQuery.isLoading;
  const error = shopQuery.error?.message ?? listingsQuery.error?.message ?? null;

  if (loading) {
    return <PageLoader />;
  }

  if (error || !shop) {
    return (
      <div className="flex flex-col flex-1 min-h-0 space-y-6 p-10 pb-16">
        <div className="text-center py-8 text-muted-foreground">{error || "Shop not found"}</div>
      </div>
    );
  }

  const a = shop.attributes;

  return (
    <div className="flex flex-col flex-1 min-h-0 space-y-6 p-10 pb-16">
      <div className="flex items-center gap-2">
        <Store className="h-6 w-6" />
        <div>
          <div className="flex items-center gap-2">
            <h2 className="text-2xl font-bold tracking-tight">{a.title || "Untitled Shop"}</h2>
            <Badge variant="secondary" className={getShopTypeBadgeVariant(a.shopType)}>
              {getShopTypeName(a.shopType)}
            </Badge>
            <Badge variant="secondary" className={getShopStateBadgeVariant(a.state)}>
              {getShopStateName(a.state)}
            </Badge>
          </div>
          <p className="text-sm text-muted-foreground">
            <Link to="/merchants" className="hover:underline">Merchants</Link>
            {" > "}
            <span>{a.title || shopId}</span>
          </p>
        </div>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Shop Information</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
            <InfoField label="Shop ID" value={shopId} mono />
            <InfoField label="Channel" value={`${tenantConfig?.attributes.worlds[a.worldId]?.name || `World ${a.worldId}`} Ch. ${a.channelId + 1}`} />
            <div className="space-y-1">
              <p className="text-sm text-muted-foreground">Map</p>
              <MapCell mapId={String(a.mapId)} tenant={activeTenant} />
            </div>
            <div className="space-y-1">
              <p className="text-sm text-muted-foreground">Owner</p>
              <Link to={`/characters/${a.characterId}`} className="text-sm font-medium text-primary hover:underline">
                {a.characterId}
              </Link>
            </div>
            <InfoField label="Position" value={`(${a.x}, ${a.y})`} />
            {a.shopType === 2 && (
              <InfoField label="Meso Balance" value={a.mesoBalance.toLocaleString()} />
            )}
            <InfoField label="Listings" value={a.listingCount} />
          </div>
        </CardContent>
      </Card>

      {a.visitors && a.visitors.length > 0 && (
        <Card>
          <CardHeader>
            <CardTitle>Visitors ({a.visitors.length})</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="flex flex-wrap gap-2">
              {a.visitors.map((characterId) => (
                <Link key={characterId} to={`/characters/${characterId}`}>
                  <Badge variant="secondary" className="cursor-pointer hover:bg-accent">
                    {characterId}
                  </Badge>
                </Link>
              ))}
            </div>
          </CardContent>
        </Card>
      )}

      <Card className="flex-1 min-h-0 flex flex-col">
        <CardHeader className="shrink-0">
          <CardTitle>Listings ({listings.length})</CardTitle>
        </CardHeader>
        <CardContent className="flex-1 min-h-0 flex flex-col">
          {listings.length === 0 ? (
            <div className="text-center py-8 text-muted-foreground">
              No listings in this shop.
            </div>
          ) : (
            <div className="rounded-md border flex-1 min-h-0 overflow-auto">
              <Table>
                <TableHeader className="sticky top-0 bg-background z-10">
                  <TableRow>
                    <TableHead className="w-10">Icon</TableHead>
                    <TableHead>Item</TableHead>
                    <TableHead>Template ID</TableHead>
                    <TableHead>Quantity</TableHead>
                    <TableHead>Bundle Size</TableHead>
                    <TableHead>Bundles</TableHead>
                    <TableHead>Price / Bundle</TableHead>
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
        </CardContent>
      </Card>
    </div>
  );
}

function ListingRow({ listing }: { listing: MerchantListing }) {
  const { activeTenant } = useTenant();
  const a = listing.attributes;

  const iconUrl = activeTenant ? getAssetIconUrl(
    activeTenant.id,
    activeTenant.attributes.region,
    activeTenant.attributes.majorVersion,
    activeTenant.attributes.minorVersion,
    'item',
    a.itemId,
  ) : '';

  return (
    <TableRow>
      <TableCell>
        {iconUrl ? (
          <img
            src={iconUrl}
            alt={String(a.itemId)}
            width={32}
            height={32}
            className="object-contain"
          />
        ) : (
          <Store className="h-8 w-8 text-muted-foreground" />
        )}
      </TableCell>
      <TableCell>
        <Link to={`/items/${a.itemId}`}>
          <ItemNameCell itemId={String(a.itemId)} tenant={activeTenant} />
        </Link>
      </TableCell>
      <TableCell>
        <span className="font-mono text-muted-foreground">{a.itemId}</span>
      </TableCell>
      <TableCell>{a.quantity}</TableCell>
      <TableCell>{a.bundleSize}</TableCell>
      <TableCell>{a.bundlesRemaining}</TableCell>
      <TableCell>
        <span className="font-mono">{a.pricePerBundle.toLocaleString()}</span>
      </TableCell>
    </TableRow>
  );
}

function InfoField({ label, value, mono }: { label: string; value: string | number; mono?: boolean }) {
  const displayValue = String(value);
  return (
    <div className="space-y-1">
      <p className="text-sm text-muted-foreground">{label}</p>
      <p className={`text-sm font-medium ${mono ? "font-mono" : ""}`}>{displayValue}</p>
    </div>
  );
}

import { useTenant } from "@/context/tenant-context";
import { Suspense, useEffect, useMemo, useState } from "react";
import { keepPreviousData, useQuery } from "@tanstack/react-query";
import { itemsService, type ItemSearchFilters, type ItemSearchPage } from "@/services/api/items.service";
import {
  getCompartmentBadgeVariant,
} from "@/types/models/item";
import {
  COMPARTMENT_OPTIONS,
  COMPARTMENT_LABELS,
  COMPARTMENT_TAXONOMY,
  CLASS_OPTIONS,
  type ClassOption,
  type Compartment,
  parseClassFilter,
  serializeClassFilter,
  subcategoryLabel,
} from "@/lib/items/taxonomy";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Package, Loader2 } from "lucide-react";
import { Link, useSearchParams } from "react-router-dom";
import { getAssetIconUrl } from "@/lib/utils/asset-url";
import { useDebounce } from "@/lib/utils/debounce";
import { Pager } from "@/components/common/Pager";

const MIN_QUERY_LENGTH = 2;
const DEBOUNCE_MS = 250;
const ANY_VALUE = "__any__";
const PAGE_SIZE = 50;

type FilterCompartment = Exclude<Compartment, "unknown">;

export function ItemsPage() {
  return (
    <Suspense>
      <ItemsPageContent />
    </Suspense>
  );
}

function ItemsPageContent() {
  const { activeTenant } = useTenant();
  const [searchParams, setSearchParams] = useSearchParams();

  const urlQ = searchParams.get("q") ?? "";
  const urlComp = (searchParams.get("comp") ?? "") as FilterCompartment | "";
  const urlSub = searchParams.get("sub") ?? "";
  const urlClassRaw = searchParams.get("class");
  const urlPageRaw = searchParams.get("page");
  const pageNumber = Math.max(1, Number.parseInt(urlPageRaw ?? "1", 10) || 1);

  const [searchInput, setSearchInput] = useState(urlQ);
  const debounced = useDebounce(searchInput.trim(), DEBOUNCE_MS);

  const compartment: FilterCompartment | "" = urlComp && (COMPARTMENT_OPTIONS as string[]).includes(urlComp) ? urlComp : "";
  const subcategory = urlSub;
  const { selected: classSelected, allClasses } = parseClassFilter(urlClassRaw);

  const writeUrlReplace = (next: { q?: string; comp?: FilterCompartment | ""; sub?: string; classFilter?: string }) => {
    const out = new URLSearchParams();
    if (next.q && next.q.length > 0) out.set("q", next.q);
    if (next.comp) out.set("comp", next.comp);
    if (next.sub) out.set("sub", next.sub);
    if (next.classFilter) out.set("class", next.classFilter);
    // page is intentionally dropped — resets to 1.
    setSearchParams(out, { replace: true });
  };

  const writeUrlPush = (nextPage: number) => {
    const out = new URLSearchParams();
    if (urlQ.length > 0) out.set("q", urlQ);
    if (compartment) out.set("comp", compartment);
    if (subcategory) out.set("sub", subcategory);
    const classStr = serializeClassFilter(classSelected, allClasses);
    if (classStr) out.set("class", classStr);
    if (nextPage > 1) out.set("page", String(nextPage));
    setSearchParams(out, { replace: false });
  };

  // Search input → URL.
  useEffect(() => {
    if (debounced.length >= MIN_QUERY_LENGTH) {
      if (debounced !== urlQ) {
        writeUrlReplace({ q: debounced, comp: compartment, sub: subcategory, classFilter: serializeClassFilter(classSelected, allClasses) });
      }
    } else if (urlQ !== "" && debounced.length === 0) {
      writeUrlReplace({ q: "", comp: compartment, sub: subcategory, classFilter: serializeClassFilter(classSelected, allClasses) });
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [debounced]);

  const onCompartmentChange = (raw: string) => {
    const next = raw === ANY_VALUE ? "" : (raw as FilterCompartment);
    writeUrlReplace({ q: urlQ, comp: next, sub: "", classFilter: next === "equipment" ? serializeClassFilter(classSelected, allClasses) : "" });
  };

  const onSubcategoryChange = (raw: string) => {
    const next = raw === ANY_VALUE ? "" : raw;
    writeUrlReplace({ q: urlQ, comp: compartment, sub: next, classFilter: serializeClassFilter(classSelected, allClasses) });
  };

  const onToggleClass = (klass: ClassOption) => {
    const next = new Set(classSelected);
    if (next.has(klass)) next.delete(klass);
    else next.add(klass);
    writeUrlReplace({ q: urlQ, comp: compartment, sub: subcategory, classFilter: serializeClassFilter(next, false) });
  };

  const onToggleAllClasses = () => {
    const next = !allClasses;
    writeUrlReplace({ q: urlQ, comp: compartment, sub: subcategory, classFilter: next ? "any" : "" });
  };

  const filters: ItemSearchFilters = useMemo(() => {
    const out: ItemSearchFilters = { pageNumber, pageSize: PAGE_SIZE };
    if (urlQ.length >= MIN_QUERY_LENGTH) out.q = urlQ;
    if (compartment) out.compartment = compartment;
    if (subcategory) out.subcategory = subcategory;
    if (allClasses) out.classes = ["any"];
    else if (classSelected.size > 0) out.classes = Array.from(classSelected);
    return out;
  }, [urlQ, compartment, subcategory, allClasses, classSelected, pageNumber]);

  const queryEnabled = !!activeTenant && (
    urlQ.length === 0 ||
    urlQ.length >= MIN_QUERY_LENGTH ||
    !!compartment || !!subcategory || allClasses || classSelected.size > 0
  );

  const itemsQuery = useQuery<ItemSearchPage, Error>({
    queryKey: [
      "items", "search",
      activeTenant?.id ?? "no-tenant",
      urlQ,
      compartment,
      subcategory,
      allClasses ? "any" : Array.from(classSelected).sort().join(","),
      pageNumber,
      PAGE_SIZE,
    ],
    queryFn: () => itemsService.searchItems(filters),
    enabled: queryEnabled,
    staleTime: 30 * 1000,
    placeholderData: keepPreviousData,
  });

  const pageData = itemsQuery.data ?? { items: [], total: 0, pageNumber, pageSize: PAGE_SIZE, lastPage: 1 };
  const items = pageData.items;
  const fetching = itemsQuery.isFetching;
  const showResults = queryEnabled;

  const firstRow = pageData.total === 0 ? 0 : (pageData.pageNumber - 1) * pageData.pageSize + 1;
  const lastRow = Math.min(pageData.pageNumber * pageData.pageSize, pageData.total);
  const statusCopy = pageData.total === 0 ? "No results." : `Showing ${firstRow}–${lastRow} of ${pageData.total} results.`;

  const handleClear = () => {
    setSearchInput("");
    writeUrlReplace({ q: "", comp: "", sub: "", classFilter: "" });
  };

  const subcategoryOptions = compartment ? COMPARTMENT_TAXONOMY[compartment] : [];

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
            Search by ID or name, or filter by compartment, subcategory, and equipment class. {statusCopy}
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex gap-4 items-end">
            <div className="flex-1 relative">
              <Input
                placeholder="Enter item ID or name..."
                value={searchInput}
                onChange={(e) => setSearchInput(e.target.value)}
                aria-label="Search items"
              />
              {fetching && (
                <Loader2 className="absolute right-3 top-1/2 -translate-y-1/2 h-4 w-4 animate-spin text-muted-foreground" />
              )}
            </div>
            <Button variant="outline" onClick={handleClear}>
              Clear
            </Button>
          </div>

          <div className="flex flex-wrap gap-3 items-center">
            <div className="flex items-center gap-2">
              <span className="text-sm text-muted-foreground">Compartment</span>
              <Select value={compartment || ANY_VALUE} onValueChange={onCompartmentChange}>
                <SelectTrigger className="w-[180px]" aria-label="Compartment">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value={ANY_VALUE}>Any</SelectItem>
                  {COMPARTMENT_OPTIONS.map((c) => (
                    <SelectItem key={c} value={c}>{COMPARTMENT_LABELS[c]}</SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

            {compartment && (
              <div className="flex items-center gap-2">
                <span className="text-sm text-muted-foreground">Subcategory</span>
                <Select value={subcategory || ANY_VALUE} onValueChange={onSubcategoryChange}>
                  <SelectTrigger className="w-[200px]" aria-label="Subcategory">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value={ANY_VALUE}>Any</SelectItem>
                    {subcategoryOptions.map((sub) => (
                      <SelectItem key={sub} value={sub}>{subcategoryLabel(sub)}</SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
            )}

            {compartment === "equipment" && (
              <div className="flex flex-wrap items-center gap-2">
                {CLASS_OPTIONS.map((klass) => (
                  <Button
                    key={klass}
                    type="button"
                    size="sm"
                    variant={classSelected.has(klass) && !allClasses ? "default" : "outline"}
                    disabled={allClasses}
                    onClick={() => onToggleClass(klass)}
                    aria-pressed={classSelected.has(klass) && !allClasses}
                  >
                    {klass.charAt(0).toUpperCase() + klass.slice(1)}
                  </Button>
                ))}
                <Button
                  type="button"
                  size="sm"
                  variant={allClasses ? "default" : "outline"}
                  onClick={onToggleAllClasses}
                  aria-pressed={allClasses}
                >
                  All Classes
                </Button>
              </div>
            )}
          </div>
        </CardContent>
      </Card>

      {showResults && (
        <Card className="flex-1 min-h-0 flex flex-col">
          <CardHeader className="shrink-0">
            <CardTitle>
              Results
              {pageData.total > 0 && (
                <span className="ml-2 text-muted-foreground font-normal">
                  ({pageData.total} {pageData.total === 1 ? "item" : "items"})
                </span>
              )}
            </CardTitle>
          </CardHeader>
          <CardContent className="flex-1 min-h-0 flex flex-col">
            {items.length === 0 ? (
              <div className="text-center py-8 text-muted-foreground">
                {fetching ? "Loading…" : "No items found matching your search criteria."}
              </div>
            ) : (
              <div className="rounded-md border flex-1 min-h-0 overflow-auto">
                <Table>
                  <TableHeader className="sticky top-0 bg-background z-10">
                    <TableRow>
                      <TableHead className="w-10">Icon</TableHead>
                      <TableHead>Name</TableHead>
                      <TableHead>Compartment</TableHead>
                      <TableHead>Subcategory</TableHead>
                      <TableHead>Type</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {items.map((item) => {
                      const iconUrl = activeTenant ? getAssetIconUrl(
                        activeTenant.id,
                        activeTenant.attributes.region,
                        activeTenant.attributes.majorVersion,
                        activeTenant.attributes.minorVersion,
                        'item',
                        parseInt(item.id),
                      ) : '';
                      return (
                        <TableRow key={item.id}>
                          <TableCell>
                            {iconUrl ? (
                              <img
                                src={iconUrl}
                                alt={item.name}
                                width={32}
                                height={32}
                                className="object-contain"
                              />
                            ) : (
                              <Package className="h-8 w-8 text-muted-foreground" />
                            )}
                          </TableCell>
                          <TableCell>
                            <TooltipProvider>
                              <Tooltip>
                                <TooltipTrigger asChild>
                                  <Link to={`/items/${item.id}`}>
                                    <Badge variant="secondary">{item.name}</Badge>
                                  </Link>
                                </TooltipTrigger>
                                <TooltipContent copyable>
                                  <p>{item.id}</p>
                                </TooltipContent>
                              </Tooltip>
                            </TooltipProvider>
                          </TableCell>
                          <TableCell>
                            <Badge variant="secondary" className={getCompartmentBadgeVariant(item.compartment)}>
                              {COMPARTMENT_LABELS[item.compartment]}
                            </Badge>
                          </TableCell>
                          <TableCell>
                            <Badge variant="secondary">
                              {item.subcategory ? subcategoryLabel(item.subcategory) : "—"}
                            </Badge>
                          </TableCell>
                          <TableCell>
                            <Badge variant="secondary">{item.type}</Badge>
                          </TableCell>
                        </TableRow>
                      );
                    })}
                  </TableBody>
                </Table>
              </div>
            )}
            {items.length > 0 && (
              <Pager
                page={pageData.pageNumber}
                lastPage={pageData.lastPage}
                total={pageData.total}
                pageSize={pageData.pageSize}
                onPageChange={writeUrlPush}
              />
            )}
          </CardContent>
        </Card>
      )}
    </div>
  );
}
